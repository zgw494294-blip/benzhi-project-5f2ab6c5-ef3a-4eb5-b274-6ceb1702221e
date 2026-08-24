package persistence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"citytree/internal/application"
	"citytree/internal/domain"
)

func NewAuditEvent(previous application.AuditEvent, entityType, entityID, action, actor, detail string, now time.Time) application.AuditEvent {
	v := application.AuditEvent{ID: domain.NewID("audit"), EntityType: entityType, EntityID: entityID, Action: action, Actor: domain.NormalizeText(actor), Detail: detail, PrevHash: previous.Hash, CreatedAt: now.UTC()}
	v.Hash = AuditHash(v)
	return v
}

func AuditHash(v application.AuditEvent) string {
	data := strings.Join([]string{v.ID, v.EntityType, v.EntityID, v.Action, v.Actor, v.Detail, v.PrevHash, timeText(v.CreatedAt)}, "\x1f")
	sum := sha256.Sum256([]byte(data))
	return hex.EncodeToString(sum[:])
}

func (r *repository) AppendAudit(ctx context.Context, v application.AuditEvent) error {
	if v.Hash == "" || AuditHash(v) != v.Hash {
		return domain.ValidationErrors{{Field: "audit.hash", Message: "审计摘要不匹配"}}
	}
	_, err := r.q.ExecContext(ctx, `INSERT INTO audit_events(id,entity_type,entity_id,action,actor,detail,prev_hash,event_hash,created_at) VALUES(?,?,?,?,?,?,?,?,?)`, v.ID, v.EntityType, v.EntityID, v.Action, v.Actor, v.Detail, v.PrevHash, v.Hash, timeText(v.CreatedAt))
	return translate(err)
}

func (r *repository) LastAudit(ctx context.Context) (application.AuditEvent, error) {
	var v application.AuditEvent
	var created string
	err := r.q.QueryRowContext(ctx, `SELECT id,entity_type,entity_id,action,actor,detail,prev_hash,event_hash,created_at FROM audit_events ORDER BY sequence DESC LIMIT 1`).Scan(&v.ID, &v.EntityType, &v.EntityID, &v.Action, &v.Actor, &v.Detail, &v.PrevHash, &v.Hash, &created)
	if err != nil {
		return v, translate(err)
	}
	v.CreatedAt, err = parseTime(created)
	return v, err
}
