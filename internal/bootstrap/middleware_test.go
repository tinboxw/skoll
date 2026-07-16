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
	h := authGuardMiddleware(policy, "/skoll", "test-secret", nil, nil, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

	token, err := security.SignJWT("test-secret", "admin", "super_admin", time.Hour, time.Now().UTC())
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
	h := authGuardMiddleware(policy, "/skoll", "test-secret", checker, nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if claims, ok := security.JWTClaimsFromContext(r.Context()); !ok || claims.Subject == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	tokenAlice, err := security.SignJWT("test-secret", "alice", "editor", time.Hour, time.Now().UTC())
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

	tokenBob, err := security.SignJWT("test-secret", "bob", "editor", time.Hour, time.Now().UTC())
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

	tokenSuper, err := security.SignJWT("test-secret", "root", "super_admin", time.Hour, time.Now().UTC())
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

func TestAuthGuardMiddlewareAuditsPermissionDenied(t *testing.T) {
	policy := AuthPolicy{Enabled: true, SkipPaths: map[string]struct{}{"/skoll/health": {}}}
	checker := &fakePermissionChecker{allowed: map[string]bool{}}
	events := memory.NewAuditEventStore()
	h := authGuardMiddleware(policy, "/skoll", "test-secret", checker, events, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	token, err := security.SignJWT("test-secret", "bob", "editor", time.Hour, time.Now().UTC())
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
		{http.MethodPost, "/skoll/v1/plugins/pharma_oa/api/stocktakes/reject", "pharma_oa.stocktake", "reject"},
		{http.MethodPost, "/skoll/v1/plugins/pharma_oa/api/purchase-requests/approve", "pharma_oa.purchase", "approve"},
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
	h := authGuardMiddleware(policy, "/skoll", "test-secret", checker, nil, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	allowedToken, err := security.SignJWT("test-secret", "approver", "manager", time.Hour, time.Now().UTC())
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

	deniedToken, err := security.SignJWT("test-secret", "viewer", "employee", time.Hour, time.Now().UTC())
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
	guarded := buildMiddlewareChain(router, logging.Discard(), policy, "/skoll", "test-secret", nil, nil)

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
