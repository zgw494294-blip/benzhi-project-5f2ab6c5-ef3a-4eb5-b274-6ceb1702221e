package application

import (
	"context"
	"time"

	"citytree/internal/domain"
)

type IdempotencyRecord struct {
	Scope       string
	Key         string
	RequestHash string
	Response    []byte
	CreatedAt   time.Time
}

type AuditEvent struct {
	ID         string    `json:"id"`
	EntityType string    `json:"entityType"`
	EntityID   string    `json:"entityID"`
	Action     string    `json:"action"`
	Actor      string    `json:"actor"`
	Detail     string    `json:"detail"`
	PrevHash   string    `json:"prevHash"`
	Hash       string    `json:"hash"`
	CreatedAt  time.Time `json:"createdAt"`
}

type Repository interface {
	InsertBatch(context.Context, domain.InspectionBatch) error
	GetBatch(context.Context, string) (domain.InspectionBatch, error)
	ListBatches(context.Context) ([]domain.InspectionBatch, error)
	UpdateBatch(context.Context, domain.InspectionBatch, int64) error

	InsertTree(context.Context, domain.TreeAsset) error
	GetTree(context.Context, string) (domain.TreeAsset, error)
	ListTreesByBatch(context.Context, string) ([]domain.TreeAsset, error)
	UpdateTree(context.Context, domain.TreeAsset, int64) error

	InsertEvidence(context.Context, domain.InspectionEvidence) error
	LatestEvidence(context.Context, string) (domain.InspectionEvidence, error)
	ListEvidenceByBatch(context.Context, string) ([]domain.InspectionEvidence, error)

	InsertTask(context.Context, domain.RemediationTask) error
	ActiveTask(context.Context, string) (domain.RemediationTask, error)
	UpdateTask(context.Context, domain.RemediationTask, int64) error
	ListTasksByBatch(context.Context, string) ([]domain.RemediationTask, error)

	InsertCertificate(context.Context, domain.CareCertificate) error
	CertificateByTree(context.Context, string) (domain.CareCertificate, error)
	ListCertificatesByBatch(context.Context, string) ([]domain.CareCertificate, error)

	GetIdempotency(context.Context, string, string) (IdempotencyRecord, error)
	InsertIdempotency(context.Context, IdempotencyRecord) error
	AppendAudit(context.Context, AuditEvent) error
	LastAudit(context.Context) (AuditEvent, error)
}

type Store interface {
	Repository
	WithinTx(context.Context, func(Repository) error) error
	Checkpoint(context.Context) error
	VerifyIntegrity(context.Context) error
	Close() error
}
