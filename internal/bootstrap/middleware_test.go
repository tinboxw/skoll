package bootstrap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	domainrbac "github.com/tinboxw/skoll/internal/domain/rbac"
	httprouter "github.com/tinboxw/skoll/internal/handler/http"
	"github.com/tinboxw/skoll/internal/plugin"
	auditrepo "github.com/tinboxw/skoll/internal/repository/audit"
	rbacsvc "github.com/tinboxw/skoll/internal/service/rbac"
	"github.com/tinboxw/skoll/internal/store/memory"
	"github.com/tinboxw/skoll/pkg/logging"
	"github.com/tinboxw/skoll/pkg/security"
)

type fakePermissionChecker struct {
	allowed map[string]bool
}

type fakePluginManager struct {
	items map[string]plugin.Info
}

func (f *fakePluginManager) Install(path string) (plugin.Info, error) {
	_ = path
	return plugin.Info{}, nil
}

func (f *fakePluginManager) Enable(pluginID string) error {
	_ = pluginID
	return nil
}

func (f *fakePluginManager) Disable(pluginID string) error {
	_ = pluginID
	return nil
}

func (f *fakePluginManager) Uninstall(pluginID string) error {
	_ = pluginID
	return nil
}

func (f *fakePluginManager) List() []plugin.Info {
	items := make([]plugin.Info, 0, len(f.items))
	for _, item := range f.items {
		items = append(items, item)
	}
	return items
}

func (f *fakePluginManager) Get(pluginID string) (plugin.Info, error) {
	item, ok := f.items[pluginID]
	if !ok {
		return plugin.Info{}, plugin.ErrPluginNotFound
	}
	return item, nil
}

func (f *fakePermissionChecker) CheckPermission(_ context.Context, in rbacsvc.CheckPermissionInput) (bool, error) {
	if f == nil {
		return false, nil
	}
	key := string(in.SubjectType) + ":" + in.SubjectID + ":" + in.Resource + ":" + in.Action
	return f.allowed[key], nil
}

func TestParseBearerToken(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		token, err := parseBearerToken("Bearer abc.def.ghi")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if token != "abc.def.ghi" {
			t.Fatalf("unexpected token: %s", token)
		}
	})

	t.Run("missing", func(t *testing.T) {
		if _, err := parseBearerToken(""); err == nil {
			t.Fatalf("expected err")
		}
	})

	t.Run("wrong scheme", func(t *testing.T) {
		if _, err := parseBearerToken("Basic abc"); err == nil {
			t.Fatalf("expected err")
		}
	})
}

func TestAuthGuardMiddlewareValidatesJWT(t *testing.T) {
	policy := AuthPolicy{Enabled: true, SkipPaths: map[string]struct{}{"/skoll/health": {}}}
	h := authGuardMiddleware(policy, "/skoll", "test-secret", nil, nil, nil, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	noAuthReq := httptest.NewRequest(http.MethodGet, "/skoll/v1/users", nil)
	noAuthResp := httptest.NewRecorder()
	h.ServeHTTP(noAuthResp, noAuthReq)
	if noAuthResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing auth, got %d", noAuthResp.Code)
	}

	badReq := httptest.NewRequest(http.MethodGet, "/skoll/v1/users", nil)
	badReq.Header.Set("Authorization", "Bearer invalid.token")
	badResp := httptest.NewRecorder()
	h.ServeHTTP(badResp, badReq)
	if badResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for bad token, got %d", badResp.Code)
	}

	token, err := security.SignJWT("test-secret", security.JWTIdentity{Subject: "admin", Role: "super_admin", Roles: []string{"super_admin"}}, time.Hour, time.Now().UTC())
	if err != nil {
		t.Fatalf("sign jwt: %v", err)
	}
	okReq := httptest.NewRequest(http.MethodGet, "/skoll/v1/users", nil)
	okReq.Header.Set("Authorization", "Bearer "+token)
	okResp := httptest.NewRecorder()
	h.ServeHTTP(okResp, okReq)
	if okResp.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid jwt, got %d", okResp.Code)
	}
}

