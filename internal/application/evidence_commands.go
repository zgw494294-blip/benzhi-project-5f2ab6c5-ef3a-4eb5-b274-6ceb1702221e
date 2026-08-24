package application

import (
	"context"
	"fmt"
	"time"

	"citytree/internal/domain"
)

type SubmitEvidenceCommand struct {
	TreeID          string           `json:"treeID"`
	ExpectedVersion int64            `json:"expectedVersion"`
	InspectedAt     time.Time        `json:"inspectedAt"`
	PhotoDigest     string           `json:"photoDigest"`
	LeafCondition   domain.Condition `json:"leafCondition"`
	TrunkDefect     domain.Condition `json:"trunkDefect"`
	PestSigns       domain.Condition `json:"pestSigns"`
	Notes           string           `json:"notes"`
	SubmittedBy     string           `json:"submittedBy"`
}

type SubmitEvidenceResult struct {
	Tree     domain.TreeAsset          `json:"tree"`
	Evidence domain.InspectionEvidence `json:"evidence"`
}

func (s *Service) SubmitEvidence(ctx context.Context, cmd SubmitEvidenceCommand, key string) (SubmitEvidenceResult, error) {
	return executeIdempotent(ctx, s, "submit-evidence:"+cmd.TreeID, key, cmd, func(repo Repository) (SubmitEvidenceResult, error) {
		tree, err := repo.GetTree(ctx, cmd.TreeID)
		if err != nil {
			return SubmitEvidenceResult{}, err
		}
		if err = checkExpected(tree.Version, cmd.ExpectedVersion); err != nil {
			return SubmitEvidenceResult{}, err
		}
		if tree.CurrentStatus != domain.TreeRegistered {
			return SubmitEvidenceResult{}, domain.ErrTransition
		}
		evidence, err := domain.NewEvidence(tree.ID, tree.BatchID, domain.EvidenceInput{InspectedAt: cmd.InspectedAt, PhotoDigest: cmd.PhotoDigest, LeafCondition: cmd.LeafCondition, TrunkDefect: cmd.TrunkDefect, PestSigns: cmd.PestSigns, Notes: cmd.Notes, SubmittedBy: cmd.SubmittedBy})
		if err != nil {
			return SubmitEvidenceResult{}, err
		}
		previous := tree.Version
		if err = tree.Transition(domain.TreeEvidenceSubmitted); err != nil {
			return SubmitEvidenceResult{}, err
		}
		if err = repo.InsertEvidence(ctx, evidence); err != nil {
			return SubmitEvidenceResult{}, err
		}
		if err = repo.UpdateTree(ctx, tree, previous); err != nil {
			return SubmitEvidenceResult{}, err
		}
		batch, err := repo.GetBatch(ctx, tree.BatchID)
		if err != nil {
			return SubmitEvidenceResult{}, err
		}
		batchVersion := batch.Version
		batch.MarkStarted()
		if batch.Version != batchVersion {
			if err = repo.UpdateBatch(ctx, batch, batchVersion); err != nil {
				return SubmitEvidenceResult{}, err
			}
		}
		detail := fmt.Sprintf("提交证据摘要 %s", evidence.Digest[:12])
		if err = appendAudit(ctx, repo, "evidence", evidence.ID, "evidence.submitted", cmd.SubmittedBy, detail, s.now()); err != nil {
			return SubmitEvidenceResult{}, err
		}
		return SubmitEvidenceResult{Tree: tree, Evidence: evidence}, nil
	})
}
