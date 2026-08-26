package stale_checkpoint_recovery

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"citytree/internal/application"
	"citytree/internal/persistence"
)

func TestDatabaseReplacementDoesNotReuseStaleCheckpoint(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "city-tree.db")
	store, err := persistence.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	cmd := application.CreateBatchCommand{Name: "旧批次", Area: "东区", Operator: "管理员"}
	cmd.Tree.Name = "一号树"
	cmd.Tree.RoadLocation = "巡检路 1 号"
	cmd.Tree.Species = "香樟"
	cmd.Tree.DiameterCM = 30
	cmd.Tree.ResponsibilityArea = "一组"
	if _, err = application.NewService(store).CreateBatch(ctx, cmd, "stale-checkpoint-1"); err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(path + ".integrity"); err != nil {
		t.Fatalf("expected integrity checkpoint fixture: %v", err)
	}
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		_ = os.Remove(path + suffix)
	}

	recovered, err := persistence.Open(ctx, path)
	if err != nil {
		t.Fatalf("database replacement recovery failed: %v", err)
	}
	defer recovered.Close()
}
