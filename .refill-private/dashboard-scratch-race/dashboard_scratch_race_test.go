package dashboard_scratch_race

import (
	"context"
	"sync"
	"testing"

	"citytree/internal/application"
	"citytree/internal/persistence"
)

func TestConcurrentDashboardCallsDoNotShareAssemblyState(t *testing.T) {
	ctx := context.Background()
	store, err := persistence.Open(ctx, t.TempDir()+"/dashboard.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	service := application.NewService(store)
	cmd := application.CreateBatchCommand{Name: "并发仪表盘", Area: "东区", Operator: "测试员"}
	cmd.Tree.Name = "树木"
	cmd.Tree.RoadLocation = "东区 1 号"
	cmd.Tree.Species = "香樟"
	cmd.Tree.DiameterCM = 32
	cmd.Tree.ResponsibilityArea = "一组"
	if _, err = service.CreateBatch(ctx, cmd, "dashboard-race-create"); err != nil {
		t.Fatal(err)
	}

	const callers = 48
	start := make(chan struct{})
	var ready sync.WaitGroup
	var done sync.WaitGroup
	ready.Add(callers)
	done.Add(callers)
	for i := 0; i < callers; i++ {
		go func() {
			defer done.Done()
			ready.Done()
			<-start
			view, callErr := service.Dashboard(ctx)
			if callErr != nil {
				t.Errorf("仪表盘查询失败: %v", callErr)
				return
			}
			if len(view.Batches) != 1 || view.TotalTrees != 1 {
				t.Errorf("仪表盘结果被并发组装污染: batches=%d trees=%d", len(view.Batches), view.TotalTrees)
			}
		}()
	}
	ready.Wait()
	close(start)
	done.Wait()
}
