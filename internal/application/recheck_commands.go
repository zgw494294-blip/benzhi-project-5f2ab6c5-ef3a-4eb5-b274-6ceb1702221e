package application

import (
	"context"
	"fmt"
	"time"

	"citytree/internal/domain"
)

type RecheckCommand struct {
	TreeID              string            `json:"treeID"`
	ExpectedTreeVersion int64             `json:"expectedTreeVersion"`
	ExpectedTaskVersion int64             `json:"expectedTaskVersion"`
	RecheckedAt         time.Time         `json:"recheckedAt"`
	Metrics             map[string]string `json:"metrics"`
	Result              string            `json:"result"`
	Inspector           string            `json:"inspector"`
}

type RecheckResult struct {
	Tree        domain.TreeAsset        `json:"tree"`
	Task        domain.RemediationTask  `json:"task"`
	Certificate *domain.CareCertificate `json:"certificate,omitempty"`
}

type recheckCacheEntry struct {
	requestHash string
	result      RecheckResult
}

func (s *Service) Recheck(ctx context.Context, cmd RecheckCommand, key string) (RecheckResult, error) {
	scope := "recheck:" + cmd.TreeID
	if cached, ok, err := s.loadRecheckResult(scope, key, cmd); err != nil || ok {
		return cached, err
	}
	result, err := executeIdempotent(ctx, s, scope, key, cmd, func(repo Repository) (RecheckResult, error) {
		tree, err := repo.GetTree(ctx, cmd.TreeID)
		if err != nil {
			return RecheckResult{}, err
		}
		task, err := repo.ActiveTask(ctx, cmd.TreeID)
		if err != nil {
			return RecheckResult{}, err
		}
		if err = checkExpected(tree.Version, cmd.ExpectedTreeVersion); err != nil {
			return RecheckResult{}, err
		}
		if err = checkExpected(task.Version, cmd.ExpectedTaskVersion); err != nil {
			return RecheckResult{}, err
		}
		input := domain.RecheckInput{RecheckedAt: cmd.RecheckedAt, Metrics: cmd.Metrics, Result: domain.NormalizeText(cmd.Result), Inspector: domain.NormalizeText(cmd.Inspector)}
		passed, err := domain.EvaluateRecheck(input)
		if err != nil {
			return RecheckResult{}, err
		}
		oldTree, oldTask := tree.Version, task.Version
		if err = task.ApplyRecheck(passed); err != nil {
			return RecheckResult{}, err
		}
		if passed {
			err = tree.Transition(domain.TreeClosed)
		} else {
			err = tree.Transition(domain.TreeAwaitingRemediation)
		}
		if err != nil {
			return RecheckResult{}, err
		}
		if err = repo.UpdateTask(ctx, task, oldTask); err != nil {
			return RecheckResult{}, err
		}
		if err = repo.UpdateTree(ctx, tree, oldTree); err != nil {
			return RecheckResult{}, err
		}
		result := RecheckResult{Tree: tree, Task: task}
		if passed {
			evidence, err := repo.LatestEvidence(ctx, tree.ID)
			if err != nil {
				return RecheckResult{}, err
			}
			certificate, err := domain.IssueCertificate(tree, evidence, task, input, s.now())
			if err != nil {
				return RecheckResult{}, err
			}
			if err = repo.InsertCertificate(ctx, certificate); err != nil {
				return RecheckResult{}, err
			}
			result.Certificate = &certificate
		}
		detail := fmt.Sprintf("复验结论 %s", input.Result)
		if err = appendAudit(ctx, repo, "tree", tree.ID, "recheck.completed", cmd.Inspector, detail, s.now()); err != nil {
			return RecheckResult{}, err
		}
		if passed {
			if err = completeBatchIfReady(ctx, repo, tree.BatchID); err != nil {
				return RecheckResult{}, err
			}
		}
		return result, nil
	})
	if err == nil {
		s.rememberRecheckResult(scope, key, cmd, result)
	}
	return result, err
}

func (s *Service) loadRecheckResult(scope, key string, cmd RecheckCommand) (RecheckResult, bool, error) {
	hash, err := requestDigest(cmd)
	if err != nil {
		return RecheckResult{}, false, err
	}
	s.recheckMu.RLock()
	entry, ok := s.recheckResults[scope+"\x00"+key]
	s.recheckMu.RUnlock()
	if !ok {
		return RecheckResult{}, false, nil
	}
	if entry.requestHash != hash {
		return RecheckResult{}, false, domain.ErrIdempotency
	}
	return cloneRecheckResult(entry.result), true, nil
}

func (s *Service) rememberRecheckResult(scope, key string, cmd RecheckCommand, result RecheckResult) {
	hash, err := requestDigest(cmd)
	if err != nil {
		return
	}
	cloned := cloneRecheckResult(result)
	s.recheckMu.Lock()
	s.recheckResults[scope+"\x00"+key] = recheckCacheEntry{requestHash: hash, result: cloned}
	s.recheckMu.Unlock()
}

func cloneRecheckResult(r RecheckResult) RecheckResult {
	clone := r
	if r.Certificate != nil {
		c := *r.Certificate
		c.Metrics = domain.CloneMetrics(c.Metrics)
		clone.Certificate = &c
	}
	return clone
}

func completeBatchIfReady(ctx context.Context, repo Repository, batchID string) error {
	trees, err := repo.ListTreesByBatch(ctx, batchID)
	if err != nil {
		return err
	}
	for _, tree := range trees {
		if tree.CurrentStatus != domain.TreeClosed {
			return nil
		}
	}
	batch, err := repo.GetBatch(ctx, batchID)
	if err != nil {
		return err
	}
	old := batch.Version
	batch.MarkCompleted()
	if batch.Version == old {
		return nil
	}
	return repo.UpdateBatch(ctx, batch, old)
}
