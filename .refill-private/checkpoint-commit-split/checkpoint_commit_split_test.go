package checkpoint_commit_split_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"citytree/internal/application"
	"citytree/internal/persistence"
)

func TestCheckpointFailureLeavesCommittedWrite(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "city-tree.db")
	store, err := persistence.Open(ctx, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// A directory at the checkpoint target makes the atomic rename fail after
	// SQLite has committed the transaction.
	if err := os.Mkdir(dbPath+".integrity", 0o755); err != nil {
		t.Fatal(err)
	}
	service := application.NewService(store)
	cmd := application.CreateBatchCommand{Name: "检查点故障批次", Area: "东区", Operator: "巡检员"}
	cmd.Tree.Name = "一号树"
	cmd.Tree.RoadLocation = "园林路 1 号"
	cmd.Tree.Species = "香樟"
	cmd.Tree.DiameterCM = 30
	cmd.Tree.ResponsibilityArea = "养护一组"

	created, err := service.CreateBatch(ctx, cmd, "checkpoint-key-01")
	if err == nil {
		t.Fatal("checkpoint failure was not triggered")
	}
	if created.Batch.ID == "" {
		t.Fatal("command did not reach the post-commit checkpoint")
	}
	if _, lookupErr := store.GetBatch(ctx, created.Batch.ID); lookupErr == nil {
		t.Fatalf("CreateBatch returned an error but batch %s is already committed", created.Batch.ID)
	}
}
