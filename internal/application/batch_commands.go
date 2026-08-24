package application

import (
	"context"
	"fmt"

	"citytree/internal/domain"
)

type CreateBatchCommand struct {
	Name string `json:"name"`
	Area string `json:"area"`
	Tree struct {
		Name               string  `json:"name"`
		RoadLocation       string  `json:"roadLocation"`
		Species            string  `json:"species"`
		DiameterCM         float64 `json:"diameterCM"`
		ResponsibilityArea string  `json:"responsibilityArea"`
	} `json:"tree"`
	Operator string `json:"operator"`
}

type CreateBatchResult struct {
	Batch domain.InspectionBatch `json:"batch"`
	Tree  domain.TreeAsset       `json:"tree"`
}

func (s *Service) CreateBatch(ctx context.Context, cmd CreateBatchCommand, key string) (CreateBatchResult, error) {
	return executeIdempotent(ctx, s, "create-batch", key, cmd, func(repo Repository) (CreateBatchResult, error) {
		now := s.now()
		batch, err := domain.NewInspectionBatch(cmd.Name, cmd.Area, now)
		if err != nil {
			return CreateBatchResult{}, err
		}
		tree, err := domain.RegisterTree(batch.ID, domain.NewTree{Name: cmd.Tree.Name, RoadLocation: cmd.Tree.RoadLocation, Species: cmd.Tree.Species, DiameterCM: cmd.Tree.DiameterCM, Responsibility: cmd.Tree.ResponsibilityArea}, now)
		if err != nil {
			return CreateBatchResult{}, err
		}
		if err = repo.InsertBatch(ctx, batch); err != nil {
			return CreateBatchResult{}, err
		}
		if err = repo.InsertTree(ctx, tree); err != nil {
			return CreateBatchResult{}, err
		}
		detail := fmt.Sprintf("登记批次 %s，首棵树木 %s", batch.Name, tree.Name)
		if err = appendAudit(ctx, repo, "batch", batch.ID, "batch.created", cmd.Operator, detail, now); err != nil {
			return CreateBatchResult{}, err
		}
		return CreateBatchResult{Batch: batch, Tree: tree}, nil
	})
}

type AddTreeCommand struct {
	BatchID            string  `json:"batchID"`
	Name               string  `json:"name"`
	RoadLocation       string  `json:"roadLocation"`
	Species            string  `json:"species"`
	DiameterCM         float64 `json:"diameterCM"`
	ResponsibilityArea string  `json:"responsibilityArea"`
	Operator           string  `json:"operator"`
}

func (s *Service) AddTree(ctx context.Context, cmd AddTreeCommand, key string) (domain.TreeAsset, error) {
	batch, err := s.store.GetBatch(ctx, cmd.BatchID)
	if err != nil {
		return domain.TreeAsset{}, err
	}
	if batch.Status == domain.BatchCompleted {
		return domain.TreeAsset{}, domain.ErrTransition
	}
	return executeIdempotent(ctx, s, "add-tree:"+cmd.BatchID, key, cmd, func(repo Repository) (domain.TreeAsset, error) {
		tree, err := domain.RegisterTree(batch.ID, domain.NewTree{Name: cmd.Name, RoadLocation: cmd.RoadLocation, Species: cmd.Species, DiameterCM: cmd.DiameterCM, Responsibility: cmd.ResponsibilityArea}, s.now())
		if err != nil {
			return domain.TreeAsset{}, err
		}
		if err = repo.InsertTree(ctx, tree); err != nil {
			return domain.TreeAsset{}, err
		}
		if err = appendAudit(ctx, repo, "tree", tree.ID, "tree.registered", cmd.Operator, "追加树木档案", s.now()); err != nil {
			return domain.TreeAsset{}, err
		}
		return tree, nil
	})
}
