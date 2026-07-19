package user

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	domainrbac "github.com/tinboxw/skoll/internal/domain/rbac"
	"github.com/tinboxw/skoll/internal/domain/shared"
	domainuser "github.com/tinboxw/skoll/internal/domain/user"
	apiv1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	rbacsvc "github.com/tinboxw/skoll/internal/service/rbac"
	usersvc "github.com/tinboxw/skoll/internal/service/user"
	"github.com/tinboxw/skoll/pkg/security"
)

type fakeUserService struct {
	createInputs []usersvc.CreateUserInput
	batchInput   usersvc.BatchCreateInput
	lastUpdate   usersvc.UpdateEmailInput
	getReturn    *domainuser.User
	updateReturn *domainuser.User
	emailReturn  *domainuser.User
	lastDisable  struct {
		id      string
		actorID string
	}
}

type fakeAuditService struct {
	lastActorID  string
	lastAction   string
	lastResource string
	lastDetail   map[string]any
}

type fakeRBACService struct {
	lastBind rbacsvc.BindRoleInput
}

func (f *fakeRBACService) CheckPermission(_ context.Context, _ rbacsvc.CheckPermissionInput) (bool, error) {
	return false, nil
}

func (f *fakeRBACService) ResolvePermission(_ context.Context, _ rbacsvc.CheckPermissionInput) (rbacsvc.PermissionDecision, error) {
	return rbacsvc.PermissionDecision{}, nil
}

func (f *fakeRBACService) ResolveDataScope(_ context.Context, _ rbacsvc.ResolveDataScopeInput) (rbacsvc.DataScopeDecision, error) {
	return rbacsvc.DataScopeDecision{Scope: domainrbac.DataScopeAll, All: true}, nil
}

func (f *fakeRBACService) SetRolePolicies(_ context.Context, _ rbacsvc.SetRolePoliciesInput) error {
	return nil
}

func (f *fakeRBACService) BindRole(_ context.Context, in rbacsvc.BindRoleInput) (*domainrbac.Binding, error) {
	f.lastBind = in
	if strings.TrimSpace(in.RoleID) == "" {
		return nil, fmt.Errorf("role id is required")
	}
	return &domainrbac.Binding{ID: shared.ID("1"), SubjectID: shared.ID(in.SubjectID), RoleID: shared.ID(in.RoleID), Scope: in.Scope}, nil
}

func (f *fakeRBACService) UnbindBinding(_ context.Context, _ string) error {
	return nil
}

func (f *fakeRBACService) ListBindings(_ context.Context, _ domainrbac.SubjectType, _ string) ([]*domainrbac.Binding, error) {
	return nil, nil
}

func (f *fakeRBACService) ListBindingsByUser(_ context.Context, _ string) ([]*domainrbac.Binding, error) {
	return nil, nil
}

func (f *fakeAuditService) Append(_ context.Context, actorID, action, resource, _ string, detail map[string]any) (*domainaudit.Record, error) {
	f.lastActorID = actorID
	f.lastAction = action
	f.lastResource = resource
	f.lastDetail = detail
	return &domainaudit.Record{ID: shared.ID("audit-1")}, nil
}

func (f *fakeAuditService) GetByID(context.Context, string) (*domainaudit.Record, error) {
	return nil, nil
}

func (f *fakeAuditService) ListByActor(context.Context, string, int) ([]*domainaudit.Record, error) {
	return nil, nil
}

func (f *fakeAuditService) ListByTimeRange(context.Context, time.Time, time.Time, int) ([]*domainaudit.Record, error) {
	return nil, nil
}

func (f *fakeAuditService) ClearByTimeRange(context.Context, time.Time, time.Time) (int, error) {
	return 0, nil
}

var _ auditsvc.Service = (*fakeAuditService)(nil)

func (f *fakeUserService) Create(_ context.Context, in usersvc.CreateUserInput) (*domainuser.User, error) {
	f.createInputs = append(f.createInputs, in)
	if strings.EqualFold(strings.TrimSpace(in.Account), "bad") {
		return nil, fmt.Errorf("invalid account")
	}
	return &domainuser.User{ID: shared.ID("1"), Account: in.Account, Name: in.Name, Email: domainuser.Email(in.Email)}, nil
}

