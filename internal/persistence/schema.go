package persistence

import (
	"context"
	"fmt"
)

const schemaVersion = 1

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL)`,
	`CREATE TABLE IF NOT EXISTS inspection_batches (
        id TEXT PRIMARY KEY, name TEXT NOT NULL, area TEXT NOT NULL, status TEXT NOT NULL,
        version INTEGER NOT NULL, created_at TEXT NOT NULL)`,
	`CREATE TABLE IF NOT EXISTS tree_assets (
        id TEXT PRIMARY KEY, batch_id TEXT NOT NULL REFERENCES inspection_batches(id), name TEXT NOT NULL,
        road_location TEXT NOT NULL, species TEXT NOT NULL, diameter_cm REAL NOT NULL,
        responsibility_area TEXT NOT NULL, current_status TEXT NOT NULL, version INTEGER NOT NULL,
        created_at TEXT NOT NULL)`,
	`CREATE TABLE IF NOT EXISTS inspection_evidence (
        id TEXT PRIMARY KEY, tree_id TEXT NOT NULL REFERENCES tree_assets(id), batch_id TEXT NOT NULL,
        inspected_at TEXT NOT NULL, photo_digest TEXT NOT NULL, leaf_condition INTEGER NOT NULL,
        trunk_defect INTEGER NOT NULL, pest_signs INTEGER NOT NULL, notes TEXT NOT NULL,
        submitted_by TEXT NOT NULL, digest TEXT NOT NULL UNIQUE, version INTEGER NOT NULL)`,
	`CREATE TABLE IF NOT EXISTS remediation_tasks (
        id TEXT PRIMARY KEY, tree_id TEXT NOT NULL REFERENCES tree_assets(id), risk_level TEXT NOT NULL,
        trigger_rule TEXT NOT NULL, recommended_action TEXT NOT NULL, assignee TEXT NOT NULL,
        due_date TEXT NOT NULL, status TEXT NOT NULL, completion_note TEXT NOT NULL DEFAULT '',
        completed_at TEXT, version INTEGER NOT NULL)`,
	`CREATE TABLE IF NOT EXISTS care_certificates (
        id TEXT PRIMARY KEY, tree_id TEXT NOT NULL UNIQUE REFERENCES tree_assets(id), rechecked_at TEXT NOT NULL,
        metrics_json TEXT NOT NULL, result TEXT NOT NULL, inspector TEXT NOT NULL,
        digest TEXT NOT NULL UNIQUE, issued_at TEXT NOT NULL)`,
	`CREATE TABLE IF NOT EXISTS idempotency_records (
        scope TEXT NOT NULL, command_key TEXT NOT NULL, request_hash TEXT NOT NULL,
        response BLOB NOT NULL, created_at TEXT NOT NULL, PRIMARY KEY(scope, command_key))`,
	`CREATE TABLE IF NOT EXISTS audit_events (
        sequence INTEGER PRIMARY KEY AUTOINCREMENT, id TEXT NOT NULL UNIQUE, entity_type TEXT NOT NULL,
        entity_id TEXT NOT NULL, action TEXT NOT NULL, actor TEXT NOT NULL, detail TEXT NOT NULL,
        prev_hash TEXT NOT NULL, event_hash TEXT NOT NULL UNIQUE, created_at TEXT NOT NULL)`,
	`CREATE INDEX IF NOT EXISTS idx_trees_batch_status ON tree_assets(batch_id, current_status)`,
	`CREATE INDEX IF NOT EXISTS idx_evidence_batch_tree ON inspection_evidence(batch_id, tree_id, inspected_at DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_tasks_tree_status_risk ON remediation_tasks(tree_id, status, risk_level)`,
	`CREATE INDEX IF NOT EXISTS idx_audit_entity ON audit_events(entity_type, entity_id, sequence)`,
}

func (s *SQLiteStore) migrate(ctx context.Context) error {
	pragmas := []string{
		`PRAGMA journal_mode=WAL`,
		`PRAGMA synchronous=FULL`,
		`PRAGMA foreign_keys=ON`,
		`PRAGMA busy_timeout=5000`,
	}
	for _, statement := range pragmas {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("设置 SQLite 参数: %w", err)
		}
	}
	for _, statement := range migrations {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("执行数据库迁移: %w", err)
		}
	}
	_, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES(?, datetime('now'))`, schemaVersion)
	return err
}
