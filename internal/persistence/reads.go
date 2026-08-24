package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"citytree/internal/application"
	"citytree/internal/domain"
)

func (r *repository) GetBatch(ctx context.Context, id string) (domain.InspectionBatch, error) {
	return scanBatch(r.q.QueryRowContext(ctx, `SELECT id,name,area,status,version,created_at FROM inspection_batches WHERE id=?`, id))
}

func (r *repository) ListBatches(ctx context.Context) ([]domain.InspectionBatch, error) {
	rows, err := r.q.QueryContext(ctx, `SELECT id,name,area,status,version,created_at FROM inspection_batches ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.InspectionBatch
	for rows.Next() {
		v, err := scanBatch(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}

func (r *repository) GetTree(ctx context.Context, id string) (domain.TreeAsset, error) {
	return scanTree(r.q.QueryRowContext(ctx, treeSelect+` WHERE id=?`, id))
}

func (r *repository) ListTreesByBatch(ctx context.Context, id string) ([]domain.TreeAsset, error) {
	rows, err := r.q.QueryContext(ctx, treeSelect+` WHERE batch_id=? ORDER BY created_at`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.TreeAsset
	for rows.Next() {
		v, err := scanTree(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}

func (r *repository) LatestEvidence(ctx context.Context, treeID string) (domain.InspectionEvidence, error) {
	return scanEvidence(r.q.QueryRowContext(ctx, evidenceSelect+` WHERE tree_id=? ORDER BY inspected_at DESC LIMIT 1`, treeID))
}

func (r *repository) ListEvidenceByBatch(ctx context.Context, batchID string) ([]domain.InspectionEvidence, error) {
	rows, err := r.q.QueryContext(ctx, evidenceSelect+` WHERE batch_id=? ORDER BY inspected_at DESC`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.InspectionEvidence
	for rows.Next() {
		v, err := scanEvidence(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}

func (r *repository) ActiveTask(ctx context.Context, treeID string) (domain.RemediationTask, error) {
	return scanTask(r.q.QueryRowContext(ctx, taskSelect+` WHERE tree_id=? ORDER BY rowid DESC LIMIT 1`, treeID))
}

func (r *repository) ListTasksByBatch(ctx context.Context, batchID string) ([]domain.RemediationTask, error) {
	rows, err := r.q.QueryContext(ctx, taskSelect+` JOIN tree_assets t ON t.id=remediation_tasks.tree_id WHERE t.batch_id=? ORDER BY due_date`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.RemediationTask
	for rows.Next() {
		v, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}

func (r *repository) CertificateByTree(ctx context.Context, treeID string) (domain.CareCertificate, error) {
	return scanCertificate(r.q.QueryRowContext(ctx, certificateSelect+` WHERE tree_id=?`, treeID))
}

func (r *repository) ListCertificatesByBatch(ctx context.Context, batchID string) ([]domain.CareCertificate, error) {
	rows, err := r.q.QueryContext(ctx, certificateSelect+` JOIN tree_assets t ON t.id=care_certificates.tree_id WHERE t.batch_id=? ORDER BY issued_at DESC`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.CareCertificate
	for rows.Next() {
		v, err := scanCertificate(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}

func (r *repository) GetIdempotency(ctx context.Context, scope, key string) (application.IdempotencyRecord, error) {
	var v application.IdempotencyRecord
	var created string
	err := r.q.QueryRowContext(ctx, `SELECT scope,command_key,request_hash,response,created_at FROM idempotency_records WHERE scope=? AND command_key=?`, scope, key).Scan(&v.Scope, &v.Key, &v.RequestHash, &v.Response, &created)
	if err != nil {
		return v, translate(err)
	}
	v.CreatedAt, err = parseTime(created)
	return v, err
}

const treeSelect = `SELECT id,batch_id,name,road_location,species,diameter_cm,responsibility_area,current_status,version,created_at FROM tree_assets`
const evidenceSelect = `SELECT id,tree_id,batch_id,inspected_at,photo_digest,leaf_condition,trunk_defect,pest_signs,notes,submitted_by,digest,version FROM inspection_evidence`
const taskSelect = `SELECT remediation_tasks.id,remediation_tasks.tree_id,risk_level,trigger_rule,recommended_action,assignee,due_date,status,completion_note,completed_at,version FROM remediation_tasks`
const certificateSelect = `SELECT care_certificates.id,care_certificates.tree_id,rechecked_at,metrics_json,result,inspector,digest,issued_at FROM care_certificates`

type scanner interface{ Scan(...any) error }

func scanBatch(s scanner) (domain.InspectionBatch, error) {
	var v domain.InspectionBatch
	var created string
	err := s.Scan(&v.ID, &v.Name, &v.Area, &v.Status, &v.Version, &created)
	if err != nil {
		return v, translate(err)
	}
	v.CreatedAt, err = parseTime(created)
	return v, err
}

func scanTree(s scanner) (domain.TreeAsset, error) {
	var v domain.TreeAsset
	var created string
	err := s.Scan(&v.ID, &v.BatchID, &v.Name, &v.RoadLocation, &v.Species, &v.DiameterCM, &v.Responsibility, &v.CurrentStatus, &v.Version, &created)
	if err != nil {
		return v, translate(err)
	}
	v.CreatedAt, err = parseTime(created)
	return v, err
}

func scanEvidence(s scanner) (domain.InspectionEvidence, error) {
	var v domain.InspectionEvidence
	var inspected string
	err := s.Scan(&v.ID, &v.TreeID, &v.BatchID, &inspected, &v.PhotoDigest, &v.LeafCondition, &v.TrunkDefect, &v.PestSigns, &v.Notes, &v.SubmittedBy, &v.Digest, &v.Version)
	if err != nil {
		return v, translate(err)
	}
	v.InspectedAt, err = parseTime(inspected)
	return v, err
}

func scanTask(s scanner) (domain.RemediationTask, error) {
	var v domain.RemediationTask
	var due string
	var completed sql.NullString
	err := s.Scan(&v.ID, &v.TreeID, &v.RiskLevel, &v.TriggerRule, &v.Recommended, &v.Assignee, &due, &v.Status, &v.CompletionNote, &completed, &v.Version)
	if err != nil {
		return v, translate(err)
	}
	v.DueDate, err = parseTime(due)
	if err == nil && completed.Valid {
		var t time.Time
		t, err = parseTime(completed.String)
		v.CompletedAt = &t
	}
	return v, err
}

func scanCertificate(s scanner) (domain.CareCertificate, error) {
	var v domain.CareCertificate
	var rechecked, issued string
	var metrics []byte
	err := s.Scan(&v.ID, &v.TreeID, &rechecked, &metrics, &v.Result, &v.Inspector, &v.Digest, &issued)
	if err != nil {
		return v, translate(err)
	}
	if err = json.Unmarshal(metrics, &v.Metrics); err != nil {
		return v, err
	}
	v.RecheckedAt, err = parseTime(rechecked)
	if err == nil {
		v.IssuedAt, err = parseTime(issued)
	}
	return v, err
}

func timeText(v time.Time) string { return v.UTC().Format(time.RFC3339Nano) }
func nullableTime(v *time.Time) any {
	if v == nil {
		return nil
	}
	return timeText(*v)
}
func parseTime(v string) (time.Time, error) { return time.Parse(time.RFC3339Nano, v) }