func (f *fakeUserService) CreateBatch(_ context.Context, in usersvc.BatchCreateInput) ([]usersvc.BatchCreateResult, error) {
	f.batchInput = in
	out := make([]usersvc.BatchCreateResult, 0, len(in.Items))
	for idx, item := range in.Items {
		if strings.EqualFold(strings.TrimSpace(item.Account), "bad") {
			out = append(out, usersvc.BatchCreateResult{Index: idx, Account: item.Account, Success: false, Message: "invalid account"})
			if in.Atomic {
				return out, nil
			}
			continue
		}
		out = append(out, usersvc.BatchCreateResult{Index: idx, Account: item.Account, Success: true, Message: "ok", ID: "1"})
	}
	return out, nil
}

func (f *fakeUserService) Get(_ context.Context, _ string) (*domainuser.User, error) {
	return f.getReturn, nil
}

func (f *fakeUserService) List(_ context.Context, _ usersvc.ListInput) ([]*domainuser.User, error) {
	return nil, nil
}

func (f *fakeUserService) Update(_ context.Context, _ usersvc.UpdateUserInput) (*domainuser.User, error) {
	return f.updateReturn, nil
}

func (f *fakeUserService) UpdateEmail(_ context.Context, in usersvc.UpdateEmailInput) (*domainuser.User, error) {
	f.lastUpdate = in
	return f.emailReturn, nil
}

func (f *fakeUserService) Disable(_ context.Context, id, actorID string) error {
	f.lastDisable.id = id
	f.lastDisable.actorID = actorID
	return nil
}

func (f *fakeUserService) Delete(_ context.Context, _ string) error {
	return nil
}

func TestUserHandlerUpdateEmailActorIDFallback(t *testing.T) {
	svc := &fakeUserService{}
	h := &UserHandler{service: svc}

	t.Run("use request actor id first", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/v1/users/u-1/email", bytes.NewBufferString(`{"email":"a@b.com","actorId":"manual-actor"}`))
		req.SetPathValue("id", "u-1")
		req = req.WithContext(security.WithJWTClaimsContext(req.Context(), &security.JWTClaims{Subject: "jwt-actor"}))
		resp := httptest.NewRecorder()

		h.updateEmail(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
		}
		if svc.lastUpdate.ActorID != "manual-actor" {
			t.Fatalf("expected manual actor id, got %q", svc.lastUpdate.ActorID)
		}
	})

	t.Run("fallback to jwt subject", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/v1/users/u-2/email", bytes.NewBufferString(`{"email":"a@b.com","actorId":""}`))
		req.SetPathValue("id", "u-2")
		req = req.WithContext(security.WithJWTClaimsContext(req.Context(), &security.JWTClaims{Subject: "jwt-actor"}))
		resp := httptest.NewRecorder()

		h.updateEmail(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
		}
		if svc.lastUpdate.ActorID != "jwt-actor" {
			t.Fatalf("expected jwt actor id, got %q", svc.lastUpdate.ActorID)
		}
	})
}