func TestAuthGuardMiddlewarePermissionChecks(t *testing.T) {
	policy := AuthPolicy{Enabled: true, SkipPaths: map[string]struct{}{"/skoll/health": {}}}
	checker := &fakePermissionChecker{allowed: map[string]bool{
		"user:alice:user:read": true,
	}}
	h := authGuardMiddleware(policy, "/skoll", "test-secret", checker, nil, nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if claims, ok := security.JWTClaimsFromContext(r.Context()); !ok || claims.Subject == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	tokenAlice, err := security.SignJWT("test-secret", security.JWTIdentity{Subject: "alice", Role: "editor", Roles: []string{"editor"}}, time.Hour, time.Now().UTC())
	if err != nil {
		t.Fatalf("sign jwt alice: %v", err)
	}
	reqAllowed := httptest.NewRequest(http.MethodGet, "/skoll/v1/users", nil)
	reqAllowed.Header.Set("Authorization", "Bearer "+tokenAlice)
	respAllowed := httptest.NewRecorder()
	h.ServeHTTP(respAllowed, reqAllowed)
	if respAllowed.Code != http.StatusOK {
		t.Fatalf("expected 200 for allowed permission, got %d", respAllowed.Code)
	}

	tokenBob, err := security.SignJWT("test-secret", security.JWTIdentity{Subject: "bob", Role: "editor", Roles: []string{"editor"}}, time.Hour, time.Now().UTC())
	if err != nil {
		t.Fatalf("sign jwt bob: %v", err)
	}
	reqDenied := httptest.NewRequest(http.MethodGet, "/skoll/v1/users", nil)
	reqDenied.Header.Set("Authorization", "Bearer "+tokenBob)
	respDenied := httptest.NewRecorder()
	h.ServeHTTP(respDenied, reqDenied)
	if respDenied.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for denied permission, got %d", respDenied.Code)
	}

	tokenSuper, err := security.SignJWT("test-secret", security.JWTIdentity{Subject: "root", Role: "super_admin", Roles: []string{"super_admin"}}, time.Hour, time.Now().UTC())
	if err != nil {
		t.Fatalf("sign jwt super: %v", err)
	}
	reqBypass := httptest.NewRequest(http.MethodGet, "/skoll/v1/users", nil)
	reqBypass.Header.Set("Authorization", "Bearer "+tokenSuper)
	respBypass := httptest.NewRecorder()
	h.ServeHTTP(respBypass, reqBypass)
	if respBypass.Code != http.StatusOK {
		t.Fatalf("expected 200 for super_admin bypass, got %d", respBypass.Code)
	}
}

func TestAuthGuardMiddlewareDataScopeSelfServiceRequiresAuthenticationOnly(t *testing.T) {
	policy := AuthPolicy{Enabled: true, SkipPaths: map[string]struct{}{}}
	h := authGuardMiddleware(policy, "/skoll", "test-secret", &fakePermissionChecker{}, nil, nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := security.JWTClaimsFromContext(r.Context())
		if !ok || claims.Subject != "alice" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	unauthenticated := httptest.NewRequest(http.MethodGet, "/skoll/v1/rbac/data-scope?resource=pharma_oa.customer&action=read", nil)
	unauthenticatedResponse := httptest.NewRecorder()
	h.ServeHTTP(unauthenticatedResponse, unauthenticated)
	if unauthenticatedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated data-scope status = %d, want %d", unauthenticatedResponse.Code, http.StatusUnauthorized)
	}

	token, err := security.SignJWT("test-secret", security.JWTIdentity{Subject: "alice", Role: "sales", Roles: []string{"sales"}}, time.Hour, time.Now().UTC())
	if err != nil {
		t.Fatalf("sign jwt: %v", err)
	}
	authenticated := httptest.NewRequest(http.MethodGet, "/skoll/v1/rbac/data-scope?resource=pharma_oa.customer&action=read", nil)
	authenticated.Header.Set("Authorization", "Bearer "+token)
	authenticatedResponse := httptest.NewRecorder()
	h.ServeHTTP(authenticatedResponse, authenticated)
	if authenticatedResponse.Code != http.StatusOK {
		t.Fatalf("authenticated data-scope status = %d, want %d", authenticatedResponse.Code, http.StatusOK)
	}
}

