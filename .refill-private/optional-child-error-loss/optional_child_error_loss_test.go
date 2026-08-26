package optional_child_error_loss_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"citytree/internal/application"
	"citytree/internal/domain"
	"citytree/internal/persistence"
	"citytree/internal/web"
)

type faultStore struct {
	application.Store
	kind  string
	fault error
}

func (s *faultStore) LatestEvidence(ctx context.Context, treeID string) (domain.InspectionEvidence, error) {
	if s.kind == "evidence" {
		return domain.InspectionEvidence{}, s.fault
	}
	return s.Store.LatestEvidence(ctx, treeID)
}

func (s *faultStore) ActiveTask(ctx context.Context, treeID string) (domain.RemediationTask, error) {
	if s.kind == "task" {
		return domain.RemediationTask{}, s.fault
	}
	return s.Store.ActiveTask(ctx, treeID)
}

func (s *faultStore) CertificateByTree(ctx context.Context, treeID string) (domain.CareCertificate, error) {
	if s.kind == "certificate" {
		return domain.CareCertificate{}, s.fault
	}
	return s.Store.CertificateByTree(ctx, treeID)
}

func TestOptionalChildReadErrorsRemainObservable(t *testing.T) {
	ctx := context.Background()
	store, err := persistence.Open(ctx, filepath.Join(t.TempDir(), "optional-errors.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	commandService := application.NewService(store)
	cmd := application.CreateBatchCommand{Name: "错误传播测试批次", Area: "东区", Operator: "测试员"}
	cmd.Tree.Name = "测试树木"
	cmd.Tree.RoadLocation = "园林路 1 号"
	cmd.Tree.Species = "香樟"
	cmd.Tree.DiameterCM = 30
	cmd.Tree.ResponsibilityArea = "一组"
	created, err := commandService.CreateBatch(ctx, cmd, "optional-error-create-01")
	if err != nil {
		t.Fatal(err)
	}

	for _, target := range []string{"tree", "batch"} {
		for _, kind := range []string{"evidence", "task", "certificate"} {
			t.Run(target+"-"+kind, func(t *testing.T) {
				fault := errors.New("optional " + kind + " resource unavailable")
				service := application.NewService(&faultStore{Store: store, kind: kind, fault: fault})

				var queryErr error
				path := "/api/trees/" + created.Tree.ID
				if target == "tree" {
					_, queryErr = service.Tree(ctx, created.Tree.ID)
				} else {
					_, queryErr = service.Batch(ctx, created.Batch.ID)
					path = "/api/batches/" + created.Batch.ID
				}
				if !errors.Is(queryErr, fault) {
					t.Errorf("%s query lost %s error: %v", target, kind, queryErr)
				}

				request := httptest.NewRequest(http.MethodGet, path, nil)
				response := httptest.NewRecorder()
				web.NewServer(service).Handler().ServeHTTP(response, request)
				body, readErr := io.ReadAll(response.Result().Body)
				if readErr != nil {
					t.Fatal(readErr)
				}
				if response.Code != http.StatusInternalServerError {
					t.Errorf("%s %s status=%d, want 500; body=%s", target, kind, response.Code, body)
				}
				if !strings.Contains(string(body), fault.Error()) {
					t.Errorf("%s %s response lost error text: %s", target, kind, body)
				}
			})
		}
	}
}
