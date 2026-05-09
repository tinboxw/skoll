package bootstrap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	domainrbac "github.com/tinboxw/skoll/internal/domain/rbac"
	rbacsvc "github.com/tinboxw/skoll/internal/service/rbac"
	"github.com/tinboxw/skoll/pkg/security"
)

type fakePermissionChecker struct {
	allowed map[string]bool
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
	policy := AuthPolicy{Enabled: true, SkipPaths: map[string]struct{}{"/health": {}}}
	h := authGuardMiddleware(policy, "test-secret", nil, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	noAuthReq := httptest.NewRequest(http.MethodGet, "/v1/users", nil)
	noAuthResp := httptest.NewRecorder()
	h.ServeHTTP(noAuthResp, noAuthReq)
	if noAuthResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing auth, got %d", noAuthResp.Code)
	}

	badReq := httptest.NewRequest(http.MethodGet, "/v1/users", nil)
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
	okReq := httptest.NewRequest(http.MethodGet, "/v1/users", nil)
	okReq.Header.Set("Authorization", "Bearer "+token)
	okResp := httptest.NewRecorder()
	h.ServeHTTP(okResp, okReq)
	if okResp.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid jwt, got %d", okResp.Code)
	}
}

func TestAuthGuardMiddlewarePermissionChecks(t *testing.T) {
	policy := AuthPolicy{Enabled: true, SkipPaths: map[string]struct{}{"/health": {}}}
	checker := &fakePermissionChecker{allowed: map[string]bool{
		"user:alice:user:read": true,
	}}
	h := authGuardMiddleware(policy, "test-secret", checker, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	reqAllowed := httptest.NewRequest(http.MethodGet, "/v1/users", nil)
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
	reqDenied := httptest.NewRequest(http.MethodGet, "/v1/users", nil)
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
	reqBypass := httptest.NewRequest(http.MethodGet, "/v1/users", nil)
	reqBypass.Header.Set("Authorization", "Bearer "+tokenSuper)
	respBypass := httptest.NewRecorder()
	h.ServeHTTP(respBypass, reqBypass)
	if respBypass.Code != http.StatusOK {
		t.Fatalf("expected 200 for super_admin bypass, got %d", respBypass.Code)
	}
}

func TestRequiredPermissionMapping(t *testing.T) {
	resource, action, guarded := requiredPermission(http.MethodDelete, "/v1/roles/abc")
	if !guarded || resource != "role" || action != "delete" {
		t.Fatalf("unexpected mapping got guarded=%v resource=%s action=%s", guarded, resource, action)
	}

	resource, action, guarded = requiredPermission(http.MethodPost, "/v1/rbac/check")
	if !guarded || resource != "permission" || action != "check" {
		t.Fatalf("unexpected rbac mapping got guarded=%v resource=%s action=%s", guarded, resource, action)
	}

	resource, action, guarded = requiredPermission(http.MethodGet, "/v1/plugins")
	if guarded || resource != "" || action != "" {
		t.Fatalf("plugin endpoint should not be guarded by RBAC mapping")
	}

	if len(defaultPermissionPolicies) < 3 {
		t.Fatalf("expected default permission policies configured")
	}

	if !isRoleBypass("super_admin") {
		t.Fatalf("expected super_admin bypass")
	}

	_ = domainrbac.SubjectUser
}
