package persistence

import (
	"context"
	"database/sql"

	"citytree/internal/application"
	"citytree/internal/domain"
)

type querier interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type repository struct{ q querier }

func (s *SQLiteStore) InsertBatch(ctx context.Context, v domain.InspectionBatch) error {
	return s.repo().InsertBatch(ctx, v)
}
func (s *SQLiteStore) GetBatch(ctx context.Context, id string) (domain.InspectionBatch, error) {
	return s.repo().GetBatch(ctx, id)
}
func (s *SQLiteStore) ListBatches(ctx context.Context) ([]domain.InspectionBatch, error) {
	return s.repo().ListBatches(ctx)
}
func (s *SQLiteStore) UpdateBatch(ctx context.Context, v domain.InspectionBatch, expected int64) error {
	return s.repo().UpdateBatch(ctx, v, expected)
}
func (s *SQLiteStore) InsertTree(ctx context.Context, v domain.TreeAsset) error {
	return s.repo().InsertTree(ctx, v)
}
func (s *SQLiteStore) GetTree(ctx context.Context, id string) (domain.TreeAsset, error) {
	return s.repo().GetTree(ctx, id)
}
func (s *SQLiteStore) ListTreesByBatch(ctx context.Context, id string) ([]domain.TreeAsset, error) {
	return s.repo().ListTreesByBatch(ctx, id)
}
func (s *SQLiteStore) UpdateTree(ctx context.Context, v domain.TreeAsset, expected int64) error {
	return s.repo().UpdateTree(ctx, v, expected)
}
func (s *SQLiteStore) InsertEvidence(ctx context.Context, v domain.InspectionEvidence) error {
	return s.repo().InsertEvidence(ctx, v)
}
func (s *SQLiteStore) LatestEvidence(ctx context.Context, id string) (domain.InspectionEvidence, error) {
	return s.repo().LatestEvidence(ctx, id)
}
func (s *SQLiteStore) ListEvidenceByBatch(ctx context.Context, id string) ([]domain.InspectionEvidence, error) {
	return s.repo().ListEvidenceByBatch(ctx, id)
}
func (s *SQLiteStore) InsertTask(ctx context.Context, v domain.RemediationTask) error {
	return s.repo().InsertTask(ctx, v)
}
func (s *SQLiteStore) ActiveTask(ctx context.Context, id string) (domain.RemediationTask, error) {
	return s.repo().ActiveTask(ctx, id)
}
func (s *SQLiteStore) UpdateTask(ctx context.Context, v domain.RemediationTask, expected int64) error {
	return s.repo().UpdateTask(ctx, v, expected)
}
func (s *SQLiteStore) ListTasksByBatch(ctx context.Context, id string) ([]domain.RemediationTask, error) {
	return s.repo().ListTasksByBatch(ctx, id)
}
func (s *SQLiteStore) InsertCertificate(ctx context.Context, v domain.CareCertificate) error {
	return s.repo().InsertCertificate(ctx, v)
}
func (s *SQLiteStore) CertificateByTree(ctx context.Context, id string) (domain.CareCertificate, error) {
	return s.repo().CertificateByTree(ctx, id)
}
func (s *SQLiteStore) ListCertificatesByBatch(ctx context.Context, id string) ([]domain.CareCertificate, error) {
	return s.repo().ListCertificatesByBatch(ctx, id)
}
func (s *SQLiteStore) GetIdempotency(ctx context.Context, scope, key string) (application.IdempotencyRecord, error) {
	return s.repo().GetIdempotency(ctx, scope, key)
}
func (s *SQLiteStore) InsertIdempotency(ctx context.Context, v application.IdempotencyRecord) error {
	return s.repo().InsertIdempotency(ctx, v)
}
func (s *SQLiteStore) AppendAudit(ctx context.Context, v application.AuditEvent) error {
	return s.repo().AppendAudit(ctx, v)
}
func (s *SQLiteStore) LastAudit(ctx context.Context) (application.AuditEvent, error) {
	return s.repo().LastAudit(ctx)
}
