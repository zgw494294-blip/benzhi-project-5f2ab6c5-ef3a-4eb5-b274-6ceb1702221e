package dashboard_leader_cancel_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"citytree/internal/application"
	"citytree/internal/domain"
)

type controlledStore struct {
	application.Store
	leaderEntered chan struct{}
	calls         atomic.Int32
}

func (s *controlledStore) ListBatches(ctx context.Context) ([]domain.InspectionBatch, error) {
	if s.calls.Add(1) == 1 {
		close(s.leaderEntered)
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return []domain.InspectionBatch{}, nil
}

type observedContext struct {
	context.Context
	doneObserved chan struct{}
	once         sync.Once
}

func (c *observedContext) Done() <-chan struct{} {
	c.once.Do(func() { close(c.doneObserved) })
	return c.Context.Done()
}

func TestHealthyDashboardFollowerSurvivesLeaderCancellation(t *testing.T) {
	store := &controlledStore{leaderEntered: make(chan struct{})}
	service := application.NewService(store)
	leaderCtx, cancelLeader := context.WithCancel(context.Background())
	leaderResult := make(chan error, 1)
	go func() {
		_, err := service.Dashboard(leaderCtx)
		leaderResult <- err
	}()
	<-store.leaderEntered

	followerCtx := &observedContext{
		Context:      context.Background(),
		doneObserved: make(chan struct{}),
	}
	followerResult := make(chan error, 1)
	go func() {
		_, err := service.Dashboard(followerCtx)
		followerResult <- err
	}()
	<-followerCtx.doneObserved

	cancelLeader()
	if err := <-leaderResult; !errors.Is(err, context.Canceled) {
		t.Fatalf("leader error = %v, want context.Canceled", err)
	}
	if err := <-followerResult; err != nil {
		t.Fatalf("healthy follower inherited canceled leader result: %v", err)
	}
	if calls := store.calls.Load(); calls != 2 {
		t.Fatalf("ListBatches calls = %d, want follower retry after canceled shared load", calls)
	}
}
