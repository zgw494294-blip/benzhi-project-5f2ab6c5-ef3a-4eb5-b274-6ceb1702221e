package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"citytree/internal/domain"
)

type Service struct {
	store         Store
	now           func() time.Time
	dashboardMu   sync.Mutex
	dashboardCall *dashboardCall
}

// maxDashboardRetries bounds how many times a healthy follower may retry the
// dashboard query after joining an in-flight call whose leader was cancelled
// before it could finish loading. This keeps retry loops finite even under
// pathological cancellation storms while still letting a single cancelled
// leader not poison healthy concurrent requests.
const maxDashboardRetries = 8

type dashboardCall struct {
	done chan struct{}
	view DashboardView
	err  error
}

func NewService(store Store) *Service {
	return &Service{store: store, now: time.Now}
}

func (s *Service) Store() Store { return s.store }

func requestDigest(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func executeIdempotent[T any](ctx context.Context, s *Service, scope, key string, request any, command func(Repository) (T, error)) (T, error) {
	var zero T
	if !domain.ValidIdempotencyKey(key) {
		return zero, domain.ValidationErrors{{Field: "idempotencyKey", Message: "长度必须在 8 到 128 之间"}}
	}
	hash, err := requestDigest(request)
	if err != nil {
		return zero, err
	}
	if cached, err := s.store.GetIdempotency(ctx, scope, key); err == nil {
		if cached.RequestHash != hash {
			return zero, domain.ErrIdempotency
		}
		var value T
		if err = json.Unmarshal(cached.Response, &value); err != nil {
			return zero, err
		}
		return value, nil
	} else if !errors.Is(err, domain.ErrNotFound) {
		return zero, err
	}
	var result T
	err = s.store.WithinTx(ctx, func(repo Repository) error {
		if cached, err := repo.GetIdempotency(ctx, scope, key); err == nil {
			if cached.RequestHash != hash {
				return domain.ErrIdempotency
			}
			return json.Unmarshal(cached.Response, &result)
		} else if !errors.Is(err, domain.ErrNotFound) {
			return err
		}
		value, err := command(repo)
		if err != nil {
			return err
		}
		result = value
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		return repo.InsertIdempotency(ctx, IdempotencyRecord{Scope: scope, Key: key, RequestHash: hash, Response: data, CreatedAt: s.now().UTC()})
	})
	return result, err
}

func appendAudit(ctx context.Context, repo Repository, entityType, entityID, action, actor, detail string, now time.Time) error {
	previous, err := repo.LastAudit(ctx)
	if errors.Is(err, domain.ErrNotFound) {
		previous = AuditEvent{}
	} else if err != nil {
		return err
	}
	event := AuditEvent{ID: domain.NewID("audit"), EntityType: entityType, EntityID: entityID, Action: action, Actor: domain.NormalizeText(actor), Detail: detail, PrevHash: previous.Hash, CreatedAt: now.UTC()}
	event.Hash = auditHash(event)
	return repo.AppendAudit(ctx, event)
}

func auditHash(v AuditEvent) string {
	data := strings.Join([]string{v.ID, v.EntityType, v.EntityID, v.Action, v.Actor, v.Detail, v.PrevHash, v.CreatedAt.UTC().Format(time.RFC3339Nano)}, "\x1f")
	sum := sha256.Sum256([]byte(data))
	return hex.EncodeToString(sum[:])
}

func checkExpected(actual, expected int64) error {
	if expected <= 0 {
		return domain.ValidationErrors{{Field: "expectedVersion", Message: "必须大于 0"}}
	}
	if actual != expected {
		return fmt.Errorf("%w: 当前版本为 %d", domain.ErrConflict, actual)
	}
	return nil
}
