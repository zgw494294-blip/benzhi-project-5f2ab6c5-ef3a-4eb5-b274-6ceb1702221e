package canceled_command_commit_test

import (
	"context"
	"errors"
	"testing"

	"citytree/internal/application"
	"citytree/internal/persistence"
)

func TestCanceledCreateDoesNotCommit(t *testing.T) {
	store, err := persistence.Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	service := application.NewService(store)
	cmd := application.CreateBatchCommand{Name: "取消请求批次", Area: "东区", Operator: "巡检员"}
	cmd.Tree.Name = "一号树"
	cmd.Tree.RoadLocation = "园林路 1 号"
	cmd.Tree.Species = "香樟"
	cmd.Tree.DiameterCM = 32
	cmd.Tree.ResponsibilityArea = "养护一组"

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, createErr := service.CreateBatch(ctx, cmd, "cancel-create-01")
	if errors.Is(createErr, context.Canceled) {
		return
	}
	if createErr != nil {
		t.Fatalf("取消后的创建应返回 context.Canceled，实际返回 %v", createErr)
	}
	view, err := service.Dashboard(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Fatalf("请求 context 已取消但命令仍提交，持久化批次数=%d", len(view.Batches))
}