func TestAuthGuardMiddlewareAuditsPermissionDenied(t *testing.T) {
	policy := AuthPolicy{Enabled: true, SkipPaths: map[string]struct{}{"/skoll/health": {}}}
	checker := &fakePermissionChecker{allowed: map[string]bool{}}
	events := memory.NewAuditEventStore()
	h := authGuardMiddleware(policy, "/skoll", "test-secret", checker, nil, events, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	token, err := security.SignJWT("test-secret", security.JWTIdentity{Subject: "bob", Role: "editor", Roles: []string{"editor"}}, time.Hour, time.Now().UTC())
	if err != nil {
		t.Fatalf("sign jwt bob: %v", err)
	}
	req := httptest.NewRequest(http.MethodDelete, "/skoll/v1/roles/r1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Request-Id", "req-denied")
	resp := httptest.NewRecorder()

	h.ServeHTTP(resp, req)

	if resp.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for denied permission, got %d", resp.Code)
	}
	items, err := events.ListEvents(context.Background(), auditrepo.EventFilter{Type: domainaudit.EventTypeSecurity})
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one security event, got %d", len(items))
	}
	event := items[0]
	if event.Action != domainaudit.AuditAction("system.security.deny") ||
		event.Result != domainaudit.EventResultDenied ||
		event.Actor.ID.String() != "bob" ||
		event.Actor.Name != "editor" ||
		event.Resource.Type != "role" ||
		event.Resource.ID != "delete" ||
		event.Metadata["reason"] != "permission_denied" ||
		event.Trace.RequestID != "req-denied" {
		t.Fatalf("unexpected denied event: %+v", event)
	}
}

func TestRequiredPermissionMapping(t *testing.T) {
	resource, action, guarded := requiredPermission(http.MethodDelete, "/skoll/v1/roles/abc", "/skoll")
	if !guarded || resource != "role" || action != "delete" {
		t.Fatalf("unexpected mapping got guarded=%v resource=%s action=%s", guarded, resource, action)
	}

	resource, action, guarded = requiredPermission(http.MethodPost, "/skoll/v1/rbac/check", "/skoll")
	if !guarded || resource != "permission" || action != "check" {
		t.Fatalf("unexpected rbac mapping got guarded=%v resource=%s action=%s", guarded, resource, action)
	}

	resource, action, guarded = requiredPermission(http.MethodGet, "/skoll/v1/plugins", "/skoll")
	if guarded || resource != "" || action != "" {
		t.Fatalf("plugin endpoint should not be guarded by RBAC mapping")
	}

	if len(defaultPermissionPolicies("/skoll")) < 3 {
		t.Fatalf("expected default permission policies configured")
	}

	if !isRoleBypass("super_admin") {
		t.Fatalf("expected super_admin bypass")
	}

	_ = domainrbac.SubjectUser
}

func TestPharmaOACriticalPermissionMapping(t *testing.T) {
	tests := []struct {
		method   string
		path     string
		resource string
		action   string
	}{
		{http.MethodPost, "/skoll/v1/pharma-oa/purchase-requests/request-1/approve", "pharma_oa.purchase", "approve"},
		{http.MethodPost, "/skoll/v1/pharma-oa/contracts/contract-1/reject", "pharma_oa.contract", "reject"},
		{http.MethodPost, "/skoll/v1/pharma-oa/quality-complaints/complaint-1/resolve", "pharma_oa.quality_complaint", "resolve"},
		{http.MethodPost, "/skoll/v1/pharma-oa/purchase-inbounds", "pharma_oa.inbound", "create"},
		{http.MethodGet, "/skoll/v1/pharma-oa/sales-outbounds/outbound-1", "pharma_oa.sales.outbound", "read"},
		{http.MethodPost, "/skoll/v1/pharma-oa/stocktakes/stocktake-1/approve", "pharma_oa.stocktake", "approve"},
		{http.MethodPost, "/skoll/v1/pharma-oa/transfers", "pharma_oa.transfer", "create"},
		{http.MethodGet, "/skoll/v1/pharma-oa/customers?includeAll=true", "pharma_oa.customer", "read"},
		{http.MethodGet, "/skoll/v1/pharma-oa/customers/qualification-reminders", "pharma_oa.customer", "reminder"},
	}
	for _, test := range tests {
		path := strings.SplitN(test.path, "?", 2)[0]
		resource, action, guarded := requiredPermission(test.method, path, "/skoll")
		if !guarded || resource != test.resource || action != test.action {
			t.Errorf("%s %s: guarded=%v resource=%s action=%s", test.method, test.path, guarded, resource, action)
		}
	}
}

