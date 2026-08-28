package add_tree_close_race_test

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

type transactionGateStore struct {
	application.Store
	entered chan struct{}
	release chan struct{}
}

func (s *transactionGateStore) WithinTx(ctx context.Context, fn func(application.Repository) error) error {
	close(s.entered)
	select {
	case <-s.release:
		return s.Store.WithinTx(ctx, fn)
	case <-ctx.Done():
		return ctx.Err()
	}
}

type addTreeOutcome struct {
	tree domain.TreeAsset
	err  error
}

func TestAddTreeCannotCommitAfterConcurrentBatchClosure(t *testing.T) {
	ctx := context.Background()
	store, err := persistence.Open(ctx, filepath.Join(t.TempDir(), "citytree.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	service := application.NewService(store)
	now := time.Now().UTC().Add(-time.Minute)

	created, err := service.CreateBatch(ctx, createBatchCommand(), "create-race-batch")
	if err != nil {
		t.Fatalf("create batch: %v", err)
	}
	evidence, err := service.SubmitEvidence(ctx, application.SubmitEvidenceCommand{
		TreeID: created.Tree.ID, ExpectedVersion: created.Tree.Version,
		InspectedAt: now, PhotoDigest: "sha256:controlled-race-evidence",
		LeafCondition: domain.ConditionSevere, TrunkDefect: domain.ConditionSevere,
		PestSigns: domain.ConditionSevere, Notes: "并发关闭测试证据", SubmittedBy: "巡检员",
	}, "evidence-race-batch")
	if err != nil {
		t.Fatalf("submit evidence: %v", err)
	}
	assessed, err := service.Assess(ctx, application.AssessCommand{
		TreeID: created.Tree.ID, ExpectedVersion: evidence.Tree.Version,
		Assignee: "养护员", DueDate: time.Now().UTC().Add(24 * time.Hour), Operator: "负责人",
	}, "assess-race-batch")
	if err != nil {
		t.Fatalf("assess tree: %v", err)
	}
	completed, err := service.CompleteRemediation(ctx, application.CompleteRemediationCommand{
		TreeID: created.Tree.ID, ExpectedTreeVersion: assessed.Tree.Version,
		ExpectedTaskVersion: assessed.Task.Version, CompletionNote: "已完成加固与消杀",
		CompletedAt: time.Now().UTC(), Operator: "养护员",
	}, "complete-race-batch")
	if err != nil {
		t.Fatalf("complete remediation: %v", err)
	}

	gate := &transactionGateStore{Store: store, entered: make(chan struct{}), release: make(chan struct{})}
	addService := application.NewService(gate)
	addDone := make(chan addTreeOutcome, 1)
	go func() {
		tree, addErr := addService.AddTree(ctx, application.AddTreeCommand{
			BatchID: created.Batch.ID, Name: "并发新增树", RoadLocation: "测试路 2 号",
			Species: "国槐", DiameterCM: 18, ResponsibilityArea: "二区", Operator: "登记员",
		}, "add-during-close")
		addDone <- addTreeOutcome{tree: tree, err: addErr}
	}()

	<-gate.entered
	_, err = service.Recheck(ctx, application.RecheckCommand{
		TreeID: created.Tree.ID, ExpectedTreeVersion: completed.Tree.Version,
		ExpectedTaskVersion: completed.Task.Version, RecheckedAt: time.Now().UTC(),
		Metrics: map[string]string{"树冠稳定": "是", "树干安全": "是", "病虫受控": "是"},
		Result:  "通过", Inspector: "复验员",
	}, "recheck-close-batch")
	if err != nil {
		t.Fatalf("close batch by recheck: %v", err)
	}
	close(gate.release)
	outcome := <-addDone
	if !errors.Is(outcome.err, domain.ErrTransition) {
		t.Fatalf("completed batch accepted concurrent tree %q: add error = %v", outcome.tree.ID, outcome.err)
	}

	view, err := service.Batch(ctx, created.Batch.ID)
	if err != nil {
		t.Fatalf("load final batch: %v", err)
	}
	if view.Batch.Status != domain.BatchCompleted || view.Total != view.Closed {
		t.Fatalf("closed batch invariant broken: status=%s total=%d closed=%d", view.Batch.Status, view.Total, view.Closed)
	}
}

func createBatchCommand() application.CreateBatchCommand {
	var cmd application.CreateBatchCommand
	cmd.Name = "并发关闭批次"
	cmd.Area = "中心城区"
	cmd.Operator = "登记员"
	cmd.Tree.Name = "待关闭树"
	cmd.Tree.RoadLocation = "测试路 1 号"
	cmd.Tree.Species = "悬铃木"
	cmd.Tree.DiameterCM = 32
	cmd.Tree.ResponsibilityArea = "一区"
	return cmd
}
