package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	domainorganization "github.com/tinboxw/skoll/internal/domain/organization"
	"github.com/tinboxw/skoll/internal/domain/shared"
	domainuser "github.com/tinboxw/skoll/internal/domain/user"
	auditrepo "github.com/tinboxw/skoll/internal/repository/audit"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	"github.com/tinboxw/skoll/internal/store/memory"
	"github.com/tinboxw/skoll/pkg/logging"
	"github.com/tinboxw/skoll/pkg/security"
)

type loginResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Token string `json:"token"`
		User  struct {
			Account          string   `json:"account"`
			Role             string   `json:"role"`
			Roles            []string `json:"roles"`
			OrganizationID   string   `json:"organizationId"`
			OrganizationPath []string `json:"organizationPath"`
		} `json:"user"`
	} `json:"data"`
}

func newSeededAuthHandler(t *testing.T) (*builtinAuthHandler, *memory.UserStore) {
	t.Helper()

	users := memory.NewUserStore()
	roles := memory.NewRoleStore()
	rbac := memory.NewRBACStore()
	organizations := memory.NewOrganizationStore()
	ensureBuiltinAuthData(context.Background(), logging.Discard(), users, roles, rbac)

	h := newBuiltinAuthHandler("test-secret", users, roles, rbac, organizations, nil, nil, logging.Discard())
	if h == nil {
		t.Fatal("expected builtin auth handler")
	}
	return h, users
}

func newSeededAuthHandlerWithEvents(t *testing.T) (*builtinAuthHandler, *memory.AuditEventStore) {
	t.Helper()

	users := memory.NewUserStore()
	roles := memory.NewRoleStore()
	rbac := memory.NewRBACStore()
	organizations := memory.NewOrganizationStore()
	events := memory.NewAuditEventStore()
	ensureBuiltinAuthData(context.Background(), logging.Discard(), users, roles, rbac)

	h := newBuiltinAuthHandler("test-secret", users, roles, rbac, organizations, nil, auditsvc.NewEventService(events), logging.Discard())
	if h == nil {
		t.Fatal("expected builtin auth handler")
	}
	return h, events
}

