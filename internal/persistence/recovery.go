package persistence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"citytree/internal/application"
	"citytree/internal/domain"
)

type checkpoint struct {
	AuditID string `json:"auditID"`
	Hash    string `json:"hash"`
	Seal    string `json:"seal"`
}

// Checkpoint writes the integrity checkpoint for the current database state
// using the store's default (non-transactional) connection.
func (s *SQLiteStore) Checkpoint(ctx context.Context) error {
	return s.writeCheckpoint(ctx, s.repo())
}

// writeCheckpoint writes the integrity checkpoint reflecting the last audit
// event visible to repo. When called with a transaction's repository, the
// checkpoint captures uncommitted audit data so it can be persisted before the
// transaction commits; this guarantees that a checkpoint write failure can roll
// the transaction back without leaving visible business data behind.
func (s *SQLiteStore) writeCheckpoint(ctx context.Context, repo application.Repository) error {
	if s.path == ":memory:" {
		return nil
	}
	last, err := repo.LastAudit(ctx)
	if errors.Is(err, domain.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	c := checkpoint{AuditID: last.ID, Hash: last.Hash}
	c.Seal = checkpointSeal(c.AuditID, c.Hash)
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.checkpointPath), ".integrity-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err = tmp.Write(data); err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err = os.Rename(name, s.checkpointPath); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(s.checkpointPath))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

// saveCheckpoint reads the current checkpoint file content so it can be
// restored by restoreCheckpoint if a transaction commit fails after a new
// checkpoint has been written.
func (s *SQLiteStore) saveCheckpoint() ([]byte, bool) {
	if s.path == ":memory:" {
		return nil, false
	}
	prev, err := os.ReadFile(s.checkpointPath)
	if err != nil {
		return nil, false
	}
	return prev, true
}

// restoreCheckpoint reverts the checkpoint file to prev when a transaction
// commit fails after writeCheckpoint has already replaced it. If no prior
// checkpoint existed the new file is removed so VerifyIntegrity treats the
// state as unchecked rather than pointing at rolled-back audit data.
func (s *SQLiteStore) restoreCheckpoint(prev []byte, hadPrev bool) {
	if s.path == ":memory:" {
		return
	}
	if !hadPrev {
		_ = os.Remove(s.checkpointPath)
		return
	}
	_ = os.WriteFile(s.checkpointPath, prev, 0o644)
	if dir, err := os.Open(filepath.Dir(s.checkpointPath)); err == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}
}

func (s *SQLiteStore) VerifyIntegrity(ctx context.Context) error {
	if s.path == ":memory:" {
		return nil
	}
	data, err := os.ReadFile(s.checkpointPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var c checkpoint
	if err = json.Unmarshal(data, &c); err != nil {
		return fmt.Errorf("恢复检查点损坏: %w", err)
	}
	if c.Seal != checkpointSeal(c.AuditID, c.Hash) {
		return fmt.Errorf("恢复检查点校验失败")
	}
	last, err := s.LastAudit(ctx)
	if err != nil {
		return fmt.Errorf("读取审计尾记录: %w", err)
	}
	if last.ID != c.AuditID || last.Hash != c.Hash {
		return fmt.Errorf("数据库与恢复检查点不一致")
	}
	return nil
}

func checkpointSeal(id, hash string) string {
	sum := sha256.Sum256([]byte(id + "\x1f" + hash))
	return hex.EncodeToString(sum[:])
}
