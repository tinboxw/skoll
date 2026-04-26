package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tinboxw/skoll/internal/integration/adminauth"
	"github.com/tinboxw/skoll/internal/module/apiregistry"
	"github.com/tinboxw/skoll/internal/module/audit"
	"github.com/tinboxw/skoll/internal/module/config"
	"github.com/tinboxw/skoll/internal/module/dictionary"
	"github.com/tinboxw/skoll/internal/module/fileservice"
	"github.com/tinboxw/skoll/internal/module/jobscheduler"
	"github.com/tinboxw/skoll/internal/module/menu"
	"github.com/tinboxw/skoll/internal/module/modgenerator"
	"github.com/tinboxw/skoll/internal/module/pluginmgr"
	"github.com/tinboxw/skoll/internal/module/rbac"
	"github.com/tinboxw/skoll/internal/module/role"
	"github.com/tinboxw/skoll/internal/module/user"
)

func TestRBACE2ESmoke_IdentityRoleAndAPIAuthorization(t *testing.T) {
	userSvc := user.NewService()
	roleSvc := role.NewService()
	menuSvc := menu.NewService()
	auditSvc := audit.NewService()
	configSvc := config.NewService()
	dictionarySvc := dictionary.NewService()
	fileSvc := fileservice.NewService(&memoryFileBackend{})
	jobSvc := jobscheduler.NewService()
	genSvc := modgenerator.NewService()
	pluginSvc := pluginmgr.NewService()
	rbacSvc := rbac.NewService()
	apiSvc := apiregistry.NewService()

	userSvc.Create("alice", "alice@example.com")
	adminRole := roleSvc.Create("admin", []string{"user.read"})

	verifier, enabled, err := adminauth.ResolveVerifier("static-token", "secret", "")
	if err != nil {
		t.Fatalf("resolve admin auth verifier failed: %v", err)
	}
	if !enabled {
		t.Fatalf("expected static-token verifier enabled")
	}

	srv := New(":0", "test-version")
	wrapper := func(next http.Handler) http.Handler {
		return adminauth.WithVerifier(WithRoleAPIAuthorizer(next, roleSvc, rbacSvc), verifier)
	}
	srv.MountAdminModuleRoutes(AdminModuleServices{
		Users:        userSvc,
		Roles:        roleSvc,
		Menus:        menuSvc,
		Audit:        auditSvc,
		Configs:      configSvc,
		Dictionaries: dictionarySvc,
		Files:        fileSvc,
		Jobs:         jobSvc,
		Generator:    genSvc,
		Plugins:      pluginSvc,
		RBAC:         rbacSvc,
		APIs:         apiSvc,
	}, wrapper)

	rbacSvc.SetRoleAPIs(adminRole.ID, []string{"GET:/admin/v1/users", "GET:/admin/v1/users/{id}"})

	unauthReq := httptest.NewRequest(http.MethodGet, "/admin/v1/users", nil)
	unauthRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(unauthRR, unauthReq)
	if unauthRR.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized without token, got %d", unauthRR.Code)
	}

	missingRoleReq := httptest.NewRequest(http.MethodGet, "/admin/v1/users", nil)
	missingRoleReq.Header.Set(adminauth.HeaderToken, "secret")
	missingRoleRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(missingRoleRR, missingRoleReq)
	if missingRoleRR.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized without role id, got %d", missingRoleRR.Code)
	}

	allowedReq := httptest.NewRequest(http.MethodGet, "/admin/v1/users", nil)
	allowedReq.Header.Set(adminauth.HeaderToken, "secret")
	allowedReq.Header.Set(HeaderAdminRoleID, "1")
	allowedRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(allowedRR, allowedReq)
	if allowedRR.Code != http.StatusOK {
		t.Fatalf("expected authorized list users status 200, got %d", allowedRR.Code)
	}
	if !strings.Contains(allowedRR.Body.String(), "alice@example.com") {
		t.Fatalf("expected user data in authorized response, got %s", allowedRR.Body.String())
	}

	allowedGetReq := httptest.NewRequest(http.MethodGet, "/admin/v1/users/1", nil)
	allowedGetReq.Header.Set(adminauth.HeaderToken, "secret")
	allowedGetReq.Header.Set(HeaderAdminRoleID, "1")
	allowedGetRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(allowedGetRR, allowedGetReq)
	if allowedGetRR.Code != http.StatusOK {
		t.Fatalf("expected authorized get user status 200, got %d", allowedGetRR.Code)
	}

	forbiddenReq := httptest.NewRequest(http.MethodPost, "/admin/v1/users", strings.NewReader(`{"name":"bob","email":"bob@example.com"}`))
	forbiddenReq.Header.Set("Content-Type", "application/json")
	forbiddenReq.Header.Set(adminauth.HeaderToken, "secret")
	forbiddenReq.Header.Set(HeaderAdminRoleID, "1")
	forbiddenRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(forbiddenRR, forbiddenReq)
	if forbiddenRR.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden for unbound api, got %d", forbiddenRR.Code)
	}
}

func TestRBACE2ESmoke_RoleAuthorizationViaJWTBridgeRoleID(t *testing.T) {
	userSvc := user.NewService()
	roleSvc := role.NewService()
	menuSvc := menu.NewService()
	auditSvc := audit.NewService()
	configSvc := config.NewService()
	dictionarySvc := dictionary.NewService()
	fileSvc := fileservice.NewService(&memoryFileBackend{})
	jobSvc := jobscheduler.NewService()
	genSvc := modgenerator.NewService()
	pluginSvc := pluginmgr.NewService()
	rbacSvc := rbac.NewService()
	apiSvc := apiregistry.NewService()

	userSvc.Create("alice", "alice@example.com")
	adminRole := roleSvc.Create("admin", []string{"user.read"})

	verifier, enabled, err := adminauth.ResolveVerifier("static-token", "secret", "")
	if err != nil {
		t.Fatalf("resolve admin auth verifier failed: %v", err)
	}
	if !enabled {
		t.Fatalf("expected static-token verifier enabled")
	}

	srv := New(":0", "test-version")
	wrapper := func(next http.Handler) http.Handler {
		return adminauth.WithVerifier(WithRoleAPIAuthorizer(next, roleSvc, rbacSvc), verifier)
	}
	srv.MountAdminModuleRoutes(AdminModuleServices{
		Users:        userSvc,
		Roles:        roleSvc,
		Menus:        menuSvc,
		Audit:        auditSvc,
		Configs:      configSvc,
		Dictionaries: dictionarySvc,
		Files:        fileSvc,
		Jobs:         jobSvc,
		Generator:    genSvc,
		Plugins:      pluginSvc,
		RBAC:         rbacSvc,
		APIs:         apiSvc,
	}, wrapper)

	rbacSvc.SetRoleAPIs(adminRole.ID, []string{"GET:/admin/v1/users"})

	req := httptest.NewRequest(http.MethodGet, "/admin/v1/users", nil)
	req.Header.Set(adminauth.HeaderToken, "secret")
	req.Header.Set(HeaderAdminJWTRoleID, "1")
	rr := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected authorized list users via jwt role bridge, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "alice@example.com") {
		t.Fatalf("expected user data in authorized response, got %s", rr.Body.String())
	}
}
