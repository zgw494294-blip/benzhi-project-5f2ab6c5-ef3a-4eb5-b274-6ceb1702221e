package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"citytree/internal/application"
	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db             *sql.DB
	path           string
	checkpointPath string
}

func Open(ctx context.Context, path string) (*SQLiteStore, error) {
	if path == "" {
		return nil, fmt.Errorf("数据库路径不能为空")
	}
	if path != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, fmt.Errorf("创建数据目录: %w", err)
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("打开 SQLite: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)
	store := &SQLiteStore{db: db, path: path, checkpointPath: path + ".integrity"}
	if err := store.migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}
	if err := store.VerifyIntegrity(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *SQLiteStore) Close() error { return s.db.Close() }

func (s *SQLiteStore) WithinTx(ctx context.Context, fn func(application.Repository) error) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	repo := &repository{q: tx}
	if err = fn(repo); err != nil {
		_ = tx.Rollback()
		return err
	}
	// Persist the integrity checkpoint for the audit tail written inside the
	// transaction before committing. If the checkpoint cannot be written
	// atomically (for example because the integrity file is unavailable), the
	// transaction is rolled back so no visible business commit is left behind:
	// callers see the failure, idempotent retries do not observe a cached
	// result, and the recovery checkpoint continues to match the database.
	if err = s.writeCheckpoint(ctx, repo); err != nil {
		_ = tx.Rollback()
		return err
	}
	// Capture the prior checkpoint so it can be restored if the commit fails
	// after the new checkpoint has already replaced the file. Without this,
	// a post-checkpoint commit failure would leave the checkpoint pointing at
	// audit data that never became durable.
	prev, hadPrev := s.saveCheckpoint()
	if err = tx.Commit(); err != nil {
		s.restoreCheckpoint(prev, hadPrev)
		return err
	}
	return nil
}

func (s *SQLiteStore) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return s.db.PingContext(ctx)
}

func (s *SQLiteStore) repo() *repository { return &repository{q: s.db} }
