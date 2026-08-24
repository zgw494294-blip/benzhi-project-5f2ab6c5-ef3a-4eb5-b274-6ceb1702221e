package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"citytree/internal/application"
	"citytree/internal/domain"
)

func (r *repository) InsertBatch(ctx context.Context, v domain.InspectionBatch) error {
	_, err := r.q.ExecContext(ctx, `INSERT INTO inspection_batches(id,name,area,status,version,created_at) VALUES(?,?,?,?,?,?)`, v.ID, v.Name, v.Area, v.Status, v.Version, timeText(v.CreatedAt))
	return translate(err)
}

func (r *repository) UpdateBatch(ctx context.Context, v domain.InspectionBatch, expected int64) error {
	result, err := r.q.ExecContext(ctx, `UPDATE inspection_batches SET status=?,version=? WHERE id=? AND version=?`, v.Status, v.Version, v.ID, expected)
	return changed(result, err)
}

func (r *repository) InsertTree(ctx context.Context, v domain.TreeAsset) error {
	_, err := r.q.ExecContext(ctx, `INSERT INTO tree_assets(id,batch_id,name,road_location,species,diameter_cm,responsibility_area,current_status,version,created_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, v.ID, v.BatchID, v.Name, v.RoadLocation, v.Species, v.DiameterCM, v.Responsibility, v.CurrentStatus, v.Version, timeText(v.CreatedAt))
	return translate(err)
}

func (r *repository) UpdateTree(ctx context.Context, v domain.TreeAsset, expected int64) error {
	result, err := r.q.ExecContext(ctx, `UPDATE tree_assets SET current_status=?,version=? WHERE id=? AND version=?`, v.CurrentStatus, v.Version, v.ID, expected)
	return changed(result, err)
}

func (r *repository) InsertEvidence(ctx context.Context, v domain.InspectionEvidence) error {
	_, err := r.q.ExecContext(ctx, `INSERT INTO inspection_evidence(id,tree_id,batch_id,inspected_at,photo_digest,leaf_condition,trunk_defect,pest_signs,notes,submitted_by,digest,version) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, v.ID, v.TreeID, v.BatchID, timeText(v.InspectedAt), v.PhotoDigest, v.LeafCondition, v.TrunkDefect, v.PestSigns, v.Notes, v.SubmittedBy, v.Digest, v.Version)
	return translate(err)
}

func (r *repository) InsertTask(ctx context.Context, v domain.RemediationTask) error {
	_, err := r.q.ExecContext(ctx, `INSERT INTO remediation_tasks(id,tree_id,risk_level,trigger_rule,recommended_action,assignee,due_date,status,completion_note,completed_at,version) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, v.ID, v.TreeID, v.RiskLevel, v.TriggerRule, v.Recommended, v.Assignee, timeText(v.DueDate), v.Status, v.CompletionNote, nullableTime(v.CompletedAt), v.Version)
	return translate(err)
}

func (r *repository) UpdateTask(ctx context.Context, v domain.RemediationTask, expected int64) error {
	result, err := r.q.ExecContext(ctx, `UPDATE remediation_tasks SET status=?,completion_note=?,completed_at=?,version=? WHERE id=? AND version=?`, v.Status, v.CompletionNote, nullableTime(v.CompletedAt), v.Version, v.ID, expected)
	return changed(result, err)
}

func (r *repository) InsertCertificate(ctx context.Context, v domain.CareCertificate) error {
	metrics, err := json.Marshal(v.Metrics)
	if err != nil {
		return err
	}
	_, err = r.q.ExecContext(ctx, `INSERT INTO care_certificates(id,tree_id,rechecked_at,metrics_json,result,inspector,digest,issued_at) VALUES(?,?,?,?,?,?,?,?)`, v.ID, v.TreeID, timeText(v.RecheckedAt), metrics, v.Result, v.Inspector, v.Digest, timeText(v.IssuedAt))
	return translate(err)
}

func (r *repository) InsertIdempotency(ctx context.Context, v application.IdempotencyRecord) error {
	_, err := r.q.ExecContext(ctx, `INSERT INTO idempotency_records(scope,command_key,request_hash,response,created_at) VALUES(?,?,?,?,?)`, v.Scope, v.Key, v.RequestHash, v.Response, timeText(v.CreatedAt))
	return translate(err)
}

func changed(result sql.Result, err error) error {
	if err != nil {
		return translate(err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrConflict
	}
	return nil
}

func translate(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	}
	return err
}
