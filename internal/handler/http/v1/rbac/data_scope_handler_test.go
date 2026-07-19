package rbac

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	rbacsvc "github.com/tinboxw/skoll/internal/service/rbac"
)

type dataScopeServiceStub struct {
	rbacsvc.Service
	decision rbacsvc.DataScopeDecision
	err      error
	input    rbacsvc.ResolveDataScopeInput
}

func (s *dataScopeServiceStub) ResolveDataScope(_ context.Context, input rbacsvc.ResolveDataScopeInput) (rbacsvc.DataScopeDecision, error) {
	s.input = input
	return s.decision, s.err
}

func TestDataScopeReturnsTrustedDecision(t *testing.T) {
	service := &dataScopeServiceStub{decision: rbacsvc.DataScopeDecision{
		Scope:         "department_tree",
		DepartmentIDs: []string{"dept-root", "dept-child"},
	}}
	mux := http.NewServeMux()
	RegisterRBACRoutes(mux, service, nil)

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/rbac/data-scope?resource=pharma_oa.customer&action=read", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if service.input.Resource != "pharma_oa.customer" || service.input.Action != "read" {
		t.Fatalf("resolver input = %+v", service.input)
	}
	var response struct {
		Data rbacsvc.DataScopeDecision `json:"data"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Data.Scope != "department_tree" || len(response.Data.DepartmentIDs) != 2 {
		t.Fatalf("decision = %+v", response.Data)
	}
}

func TestDataScopeRejectsInvalidOrDeniedRequests(t *testing.T) {
	service := &dataScopeServiceStub{}
	mux := http.NewServeMux()
	RegisterRBACRoutes(mux, service, nil)

	missing := httptest.NewRecorder()
	mux.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/v1/rbac/data-scope?resource=user", nil))
	if missing.Code != http.StatusBadRequest {
		t.Fatalf("missing action status = %d, want %d", missing.Code, http.StatusBadRequest)
	}

	service.err = errors.New("scope denied")
	denied := httptest.NewRecorder()
	mux.ServeHTTP(denied, httptest.NewRequest(http.MethodGet, "/v1/rbac/data-scope?resource=user&action=read", nil))
	if denied.Code != http.StatusForbidden {
		t.Fatalf("denied status = %d, want %d", denied.Code, http.StatusForbidden)
	}
}