func TestUserHandlerCreateActorIDFallback(t *testing.T) {
	svc := &fakeUserService{}
	h := &UserHandler{service: svc}

	req := httptest.NewRequest(http.MethodPost, "/v1/users", bytes.NewBufferString(`{"account":"u1","name":"U1","email":"u1@example.com","passwordHash":"password-1234"}`))
	req = req.WithContext(security.WithJWTClaimsContext(req.Context(), &security.JWTClaims{Subject: "jwt-creator"}))
	resp := httptest.NewRecorder()

	h.create(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	if len(svc.createInputs) != 1 {
		t.Fatalf("expected one create input, got %d", len(svc.createInputs))
	}
	if svc.createInputs[0].ActorID != "jwt-creator" {
		t.Fatalf("expected fallback actor id from jwt, got %q", svc.createInputs[0].ActorID)
	}
}

func TestUserHandlerCreateBatchActorIDFallback(t *testing.T) {
	svc := &fakeUserService{}
	h := &UserHandler{service: svc}

	req := httptest.NewRequest(http.MethodPost, "/v1/users/batch", bytes.NewBufferString(`{"items":[{"account":"u1","name":"U1","email":"u1@example.com","passwordHash":"password-1234"}]}`))
	req = req.WithContext(security.WithJWTClaimsContext(req.Context(), &security.JWTClaims{Subject: "jwt-batch"}))
	resp := httptest.NewRecorder()

	h.createBatch(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	if len(svc.batchInput.Items) != 1 {
		t.Fatalf("expected one batch item, got %d", len(svc.batchInput.Items))
	}
	if svc.batchInput.Items[0].ActorID != "jwt-batch" {
		t.Fatalf("expected fallback actor id from jwt, got %q", svc.batchInput.Items[0].ActorID)
	}
}

func TestUserHandlerDisableActorIDFallback(t *testing.T) {
	svc := &fakeUserService{}
	h := &UserHandler{service: svc}

	req := httptest.NewRequest(http.MethodPost, "/v1/users/u-3/disable", bytes.NewBufferString(`{}`))
	req.SetPathValue("id", "u-3")
	req = req.WithContext(security.WithJWTClaimsContext(req.Context(), &security.JWTClaims{Subject: "jwt-disabler"}))
	resp := httptest.NewRecorder()

	h.disable(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	if svc.lastDisable.actorID != "jwt-disabler" {
		t.Fatalf("expected jwt actor id, got %q", svc.lastDisable.actorID)
	}

	var body apiv1.Response
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != "ok" {
		t.Fatalf("unexpected response code: %s", body.Code)
	}
}

func TestUserHandlerCreateBatch(t *testing.T) {
	svc := &fakeUserService{}
	h := &UserHandler{service: svc}

	req := httptest.NewRequest(http.MethodPost, "/v1/users/batch", bytes.NewBufferString(`{"items":[{"account":"u1","name":"U1","email":"u1@example.com","passwordHash":"password-1234"},{"account":"bad","name":"Bad","email":"bad@example.com","passwordHash":"password-1234"}]}`))
	resp := httptest.NewRecorder()

	h.createBatch(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}

	var body struct {
		Code string `json:"code"`
		Data struct {
			SuccessCount int `json:"successCount"`
			FailureCount int `json:"failureCount"`
			Results      []struct {
				Account string `json:"account"`
				Success bool   `json:"success"`
			} `json:"results"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != "ok" {
		t.Fatalf("unexpected response code: %s", body.Code)
	}
	if body.Data.SuccessCount != 1 || body.Data.FailureCount != 1 {
		t.Fatalf("unexpected counters: success=%d failure=%d", body.Data.SuccessCount, body.Data.FailureCount)
	}
	if len(body.Data.Results) != 2 {
		t.Fatalf("unexpected results length: %d", len(body.Data.Results))
	}
}

func TestUserHandlerCreateBatchAtomicFlag(t *testing.T) {
	svc := &fakeUserService{}
	h := &UserHandler{service: svc}

	req := httptest.NewRequest(http.MethodPost, "/v1/users/batch", bytes.NewBufferString(`{"atomic":true,"items":[{"account":"u1","name":"U1","email":"u1@example.com","passwordHash":"password-1234"}]}`))
	resp := httptest.NewRecorder()

	h.createBatch(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	if !svc.batchInput.Atomic {
		t.Fatalf("expected atomic batch flag to be true")
	}
}

func TestUserHandlerCreateBatchValidationErrors(t *testing.T) {
	svc := &fakeUserService{}
	h := &UserHandler{service: svc}

	t.Run("invalid json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/users/batch", bytes.NewBufferString(`{"items":`))
		resp := httptest.NewRecorder()

		h.createBatch(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", resp.Code, resp.Body.String())
		}
	})

	t.Run("empty items", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/users/batch", bytes.NewBufferString(`{"items":[]}`))
		resp := httptest.NewRecorder()

		h.createBatch(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", resp.Code, resp.Body.String())
		}
	})
}

func TestUserHandlerAssignRoleValidations(t *testing.T) {
	t.Run("rbac service not configured", func(t *testing.T) {
		h := &UserHandler{service: &fakeUserService{}, rbacService: nil}
		req := httptest.NewRequest(http.MethodPost, "/v1/users/u-1/roles", bytes.NewBufferString(`{"roleId":"r1"}`))
		req.SetPathValue("id", "u-1")
		resp := httptest.NewRecorder()

		h.assignRole(resp, req)

		if resp.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503, got %d body=%s", resp.Code, resp.Body.String())
		}
	})

	t.Run("missing role id", func(t *testing.T) {
		h := &UserHandler{service: &fakeUserService{}, rbacService: &fakeRBACService{}}
		req := httptest.NewRequest(http.MethodPost, "/v1/users/u-1/roles", bytes.NewBufferString(`{"scope":"self"}`))
		req.SetPathValue("id", "u-1")
		resp := httptest.NewRecorder()

		h.assignRole(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", resp.Code, resp.Body.String())
		}
	})

	t.Run("success with default scope", func(t *testing.T) {
		rbac := &fakeRBACService{}
		h := &UserHandler{service: &fakeUserService{}, rbacService: rbac}
		req := httptest.NewRequest(http.MethodPost, "/v1/users/u-1/roles", bytes.NewBufferString(`{"roleId":"r-1"}`))
		req.SetPathValue("id", "u-1")
		resp := httptest.NewRecorder()

		h.assignRole(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
		}
		if rbac.lastBind.Scope != domainrbac.DataScopeSelf {
			t.Fatalf("expected default self scope, got %q", rbac.lastBind.Scope)
		}
	})
}

func TestRegisterUserRoutesNilServiceNoRegistration(t *testing.T) {
	mux := http.NewServeMux()
	RegisterUserRoutes(mux, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/users", nil)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when service is nil, got %d", resp.Code)
	}
}

func TestUserHandlerUpdateEmailAuditDetail(t *testing.T) {
	audit := &fakeAuditService{}
	svc := &fakeUserService{
		getReturn:   &domainuser.User{ID: shared.ID("u-1"), Email: domainuser.Email("old@example.com")},
		emailReturn: &domainuser.User{ID: shared.ID("u-1"), Email: domainuser.Email("new@example.com")},
	}
	h := &UserHandler{service: svc, audit: audit}

	req := httptest.NewRequest(http.MethodPatch, "/v1/users/u-1/email", bytes.NewBufferString(`{"email":"new@example.com","actorId":""}`))
	req.SetPathValue("id", "u-1")
	req = req.WithContext(security.WithJWTClaimsContext(req.Context(), &security.JWTClaims{Subject: "actor-1"}))
	resp := httptest.NewRecorder()

	h.updateEmail(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	if audit.lastAction != "update_email" || audit.lastResource != "user" || audit.lastActorID != "actor-1" {
		t.Fatalf("unexpected audit metadata: actor=%q action=%q resource=%q", audit.lastActorID, audit.lastAction, audit.lastResource)
	}
	before, ok := audit.lastDetail["before"].(map[string]any)
	if !ok || strings.TrimSpace(fmt.Sprint(before["email"])) != "old@example.com" {
		t.Fatalf("unexpected before detail: %+v", audit.lastDetail)
	}
	after, ok := audit.lastDetail["after"].(map[string]any)
	if !ok || strings.TrimSpace(fmt.Sprint(after["email"])) != "new@example.com" {
		t.Fatalf("unexpected after detail: %+v", audit.lastDetail)
	}
}

func TestUserHandlerUpdateAuditDetail(t *testing.T) {
	audit := &fakeAuditService{}
	svc := &fakeUserService{
		getReturn:    &domainuser.User{ID: shared.ID("u-2"), Name: "old", Email: domainuser.Email("old@example.com"), Status: "active"},
		updateReturn: &domainuser.User{ID: shared.ID("u-2"), Name: "new", Email: domainuser.Email("new@example.com"), Status: "disabled"},
	}
	h := &UserHandler{service: svc, audit: audit}

	req := httptest.NewRequest(http.MethodPut, "/v1/users/u-2", bytes.NewBufferString(`{"name":"new","email":"new@example.com","status":"disabled"}`))
	req.SetPathValue("id", "u-2")
	resp := httptest.NewRecorder()

	h.update(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	if audit.lastAction != "update" || audit.lastResource != "user" {
		t.Fatalf("unexpected audit metadata: action=%q resource=%q", audit.lastAction, audit.lastResource)
	}
	before, ok := audit.lastDetail["before"].(map[string]any)
	if !ok || strings.TrimSpace(fmt.Sprint(before["name"])) != "old" || strings.TrimSpace(fmt.Sprint(before["status"])) != "active" {
		t.Fatalf("unexpected before detail: %+v", audit.lastDetail)
	}
	after, ok := audit.lastDetail["after"].(map[string]any)
	if !ok || strings.TrimSpace(fmt.Sprint(after["name"])) != "new" || strings.TrimSpace(fmt.Sprint(after["status"])) != "disabled" {
		t.Fatalf("unexpected after detail: %+v", audit.lastDetail)
	}
}
