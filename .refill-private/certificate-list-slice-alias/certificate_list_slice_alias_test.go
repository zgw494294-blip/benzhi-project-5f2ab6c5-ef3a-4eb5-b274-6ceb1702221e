package certificate_list_slice_alias_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"citytree/internal/application"
	"citytree/internal/domain"
	"citytree/internal/persistence"
)

func TestCertificateListResultSurvivesSubsequentRead(t *testing.T) {
	ctx := context.Background()
	store, err := persistence.Open(ctx, filepath.Join(t.TempDir(), "trees.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := application.NewService(store)
	first := createCertifiedTree(t, service, "东区", "一号树", "certificate-list-a")
	second := createCertifiedTree(t, service, "西区", "二号树", "certificate-list-b")

	firstList, err := store.ListCertificatesByBatch(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	if len(firstList) != 1 || firstList[0].TreeID == "" {
		t.Fatalf("unexpected first certificate list: %+v", firstList)
	}
	wantTreeID := firstList[0].TreeID
	if _, err = store.ListCertificatesByBatch(ctx, second); err != nil {
		t.Fatal(err)
	}
	if firstList[0].TreeID != wantTreeID {
		t.Fatalf("first list was overwritten by a later read: got %q want %q", firstList[0].TreeID, wantTreeID)
	}
}

func createCertifiedTree(t *testing.T, service *application.Service, area, name, keyPrefix string) string {
	t.Helper()
	ctx := context.Background()
	cmd := application.CreateBatchCommand{Name: "批次-" + area, Area: area, Operator: "测试员"}
	cmd.Tree.Name = name
	cmd.Tree.RoadLocation = area + "主路"
	cmd.Tree.Species = "香樟"
	cmd.Tree.DiameterCM = 30
	cmd.Tree.ResponsibilityArea = "养护一组"
	created, err := service.CreateBatch(ctx, cmd, keyPrefix+"-create")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Add(-time.Minute)
	evidence, err := service.SubmitEvidence(ctx, application.SubmitEvidenceCommand{
		TreeID: created.Tree.ID, ExpectedVersion: created.Tree.Version, InspectedAt: now,
		PhotoDigest: "sha256:" + keyPrefix, LeafCondition: domain.ConditionSevere,
		TrunkDefect: domain.ConditionModerate, PestSigns: domain.ConditionModerate,
		Notes: "结构化巡检记录", SubmittedBy: "巡检员",
	}, keyPrefix+"-evidence")
	if err != nil {
		t.Fatal(err)
	}
	assessed, err := service.Assess(ctx, application.AssessCommand{
		TreeID: evidence.Tree.ID, ExpectedVersion: evidence.Tree.Version,
		Assignee: "修复负责人", DueDate: time.Now().Add(time.Hour), Operator: "负责人",
	}, keyPrefix+"-assess")
	if err != nil {
		t.Fatal(err)
	}
	completed, err := service.CompleteRemediation(ctx, application.CompleteRemediationCommand{
		TreeID: assessed.Tree.ID, ExpectedTreeVersion: assessed.Tree.Version,
		ExpectedTaskVersion: assessed.Task.Version, CompletionNote: "完成修复",
		CompletedAt: time.Now(), Operator: "修复人员",
	}, keyPrefix+"-complete")
	if err != nil {
		t.Fatal(err)
	}
	rechecked, err := service.Recheck(ctx, application.RecheckCommand{
		TreeID: completed.Tree.ID, ExpectedTreeVersion: completed.Tree.Version,
		ExpectedTaskVersion: completed.Task.Version, RecheckedAt: time.Now(),
		Metrics: map[string]string{"树冠稳定": "是", "树干安全": "是", "病虫受控": "是"},
		Result:  "通过", Inspector: "复验员",
	}, keyPrefix+"-recheck")
	if err != nil || rechecked.Certificate == nil {
		t.Fatalf("recheck failed: %v", err)
	}
	return created.Batch.ID
}
