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

	"citytree/internal/domain"
)

type checkpoint struct {
	AuditID string `json:"auditID"`
	Hash    string `json:"hash"`
	Seal    string `json:"seal"`
}

func (s *SQLiteStore) Checkpoint(ctx context.Context) error {
	if s.path == ":memory:" {
		return nil
	}
	last, err := s.LastAudit(ctx)
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