func TestAuthGuardMiddlewareEnforcesPharmaOACriticalPermission(t *testing.T) {
	policy := AuthPolicy{Enabled: true, SkipPaths: map[string]struct{}{}}
	checker := &fakePermissionChecker{allowed: map[string]bool{
		"user:approver:pharma_oa.purchase:approve": true,
	}}
	resolver, err := plugin.NewRoutePermissionRegistry([]plugin.RouteExtension{{
		Method: http.MethodPost, Path: "/v1/plugins/pharma_oa/api/purchase-requests/approve", Permission: "pharma_oa.purchase.approve", Source: "plugin.pharma_oa",
	}})
	if err != nil {
		t.Fatalf("create route permission resolver: %v", err)
	}
	h := authGuardMiddleware(policy, "/skoll", "test-secret", checker, resolver, nil, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	allowedToken, err := security.SignJWT("test-secret", security.JWTIdentity{Subject: "approver", Role: "manager", Roles: []string{"manager"}}, time.Hour, time.Now().UTC())
	if err != nil {
		t.Fatalf("sign allowed jwt: %v", err)
	}
	allowedReq := httptest.NewRequest(http.MethodPost, "/skoll/v1/pharma-oa/purchase-requests/request-1/approve", nil)
	allowedReq.Header.Set("Authorization", "Bearer "+allowedToken)
	allowedResp := httptest.NewRecorder()
	h.ServeHTTP(allowedResp, allowedReq)
	if allowedResp.Code != http.StatusOK {
		t.Fatalf("allowed status=%d body=%s", allowedResp.Code, allowedResp.Body.String())
	}
	deniedToken, err := security.SignJWT("test-secret", security.JWTIdentity{Subject: "viewer", Role: "employee", Roles: []string{"employee"}}, time.Hour, time.Now().UTC())
	if err != nil {
		t.Fatalf("sign denied jwt: %v", err)
	}
	deniedReq := httptest.NewRequest(http.MethodPost, "/skoll/v1/plugins/pharma_oa/api/purchase-requests/approve", nil)
	deniedReq.Header.Set("Authorization", "Bearer "+deniedToken)
	deniedResp := httptest.NewRecorder()
	h.ServeHTTP(deniedResp, deniedReq)
	if deniedResp.Code != http.StatusForbidden {
		t.Fatalf("denied status=%d body=%s", deniedResp.Code, deniedResp.Body.String())
	}
}

func TestAuthGuardMiddlewarePluginRouteFailsClosed(t *testing.T) {
	policy := AuthPolicy{Enabled: false, SkipPaths: map[string]struct{}{}}
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	path := "/skoll/v1/plugins/demo/api/items"

	anonymous := authGuardMiddleware(policy, "/skoll", "test-secret", nil, nil, nil, next)
	anonymousResp := httptest.NewRecorder()
	anonymous.ServeHTTP(anonymousResp, httptest.NewRequest(http.MethodGet, path, nil))
	if anonymousResp.Code != http.StatusUnauthorized {
		t.Fatalf("plugin API must require JWT when global auth is disabled: status=%d body=%s", anonymousResp.Code, anonymousResp.Body.String())
	}

	token, err := security.SignJWT("test-secret", security.JWTIdentity{Subject: "alice", Role: "editor", Roles: []string{"editor"}}, time.Hour, time.Now().UTC())
	if err != nil {
		t.Fatalf("sign editor jwt: %v", err)
	}
	events := memory.NewAuditEventStore()
	missingResolver := authGuardMiddleware(policy, "/skoll", "test-secret", nil, nil, events, next)
	missingReq := httptest.NewRequest(http.MethodGet, path, nil)
	missingReq.Header.Set("Authorization", "Bearer "+token)
	missingResp := httptest.NewRecorder()
	missingResolver.ServeHTTP(missingResp, missingReq)
	if missingResp.Code != http.StatusForbidden || !strings.Contains(missingResp.Body.String(), "插件路由权限不可用") {
		t.Fatalf("missing resolver status=%d body=%s", missingResp.Code, missingResp.Body.String())
	}
	assertPermissionDeniedReason(t, events, "plugin.route", "resolve", "plugin_route_resolver_not_configured")

	emptyResolver, err := plugin.NewRoutePermissionRegistry(nil)
	if err != nil {
		t.Fatalf("create empty route permission resolver: %v", err)
	}
	superToken, err := security.SignJWT("test-secret", security.JWTIdentity{Subject: "root", Role: "super_admin", Roles: []string{"super_admin"}}, time.Hour, time.Now().UTC())
	if err != nil {
		t.Fatalf("sign super admin jwt: %v", err)
	}
	undeclared := authGuardMiddleware(policy, "/skoll", "test-secret", nil, emptyResolver, nil, next)
	undeclaredReq := httptest.NewRequest(http.MethodGet, path, nil)
	undeclaredReq.Header.Set("Authorization", "Bearer "+superToken)
	undeclaredReq.Header.Set("Accept-Language", "en-US")
	undeclaredResp := httptest.NewRecorder()
	undeclared.ServeHTTP(undeclaredResp, undeclaredReq)
	if undeclaredResp.Code != http.StatusForbidden || !strings.Contains(undeclaredResp.Body.String(), "plugin route permission is not declared") {
		t.Fatalf("undeclared super admin status=%d body=%s", undeclaredResp.Code, undeclaredResp.Body.String())
	}
}

func TestAuthGuardMiddlewarePluginRoutePermissionMatrixAndCustomPrefix(t *testing.T) {
	resolver, err := plugin.NewRoutePermissionRegistry([]plugin.RouteExtension{{
		Method: http.MethodGet, Path: "/v1/plugins/demo/api/items", Permission: "demo.items.read", AuditAction: "demo.items.read", Source: "plugin.demo",
	}})
	if err != nil {
		t.Fatalf("create route permission resolver: %v", err)
	}
	checker := &fakePermissionChecker{allowed: map[string]bool{
		"user:alice:demo.items:read": true,
	}}
	events := memory.NewAuditEventStore()
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	h := authGuardMiddleware(AuthPolicy{Enabled: true, SkipPaths: map[string]struct{}{}}, "/gateway", "test-secret", checker, resolver, events, next)

	aliceToken, err := security.SignJWT("test-secret", security.JWTIdentity{Subject: "alice", Role: "editor", Roles: []string{"editor"}}, time.Hour, time.Now().UTC())
	if err != nil {
		t.Fatalf("sign alice jwt: %v", err)
	}
	allowedReq := httptest.NewRequest(http.MethodGet, "/gateway/v1/plugins/demo/api/items", nil)
	allowedReq.Header.Set("Authorization", "Bearer "+aliceToken)
	allowedResp := httptest.NewRecorder()
	h.ServeHTTP(allowedResp, allowedReq)
	if allowedResp.Code != http.StatusOK {
		t.Fatalf("allowed status=%d body=%s", allowedResp.Code, allowedResp.Body.String())
	}
	assertPluginRouteAudit(t, events, "demo.items.read", "demo.items.read", http.StatusOK)

	bobToken, err := security.SignJWT("test-secret", security.JWTIdentity{Subject: "bob", Role: "viewer", Roles: []string{"viewer"}}, time.Hour, time.Now().UTC())
	if err != nil {
		t.Fatalf("sign bob jwt: %v", err)
	}
	deniedReq := httptest.NewRequest(http.MethodGet, "/gateway/v1/plugins/demo/api/items", nil)
	deniedReq.Header.Set("Authorization", "Bearer "+bobToken)
	deniedResp := httptest.NewRecorder()
	h.ServeHTTP(deniedResp, deniedReq)
	if deniedResp.Code != http.StatusForbidden || !strings.Contains(deniedResp.Body.String(), "插件路由权限不足") {
		t.Fatalf("denied status=%d body=%s", deniedResp.Code, deniedResp.Body.String())
	}
	assertPermissionDeniedReason(t, events, "demo.items", "read", "permission_denied")

	superToken, err := security.SignJWT("test-secret", security.JWTIdentity{Subject: "root", Role: "super_admin", Roles: []string{"super_admin"}}, time.Hour, time.Now().UTC())
	if err != nil {
		t.Fatalf("sign super admin jwt: %v", err)
	}
	superReq := httptest.NewRequest(http.MethodGet, "/gateway/v1/plugins/demo/api/items", nil)
	superReq.Header.Set("Authorization", "Bearer "+superToken)
	superResp := httptest.NewRecorder()
	h.ServeHTTP(superResp, superReq)
	if superResp.Code != http.StatusOK {
		t.Fatalf("declared super admin route status=%d body=%s", superResp.Code, superResp.Body.String())
	}
}

func TestPluginRoutePermissionHelpers(t *testing.T) {
	for _, tt := range []struct {
		permission string
		resource   string
		action     string
		ok         bool
	}{
		{permission: "pharma_oa.sales.outbound.read", resource: "pharma_oa.sales.outbound", action: "read", ok: true},
		{permission: "reports:invoice:export", resource: "reports:invoice", action: "export", ok: true},
		{permission: "invalid", ok: false},
	} {
		resource, action, ok := splitPermissionKey(tt.permission)
		if resource != tt.resource || action != tt.action || ok != tt.ok {
			t.Errorf("splitPermissionKey(%q) = (%q, %q, %v)", tt.permission, resource, action, ok)
		}
	}
	if path, ok := pluginBusinessRoutePath("/gateway/v1/plugins/demo/api/items", "/gateway"); !ok || path != "/v1/plugins/demo/api/items" {
		t.Fatalf("unexpected custom-prefix plugin path: %q, ok=%v", path, ok)
	}
	for _, path := range []string{
		"/gateway/v1/plugins/demo/page",
		"/gateway/v1/plugins/demo/assets/app.js",
		"/gateway/v1/plugins",
		"/gateway/v1/plugins/demo/api",
	} {
		if _, ok := pluginBusinessRoutePath(path, "/gateway"); ok {
			t.Errorf("path must not be treated as plugin business API: %s", path)
		}
	}
}

func TestPluginManagerBuildsRoutePermissionsFromEnabledRuntimeManifests(t *testing.T) {
	manager := &pluginManagerWithExtensions{
		Manager: &fakePluginManager{items: map[string]plugin.Info{
			"enabled": {
				ID: "enabled", State: plugin.StateEnabled,
				APIContract: &plugin.APIContract{Routes: []plugin.APIRoute{{Method: http.MethodGet, Path: "/v1/plugins/enabled/api/items", Permission: "enabled.items.read", AuditAction: "enabled.items.read"}}},
			},
			"disabled": {
				ID: "disabled", State: plugin.StateDisabled,
				APIContract: &plugin.APIContract{Routes: []plugin.APIRoute{{Method: http.MethodGet, Path: "/v1/plugins/disabled/api/items", Permission: "disabled.items.read", AuditAction: "disabled.items.read"}}},
			},
		}},
		routePermissions: mustEmptyRoutePermissionRegistry(),
	}
	if err := manager.refreshRoutePermissions(); err != nil {
		t.Fatalf("refresh route permissions: %v", err)
	}
	descriptor, ok := manager.ResolveRoutePermission(http.MethodGet, "/v1/plugins/enabled/api/items")
	if !ok || descriptor.Permission != "enabled.items.read" {
		t.Fatalf("unexpected enabled descriptor: %#v, ok=%v", descriptor, ok)
	}
	if _, ok := manager.ResolveRoutePermission(http.MethodGet, "/v1/plugins/disabled/api/items"); ok {
		t.Fatal("disabled plugin route must not enter permission resolver")
	}
}

func assertPermissionDeniedReason(t *testing.T, events *memory.AuditEventStore, resource, action, reason string) {
	t.Helper()
	items, err := events.ListEvents(context.Background(), auditrepo.EventFilter{Type: domainaudit.EventTypeSecurity})
	if err != nil {
		t.Fatalf("list permission denied events: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("permission denied events = %d, want 1", len(items))
	}
	event := items[0]
	if event.Resource.Type != resource || event.Resource.ID != action || event.Metadata["reason"] != reason {
		t.Fatalf("unexpected permission denied event: %+v", event)
	}
}

func assertPluginRouteAudit(t *testing.T, events *memory.AuditEventStore, auditAction, permission string, status int) {
	t.Helper()
	items, err := events.ListEvents(context.Background(), auditrepo.EventFilter{Type: domainaudit.EventTypePlugin})
	if err != nil {
		t.Fatalf("list plugin route audit events: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("plugin route audit events = %d, want 1", len(items))
	}
	event := items[0]
	if event.Action != domainaudit.AuditAction(auditAction) || event.Result != domainaudit.EventResultSuccess || event.Metadata["permission"] != permission || event.Metadata["status"] != status {
		t.Fatalf("unexpected plugin route audit event: %+v", event)
	}
}

func TestBuildMiddlewareChainPluginPageBypassesAuth(t *testing.T) {
	tmp := t.TempDir()
	pluginDir := filepath.Join(tmp, "demo-frontend")
	if err := os.MkdirAll(filepath.Join(pluginDir, "static"), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "static", "index.html"), []byte("<html><head></head><body>demo</body></html>"), 0o644); err != nil {
		t.Fatalf("write index failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "static", "app.js"), []byte("console.log('ok')"), 0o644); err != nil {
		t.Fatalf("write app.js failed: %v", err)
	}

	router := httprouter.NewRouter(httprouter.Dependencies{PluginManager: &fakePluginManager{items: map[string]plugin.Info{
		"demo-frontend": {
			ID:            "demo-frontend",
			Name:          "Demo Frontend",
			Version:       "0.1.0",
			State:         plugin.StateEnabled,
			Source:        pluginDir,
			UIMode:        plugin.UIModeFrontendOnly,
			FrontendEntry: "/plugins/demo-frontend",
		},
	}}})

	policy := AuthPolicy{Enabled: true, SkipPaths: map[string]struct{}{
		"/skoll/health":        {},
		"/skoll/ready":         {},
		"/skoll/v1/plugins":    {},
		"/skoll/v1/auth/login": {},
	}}
	guarded := buildMiddlewareChain(router, logging.Discard(), policy, "/skoll", "test-secret", nil, nil, nil)

	pageReq := httptest.NewRequest(http.MethodGet, "/skoll/v1/plugins/demo-frontend/page", nil)
	pageResp := httptest.NewRecorder()
	guarded.ServeHTTP(pageResp, pageReq)
	if pageResp.Code != http.StatusOK {
		t.Fatalf("page status=%d body=%s", pageResp.Code, pageResp.Body.String())
	}

	assetReq := httptest.NewRequest(http.MethodGet, "/skoll/v1/plugins/demo-frontend/assets/app.js", nil)
	assetResp := httptest.NewRecorder()
	guarded.ServeHTTP(assetResp, assetReq)
	if assetResp.Code != http.StatusOK {
		t.Fatalf("asset status=%d body=%s", assetResp.Code, assetResp.Body.String())
	}

	pluginAPIReq := httptest.NewRequest(http.MethodPost, "/skoll/v1/plugins/demo-frontend/api/action", nil)
	pluginAPIResp := httptest.NewRecorder()
	guarded.ServeHTTP(pluginAPIResp, pluginAPIReq)
	if pluginAPIResp.Code != http.StatusUnauthorized {
		t.Fatalf("plugin API status=%d body=%s", pluginAPIResp.Code, pluginAPIResp.Body.String())
	}

	protectedReq := httptest.NewRequest(http.MethodGet, "/skoll/v1/system/settings", nil)
	protectedResp := httptest.NewRecorder()
	guarded.ServeHTTP(protectedResp, protectedReq)
	if protectedResp.Code != http.StatusUnauthorized {
		t.Fatalf("protected status=%d body=%s", protectedResp.Code, protectedResp.Body.String())
	}
}
