package application

import (
	"context"
	"fmt"
	"time"

	"citytree/internal/domain"
)

type AssessCommand struct {
	TreeID          string    `json:"treeID"`
	ExpectedVersion int64     `json:"expectedVersion"`
	Assignee        string    `json:"assignee"`
	DueDate         time.Time `json:"dueDate"`
	Operator        string    `json:"operator"`
}

type AssessResult struct {
	Tree       domain.TreeAsset        `json:"tree"`
	Assessment domain.RiskAssessment   `json:"assessment"`
	Task       *domain.RemediationTask `json:"task,omitempty"`
}

func (s *Service) Assess(ctx context.Context, cmd AssessCommand, key string) (AssessResult, error) {
	return executeIdempotent(ctx, s, "assess:"+cmd.TreeID, key, cmd, func(repo Repository) (AssessResult, error) {
		tree, err := repo.GetTree(ctx, cmd.TreeID)
		if err != nil {
			return AssessResult{}, err
		}
		if err = checkExpected(tree.Version, cmd.ExpectedVersion); err != nil {
			return AssessResult{}, err
		}
		if tree.CurrentStatus != domain.TreeEvidenceSubmitted {
			return AssessResult{}, domain.ErrTransition
		}
		evidence, err := repo.LatestEvidence(ctx, tree.ID)
		if err != nil {
			return AssessResult{}, err
		}
		assessment, err := domain.AssessEvidence(evidence)
		if err != nil {
			return AssessResult{}, err
		}
		oldVersion := tree.Version
		result := AssessResult{Assessment: assessment}
		if assessment.Level.RequiresTask() {
			task, err := domain.AssignRemediation(tree.ID, assessment, cmd.Assignee, cmd.DueDate, s.now())
			if err != nil {
				return AssessResult{}, err
			}
			if err = tree.Transition(domain.TreeAwaitingRemediation); err != nil {
				return AssessResult{}, err
			}
			if err = repo.InsertTask(ctx, task); err != nil {
				return AssessResult{}, err
			}
			result.Task = &task
		} else {
			if err = tree.Transition(domain.TreeClosed); err != nil {
				return AssessResult{}, err
			}
		}
		if err = repo.UpdateTree(ctx, tree, oldVersion); err != nil {
			return AssessResult{}, err
		}
		if !assessment.Level.RequiresTask() {
			if err = completeBatchIfReady(ctx, repo, tree.BatchID); err != nil {
				return AssessResult{}, err
			}
		}
		detail := fmt.Sprintf("风险等级 %s，得分 %d", assessment.Level, assessment.Score)
		if err = appendAudit(ctx, repo, "tree", tree.ID, "risk.assessed", cmd.Operator, detail, s.now()); err != nil {
			return AssessResult{}, err
		}
		result.Tree = tree
		return result, nil
	})
}

type CompleteRemediationCommand struct {
	TreeID              string    `json:"treeID"`
	ExpectedTreeVersion int64     `json:"expectedTreeVersion"`
	ExpectedTaskVersion int64     `json:"expectedTaskVersion"`
	CompletionNote      string    `json:"completionNote"`
	CompletedAt         time.Time `json:"completedAt"`
	Operator            string    `json:"operator"`
}

type CompleteRemediationResult struct {
	Tree domain.TreeAsset       `json:"tree"`
	Task domain.RemediationTask `json:"task"`
}

func (s *Service) CompleteRemediation(ctx context.Context, cmd CompleteRemediationCommand, key string) (CompleteRemediationResult, error) {
	return executeIdempotent(ctx, s, "complete-remediation:"+cmd.TreeID, key, cmd, func(repo Repository) (CompleteRemediationResult, error) {
		tree, err := repo.GetTree(ctx, cmd.TreeID)
		if err != nil {
			return CompleteRemediationResult{}, err
		}
		task, err := repo.ActiveTask(ctx, cmd.TreeID)
		if err != nil {
			return CompleteRemediationResult{}, err
		}
		if err = checkExpected(tree.Version, cmd.ExpectedTreeVersion); err != nil {
			return CompleteRemediationResult{}, err
		}
		if err = checkExpected(task.Version, cmd.ExpectedTaskVersion); err != nil {
			return CompleteRemediationResult{}, err
		}
		oldTree, oldTask := tree.Version, task.Version
		if err = task.Complete(cmd.CompletionNote, cmd.CompletedAt); err != nil {
			return CompleteRemediationResult{}, err
		}
		if err = tree.Transition(domain.TreeAwaitingRecheck); err != nil {
			return CompleteRemediationResult{}, err
		}
		if err = repo.UpdateTask(ctx, task, oldTask); err != nil {
			return CompleteRemediationResult{}, err
		}
		if err = repo.UpdateTree(ctx, tree, oldTree); err != nil {
			return CompleteRemediationResult{}, err
		}
		if err = appendAudit(ctx, repo, "task", task.ID, "remediation.completed", cmd.Operator, task.CompletionNote, s.now()); err != nil {
			return CompleteRemediationResult{}, err
		}
		return CompleteRemediationResult{Tree: tree, Task: task}, nil
	})
}
