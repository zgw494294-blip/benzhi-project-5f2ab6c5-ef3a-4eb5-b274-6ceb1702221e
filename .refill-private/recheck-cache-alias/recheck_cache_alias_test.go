package recheck_cache_alias_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"citytree/internal/application"
	"citytree/internal/domain"
	"citytree/internal/persistence"
)

func TestRecheckReplayIsolatedFromReturnedMetricsMutation(t *testing.T) {
	ctx := context.Background()
	store, err := persistence.Open(ctx, filepath.Join(t.TempDir(), "recheck-cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := application.NewService(store)
	now := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)

	create := application.CreateBatchCommand{Name: "缓存隔离复现批次", Area: "东区", Operator: "巡检员"}
	create.Tree.Name = "复现树木"
	create.Tree.RoadLocation = "园林路 1 号"
	create.Tree.Species = "香樟"
	create.Tree.DiameterCM = 32
	create.Tree.ResponsibilityArea = "养护一组"
	created, err := service.CreateBatch(ctx, create, "alias-create-0001")
	if err != nil {
		t.Fatal(err)
	}

	evidence, err := service.SubmitEvidence(ctx, application.SubmitEvidenceCommand{
		TreeID:          created.Tree.ID,
		ExpectedVersion: created.Tree.Version,
		InspectedAt:     now,
		PhotoDigest:     "sha256:alias-repro-photo",
		LeafCondition:   domain.ConditionModerate,
		TrunkDefect:     domain.ConditionModerate,
		PestSigns:       domain.ConditionModerate,
		Notes:           "叶片、树干和病虫害均需处置",
		SubmittedBy:     "巡检员",
	}, "alias-evidence-01")
	if err != nil {
		t.Fatal(err)
	}
	assessed, err := service.Assess(ctx, application.AssessCommand{
		TreeID:          created.Tree.ID,
		ExpectedVersion: evidence.Tree.Version,
		Assignee:        "养护员",
		DueDate:         now.Add(73 * time.Hour),
		Operator:        "负责人",
	}, "alias-assess-0001")
	if err != nil {
		t.Fatal(err)
	}
	completed, err := service.CompleteRemediation(ctx, application.CompleteRemediationCommand{
		TreeID:              created.Tree.ID,
		ExpectedTreeVersion: assessed.Tree.Version,
		ExpectedTaskVersion: assessed.Task.Version,
		CompletionNote:      "完成病枝清理、树干加固和病虫害处置",
		CompletedAt:         now.Add(30 * time.Minute),
		Operator:            "养护员",
	}, "alias-remedy-0001")
	if err != nil {
		t.Fatal(err)
	}

	metrics := map[string]string{
		"树冠稳定": "病枝已清除",
		"树干安全": "支撑牢固",
		"病虫受控": "未见新增虫孔",
	}
	command := application.RecheckCommand{
		TreeID:              created.Tree.ID,
		ExpectedTreeVersion: completed.Tree.Version,
		ExpectedTaskVersion: completed.Task.Version,
		RecheckedAt:         now.Add(45 * time.Minute),
		Metrics:             metrics,
		Result:              "通过",
		Inspector:           "复验员",
	}
	first, err := service.Recheck(ctx, command, "alias-recheck-0001")
	if err != nil {
		t.Fatal(err)
	}
	if first.Certificate == nil {
		t.Fatal("首次复验未签发凭据")
	}
	replayCommand := command
	replayCommand.Metrics = map[string]string{
		"树冠稳定": "病枝已清除",
		"树干安全": "支撑牢固",
		"病虫受控": "未见新增虫孔",
	}

	first.Certificate.Metrics["树干安全"] = "调用方误改"
	replayed, err := service.Recheck(ctx, replayCommand, "alias-recheck-0001")
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Certificate == nil {
		t.Fatal("幂等重放未返回凭据")
	}
	if got := replayed.Certificate.Metrics["树干安全"]; got != "支撑牢固" {
		t.Fatalf("幂等缓存被已返回对象污染: tree safety metric = %q", got)
	}
	if !domain.VerifyCertificate(*replayed.Certificate, evidence.Evidence.Digest, completed.Task.ID) {
		t.Fatal("幂等缓存返回了摘要已失效的养护凭据")
	}
}
