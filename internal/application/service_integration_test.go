package application_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"citytree/internal/application"
	"citytree/internal/domain"
	"citytree/internal/persistence"
)

func TestWorkflowAndIdempotency(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")
	store, err := persistence.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := application.NewService(store)
	cmd := application.CreateBatchCommand{Name: "测试批次", Area: "东区", Operator: "管理员"}
	cmd.Tree.Name = "一号树"
	cmd.Tree.RoadLocation = "园林路 1 号"
	cmd.Tree.Species = "香樟"
	cmd.Tree.DiameterCM = 30
	cmd.Tree.ResponsibilityArea = "一组"
	first, err := service.CreateBatch(ctx, cmd, "create-key-0001")
	if err != nil {
		t.Fatal(err)
	}
	again, err := service.CreateBatch(ctx, cmd, "create-key-0001")
	if err != nil {
		t.Fatal(err)
	}
	if first.Batch.ID != again.Batch.ID {
		t.Fatal("幂等重试创建了不同批次")
	}
	changed := cmd
	changed.Area = "西区"
	if _, err = service.CreateBatch(ctx, changed, "create-key-0001"); !errors.Is(err, domain.ErrIdempotency) {
		t.Fatalf("expected idempotency conflict, got %v", err)
	}
	now := time.Now().Add(-time.Minute)
	evidence, err := service.SubmitEvidence(ctx, application.SubmitEvidenceCommand{TreeID: first.Tree.ID, ExpectedVersion: first.Tree.Version, InspectedAt: now, PhotoDigest: "sha256:test", LeafCondition: domain.ConditionMild, TrunkDefect: domain.ConditionHealthy, PestSigns: domain.ConditionHealthy, Notes: "长势基本健康", SubmittedBy: "巡检员"}, "evidence-key-01")
	if err != nil {
		t.Fatal(err)
	}
	assessed, err := service.Assess(ctx, application.AssessCommand{TreeID: first.Tree.ID, ExpectedVersion: evidence.Tree.Version, Assignee: "", DueDate: now.Add(time.Hour), Operator: "负责人"}, "assess-key-0001")
	if err != nil {
		t.Fatal(err)
	}
	if assessed.Assessment.Level != domain.RiskLow || assessed.Tree.CurrentStatus != domain.TreeClosed {
		t.Fatalf("unexpected low-risk result: %+v", assessed)
	}
	view, err := service.Batch(ctx, first.Batch.ID)
	if err != nil {
		t.Fatal(err)
	}
	if view.Batch.Status != domain.BatchCompleted {
		t.Fatalf("batch status=%s", view.Batch.Status)
	}
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = persistence.Open(ctx, path)
	if err != nil {
		t.Fatalf("重启恢复失败: %v", err)
	}
	defer store.Close()
	if err = store.VerifyIntegrity(ctx); err != nil {
		t.Fatalf("完整性校验失败: %v", err)
	}
}
