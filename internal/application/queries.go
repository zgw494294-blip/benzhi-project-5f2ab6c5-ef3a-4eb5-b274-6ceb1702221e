package application

import (
	"context"
	"errors"

	"citytree/internal/domain"
)

func (s *Service) Dashboard(ctx context.Context) (DashboardView, error) {
	call, leader := s.beginDashboardCall()
	if !leader {
		select {
		case <-ctx.Done():
			return DashboardView{}, ctx.Err()
		case <-call.done:
			return call.view, call.err
		}
	}
	call.view, call.err = s.loadDashboard(ctx)
	s.finishDashboardCall(call)
	return call.view, call.err
}

func (s *Service) beginDashboardCall() (*dashboardCall, bool) {
	s.dashboardMu.Lock()
	defer s.dashboardMu.Unlock()
	if s.dashboardCall != nil {
		return s.dashboardCall, false
	}
	call := &dashboardCall{done: make(chan struct{})}
	s.dashboardCall = call
	return call, true
}

func (s *Service) finishDashboardCall(call *dashboardCall) {
	s.dashboardMu.Lock()
	close(call.done)
	if s.dashboardCall == call {
		s.dashboardCall = nil
	}
	s.dashboardMu.Unlock()
}

func (s *Service) loadDashboard(ctx context.Context) (DashboardView, error) {
	batches, err := s.store.ListBatches(ctx)
	if err != nil {
		return DashboardView{}, err
	}
	view := DashboardView{Batches: make([]BatchSummary, 0, len(batches))}
	for _, batch := range batches {
		detail, err := s.Batch(ctx, batch.ID)
		if err != nil {
			return DashboardView{}, err
		}
		summary := BatchSummary{Batch: batch, TreeCount: detail.Total, Closed: detail.Closed, RiskCount: detail.RiskTrees}
		if detail.Total > 0 {
			summary.Completion = detail.Closed * 100 / detail.Total
		}
		view.TotalTrees += detail.Total
		view.Certificates += len(detail.Certificates)
		for _, tree := range detail.Trees {
			if tree.Task != nil && tree.Task.Status != domain.TaskClosed {
				view.OpenTasks++
			}
		}
		view.Batches = append(view.Batches, summary)
	}
	return view, nil
}

func (s *Service) Batch(ctx context.Context, id string) (BatchView, error) {
	batch, err := s.store.GetBatch(ctx, id)
	if err != nil {
		return BatchView{}, err
	}
	trees, err := s.store.ListTreesByBatch(ctx, id)
	if err != nil {
		return BatchView{}, err
	}
	view := BatchView{Batch: batch, Total: len(trees), Trees: make([]TreeView, 0, len(trees))}
	for _, tree := range trees {
		item := TreeView{Tree: tree}
		if evidence, e := s.store.LatestEvidence(ctx, tree.ID); e == nil {
			item.Evidence = &evidence
		} else if !errors.Is(e, domain.ErrNotFound) {
			return BatchView{}, e
		}
		if task, e := s.store.ActiveTask(ctx, tree.ID); e == nil {
			item.Task = &task
			if task.RiskLevel.RequiresTask() {
				view.RiskTrees++
			}
		} else if !errors.Is(e, domain.ErrNotFound) {
			return BatchView{}, e
		}
		if cert, e := s.store.CertificateByTree(ctx, tree.ID); e == nil {
			item.Certificate = &cert
			view.Certificates = append(view.Certificates, cert)
		} else if !errors.Is(e, domain.ErrNotFound) {
			return BatchView{}, e
		}
		if tree.CurrentStatus == domain.TreeClosed {
			view.Closed++
		}
		view.Trees = append(view.Trees, item)
	}
	return view, nil
}

func (s *Service) Tree(ctx context.Context, id string) (TreeView, error) {
	tree, err := s.store.GetTree(ctx, id)
	if err != nil {
		return TreeView{}, err
	}
	view := TreeView{Tree: tree}
	if v, e := s.store.LatestEvidence(ctx, id); e == nil {
		view.Evidence = &v
	} else if !errors.Is(e, domain.ErrNotFound) {
		return TreeView{}, e
	}
	if v, e := s.store.ActiveTask(ctx, id); e == nil {
		view.Task = &v
	} else if !errors.Is(e, domain.ErrNotFound) {
		return TreeView{}, e
	}
	if v, e := s.store.CertificateByTree(ctx, id); e == nil {
		view.Certificate = &v
	} else if !errors.Is(e, domain.ErrNotFound) {
		return TreeView{}, e
	}
	return view, nil
}