func TestBuiltinAuthLoginSuccessWithSeededAdminCredentials(t *testing.T) {
	h, users := newSeededAuthHandler(t)

	seeded, err := users.GetByAccount(context.Background(), "admin")
	if err != nil {
		t.Fatalf("load seeded admin: %v", err)
	}
	if seeded == nil {
		t.Fatal("expected seeded admin user")
	}
	if !domainuser.VerifyPassword("Admin@123456", seeded.Password) {
		t.Fatalf("expected seeded admin password to match")
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(`{"account":"admin","password":"Admin@123456"}`))
	resp := httptest.NewRecorder()

	h.handleLogin(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var body loginResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if body.Data.Token == "" {
		t.Fatalf("expected token in response")
	}
	if body.Data.User.Account != "admin" {
		t.Fatalf("expected admin account in response, got %q", body.Data.User.Account)
	}
}

func TestBuiltinAuthLoginRejectsWrongPassword(t *testing.T) {
	h, _ := newSeededAuthHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(`{"account":"admin","password":"admin"}`))
	resp := httptest.NewRecorder()

	h.handleLogin(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestBuiltinAuthLoginRejectsIdentifierFieldPayload(t *testing.T) {
	h, _ := newSeededAuthHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(`{"identifier":"admin","password":"Admin@123456"}`))
	resp := httptest.NewRecorder()

	h.handleLogin(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestBuiltinAuthLoginSignsTrustedOrganizationAndRoleClaims(t *testing.T) {
	users := memory.NewUserStore()
	roles := memory.NewRoleStore()
	rbac := memory.NewRBACStore()
	organizations := memory.NewOrganizationStore()
	ensureBuiltinAuthData(context.Background(), logging.Discard(), users, roles, rbac)

	now := time.Now().UTC()
	root, err := domainorganization.NewDepartment(domainorganization.DepartmentInput{
		ID: shared.ID("org-root"), Code: "root", Name: "Headquarters", Status: domainorganization.StatusEnabled, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("create root organization: %v", err)
	}
	child, err := domainorganization.NewDepartment(domainorganization.DepartmentInput{
		ID: shared.ID("org-sales"), ParentID: root.ID, Code: "sales", Name: "Sales", Status: domainorganization.StatusEnabled, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("create child organization: %v", err)
	}
	if err := organizations.SaveDepartment(context.Background(), root); err != nil {
		t.Fatalf("save root organization: %v", err)
	}
	if err := organizations.SaveDepartment(context.Background(), child); err != nil {
		t.Fatalf("save child organization: %v", err)
	}
	admin, err := users.GetByAccount(context.Background(), "admin")
	if err != nil || admin == nil {
		t.Fatalf("load seeded admin: %v", err)
	}
	admin.SetOrganization(child.ID.String(), "", now)
	if err := users.Save(context.Background(), admin); err != nil {
		t.Fatalf("save admin organization: %v", err)
	}

	h := newBuiltinAuthHandler("test-secret", users, roles, rbac, organizations, nil, nil, logging.Discard())
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(`{"account":"admin","password":"Admin@123456","organizationId":"forged","organizationPath":["forged"],"role":"forged","roles":["forged"]}`))
	resp := httptest.NewRecorder()
	h.handleLogin(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var body loginResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	claims, err := security.ParseJWT("test-secret", body.Data.Token)
	if err != nil {
		t.Fatalf("parse login token: %v", err)
	}
	if claims.Subject != admin.ID.String() || claims.OrganizationID != "org-sales" || claims.Role != "super_admin" {
		t.Fatalf("unexpected trusted claims: %+v", claims)
	}
	if len(claims.OrganizationPath) != 2 || claims.OrganizationPath[0] != "org-root" || claims.OrganizationPath[1] != "org-sales" {
		t.Fatalf("unexpected organization path: %v", claims.OrganizationPath)
	}
	if len(claims.Roles) != 1 || claims.Roles[0] != "super_admin" {
		t.Fatalf("unexpected role claims: %v", claims.Roles)
	}
	if body.Data.User.OrganizationID != claims.OrganizationID || len(body.Data.User.OrganizationPath) != 2 || len(body.Data.User.Roles) != 1 {
		t.Fatalf("login profile does not match token claims: %+v", body.Data.User)
	}
}

func TestBuiltinAuthLoginRejectsBrokenTrustedOrganization(t *testing.T) {
	h, users := newSeededAuthHandler(t)
	admin, err := users.GetByAccount(context.Background(), "admin")
	if err != nil || admin == nil {
		t.Fatalf("load seeded admin: %v", err)
	}
	admin.SetOrganization("missing-organization", "", time.Now().UTC())
	if err := users.Save(context.Background(), admin); err != nil {
		t.Fatalf("save admin organization: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(`{"account":"admin","password":"Admin@123456"}`))
	resp := httptest.NewRecorder()
	h.handleLogin(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", resp.Code, resp.Body.String())
	}
	var body loginResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body.Code != "invalid_organization" {
		t.Fatalf("expected invalid_organization, got %q", body.Code)
	}
}

func TestBuiltinAuthLoginWritesSuccessEvent(t *testing.T) {
	h, events := newSeededAuthHandlerWithEvents(t)

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(`{"account":"admin","password":"Admin@123456"}`))
	req.RemoteAddr = "203.0.113.10:4567"
	req.Header.Set("User-Agent", "auth-test")
	req.Header.Set("X-Request-Id", "req-login")
	resp := httptest.NewRecorder()

	h.handleLogin(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}
	items, err := events.ListEvents(context.Background(), auditrepo.EventFilter{Type: domainaudit.EventTypeLogin})
	if err != nil {
		t.Fatalf("list login events: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one login event, got %d", len(items))
	}
	event := items[0]
	if event.Action != domainaudit.AuditAction("auth.session.login") ||
		event.Result != domainaudit.EventResultSuccess ||
		event.Risk != domainaudit.EventRiskLow ||
		event.Actor.Type != "user" ||
		event.Resource.Type != "auth_session" ||
		event.Trace.RequestID != "req-login" ||
		event.Trace.IP != "203.0.113.10" ||
		event.SourceData["kind"] != "login_log" ||
		event.SourceData["account"] != "admin" ||
		event.SourceData["result"] != "success" {
		t.Fatalf("unexpected login event: %+v", event)
	}
}

func TestBuiltinAuthLoginWritesFailureEvent(t *testing.T) {
	h, events := newSeededAuthHandlerWithEvents(t)

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(`{"account":"admin","password":"bad"}`))
	resp := httptest.NewRecorder()

	h.handleLogin(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", resp.Code, resp.Body.String())
	}
	items, err := events.ListEvents(context.Background(), auditrepo.EventFilter{Type: domainaudit.EventTypeLogin})
	if err != nil {
		t.Fatalf("list login events: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one login event, got %d", len(items))
	}
	event := items[0]
	if event.Action != domainaudit.AuditAction("auth.session.login_failed") ||
		event.Result != domainaudit.EventResultFailure ||
		event.Risk != domainaudit.EventRiskMedium ||
		event.Actor.Type != "account" ||
		event.Actor.ID.String() != "admin" ||
		event.SourceData["failureReason"] != "invalid_credentials" ||
		event.SourceData["result"] != "failure" {
		t.Fatalf("unexpected login failure event: %+v", event)
	}
}
