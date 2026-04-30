package app

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
	"github.com/tinboxw/skoll/internal/module/releasegov"
	"github.com/tinboxw/skoll/internal/module/role"
	"github.com/tinboxw/skoll/internal/module/user"
)

type memoryFileBackend struct {
	items map[string][]byte
}

func (b *memoryFileBackend) Save(_ string, content []byte) (string, error) {
	if b.items == nil {
		b.items = make(map[string][]byte)
	}
	key := fmt.Sprintf("f-%d", len(b.items)+1)
	b.items[key] = append([]byte(nil), content...)
	return key, nil
}

func (b *memoryFileBackend) Open(storageKey string) ([]byte, error) {
	out, ok := b.items[storageKey]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return append([]byte(nil), out...), nil
}

func testAdminModuleServices() AdminModuleServices {
	return AdminModuleServices{
		Users:        user.NewService(),
		Roles:        role.NewService(),
		Menus:        menu.NewService(),
		Audit:        audit.NewService(),
		Configs:      config.NewService(),
		Dictionaries: dictionary.NewService(),
		Files:        fileservice.NewService(&memoryFileBackend{}),
		Jobs:         jobscheduler.NewService(),
		Generator:    modgenerator.NewService(),
		Plugins:      pluginmgr.NewService(),
		RBAC:         rbac.NewService(),
		APIs:         apiregistry.NewService(),
		Releases:     releasegov.NewService(),
	}
}

func TestAdminUserRoutes_CreateListGet(t *testing.T) {
	srv := New(":0", "test-version")
	srv.MountAdminModuleRoutes(testAdminModuleServices(), nil)

	createReq := httptest.NewRequest(http.MethodPost, "/admin/v1/users", strings.NewReader(`{"name":"alice","email":"alice@example.com"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(createRR, createReq)
	if createRR.Code != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d", createRR.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/admin/v1/users", nil)
	listRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(listRR, listReq)
	if listRR.Code != http.StatusOK {
		t.Fatalf("expected list status 200, got %d", listRR.Code)
	}

	var users []map[string]any
	if err := json.Unmarshal(listRR.Body.Bytes(), &users); err != nil {
		t.Fatalf("unmarshal list response failed: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}

	getReq := httptest.NewRequest(http.MethodGet, "/admin/v1/users/1", nil)
	getRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(getRR, getReq)
	if getRR.Code != http.StatusOK {
		t.Fatalf("expected get status 200, got %d", getRR.Code)
	}
	if !strings.Contains(getRR.Body.String(), `"Email":"alice@example.com"`) {
		t.Fatalf("expected user email in response, got %s", getRR.Body.String())
	}
}

func TestAdminBulkUserCreateAtomic(t *testing.T) {
	srv := New(":0", "test-version")
	srv.MountAdminModuleRoutes(testAdminModuleServices(), nil)

	bulkReq := httptest.NewRequest(http.MethodPost, "/admin/v1/users/bulk", strings.NewReader(`{"items":[{"name":"alice","email":"alice@example.com"},{"name":"bob","email":"bob@example.com"}]}`))
	bulkReq.Header.Set("Content-Type", "application/json")
	bulkRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(bulkRR, bulkReq)
	if bulkRR.Code != http.StatusCreated {
		t.Fatalf("expected bulk create status 201, got %d body=%s", bulkRR.Code, bulkRR.Body.String())
	}
	if !strings.Contains(bulkRR.Body.String(), `"atomic":true`) {
		t.Fatalf("expected atomic true in bulk create response, got %s", bulkRR.Body.String())
	}

	invalidReq := httptest.NewRequest(http.MethodPost, "/admin/v1/users/bulk", strings.NewReader(`{"items":[{"name":"ok","email":"ok@example.com"},{"name":"","email":"bad@example.com"}]}`))
	invalidReq.Header.Set("Content-Type", "application/json")
	invalidRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(invalidRR, invalidReq)
	if invalidRR.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid bulk create status 400, got %d", invalidRR.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/admin/v1/users", nil)
	listRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(listRR, listReq)
	if !strings.Contains(listRR.Body.String(), `"Email":"alice@example.com"`) || !strings.Contains(listRR.Body.String(), `"Email":"bob@example.com"`) {
		t.Fatalf("expected only valid bulk-created users persisted, got %s", listRR.Body.String())
	}
}

func TestAdminUserSecurityRoutes(t *testing.T) {
	srv := New(":0", "test-version")
	srv.MountAdminModuleRoutes(testAdminModuleServices(), nil)

	createReq := httptest.NewRequest(http.MethodPost, "/admin/v1/users", strings.NewReader(`{"name":"alice","email":"alice@example.com"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(createRR, createReq)
	if createRR.Code != http.StatusCreated {
		t.Fatalf("expected user create status 201, got %d", createRR.Code)
	}

	rotateReq := httptest.NewRequest(http.MethodPost, "/admin/v1/users/1/password/rotate", strings.NewReader(`{"min_interval_minutes":0}`))
	rotateReq.Header.Set("Content-Type", "application/json")
	rotateRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rotateRR, rotateReq)
	if rotateRR.Code != http.StatusOK {
		t.Fatalf("expected password rotate status 200, got %d", rotateRR.Code)
	}

	loginFailureReq := httptest.NewRequest(http.MethodPost, "/admin/v1/users/1/login-failures", strings.NewReader(`{"lock_threshold":1,"lock_duration_minutes":15}`))
	loginFailureReq.Header.Set("Content-Type", "application/json")
	loginFailureRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(loginFailureRR, loginFailureReq)
	if loginFailureRR.Code != http.StatusOK {
		t.Fatalf("expected login failure status 200, got %d", loginFailureRR.Code)
	}
	if !strings.Contains(loginFailureRR.Body.String(), `"locked_until_unix_sec":`) {
		t.Fatalf("expected lock info in response, got %s", loginFailureRR.Body.String())
	}

	mfaReq := httptest.NewRequest(http.MethodPost, "/admin/v1/users/1/mfa", strings.NewReader(`{"enabled":true,"provider":"totp"}`))
	mfaReq.Header.Set("Content-Type", "application/json")
	mfaRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(mfaRR, mfaReq)
	if mfaRR.Code != http.StatusOK {
		t.Fatalf("expected mfa update status 200, got %d", mfaRR.Code)
	}
	if !strings.Contains(mfaRR.Body.String(), `"mfa_enabled":true`) {
		t.Fatalf("expected mfa enabled in response, got %s", mfaRR.Body.String())
	}

	revokeReq := httptest.NewRequest(http.MethodPost, "/admin/v1/sessions/revoke", strings.NewReader(`{"session_id":"sess-1","reason":"manual"}`))
	revokeReq.Header.Set("Content-Type", "application/json")
	revokeRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(revokeRR, revokeReq)
	if revokeRR.Code != http.StatusOK {
		t.Fatalf("expected session revoke status 200, got %d", revokeRR.Code)
	}

	anomalyReq := httptest.NewRequest(http.MethodPost, "/admin/v1/sessions/anomalies", strings.NewReader(`{"session_id":"sess-1","category":"geo_jump","detail":"ip changed"}`))
	anomalyReq.Header.Set("Content-Type", "application/json")
	anomalyRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(anomalyRR, anomalyReq)
	if anomalyRR.Code != http.StatusCreated {
		t.Fatalf("expected session anomaly status 201, got %d", anomalyRR.Code)
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/admin/v1/sessions/sess-1/status", nil)
	statusRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(statusRR, statusReq)
	if statusRR.Code != http.StatusOK {
		t.Fatalf("expected session status 200, got %d", statusRR.Code)
	}
	if !strings.Contains(statusRR.Body.String(), `"revoked":true`) {
		t.Fatalf("expected revoked session status, got %s", statusRR.Body.String())
	}

	auditReq := httptest.NewRequest(http.MethodGet, "/admin/v1/audit-logs?page=1&size=20&actor=security", nil)
	auditRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(auditRR, auditReq)
	if auditRR.Code != http.StatusOK {
		t.Fatalf("expected audit query status 200, got %d", auditRR.Code)
	}
	if !strings.Contains(auditRR.Body.String(), `"total":5`) {
		t.Fatalf("expected security audit events in response, got %s", auditRR.Body.String())
	}
}

func TestAdminAuthLifecycleRoutes(t *testing.T) {
	srv := New(":0", "test-version")
	srv.MountAdminModuleRoutes(testAdminModuleServices(), nil)

	createReq := httptest.NewRequest(http.MethodPost, "/admin/v1/users", strings.NewReader(`{"name":"alice","email":"alice@example.com"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(createRR, createReq)
	if createRR.Code != http.StatusCreated {
		t.Fatalf("expected user create status 201, got %d", createRR.Code)
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/admin/v1/auth/login", strings.NewReader(`{"user_id":1,"role_id":1,"claims_version":"v2"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(loginRR, loginReq)
	if loginRR.Code != http.StatusOK {
		t.Fatalf("expected login status 200, got %d body=%s", loginRR.Code, loginRR.Body.String())
	}

	var loginResp map[string]any
	if err := json.Unmarshal(loginRR.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("unmarshal login response failed: %v", err)
	}
	sessionID, _ := loginResp["session_id"].(string)
	refreshToken, _ := loginResp["refresh_token"].(string)
	if sessionID == "" || refreshToken == "" {
		t.Fatalf("expected non-empty login payload, got %+v", loginResp)
	}

	refreshReq := httptest.NewRequest(http.MethodPost, "/admin/v1/auth/refresh", strings.NewReader(`{"refresh_token":"`+refreshToken+`"}`))
	refreshReq.Header.Set("Content-Type", "application/json")
	refreshRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(refreshRR, refreshReq)
	if refreshRR.Code != http.StatusOK {
		t.Fatalf("expected refresh status 200, got %d body=%s", refreshRR.Code, refreshRR.Body.String())
	}

	sessionReq := httptest.NewRequest(http.MethodGet, "/admin/v1/auth/sessions/"+sessionID, nil)
	sessionRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(sessionRR, sessionReq)
	if sessionRR.Code != http.StatusOK {
		t.Fatalf("expected auth session get status 200, got %d", sessionRR.Code)
	}

	logoutReq := httptest.NewRequest(http.MethodPost, "/admin/v1/auth/logout", strings.NewReader(`{"session_id":"`+sessionID+`","reason":"manual"}`))
	logoutReq.Header.Set("Content-Type", "application/json")
	logoutRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(logoutRR, logoutReq)
	if logoutRR.Code != http.StatusOK {
		t.Fatalf("expected logout status 200, got %d body=%s", logoutRR.Code, logoutRR.Body.String())
	}
	if !strings.Contains(logoutRR.Body.String(), `"revoked":true`) {
		t.Fatalf("expected revoked auth session, got %s", logoutRR.Body.String())
	}
}

func TestAdminRoleAndMenuRoutes(t *testing.T) {
	srv := New(":0", "test-version")
	srv.MountAdminModuleRoutes(testAdminModuleServices(), nil)

	roleReq := httptest.NewRequest(http.MethodPost, "/admin/v1/roles", strings.NewReader(`{"name":"ops","permissions":["user.read","menu.read"]}`))
	roleReq.Header.Set("Content-Type", "application/json")
	roleRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(roleRR, roleReq)
	if roleRR.Code != http.StatusCreated {
		t.Fatalf("expected role create status 201, got %d", roleRR.Code)
	}

	menuReq := httptest.NewRequest(http.MethodPost, "/admin/v1/menus", strings.NewReader(`{"title":"Dashboard","path":"/dashboard","order":1}`))
	menuReq.Header.Set("Content-Type", "application/json")
	menuRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(menuRR, menuReq)
	if menuRR.Code != http.StatusCreated {
		t.Fatalf("expected menu create status 201, got %d", menuRR.Code)
	}

	roleGetReq := httptest.NewRequest(http.MethodGet, "/admin/v1/roles/1", nil)
	roleGetRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(roleGetRR, roleGetReq)
	if roleGetRR.Code != http.StatusOK {
		t.Fatalf("expected role get status 200, got %d", roleGetRR.Code)
	}

	menuListReq := httptest.NewRequest(http.MethodGet, "/admin/v1/menus", nil)
	menuListRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(menuListRR, menuListReq)
	if menuListRR.Code != http.StatusOK {
		t.Fatalf("expected menu list status 200, got %d", menuListRR.Code)
	}
}

func TestAdminAuditRoutes_RecentWithLimit(t *testing.T) {
	srv := New(":0", "test-version")
	srv.MountAdminModuleRoutes(testAdminModuleServices(), nil)

	for i := 0; i < 3; i++ {
		payload := []byte(`{"actor":"system","action":"create","target":"user"}`)
		req := httptest.NewRequest(http.MethodPost, "/admin/v1/audit-logs", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		srv.httpServer.Handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("expected audit append status 201, got %d", rr.Code)
		}
	}

	recentReq := httptest.NewRequest(http.MethodGet, "/admin/v1/audit-logs?limit=2", nil)
	recentRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(recentRR, recentReq)
	if recentRR.Code != http.StatusOK {
		t.Fatalf("expected recent status 200, got %d", recentRR.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(recentRR.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal recent response failed: %v", err)
	}
	items, ok := payload["items"].([]any)
	if !ok {
		t.Fatalf("expected items array payload, got %v", payload)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 recent records, got %d", len(items))
	}

	filteredReq := httptest.NewRequest(http.MethodGet, "/admin/v1/audit-logs?page=1&size=10&actor=system&action=create&q=user", nil)
	filteredRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(filteredRR, filteredReq)
	if filteredRR.Code != http.StatusOK {
		t.Fatalf("expected filtered status 200, got %d", filteredRR.Code)
	}
	if !strings.Contains(filteredRR.Body.String(), `"total":3`) {
		t.Fatalf("expected filtered total field in response, got %s", filteredRR.Body.String())
	}

	profileReq := httptest.NewRequest(http.MethodGet, "/admin/v1/audit-logs/profile", nil)
	profileRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(profileRR, profileReq)
	if profileRR.Code != http.StatusOK {
		t.Fatalf("expected audit profile status 200, got %d", profileRR.Code)
	}
	if !strings.Contains(profileRR.Body.String(), `"max_size":200`) {
		t.Fatalf("expected audit profile max size field, got %s", profileRR.Body.String())
	}

	controlProfileReq := httptest.NewRequest(http.MethodGet, "/admin/v1/admin-ops/control-profile", nil)
	controlProfileRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(controlProfileRR, controlProfileReq)
	if controlProfileRR.Code != http.StatusOK {
		t.Fatalf("expected control profile status 200, got %d", controlProfileRR.Code)
	}
	if !strings.Contains(controlProfileRR.Body.String(), `"audit_retention_days":180`) {
		t.Fatalf("expected retention profile payload, got %s", controlProfileRR.Body.String())
	}
}

func TestAdminRoutes_AuthWrapper(t *testing.T) {
	srv := New(":0", "test-version")
	wrapper := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Admin-Token") != "secret" {
				respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
	srv.MountAdminModuleRoutes(testAdminModuleServices(), wrapper)

	createReq := httptest.NewRequest(http.MethodPost, "/admin/v1/users", strings.NewReader(`{"name":"alice","email":"alice@example.com"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("X-Admin-Token", "secret")
	createRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(createRR, createReq)
	if createRR.Code != http.StatusCreated {
		t.Fatalf("expected setup user create status 201, got %d", createRR.Code)
	}

	publicLoginReq := httptest.NewRequest(http.MethodPost, "/admin/v1/auth/login", strings.NewReader(`{"user_id":1,"role_id":1}`))
	publicLoginReq.Header.Set("Content-Type", "application/json")
	publicLoginRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(publicLoginRR, publicLoginReq)
	if publicLoginRR.Code != http.StatusOK {
		t.Fatalf("expected public auth login route bypass wrapper, got %d body=%s", publicLoginRR.Code, publicLoginRR.Body.String())
	}

	unauthReq := httptest.NewRequest(http.MethodGet, "/admin/v1/users", nil)
	unauthRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(unauthRR, unauthReq)
	if unauthRR.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized status 401, got %d", unauthRR.Code)
	}

	authReq := httptest.NewRequest(http.MethodGet, "/admin/v1/users", nil)
	authReq.Header.Set("X-Admin-Token", "secret")
	authRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(authRR, authReq)
	if authRR.Code != http.StatusOK {
		t.Fatalf("expected authorized status 200, got %d", authRR.Code)
	}
}

func BenchmarkAdminUsersListEndpoint(b *testing.B) {
	srv := New(":0", "bench")
	users := user.NewService()
	for i := 0; i < 100; i++ {
		users.Create("user", "user@example.com")
	}
	services := testAdminModuleServices()
	services.Users = users
	srv.MountAdminModuleRoutes(services, nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/v1/users", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		srv.httpServer.Handler.ServeHTTP(rr, req)
	}
}

func BenchmarkAdminAuthLoginEndpoint(b *testing.B) {
	srv := New(":0", "bench")
	services := testAdminModuleServices()
	services.Users.Create("alice", "alice@example.com")
	srv.MountAdminModuleRoutes(services, nil)

	req := httptest.NewRequest(http.MethodPost, "/admin/v1/auth/login", strings.NewReader(`{"user_id":1,"role_id":1,"claims_version":"v2"}`))
	req.Header.Set("Content-Type", "application/json")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		srv.httpServer.Handler.ServeHTTP(rr, req)
	}
}

func TestRoleBindingRoutes_MenuAndAPI(t *testing.T) {
	srv := New(":0", "test-version")
	srv.MountAdminModuleRoutes(testAdminModuleServices(), nil)

	createRoleReq := httptest.NewRequest(http.MethodPost, "/admin/v1/roles", strings.NewReader(`{"name":"ops","permissions":["user.read"]}`))
	createRoleReq.Header.Set("Content-Type", "application/json")
	createRoleRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(createRoleRR, createRoleReq)
	if createRoleRR.Code != http.StatusCreated {
		t.Fatalf("expected role create status 201, got %d", createRoleRR.Code)
	}

	createMenuReq1 := httptest.NewRequest(http.MethodPost, "/admin/v1/menus", strings.NewReader(`{"title":"Dashboard","path":"/dashboard","order":1}`))
	createMenuReq1.Header.Set("Content-Type", "application/json")
	createMenuRR1 := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(createMenuRR1, createMenuReq1)
	if createMenuRR1.Code != http.StatusCreated {
		t.Fatalf("expected menu create status 201, got %d", createMenuRR1.Code)
	}

	createMenuReq2 := httptest.NewRequest(http.MethodPost, "/admin/v1/menus", strings.NewReader(`{"title":"System","path":"/system","order":2}`))
	createMenuReq2.Header.Set("Content-Type", "application/json")
	createMenuRR2 := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(createMenuRR2, createMenuReq2)
	if createMenuRR2.Code != http.StatusCreated {
		t.Fatalf("expected menu create status 201, got %d", createMenuRR2.Code)
	}

	setMenusReq := httptest.NewRequest(http.MethodPut, "/admin/v1/roles/1/menus", strings.NewReader(`{"menu_ids":[2,1,1]}`))
	setMenusReq.Header.Set("Content-Type", "application/json")
	setMenusRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(setMenusRR, setMenusReq)
	if setMenusRR.Code != http.StatusOK {
		t.Fatalf("expected set role menus status 200, got %d", setMenusRR.Code)
	}

	getMenusReq := httptest.NewRequest(http.MethodGet, "/admin/v1/roles/1/menus", nil)
	getMenusRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(getMenusRR, getMenusReq)
	if getMenusRR.Code != http.StatusOK {
		t.Fatalf("expected get role menus status 200, got %d", getMenusRR.Code)
	}
	if !strings.Contains(getMenusRR.Body.String(), `"menu_ids":[1,2]`) {
		t.Fatalf("expected sorted deduplicated menu ids, got %s", getMenusRR.Body.String())
	}

	setAPIsReq := httptest.NewRequest(http.MethodPut, "/admin/v1/roles/1/apis", strings.NewReader(`{"apis":["POST:/admin/v1/users","GET:/admin/v1/users","GET:/admin/v1/users"]}`))
	setAPIsReq.Header.Set("Content-Type", "application/json")
	setAPIsRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(setAPIsRR, setAPIsReq)
	if setAPIsRR.Code != http.StatusOK {
		t.Fatalf("expected set role apis status 200, got %d", setAPIsRR.Code)
	}

	getAPIsReq := httptest.NewRequest(http.MethodGet, "/admin/v1/roles/1/apis", nil)
	getAPIsRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(getAPIsRR, getAPIsReq)
	if getAPIsRR.Code != http.StatusOK {
		t.Fatalf("expected get role apis status 200, got %d", getAPIsRR.Code)
	}
	if !strings.Contains(getAPIsRR.Body.String(), `"apis":["GET:/admin/v1/users","POST:/admin/v1/users"]`) {
		t.Fatalf("expected sorted deduplicated api bindings, got %s", getAPIsRR.Body.String())
	}

	setPoliciesReq := httptest.NewRequest(http.MethodPut, "/admin/v1/roles/1/policies", strings.NewReader(`{"rules":[{"api":"GET:/admin/v1/users","effect":"allow","require_verified":true,"require_claims_version":"v2"},{"api":"POST:/admin/v1/users","effect":"deny"}]}`))
	setPoliciesReq.Header.Set("Content-Type", "application/json")
	setPoliciesRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(setPoliciesRR, setPoliciesReq)
	if setPoliciesRR.Code != http.StatusOK {
		t.Fatalf("expected set role policies status 200, got %d", setPoliciesRR.Code)
	}

	getPoliciesReq := httptest.NewRequest(http.MethodGet, "/admin/v1/roles/1/policies", nil)
	getPoliciesRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(getPoliciesRR, getPoliciesReq)
	if getPoliciesRR.Code != http.StatusOK {
		t.Fatalf("expected get role policies status 200, got %d", getPoliciesRR.Code)
	}
	if !strings.Contains(getPoliciesRR.Body.String(), `"effect":"allow"`) {
		t.Fatalf("expected role policy bindings in response, got %s", getPoliciesRR.Body.String())
	}

	setDataScopeReq := httptest.NewRequest(http.MethodPut, "/admin/v1/roles/1/data-scope", strings.NewReader(`{"tenant_ids":["tenant-b","tenant-a","tenant-a"],"require_owner_match":true,"cross_tenant_admin_allow":["ops@example.com","ops@example.com"]}`))
	setDataScopeReq.Header.Set("Content-Type", "application/json")
	setDataScopeRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(setDataScopeRR, setDataScopeReq)
	if setDataScopeRR.Code != http.StatusOK {
		t.Fatalf("expected set role data scope status 200, got %d", setDataScopeRR.Code)
	}

	getDataScopeReq := httptest.NewRequest(http.MethodGet, "/admin/v1/roles/1/data-scope", nil)
	getDataScopeRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(getDataScopeRR, getDataScopeReq)
	if getDataScopeRR.Code != http.StatusOK {
		t.Fatalf("expected get role data scope status 200, got %d", getDataScopeRR.Code)
	}
	if !strings.Contains(getDataScopeRR.Body.String(), `"tenant_ids":["tenant-a","tenant-b"]`) {
		t.Fatalf("expected normalized tenant scope in response, got %s", getDataScopeRR.Body.String())
	}
	if !strings.Contains(getDataScopeRR.Body.String(), `"cross_tenant_admin_allow":["ops@example.com"]`) {
		t.Fatalf("expected normalized cross tenant whitelist in response, got %s", getDataScopeRR.Body.String())
	}

	diffReq := httptest.NewRequest(http.MethodPost, "/admin/v1/roles/1/permissions/diff", strings.NewReader(`{"menu_ids":[1,2],"apis":["GET:/admin/v1/users","POST:/admin/v1/users"],"rules":[{"api":"GET:/admin/v1/users","effect":"allow"}],"data_scope":{"tenant_ids":["tenant-a","tenant-b","tenant-c"],"require_owner_match":false,"cross_tenant_admin_allow":["ops@example.com","root@example.com"]},"permission_contract":{"version":"v2","items":[{"menu_id":1,"route":"/dashboard","buttons":["view"]}]}}`))
	diffReq.Header.Set("Content-Type", "application/json")
	diffRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(diffRR, diffReq)
	if diffRR.Code != http.StatusOK {
		t.Fatalf("expected permissions diff status 200, got %d body=%s", diffRR.Code, diffRR.Body.String())
	}
	if !strings.Contains(diffRR.Body.String(), `"data_scope_changed":true`) {
		t.Fatalf("expected data scope drift in diff response, got %s", diffRR.Body.String())
	}

	checkReq := httptest.NewRequest(http.MethodPost, "/admin/v1/roles/1/permissions/check", strings.NewReader(`{"menu_ids":[1,2],"apis":["GET:/admin/v1/users","POST:/admin/v1/users"],"rules":[{"api":"GET:/admin/v1/users","effect":"allow"}],"data_scope":{"tenant_ids":["tenant-a","tenant-b","tenant-c"],"require_owner_match":false,"cross_tenant_admin_allow":["ops@example.com","root@example.com"]},"permission_contract":{"version":"v2","items":[{"menu_id":1,"route":"/dashboard","buttons":["view"]}]}}`))
	checkReq.Header.Set("Content-Type", "application/json")
	checkRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(checkRR, checkReq)
	if checkRR.Code != http.StatusOK {
		t.Fatalf("expected permissions check status 200, got %d body=%s", checkRR.Code, checkRR.Body.String())
	}
	if !strings.Contains(checkRR.Body.String(), `"blocking":true`) {
		t.Fatalf("expected blocking permission check, got %s", checkRR.Body.String())
	}

	setContractReq := httptest.NewRequest(http.MethodPut, "/admin/v1/roles/1/permission-contract", strings.NewReader(`{"version":"v2","items":[{"menu_id":1,"route":"/dashboard","buttons":["view"]},{"menu_id":2,"route":"/system","buttons":["create","delete","create"]}]}`))
	setContractReq.Header.Set("Content-Type", "application/json")
	setContractRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(setContractRR, setContractReq)
	if setContractRR.Code != http.StatusOK {
		t.Fatalf("expected set permission contract status 200, got %d", setContractRR.Code)
	}
	if !strings.Contains(setContractRR.Body.String(), `"version":"v2"`) {
		t.Fatalf("expected permission contract version in response, got %s", setContractRR.Body.String())
	}

	getContractReq := httptest.NewRequest(http.MethodGet, "/admin/v1/roles/1/permission-contract", nil)
	getContractRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(getContractRR, getContractReq)
	if getContractRR.Code != http.StatusOK {
		t.Fatalf("expected get permission contract status 200, got %d", getContractRR.Code)
	}
	if !strings.Contains(getContractRR.Body.String(), `"buttons":["create","delete"]`) {
		t.Fatalf("expected normalized button list in contract, got %s", getContractRR.Body.String())
	}

	checkContractReq := httptest.NewRequest(http.MethodPost, "/admin/v1/roles/1/permission-contract/consistency-check", nil)
	checkContractRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(checkContractRR, checkContractReq)
	if checkContractRR.Code != http.StatusOK {
		t.Fatalf("expected permission contract consistency status 200, got %d", checkContractRR.Code)
	}
	if !strings.Contains(checkContractRR.Body.String(), `"passed":true`) {
		t.Fatalf("expected consistency check passed, got %s", checkContractRR.Body.String())
	}

	listRegistryReq := httptest.NewRequest(http.MethodGet, "/admin/v1/apis", nil)
	listRegistryRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(listRegistryRR, listRegistryReq)
	if listRegistryRR.Code != http.StatusOK {
		t.Fatalf("expected list api registry status 200, got %d", listRegistryRR.Code)
	}
	if !strings.Contains(listRegistryRR.Body.String(), `"GET:/admin/v1/roles/{id}/apis"`) {
		t.Fatalf("expected registered apis in response, got %s", listRegistryRR.Body.String())
	}

	invalidAPIReq := httptest.NewRequest(http.MethodPut, "/admin/v1/roles/1/apis", strings.NewReader(`{"apis":["DELETE:/admin/v1/users"]}`))
	invalidAPIReq.Header.Set("Content-Type", "application/json")
	invalidAPIRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(invalidAPIRR, invalidAPIReq)
	if invalidAPIRR.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid api binding status 400, got %d", invalidAPIRR.Code)
	}
}

func TestRolePolicySnapshotAndRollbackRoutes(t *testing.T) {
	srv := New(":0", "test-version")
	srv.MountAdminModuleRoutes(testAdminModuleServices(), nil)

	createRoleReq := httptest.NewRequest(http.MethodPost, "/admin/v1/roles", strings.NewReader(`{"name":"ops","permissions":["user.read"]}`))
	createRoleReq.Header.Set("Content-Type", "application/json")
	createRoleRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(createRoleRR, createRoleReq)
	if createRoleRR.Code != http.StatusCreated {
		t.Fatalf("expected role create status 201, got %d", createRoleRR.Code)
	}

	setV1Req := httptest.NewRequest(http.MethodPut, "/admin/v1/roles/1/policies", strings.NewReader(`{"rules":[{"api":"GET:/admin/v1/users","effect":"allow"}]}`))
	setV1Req.Header.Set("Content-Type", "application/json")
	setV1RR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(setV1RR, setV1Req)
	if setV1RR.Code != http.StatusOK {
		t.Fatalf("expected set policies v1 status 200, got %d body=%s", setV1RR.Code, setV1RR.Body.String())
	}

	snapV1Req := httptest.NewRequest(http.MethodPost, "/admin/v1/roles/1/policies/snapshots", nil)
	snapV1RR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(snapV1RR, snapV1Req)
	if snapV1RR.Code != http.StatusCreated || !strings.Contains(snapV1RR.Body.String(), `"version":"v1"`) {
		t.Fatalf("expected snapshot v1 created, got code=%d body=%s", snapV1RR.Code, snapV1RR.Body.String())
	}

	setV2Req := httptest.NewRequest(http.MethodPut, "/admin/v1/roles/1/policies", strings.NewReader(`{"rules":[{"api":"POST:/admin/v1/users","effect":"deny"}]}`))
	setV2Req.Header.Set("Content-Type", "application/json")
	setV2RR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(setV2RR, setV2Req)
	if setV2RR.Code != http.StatusOK {
		t.Fatalf("expected set policies v2 status 200, got %d body=%s", setV2RR.Code, setV2RR.Body.String())
	}

	snapV2Req := httptest.NewRequest(http.MethodPost, "/admin/v1/roles/1/policies/snapshots", nil)
	snapV2RR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(snapV2RR, snapV2Req)
	if snapV2RR.Code != http.StatusCreated || !strings.Contains(snapV2RR.Body.String(), `"version":"v2"`) {
		t.Fatalf("expected snapshot v2 created, got code=%d body=%s", snapV2RR.Code, snapV2RR.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/admin/v1/roles/1/policies/snapshots", nil)
	listRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(listRR, listReq)
	if listRR.Code != http.StatusOK {
		t.Fatalf("expected list snapshots status 200, got %d", listRR.Code)
	}
	if !strings.Contains(listRR.Body.String(), `"version":"v1"`) || !strings.Contains(listRR.Body.String(), `"version":"v2"`) {
		t.Fatalf("expected snapshot versions in list, got %s", listRR.Body.String())
	}

	rollbackReq := httptest.NewRequest(http.MethodPost, "/admin/v1/roles/1/policies/rollback", strings.NewReader(`{"snapshot_version":"v1","approver":"security.lead"}`))
	rollbackReq.Header.Set("Content-Type", "application/json")
	rollbackRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rollbackRR, rollbackReq)
	if rollbackRR.Code != http.StatusOK {
		t.Fatalf("expected rollback status 200, got %d body=%s", rollbackRR.Code, rollbackRR.Body.String())
	}
	if !strings.Contains(rollbackRR.Body.String(), `"api":"GET:/admin/v1/users"`) {
		t.Fatalf("expected rollback to restore v1 policy, got %s", rollbackRR.Body.String())
	}

	missingApproverReq := httptest.NewRequest(http.MethodPost, "/admin/v1/roles/1/policies/rollback", strings.NewReader(`{"snapshot_version":"v1"}`))
	missingApproverReq.Header.Set("Content-Type", "application/json")
	missingApproverRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(missingApproverRR, missingApproverReq)
	if missingApproverRR.Code != http.StatusBadRequest {
		t.Fatalf("expected rollback approver required status 400, got %d", missingApproverRR.Code)
	}
}

func TestRolePolicyPersistenceExportImportRoutes(t *testing.T) {
	srv := New(":0", "test-version")
	srv.MountAdminModuleRoutes(testAdminModuleServices(), nil)

	createRoleReq := httptest.NewRequest(http.MethodPost, "/admin/v1/roles", strings.NewReader(`{"name":"ops","permissions":["user.read"]}`))
	createRoleReq.Header.Set("Content-Type", "application/json")
	createRoleRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(createRoleRR, createRoleReq)
	if createRoleRR.Code != http.StatusCreated {
		t.Fatalf("expected role create status 201, got %d", createRoleRR.Code)
	}

	createMenuReq := httptest.NewRequest(http.MethodPost, "/admin/v1/menus", strings.NewReader(`{"title":"Dashboard","path":"/dashboard","order":1}`))
	createMenuReq.Header.Set("Content-Type", "application/json")
	createMenuRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(createMenuRR, createMenuReq)
	if createMenuRR.Code != http.StatusCreated {
		t.Fatalf("expected menu create status 201, got %d", createMenuRR.Code)
	}

	setPoliciesReq := httptest.NewRequest(http.MethodPut, "/admin/v1/roles/1/policies", strings.NewReader(`{"rules":[{"api":"GET:/admin/v1/users","effect":"allow"}]}`))
	setPoliciesReq.Header.Set("Content-Type", "application/json")
	setPoliciesRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(setPoliciesRR, setPoliciesReq)
	if setPoliciesRR.Code != http.StatusOK {
		t.Fatalf("expected set policies status 200, got %d", setPoliciesRR.Code)
	}

	exportReq := httptest.NewRequest(http.MethodGet, "/admin/v1/roles/1/policies/persistence/export", nil)
	exportRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(exportRR, exportReq)
	if exportRR.Code != http.StatusOK {
		t.Fatalf("expected export status 200, got %d body=%s", exportRR.Code, exportRR.Body.String())
	}
	if !strings.Contains(exportRR.Body.String(), `"role_id":1`) {
		t.Fatalf("expected role id in export response, got %s", exportRR.Body.String())
	}

	importReq := httptest.NewRequest(http.MethodPost, "/admin/v1/roles/1/policies/persistence/import", strings.NewReader(`{"operator":"security.lead","bundle":{"menu_ids":[1],"apis":["GET:/admin/v1/users"],"rules":[{"api":"GET:/admin/v1/users","effect":"allow"}],"data_scope":{"tenant_ids":["tenant-a"],"require_owner_match":true,"cross_tenant_admin_allow":["ops@example.com"]},"permission_contract":{"version":"v2","items":[{"menu_id":1,"route":"/dashboard","buttons":["view"]}]}}}`))
	importReq.Header.Set("Content-Type", "application/json")
	importRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(importRR, importReq)
	if importRR.Code != http.StatusOK {
		t.Fatalf("expected import status 200, got %d body=%s", importRR.Code, importRR.Body.String())
	}
	if !strings.Contains(importRR.Body.String(), `"cross_tenant_admin_allow":["ops@example.com"]`) {
		t.Fatalf("expected imported cross-tenant allow list, got %s", importRR.Body.String())
	}

	invalidImportReq := httptest.NewRequest(http.MethodPost, "/admin/v1/roles/1/policies/persistence/import", strings.NewReader(`{"bundle":{"apis":["GET:/admin/v1/users"]}}`))
	invalidImportReq.Header.Set("Content-Type", "application/json")
	invalidImportRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(invalidImportRR, invalidImportReq)
	if invalidImportRR.Code != http.StatusBadRequest {
		t.Fatalf("expected import operator-required status 400, got %d", invalidImportRR.Code)
	}
}

func TestConfigAndDictionaryRoutes(t *testing.T) {
	srv := New(":0", "test-version")
	srv.MountAdminModuleRoutes(testAdminModuleServices(), nil)

	configReq := httptest.NewRequest(http.MethodPost, "/admin/v1/configs", strings.NewReader(`{"key":"system.theme","value":"aurora","description":"ui theme"}`))
	configReq.Header.Set("Content-Type", "application/json")
	configRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(configRR, configReq)
	if configRR.Code != http.StatusCreated {
		t.Fatalf("expected config create status 201, got %d", configRR.Code)
	}

	configGetReq := httptest.NewRequest(http.MethodGet, "/admin/v1/configs/system.theme", nil)
	configGetRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(configGetRR, configGetReq)
	if configGetRR.Code != http.StatusOK {
		t.Fatalf("expected config get status 200, got %d", configGetRR.Code)
	}

	dictReq := httptest.NewRequest(http.MethodPost, "/admin/v1/dictionaries", strings.NewReader(`{"type":"status","label":"Enabled","value":"1","sort":10}`))
	dictReq.Header.Set("Content-Type", "application/json")
	dictRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(dictRR, dictReq)
	if dictRR.Code != http.StatusCreated {
		t.Fatalf("expected dictionary create status 201, got %d", dictRR.Code)
	}

	dictListReq := httptest.NewRequest(http.MethodGet, "/admin/v1/dictionaries?type=status", nil)
	dictListRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(dictListRR, dictListReq)
	if dictListRR.Code != http.StatusOK {
		t.Fatalf("expected dictionary list status 200, got %d", dictListRR.Code)
	}
	if !strings.Contains(dictListRR.Body.String(), `"Type":"status"`) {
		t.Fatalf("expected filtered dictionary items, got %s", dictListRR.Body.String())
	}

	dictGetReq := httptest.NewRequest(http.MethodGet, "/admin/v1/dictionaries/1", nil)
	dictGetRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(dictGetRR, dictGetReq)
	if dictGetRR.Code != http.StatusOK {
		t.Fatalf("expected dictionary get status 200, got %d", dictGetRR.Code)
	}

	bulkConfigReq := httptest.NewRequest(http.MethodPost, "/admin/v1/configs/bulk", strings.NewReader(`{"items":[{"key":"feature.a","value":"on","description":"a"},{"key":"feature.b","value":"off","description":"b"}]}`))
	bulkConfigReq.Header.Set("Content-Type", "application/json")
	bulkConfigRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(bulkConfigRR, bulkConfigReq)
	if bulkConfigRR.Code != http.StatusCreated {
		t.Fatalf("expected bulk config upsert status 201, got %d body=%s", bulkConfigRR.Code, bulkConfigRR.Body.String())
	}

	bulkDictReq := httptest.NewRequest(http.MethodPost, "/admin/v1/dictionaries/bulk", strings.NewReader(`{"items":[{"type":"status","label":"Disabled","value":"0","sort":20},{"type":"status","label":"Archived","value":"2","sort":30}]}`))
	bulkDictReq.Header.Set("Content-Type", "application/json")
	bulkDictRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(bulkDictRR, bulkDictReq)
	if bulkDictRR.Code != http.StatusCreated {
		t.Fatalf("expected bulk dictionary create status 201, got %d body=%s", bulkDictRR.Code, bulkDictRR.Body.String())
	}

	configQueryReq := httptest.NewRequest(http.MethodGet, "/admin/v1/configs/query?key_prefix=feature.&size=1&page=1", nil)
	configQueryRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(configQueryRR, configQueryReq)
	if configQueryRR.Code != http.StatusOK {
		t.Fatalf("expected config query status 200, got %d body=%s", configQueryRR.Code, configQueryRR.Body.String())
	}
	if !strings.Contains(configQueryRR.Body.String(), `"total":2`) || !strings.Contains(configQueryRR.Body.String(), `"has_next":true`) {
		t.Fatalf("expected paged config query payload, got %s", configQueryRR.Body.String())
	}

	dictQueryReq := httptest.NewRequest(http.MethodGet, "/admin/v1/dictionaries/query?type=status&q=archived&enabled=true", nil)
	dictQueryRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(dictQueryRR, dictQueryReq)
	if dictQueryRR.Code != http.StatusOK {
		t.Fatalf("expected dictionary query status 200, got %d body=%s", dictQueryRR.Code, dictQueryRR.Body.String())
	}
	if !strings.Contains(dictQueryRR.Body.String(), `"total":1`) || !strings.Contains(dictQueryRR.Body.String(), `"Label":"Archived"`) {
		t.Fatalf("expected filtered dictionary query payload, got %s", dictQueryRR.Body.String())
	}
}

func TestFileRoutes_UploadListGetDownload(t *testing.T) {
	srv := New(":0", "test-version")
	srv.MountAdminModuleRoutes(testAdminModuleServices(), nil)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "hello.txt")
	if err != nil {
		t.Fatalf("create form file failed: %v", err)
	}
	if _, err := part.Write([]byte("hello skoll")); err != nil {
		t.Fatalf("write multipart content failed: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer failed: %v", err)
	}

	uploadReq := httptest.NewRequest(http.MethodPost, "/admin/v1/files", &body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	uploadRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(uploadRR, uploadReq)
	if uploadRR.Code != http.StatusCreated {
		t.Fatalf("expected upload status 201, got %d", uploadRR.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/admin/v1/files", nil)
	listRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(listRR, listReq)
	if listRR.Code != http.StatusOK {
		t.Fatalf("expected file list status 200, got %d", listRR.Code)
	}
	if !strings.Contains(listRR.Body.String(), `"Name":"hello.txt"`) {
		t.Fatalf("expected uploaded file in list, got %s", listRR.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/admin/v1/files/1", nil)
	getRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(getRR, getReq)
	if getRR.Code != http.StatusOK {
		t.Fatalf("expected file get status 200, got %d", getRR.Code)
	}

	downloadReq := httptest.NewRequest(http.MethodGet, "/admin/v1/files/1/download", nil)
	downloadRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(downloadRR, downloadReq)
	if downloadRR.Code != http.StatusOK {
		t.Fatalf("expected file download status 200, got %d", downloadRR.Code)
	}
	if downloadRR.Body.String() != "hello skoll" {
		t.Fatalf("unexpected downloaded body: %s", downloadRR.Body.String())
	}
}

func TestJobRoutes_CreateRunHistory(t *testing.T) {
	srv := New(":0", "test-version")
	srv.MountAdminModuleRoutes(testAdminModuleServices(), nil)

	createReq := httptest.NewRequest(http.MethodPost, "/admin/v1/jobs", strings.NewReader(`{"name":"daily-sync","schedule":"0 0 * * *"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(createRR, createReq)
	if createRR.Code != http.StatusCreated {
		t.Fatalf("expected create job status 201, got %d", createRR.Code)
	}

	runReq := httptest.NewRequest(http.MethodPost, "/admin/v1/jobs/1/run", nil)
	runRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(runRR, runReq)
	if runRR.Code != http.StatusOK {
		t.Fatalf("expected run job status 200, got %d", runRR.Code)
	}

	historyReq := httptest.NewRequest(http.MethodGet, "/admin/v1/jobs/1/history?limit=10", nil)
	historyRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(historyRR, historyReq)
	if historyRR.Code != http.StatusOK {
		t.Fatalf("expected job history status 200, got %d", historyRR.Code)
	}
	if !strings.Contains(historyRR.Body.String(), `"Status":"success"`) {
		t.Fatalf("expected successful run in history, got %s", historyRR.Body.String())
	}
}

func TestGeneratorRoutes_ModuleScaffoldPreview(t *testing.T) {
	srv := New(":0", "test-version")
	srv.MountAdminModuleRoutes(testAdminModuleServices(), nil)

	req := httptest.NewRequest(http.MethodPost, "/admin/v1/generator/modules", strings.NewReader(`{"module":"billing","template_version":"v2","form_schema":{"version":"v2","fields":[{"name":"name","type":"string","required":true},{"name":"status","type":"select"}]}}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected generator status 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"module":"billing"`) {
		t.Fatalf("expected module name in generator result, got %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"internal/module/billing/service.go"`) {
		t.Fatalf("expected generated service artifact path, got %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"template_version":"v2"`) {
		t.Fatalf("expected template compatibility metadata, got %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"form_schema"`) {
		t.Fatalf("expected form schema in generator response, got %s", rr.Body.String())
	}
}

func TestPluginRoutes_InstallAndToggle(t *testing.T) {
	srv := New(":0", "test-version")
	srv.MountAdminModuleRoutes(testAdminModuleServices(), nil)

	installReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/manifests", strings.NewReader(`{"name":"audit-ext","version":"1.0.0","hooks":["on_boot"]}`))
	installReq.Header.Set("Content-Type", "application/json")
	installRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(installRR, installReq)
	if installRR.Code != http.StatusCreated {
		t.Fatalf("expected install status 201, got %d", installRR.Code)
	}

	disableReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/audit-ext/disable", nil)
	disableRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(disableRR, disableReq)
	if disableRR.Code != http.StatusOK {
		t.Fatalf("expected disable status 200, got %d", disableRR.Code)
	}
	if !strings.Contains(disableRR.Body.String(), `"enabled":false`) {
		t.Fatalf("expected plugin disabled, got %s", disableRR.Body.String())
	}

	enableReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/audit-ext/enable", nil)
	enableRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(enableRR, enableReq)
	if enableRR.Code != http.StatusOK {
		t.Fatalf("expected enable status 200, got %d", enableRR.Code)
	}
	if !strings.Contains(enableRR.Body.String(), `"enabled":true`) {
		t.Fatalf("expected plugin enabled, got %s", enableRR.Body.String())
	}

	packageReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/packages/install", strings.NewReader(`{"name":"audit-ext","version":"1.1.0","package_url":"https://example.com/plugins/audit-ext-1.1.0.tgz","package_hash":"sha256:abcd","signature":"sig:sha256:abcd","hooks":["on_boot"]}`))
	packageReq.Header.Set("Content-Type", "application/json")
	packageRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(packageRR, packageReq)
	if packageRR.Code != http.StatusCreated {
		t.Fatalf("expected package install status 201, got %d", packageRR.Code)
	}
	if !strings.Contains(packageRR.Body.String(), `"package_url":"https://example.com/plugins/audit-ext-1.1.0.tgz"`) {
		t.Fatalf("expected package metadata in response, got %s", packageRR.Body.String())
	}

	versionReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/audit-ext/version-check", strings.NewReader(`{"latest_version":"1.2.0"}`))
	versionReq.Header.Set("Content-Type", "application/json")
	versionRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(versionRR, versionReq)
	if versionRR.Code != http.StatusOK {
		t.Fatalf("expected version check status 200, got %d", versionRR.Code)
	}
	if !strings.Contains(versionRR.Body.String(), `"update_available":true`) {
		t.Fatalf("expected update availability in response, got %s", versionRR.Body.String())
	}

	badUpgradeReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/audit-ext/upgrade", strings.NewReader(`{"target_version":"1.2.0","package_url":"https://example.com/plugins/audit-ext-1.2.0.tgz","package_hash":"sha256:efgh","signature":"bad"}`))
	badUpgradeReq.Header.Set("Content-Type", "application/json")
	badUpgradeRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(badUpgradeRR, badUpgradeReq)
	if badUpgradeRR.Code != http.StatusOK {
		t.Fatalf("expected bad signature upgrade to return rollback result status 200, got %d", badUpgradeRR.Code)
	}
	if !strings.Contains(badUpgradeRR.Body.String(), `"rolled_back":true`) {
		t.Fatalf("expected rollback result for failed upgrade, got %s", badUpgradeRR.Body.String())
	}

	upgradeReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/audit-ext/upgrade", strings.NewReader(`{"target_version":"1.2.0","package_url":"https://example.com/plugins/audit-ext-1.2.0.tgz","package_hash":"sha256:efgh","signature":"sig:sha256:efgh"}`))
	upgradeReq.Header.Set("Content-Type", "application/json")
	upgradeRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(upgradeRR, upgradeReq)
	if upgradeRR.Code != http.StatusOK {
		t.Fatalf("expected upgrade status 200, got %d", upgradeRR.Code)
	}
	if !strings.Contains(upgradeRR.Body.String(), `"succeeded":true`) {
		t.Fatalf("expected successful upgrade result, got %s", upgradeRR.Body.String())
	}

	hookRegisterReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/hooks/register", strings.NewReader(`{"name":"on_user_created","namespace":"billing","version":"1.0.0","order":10,"timeout_millis":900,"retry_limit":2,"dead_letter":true}`))
	hookRegisterReq.Header.Set("Content-Type", "application/json")
	hookRegisterRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(hookRegisterRR, hookRegisterReq)
	if hookRegisterRR.Code != http.StatusCreated {
		t.Fatalf("expected hook register status 201, got %d body=%s", hookRegisterRR.Code, hookRegisterRR.Body.String())
	}

	hookListReq := httptest.NewRequest(http.MethodGet, "/admin/v1/plugins/hooks", nil)
	hookListRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(hookListRR, hookListReq)
	if hookListRR.Code != http.StatusOK {
		t.Fatalf("expected hook list status 200, got %d", hookListRR.Code)
	}
	if !strings.Contains(hookListRR.Body.String(), `"namespace":"billing"`) {
		t.Fatalf("expected hook namespace in list, got %s", hookListRR.Body.String())
	}

	hookOrderReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/hooks/billing/on_user_created/order", strings.NewReader(`{"order":30}`))
	hookOrderReq.Header.Set("Content-Type", "application/json")
	hookOrderRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(hookOrderRR, hookOrderReq)
	if hookOrderRR.Code != http.StatusOK {
		t.Fatalf("expected hook order status 200, got %d body=%s", hookOrderRR.Code, hookOrderRR.Body.String())
	}
	if !strings.Contains(hookOrderRR.Body.String(), `"order":30`) {
		t.Fatalf("expected updated hook order, got %s", hookOrderRR.Body.String())
	}

	hookRuntimeReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/hooks/billing/on_user_created/runtime", strings.NewReader(`{"timeout_millis":1200,"retry_limit":3,"dead_letter":false}`))
	hookRuntimeReq.Header.Set("Content-Type", "application/json")
	hookRuntimeRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(hookRuntimeRR, hookRuntimeReq)
	if hookRuntimeRR.Code != http.StatusOK {
		t.Fatalf("expected hook runtime status 200, got %d body=%s", hookRuntimeRR.Code, hookRuntimeRR.Body.String())
	}
	if !strings.Contains(hookRuntimeRR.Body.String(), `"timeout_millis":1200`) || !strings.Contains(hookRuntimeRR.Body.String(), `"dead_letter":false`) {
		t.Fatalf("expected updated runtime policy, got %s", hookRuntimeRR.Body.String())
	}

	hookDisableReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/hooks/billing/on_user_created/disable", nil)
	hookDisableRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(hookDisableRR, hookDisableReq)
	if hookDisableRR.Code != http.StatusOK {
		t.Fatalf("expected hook disable status 200, got %d", hookDisableRR.Code)
	}
	if !strings.Contains(hookDisableRR.Body.String(), `"enabled":false`) {
		t.Fatalf("expected hook disabled, got %s", hookDisableRR.Body.String())
	}

	hookEnableReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/hooks/billing/on_user_created/enable", nil)
	hookEnableRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(hookEnableRR, hookEnableReq)
	if hookEnableRR.Code != http.StatusOK {
		t.Fatalf("expected hook enable status 200, got %d", hookEnableRR.Code)
	}

	hookRuntimeDlqReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/hooks/billing/on_user_created/runtime", strings.NewReader(`{"timeout_millis":1200,"retry_limit":3,"dead_letter":true}`))
	hookRuntimeDlqReq.Header.Set("Content-Type", "application/json")
	hookRuntimeDlqRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(hookRuntimeDlqRR, hookRuntimeDlqReq)
	if hookRuntimeDlqRR.Code != http.StatusOK {
		t.Fatalf("expected hook runtime update status 200, got %d", hookRuntimeDlqRR.Code)
	}

	hookDiagReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/hooks/billing/on_user_created/execute-diagnostic", strings.NewReader(`{"fail_times":10}`))
	hookDiagReq.Header.Set("Content-Type", "application/json")
	hookDiagRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(hookDiagRR, hookDiagReq)
	if hookDiagRR.Code != http.StatusOK {
		t.Fatalf("expected hook diagnostic status 200, got %d body=%s", hookDiagRR.Code, hookDiagRR.Body.String())
	}
	if !strings.Contains(hookDiagRR.Body.String(), `"dead_lettered":true`) {
		t.Fatalf("expected dead-lettered diagnostic result, got %s", hookDiagRR.Body.String())
	}

	dlqReq := httptest.NewRequest(http.MethodGet, "/admin/v1/plugins/hooks/dead-letters", nil)
	dlqRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(dlqRR, dlqReq)
	if dlqRR.Code != http.StatusOK {
		t.Fatalf("expected dead-letter list status 200, got %d", dlqRR.Code)
	}
	if !strings.Contains(dlqRR.Body.String(), `"namespace":"billing"`) {
		t.Fatalf("expected dead-letter payload in response, got %s", dlqRR.Body.String())
	}

	compatReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/compatibility-check", strings.NewReader(`{"name":"audit-ext","version":"1.2.0","dependencies":[{"name":"audit-ext","min_version":"1.0.0"}]}`))
	compatReq.Header.Set("Content-Type", "application/json")
	compatRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(compatRR, compatReq)
	if compatRR.Code != http.StatusOK {
		t.Fatalf("expected compatibility status 200, got %d body=%s", compatRR.Code, compatRR.Body.String())
	}
	if !strings.Contains(compatRR.Body.String(), `"compatible":false`) {
		t.Fatalf("expected compatibility blockers for self dependency, got %s", compatRR.Body.String())
	}

	removeReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/audit-ext/remove", nil)
	removeRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(removeRR, removeReq)
	if removeRR.Code != http.StatusOK {
		t.Fatalf("expected remove status 200, got %d body=%s", removeRR.Code, removeRR.Body.String())
	}
	if !strings.Contains(removeRR.Body.String(), `"succeeded":true`) {
		t.Fatalf("expected successful remove result, got %s", removeRR.Body.String())
	}

	removeAgainReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/audit-ext/remove", nil)
	removeAgainRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(removeAgainRR, removeAgainReq)
	if removeAgainRR.Code != http.StatusOK {
		t.Fatalf("expected idempotent remove status 200, got %d", removeAgainRR.Code)
	}
	if !strings.Contains(removeAgainRR.Body.String(), `"idempotent":true`) {
		t.Fatalf("expected idempotent remove result, got %s", removeAgainRR.Body.String())
	}

	trustRootsReq := httptest.NewRequest(http.MethodPut, "/admin/v1/plugins/marketplace/trust-roots", strings.NewReader(`{"roots":["corp-root","backup-root","corp-root"]}`))
	trustRootsReq.Header.Set("Content-Type", "application/json")
	trustRootsRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(trustRootsRR, trustRootsReq)
	if trustRootsRR.Code != http.StatusOK {
		t.Fatalf("expected trust roots status 200, got %d body=%s", trustRootsRR.Code, trustRootsRR.Body.String())
	}

	expiresAt := time.Now().UTC().Add(15 * time.Minute)
	payload := `{"source":"official","signed_by":"corp-root","signature":"%s","expires_at_unix_sec":%d,"packages":[{"name":"audit-ext","version":"1.2.0","package_url":"https://example.com/audit-ext-1.2.0.tgz","package_hash":"sha256:abc"}]}`
	sig := testMarketplaceIndexSignature("official", "corp-root", expiresAt.Unix(), []string{"audit-ext@1.2.0#sha256:abc"})
	ingestReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/marketplace/index/ingest", strings.NewReader(fmt.Sprintf(payload, sig, expiresAt.Unix())))
	ingestReq.Header.Set("Content-Type", "application/json")
	ingestRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(ingestRR, ingestReq)
	if ingestRR.Code != http.StatusOK {
		t.Fatalf("expected marketplace ingest status 200, got %d body=%s", ingestRR.Code, ingestRR.Body.String())
	}
	if !strings.Contains(ingestRR.Body.String(), `"accepted":true`) {
		t.Fatalf("expected accepted ingest result, got %s", ingestRR.Body.String())
	}

	sourcesReq := httptest.NewRequest(http.MethodGet, "/admin/v1/plugins/marketplace/index/sources", nil)
	sourcesRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(sourcesRR, sourcesReq)
	if sourcesRR.Code != http.StatusOK {
		t.Fatalf("expected index sources status 200, got %d", sourcesRR.Code)
	}
	if !strings.Contains(sourcesRR.Body.String(), `"source":"official"`) {
		t.Fatalf("expected indexed source in response, got %s", sourcesRR.Body.String())
	}

	solverReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/dependency-solver/resolve", strings.NewReader(`{"items":[{"name":"billing-ext","version":"1.0.0","dependencies":[{"name":"audit-ext","min_version":"2.0.0"}]},{"name":"report-ext","version":"1.0.0","dependencies":[{"name":"missing-ext","min_version":"1.0.0"}]}]}`))
	solverReq.Header.Set("Content-Type", "application/json")
	solverRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(solverRR, solverReq)
	if solverRR.Code != http.StatusOK {
		t.Fatalf("expected dependency solver status 200, got %d body=%s", solverRR.Code, solverRR.Body.String())
	}
	if !strings.Contains(solverRR.Body.String(), `"deterministic":true`) || !strings.Contains(solverRR.Body.String(), `"conflicts"`) {
		t.Fatalf("expected dependency solver diagnostics payload, got %s", solverRR.Body.String())
	}

	reinstallReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/manifests", strings.NewReader(`{"name":"audit-ext","version":"1.2.0","hooks":["on_boot"]}`))
	reinstallReq.Header.Set("Content-Type", "application/json")
	reinstallRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(reinstallRR, reinstallReq)
	if reinstallRR.Code != http.StatusCreated {
		t.Fatalf("expected reinstall status 201, got %d body=%s", reinstallRR.Code, reinstallRR.Body.String())
	}

	txUpgradeReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/audit-ext/upgrade/transaction", strings.NewReader(`{"transaction_id":"tx-api-1","target_version":"1.3.0","package_url":"https://example.com/plugins/audit-ext-1.3.0.tgz","package_hash":"sha256:aa130","signature":"sig:sha256:aa130"}`))
	txUpgradeReq.Header.Set("Content-Type", "application/json")
	txUpgradeRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(txUpgradeRR, txUpgradeReq)
	if txUpgradeRR.Code != http.StatusOK {
		t.Fatalf("expected transactional upgrade status 200, got %d body=%s", txUpgradeRR.Code, txUpgradeRR.Body.String())
	}
	if !strings.Contains(txUpgradeRR.Body.String(), `"transaction_id":"tx-api-1"`) || !strings.Contains(txUpgradeRR.Body.String(), `"succeeded":true`) {
		t.Fatalf("expected transactional upgrade success payload, got %s", txUpgradeRR.Body.String())
	}

	provenanceReq := httptest.NewRequest(http.MethodGet, "/admin/v1/plugins/upgrade/provenance?limit=1", nil)
	provenanceRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(provenanceRR, provenanceReq)
	if provenanceRR.Code != http.StatusOK {
		t.Fatalf("expected provenance list status 200, got %d body=%s", provenanceRR.Code, provenanceRR.Body.String())
	}
	if !strings.Contains(provenanceRR.Body.String(), `"transaction_id":"tx-api-1"`) {
		t.Fatalf("expected provenance payload to include tx-api-1, got %s", provenanceRR.Body.String())
	}

	provenanceBadLimitReq := httptest.NewRequest(http.MethodGet, "/admin/v1/plugins/upgrade/provenance?limit=0", nil)
	provenanceBadLimitRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(provenanceBadLimitRR, provenanceBadLimitReq)
	if provenanceBadLimitRR.Code != http.StatusBadRequest {
		t.Fatalf("expected provenance bad limit status 400, got %d", provenanceBadLimitRR.Code)
	}
}

func testMarketplaceIndexSignature(source, signedBy string, expiresAtUnix int64, pkgDigests []string) string {
	raw := source + "|" + signedBy + "|" + fmt.Sprintf("%d", expiresAtUnix) + "|" + fmt.Sprintf("%d", len(pkgDigests))
	for _, digest := range pkgDigests {
		raw += "|" + digest
	}
	sum := sha256.Sum256([]byte(raw))
	return "idxsig:" + hex.EncodeToString(sum[:])
}

func TestSystemStatusRoute_AggregatesModuleCounts(t *testing.T) {
	srv := New(":0", "test-version")
	srv.MountAdminModuleRoutes(testAdminModuleServices(), nil)

	createUserReq := httptest.NewRequest(http.MethodPost, "/admin/v1/users", strings.NewReader(`{"name":"alice","email":"alice@example.com"}`))
	createUserReq.Header.Set("Content-Type", "application/json")
	createUserRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(createUserRR, createUserReq)
	if createUserRR.Code != http.StatusCreated {
		t.Fatalf("expected user create status 201, got %d", createUserRR.Code)
	}

	createRoleReq := httptest.NewRequest(http.MethodPost, "/admin/v1/roles", strings.NewReader(`{"name":"ops","permissions":["user.read"]}`))
	createRoleReq.Header.Set("Content-Type", "application/json")
	createRoleRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(createRoleRR, createRoleReq)
	if createRoleRR.Code != http.StatusCreated {
		t.Fatalf("expected role create status 201, got %d", createRoleRR.Code)
	}

	createPluginReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/manifests", strings.NewReader(`{"name":"audit-ext","version":"1.0.0","hooks":["on_boot"]}`))
	createPluginReq.Header.Set("Content-Type", "application/json")
	createPluginRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(createPluginRR, createPluginReq)
	if createPluginRR.Code != http.StatusCreated {
		t.Fatalf("expected plugin install status 201, got %d", createPluginRR.Code)
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/admin/v1/system/status", nil)
	statusRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(statusRR, statusReq)
	if statusRR.Code != http.StatusOK {
		t.Fatalf("expected system status 200, got %d", statusRR.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(statusRR.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal status response failed: %v", err)
	}
	if got, _ := payload["user_count"].(float64); got != 1 {
		t.Fatalf("expected user_count=1, got %v", payload["user_count"])
	}
	if got, _ := payload["role_count"].(float64); got != 1 {
		t.Fatalf("expected role_count=1, got %v", payload["role_count"])
	}
	if got, _ := payload["plugin_count"].(float64); got != 1 {
		t.Fatalf("expected plugin_count=1, got %v", payload["plugin_count"])
	}
	if got, ok := payload["api_entry_count"].(float64); !ok || got <= 0 {
		t.Fatalf("expected api_entry_count > 0, got %v", payload["api_entry_count"])
	}
}

func TestDatabaseOpsGovernanceRoutes(t *testing.T) {
	srv := New(":0", "test-version")
	srv.MountAdminModuleRoutes(testAdminModuleServices(), nil)

	planReq := httptest.NewRequest(http.MethodPost, "/admin/v1/db/migrations/plan", strings.NewReader(`{"from_version":"2026.04","to_version":"2026.05","steps":["add_table_users","add_index_users_email"]}`))
	planReq.Header.Set("Content-Type", "application/json")
	planRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(planRR, planReq)
	if planRR.Code != http.StatusOK {
		t.Fatalf("expected migration plan status 200, got %d", planRR.Code)
	}

	driftReq := httptest.NewRequest(http.MethodPost, "/admin/v1/db/migrations/drift-detect", strings.NewReader(`{"from_version":"2026.04","to_version":"2026.05","expected_steps":["add_table_users","add_index_users_email"],"applied_steps":["add_table_users","hotfix_sessions_index"]}`))
	driftReq.Header.Set("Content-Type", "application/json")
	driftRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(driftRR, driftReq)
	if driftRR.Code != http.StatusOK {
		t.Fatalf("expected drift detect status 200, got %d body=%s", driftRR.Code, driftRR.Body.String())
	}
	if !strings.Contains(driftRR.Body.String(), `"drift_detected":true`) || !strings.Contains(driftRR.Body.String(), `"impact_grade":"high"`) {
		t.Fatalf("expected drift detected payload with impact grade, got %s", driftRR.Body.String())
	}

	driftBadReq := httptest.NewRequest(http.MethodPost, "/admin/v1/db/migrations/drift-detect", strings.NewReader(`{"from_version":"2026.04","to_version":"2026.05"}`))
	driftBadReq.Header.Set("Content-Type", "application/json")
	driftBadRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(driftBadRR, driftBadReq)
	if driftBadRR.Code != http.StatusBadRequest {
		t.Fatalf("expected drift detect validation status 400, got %d", driftBadRR.Code)
	}

	backupReq := httptest.NewRequest(http.MethodPost, "/admin/v1/db/backup", strings.NewReader(`{"backup_id":"bk-001","reason":"pre-release"}`))
	backupReq.Header.Set("Content-Type", "application/json")
	backupRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(backupRR, backupReq)
	if backupRR.Code != http.StatusCreated {
		t.Fatalf("expected backup status 201, got %d", backupRR.Code)
	}

	catalogReq := httptest.NewRequest(http.MethodGet, "/admin/v1/db/backups/catalog?limit=10", nil)
	catalogRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(catalogRR, catalogReq)
	if catalogRR.Code != http.StatusOK {
		t.Fatalf("expected backup catalog status 200, got %d body=%s", catalogRR.Code, catalogRR.Body.String())
	}
	if !strings.Contains(catalogRR.Body.String(), `"backup_id":"bk-001"`) {
		t.Fatalf("expected backup catalog item, got %s", catalogRR.Body.String())
	}

	restoreDrillForbiddenReq := httptest.NewRequest(http.MethodPost, "/admin/v1/db/restore/drills", strings.NewReader(`{"backup_id":"bk-001","confirm_token":"","expected_max_rto_ms":500,"expected_max_rpo_ms":300}`))
	restoreDrillForbiddenReq.Header.Set("Content-Type", "application/json")
	restoreDrillForbiddenRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(restoreDrillForbiddenRR, restoreDrillForbiddenReq)
	if restoreDrillForbiddenRR.Code != http.StatusForbidden {
		t.Fatalf("expected restore drill forbidden status 403, got %d", restoreDrillForbiddenRR.Code)
	}

	restoreDrillReq := httptest.NewRequest(http.MethodPost, "/admin/v1/db/restore/drills", strings.NewReader(`{"backup_id":"bk-001","confirm_token":"I_UNDERSTAND","expected_max_rto_ms":500,"expected_max_rpo_ms":300}`))
	restoreDrillReq.Header.Set("Content-Type", "application/json")
	restoreDrillRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(restoreDrillRR, restoreDrillReq)
	if restoreDrillRR.Code != http.StatusOK {
		t.Fatalf("expected restore drill status 200, got %d body=%s", restoreDrillRR.Code, restoreDrillRR.Body.String())
	}
	if !strings.Contains(restoreDrillRR.Body.String(), `"rto_compliant":true`) || !strings.Contains(restoreDrillRR.Body.String(), `"rpo_compliant":true`) || !strings.Contains(restoreDrillRR.Body.String(), `"data_check_passed":true`) {
		t.Fatalf("expected restore drill RTO/RPO/data-check payload, got %s", restoreDrillRR.Body.String())
	}

	restoreDrillListReq := httptest.NewRequest(http.MethodGet, "/admin/v1/db/restore/drills?limit=10", nil)
	restoreDrillListRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(restoreDrillListRR, restoreDrillListReq)
	if restoreDrillListRR.Code != http.StatusOK {
		t.Fatalf("expected restore drill list status 200, got %d body=%s", restoreDrillListRR.Code, restoreDrillListRR.Body.String())
	}
	if !strings.Contains(restoreDrillListRR.Body.String(), `"drill_id":"drill-`) {
		t.Fatalf("expected drill evidence list payload, got %s", restoreDrillListRR.Body.String())
	}

	restoreForbiddenReq := httptest.NewRequest(http.MethodPost, "/admin/v1/db/restore", strings.NewReader(`{"backup_id":"bk-001","confirm_token":""}`))
	restoreForbiddenReq.Header.Set("Content-Type", "application/json")
	restoreForbiddenRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(restoreForbiddenRR, restoreForbiddenReq)
	if restoreForbiddenRR.Code != http.StatusForbidden {
		t.Fatalf("expected restore forbidden status 403, got %d", restoreForbiddenRR.Code)
	}

	restoreReq := httptest.NewRequest(http.MethodPost, "/admin/v1/db/restore", strings.NewReader(`{"backup_id":"bk-001","confirm_token":"I_UNDERSTAND"}`))
	restoreReq.Header.Set("Content-Type", "application/json")
	restoreRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(restoreRR, restoreReq)
	if restoreRR.Code != http.StatusOK {
		t.Fatalf("expected restore status 200, got %d", restoreRR.Code)
	}

	sqlBlockedReq := httptest.NewRequest(http.MethodPost, "/admin/v1/db/sql/execute", strings.NewReader(`{"sql":"DROP TABLE users"}`))
	sqlBlockedReq.Header.Set("Content-Type", "application/json")
	sqlBlockedRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(sqlBlockedRR, sqlBlockedReq)
	if sqlBlockedRR.Code != http.StatusForbidden {
		t.Fatalf("expected dangerous sql blocked status 403, got %d", sqlBlockedRR.Code)
	}

	readOnlyReq := httptest.NewRequest(http.MethodPost, "/admin/v1/db/sql/execute", strings.NewReader(`{"sql":"SELECT 1","sql_class":"read_only"}`))
	readOnlyReq.Header.Set("Content-Type", "application/json")
	readOnlyRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(readOnlyRR, readOnlyReq)
	if readOnlyRR.Code != http.StatusOK || !strings.Contains(readOnlyRR.Body.String(), `"sql_class":"read_only"`) {
		t.Fatalf("expected read-only sql approved, got status=%d body=%s", readOnlyRR.Code, readOnlyRR.Body.String())
	}

	writeGuardMissingConfirmReq := httptest.NewRequest(http.MethodPost, "/admin/v1/db/sql/execute", strings.NewReader(`{"sql":"UPDATE users SET status='ok'","sql_class":"write_guarded"}`))
	writeGuardMissingConfirmReq.Header.Set("Content-Type", "application/json")
	writeGuardMissingConfirmRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(writeGuardMissingConfirmRR, writeGuardMissingConfirmReq)
	if writeGuardMissingConfirmRR.Code != http.StatusForbidden {
		t.Fatalf("expected write-guarded confirmation required status 403, got %d", writeGuardMissingConfirmRR.Code)
	}

	writeGuardReq := httptest.NewRequest(http.MethodPost, "/admin/v1/db/sql/execute", strings.NewReader(`{"sql":"UPDATE users SET status='ok'","sql_class":"write_guarded","confirm_token":"I_UNDERSTAND"}`))
	writeGuardReq.Header.Set("Content-Type", "application/json")
	writeGuardRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(writeGuardRR, writeGuardReq)
	if writeGuardRR.Code != http.StatusOK || !strings.Contains(writeGuardRR.Body.String(), `"sql_class":"write_guarded"`) {
		t.Fatalf("expected write-guarded sql approved, got status=%d body=%s", writeGuardRR.Code, writeGuardRR.Body.String())
	}

	sqlAllowedReq := httptest.NewRequest(http.MethodPost, "/admin/v1/db/sql/execute", strings.NewReader(`{"sql":"DELETE FROM sessions WHERE expired=1","sql_class":"destructive_confirmed","allow_dangerous":true,"confirm_token":"I_UNDERSTAND","confirm_token_dual":"CONFIRM_DESTRUCTIVE_SQL"}`))
	sqlAllowedReq.Header.Set("Content-Type", "application/json")
	sqlAllowedRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(sqlAllowedRR, sqlAllowedReq)
	if sqlAllowedRR.Code != http.StatusOK {
		t.Fatalf("expected controlled sql status 200, got %d", sqlAllowedRR.Code)
	}
	if !strings.Contains(sqlAllowedRR.Body.String(), `"allowed":true`) || !strings.Contains(sqlAllowedRR.Body.String(), `"sql_class":"destructive_confirmed"`) {
		t.Fatalf("expected allowed sql response, got %s", sqlAllowedRR.Body.String())
	}
}

func TestMultiInstanceConsistencyRoutes(t *testing.T) {
	srv := New(":0", "test-version")
	srv.MountAdminModuleRoutes(testAdminModuleServices(), nil)

	createJobReq := httptest.NewRequest(http.MethodPost, "/admin/v1/jobs", strings.NewReader(`{"name":"daily-sync","schedule":"0 0 * * *"}`))
	createJobReq.Header.Set("Content-Type", "application/json")
	createJobRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(createJobRR, createJobReq)
	if createJobRR.Code != http.StatusCreated {
		t.Fatalf("expected create job status 201, got %d", createJobRR.Code)
	}

	heartbeatReq := httptest.NewRequest(http.MethodPost, "/admin/v1/sessions/consistency/heartbeat", strings.NewReader(`{"session_id":"sess-1","instance_id":"node-a","version":10}`))
	heartbeatReq.Header.Set("Content-Type", "application/json")
	heartbeatRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(heartbeatRR, heartbeatReq)
	if heartbeatRR.Code != http.StatusOK {
		t.Fatalf("expected heartbeat status 200, got %d", heartbeatRR.Code)
	}

	conflictReq := httptest.NewRequest(http.MethodPost, "/admin/v1/sessions/consistency/heartbeat", strings.NewReader(`{"session_id":"sess-1","instance_id":"node-b","version":10}`))
	conflictReq.Header.Set("Content-Type", "application/json")
	conflictRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(conflictRR, conflictReq)
	if conflictRR.Code != http.StatusOK || !strings.Contains(conflictRR.Body.String(), `"consistent":false`) {
		t.Fatalf("expected writer conflict response, got status=%d body=%s", conflictRR.Code, conflictRR.Body.String())
	}

	sessionStatusReq := httptest.NewRequest(http.MethodGet, "/admin/v1/sessions/sess-1/consistency", nil)
	sessionStatusRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(sessionStatusRR, sessionStatusReq)
	if sessionStatusRR.Code != http.StatusOK {
		t.Fatalf("expected session consistency status 200, got %d", sessionStatusRR.Code)
	}

	claimReq := httptest.NewRequest(http.MethodPost, "/admin/v1/jobs/1/dispatch-claim", strings.NewReader(`{"execution_key":"job-1:20260426T100000Z","instance_id":"node-a"}`))
	claimReq.Header.Set("Content-Type", "application/json")
	claimRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(claimRR, claimReq)
	if claimRR.Code != http.StatusOK || !strings.Contains(claimRR.Body.String(), `"claimed":true`) {
		t.Fatalf("expected dispatch claim accepted, got status=%d body=%s", claimRR.Code, claimRR.Body.String())
	}

	dupReq := httptest.NewRequest(http.MethodPost, "/admin/v1/jobs/1/dispatch-claim", strings.NewReader(`{"execution_key":"job-1:20260426T100000Z","instance_id":"node-b"}`))
	dupReq.Header.Set("Content-Type", "application/json")
	dupRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(dupRR, dupReq)
	if dupRR.Code != http.StatusOK || !strings.Contains(dupRR.Body.String(), `"duplicate_blocked":true`) {
		t.Fatalf("expected duplicate blocked response, got status=%d body=%s", dupRR.Code, dupRR.Body.String())
	}

	claimStatusReq := httptest.NewRequest(http.MethodGet, "/admin/v1/job-dispatch-claims/job-1:20260426T100000Z", nil)
	claimStatusRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(claimStatusRR, claimStatusReq)
	if claimStatusRR.Code != http.StatusOK {
		t.Fatalf("expected claim status 200, got %d", claimStatusRR.Code)
	}
}

func TestReleaseGovernanceRoutes(t *testing.T) {
	srv := New(":0", "test-version")
	srv.MountAdminModuleRoutes(testAdminModuleServices(), nil)

	evidenceReq := httptest.NewRequest(http.MethodPost, "/admin/v1/release-governance/evidence", strings.NewReader(`{"milestone":"E8-step1","go_test_passed":true,"go_race_passed":true,"readme_synced":true,"benchmark_ns_per_op":3300,"baseline_ns_per_op":3000,"benchmark_command":"go test -bench=BenchmarkAdminUsersListEndpoint -benchmem ./internal/app"}`))
	evidenceReq.Header.Set("Content-Type", "application/json")
	evidenceRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(evidenceRR, evidenceReq)
	if evidenceRR.Code != http.StatusCreated {
		t.Fatalf("expected evidence submit status 201, got %d", evidenceRR.Code)
	}

	scoreReq := httptest.NewRequest(http.MethodGet, "/admin/v1/release-governance/scorecard/E8-step1?allowed_regression=0.15", nil)
	scoreRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(scoreRR, scoreReq)
	if scoreRR.Code != http.StatusOK {
		t.Fatalf("expected release scorecard status 200, got %d", scoreRR.Code)
	}
	if !strings.Contains(scoreRR.Body.String(), `"release_ready":true`) {
		t.Fatalf("expected release_ready=true in scorecard, got %s", scoreRR.Body.String())
	}
}

func TestRuntimeMetricsRoute_ReturnsSnapshot(t *testing.T) {
	srv := New(":0", "test-version")
	srv.MountAdminModuleRoutes(testAdminModuleServices(), nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/v1/system/runtime-metrics", nil)
	rr := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected runtime metrics status 200, got %d", rr.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal runtime metrics response failed: %v", err)
	}
	if got, ok := payload["goroutines"].(float64); !ok || got < 1 {
		t.Fatalf("expected goroutines >= 1, got %v", payload["goroutines"])
	}
	if _, ok := payload["memory_alloc_bytes"].(float64); !ok {
		t.Fatalf("expected memory_alloc_bytes field, got %v", payload["memory_alloc_bytes"])
	}
	if got, ok := payload["snapshot_unix_sec"].(float64); !ok || got <= 0 {
		t.Fatalf("expected snapshot_unix_sec > 0, got %v", payload["snapshot_unix_sec"])
	}
}

func TestNodeHealthRoute_ReturnsDependencyStatus(t *testing.T) {
	srv := New(":0", "test-version")
	srv.MountAdminModuleRoutes(testAdminModuleServices(), nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/v1/system/node-health", nil)
	rr := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected node health status 200, got %d", rr.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal node health response failed: %v", err)
	}
	if got, ok := payload["node_status"].(string); !ok || got != "up" {
		t.Fatalf("expected node_status=up, got %v", payload["node_status"])
	}
	deps, ok := payload["dependencies"].([]any)
	if !ok || len(deps) == 0 {
		t.Fatalf("expected non-empty dependencies list, got %v", payload["dependencies"])
	}
}

func TestDashboardAggregateRoute_ReturnsUnifiedSnapshot(t *testing.T) {
	resetDashboardJWTProvenanceMetricsForTest()
	srv := New(":0", "test-version")
	srv.MountAdminModuleRoutes(testAdminModuleServices(), nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/v1/system/dashboard", nil)
	rr := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected dashboard aggregate status 200, got %d", rr.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal dashboard aggregate response failed: %v", err)
	}
	if got, ok := payload["generated_at_unix_sec"].(float64); !ok || got <= 0 {
		t.Fatalf("expected generated_at_unix_sec > 0, got %v", payload["generated_at_unix_sec"])
	}
	contract, ok := payload["contract"].(map[string]any)
	if !ok {
		t.Fatalf("expected contract object, got %v", payload["contract"])
	}
	if got, ok := contract["name"].(string); !ok || got != "dashboard-ui-bootstrap" {
		t.Fatalf("expected contract.name=dashboard-ui-bootstrap, got %v", contract["name"])
	}
	if got, ok := contract["version"].(string); !ok || got != "v1" {
		t.Fatalf("expected contract.version=v1, got %v", contract["version"])
	}
	sections, ok := contract["required_sections"].([]any)
	if !ok || len(sections) < 7 {
		t.Fatalf("expected contract.required_sections with 7 entries, got %v", contract["required_sections"])
	}
	authSession, ok := payload["auth_session"].(map[string]any)
	if !ok {
		t.Fatalf("expected auth_session object, got %v", payload["auth_session"])
	}
	if got, ok := authSession["auth_mode_hint"].(string); !ok || got != "none" {
		t.Fatalf("expected auth_session.auth_mode_hint=none, got %v", authSession["auth_mode_hint"])
	}
	authObs, ok := payload["auth_observability"].(map[string]any)
	if !ok {
		t.Fatalf("expected auth_observability object, got %v", payload["auth_observability"])
	}
	if _, ok := authObs["total_failure"].(float64); !ok {
		t.Fatalf("expected auth_observability.total_failure field, got %v", authObs["total_failure"])
	}
	authActionability, ok := payload["auth_actionability"].(map[string]any)
	if !ok {
		t.Fatalf("expected auth_actionability object, got %v", payload["auth_actionability"])
	}
	if got, ok := authActionability["recommended_auth_mode"].(string); !ok || got != "hmac-sha256" {
		t.Fatalf("expected auth_actionability.recommended_auth_mode=hmac-sha256, got %v", authActionability["recommended_auth_mode"])
	}
	if _, ok := authActionability["next_actions"].([]any); !ok {
		t.Fatalf("expected auth_actionability.next_actions array, got %v", authActionability["next_actions"])
	}
	jwtBootstrap, ok := payload["jwt_session_bootstrap"].(map[string]any)
	if !ok {
		t.Fatalf("expected jwt_session_bootstrap object, got %v", payload["jwt_session_bootstrap"])
	}
	if got, ok := jwtBootstrap["token_format"].(string); !ok || got != "none" {
		t.Fatalf("expected jwt_session_bootstrap.token_format=none, got %v", jwtBootstrap["token_format"])
	}
	if got, ok := jwtBootstrap["session_state"].(string); !ok || got != "none" {
		t.Fatalf("expected jwt_session_bootstrap.session_state=none, got %v", jwtBootstrap["session_state"])
	}
	if got, ok := jwtBootstrap["verification_state"].(string); !ok || got != "not_present" {
		t.Fatalf("expected jwt_session_bootstrap.verification_state=not_present, got %v", jwtBootstrap["verification_state"])
	}
	bridge, ok := jwtBootstrap["middleware_bridge"].(map[string]any)
	if !ok {
		t.Fatalf("expected jwt_session_bootstrap.middleware_bridge object, got %v", jwtBootstrap["middleware_bridge"])
	}
	if got, ok := bridge["source"].(string); !ok || got != "none" {
		t.Fatalf("expected middleware_bridge.source=none, got %v", bridge["source"])
	}
	if got, ok := bridge["source_provenance"].([]any); !ok || len(got) != 0 {
		t.Fatalf("expected empty middleware_bridge.source_provenance, got %v", bridge["source_provenance"])
	}
	auditExport, ok := jwtBootstrap["provenance_audit_export"].(map[string]any)
	if !ok {
		t.Fatalf("expected jwt_session_bootstrap.provenance_audit_export object, got %v", jwtBootstrap["provenance_audit_export"])
	}
	if got, ok := auditExport["enabled"].(bool); !ok || got {
		t.Fatalf("expected provenance_audit_export.enabled=false, got %v", auditExport["enabled"])
	}
	if got, ok := auditExport["verification_state"].(string); !ok || got != "not_present" {
		t.Fatalf("expected provenance_audit_export.verification_state=not_present, got %v", auditExport["verification_state"])
	}
	opsMetrics, ok := auditExport["operational_metrics"].(map[string]any)
	if !ok {
		t.Fatalf("expected provenance_audit_export.operational_metrics object, got %v", auditExport["operational_metrics"])
	}
	if got, ok := opsMetrics["exports_total"].(float64); !ok || got < 1 {
		t.Fatalf("expected operational_metrics.exports_total >= 1, got %v", opsMetrics["exports_total"])
	}
	if got, ok := opsMetrics["disabled_total"].(float64); !ok || got < 1 {
		t.Fatalf("expected operational_metrics.disabled_total >= 1, got %v", opsMetrics["disabled_total"])
	}
	slo, ok := auditExport["slo_dashboard"].(map[string]any)
	if !ok {
		t.Fatalf("expected provenance_audit_export.slo_dashboard object, got %v", auditExport["slo_dashboard"])
	}
	if got, ok := slo["window"].(string); !ok || got != "30d" {
		t.Fatalf("expected slo_dashboard.window=30d, got %v", slo["window"])
	}
	if got, ok := slo["target_reliability"].(float64); !ok || got != 0.99 {
		t.Fatalf("expected slo_dashboard.target_reliability=0.99, got %v", slo["target_reliability"])
	}
	budget, ok := auditExport["error_budget_policy"].(map[string]any)
	if !ok {
		t.Fatalf("expected provenance_audit_export.error_budget_policy object, got %v", auditExport["error_budget_policy"])
	}
	if got, ok := budget["window"].(string); !ok || got != "30d" {
		t.Fatalf("expected error_budget_policy.window=30d, got %v", budget["window"])
	}
	status, ok := payload["status"].(map[string]any)
	if !ok {
		t.Fatalf("expected status object, got %v", payload["status"])
	}
	if _, ok := status["user_count"].(float64); !ok {
		t.Fatalf("expected status.user_count field, got %v", status["user_count"])
	}
	runtimeMetrics, ok := payload["runtime_metrics"].(map[string]any)
	if !ok {
		t.Fatalf("expected runtime_metrics object, got %v", payload["runtime_metrics"])
	}
	if got, ok := runtimeMetrics["goroutines"].(float64); !ok || got < 1 {
		t.Fatalf("expected runtime_metrics.goroutines >= 1, got %v", runtimeMetrics["goroutines"])
	}
	nodeHealth, ok := payload["node_health"].(map[string]any)
	if !ok {
		t.Fatalf("expected node_health object, got %v", payload["node_health"])
	}
	if got, ok := nodeHealth["node_status"].(string); !ok || got == "" {
		t.Fatalf("expected node_health.node_status, got %v", nodeHealth["node_status"])
	}
}

func TestDashboardAggregateRoute_AuthSessionAlignmentWithHeaders(t *testing.T) {
	srv := New(":0", "test-version")
	srv.MountAdminModuleRoutes(testAdminModuleServices(), nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/v1/system/dashboard", nil)
	req.Header.Set("X-Admin-Token", "secret")
	req.Header.Set(HeaderAdminRoleID, "1")
	rr := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected dashboard aggregate status 200, got %d", rr.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal dashboard aggregate response failed: %v", err)
	}
	authSession, ok := payload["auth_session"].(map[string]any)
	if !ok {
		t.Fatalf("expected auth_session object, got %v", payload["auth_session"])
	}
	if got, ok := authSession["authenticated"].(bool); !ok || !got {
		t.Fatalf("expected auth_session.authenticated=true, got %v", authSession["authenticated"])
	}
	if got, ok := authSession["auth_mode_hint"].(string); !ok || got != "static-token" {
		t.Fatalf("expected auth_session.auth_mode_hint=static-token, got %v", authSession["auth_mode_hint"])
	}
	if got, ok := authSession["role_id"].(string); !ok || got != "1" {
		t.Fatalf("expected auth_session.role_id=1, got %v", authSession["role_id"])
	}
	authObs, ok := payload["auth_observability"].(map[string]any)
	if !ok {
		t.Fatalf("expected auth_observability object, got %v", payload["auth_observability"])
	}
	if got, ok := authObs["total_failure"].(float64); !ok || got < 0 {
		t.Fatalf("expected auth_observability.total_failure >= 0, got %v", authObs["total_failure"])
	}
	authActionability, ok := payload["auth_actionability"].(map[string]any)
	if !ok {
		t.Fatalf("expected auth_actionability object, got %v", payload["auth_actionability"])
	}
	if got, ok := authActionability["severity"].(string); !ok || got == "" {
		t.Fatalf("expected auth_actionability.severity, got %v", authActionability["severity"])
	}
	if got, ok := authActionability["docs"].([]any); !ok || len(got) == 0 {
		t.Fatalf("expected auth_actionability.docs list, got %v", authActionability["docs"])
	}
}

func TestDashboardAggregateRoute_JWTSessionBootstrapFromBearerToken(t *testing.T) {
	srv := New(":0", "test-version")
	srv.MountAdminModuleRoutes(testAdminModuleServices(), nil)

	expiresAt := time.Now().UTC().Add(15 * time.Minute).Unix()
	token := testUnsignedJWT(map[string]any{
		"sub": "user-42",
		"iss": "skoll-test",
		"aud": []string{"admin-ui", "ops"},
		"iat": time.Now().UTC().Unix(),
		"exp": expiresAt,
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/v1/system/dashboard", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected dashboard aggregate status 200, got %d", rr.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal dashboard aggregate response failed: %v", err)
	}
	jwtBootstrap, ok := payload["jwt_session_bootstrap"].(map[string]any)
	if !ok {
		t.Fatalf("expected jwt_session_bootstrap object, got %v", payload["jwt_session_bootstrap"])
	}
	if got, ok := jwtBootstrap["token_format"].(string); !ok || got != "bearer-jwt" {
		t.Fatalf("expected jwt_session_bootstrap.token_format=bearer-jwt, got %v", jwtBootstrap["token_format"])
	}
	if got, ok := jwtBootstrap["subject"].(string); !ok || got != "user-42" {
		t.Fatalf("expected jwt_session_bootstrap.subject=user-42, got %v", jwtBootstrap["subject"])
	}
	if got, ok := jwtBootstrap["issuer"].(string); !ok || got != "skoll-test" {
		t.Fatalf("expected jwt_session_bootstrap.issuer=skoll-test, got %v", jwtBootstrap["issuer"])
	}
	if got, ok := jwtBootstrap["expires_at_unix_sec"].(float64); !ok || int64(got) != expiresAt {
		t.Fatalf("expected jwt_session_bootstrap.expires_at_unix_sec=%d, got %v", expiresAt, jwtBootstrap["expires_at_unix_sec"])
	}
	if got, ok := jwtBootstrap["session_state"].(string); !ok || got != "active" {
		t.Fatalf("expected jwt_session_bootstrap.session_state=active, got %v", jwtBootstrap["session_state"])
	}
	if got, ok := jwtBootstrap["verification_state"].(string); !ok || got != "unverified" {
		t.Fatalf("expected jwt_session_bootstrap.verification_state=unverified, got %v", jwtBootstrap["verification_state"])
	}
	if got, ok := jwtBootstrap["trust_level"].(string); !ok || got != "low" {
		t.Fatalf("expected jwt_session_bootstrap.trust_level=low, got %v", jwtBootstrap["trust_level"])
	}
	bridge, ok := jwtBootstrap["middleware_bridge"].(map[string]any)
	if !ok {
		t.Fatalf("expected jwt_session_bootstrap.middleware_bridge object, got %v", jwtBootstrap["middleware_bridge"])
	}
	if got, ok := bridge["source"].(string); !ok || got != "none" {
		t.Fatalf("expected middleware_bridge.source=none, got %v", bridge["source"])
	}
	if got, ok := jwtBootstrap["refresh_recommended"].(bool); !ok || got {
		t.Fatalf("expected jwt_session_bootstrap.refresh_recommended=false, got %v", jwtBootstrap["refresh_recommended"])
	}
}

func TestDashboardAggregateRoute_JWTMiddlewareBridgePromotesVerifiedTrust(t *testing.T) {
	resetDashboardJWTProvenanceMetricsForTest()
	srv := New(":0", "test-version")
	srv.MountAdminModuleRoutes(testAdminModuleServices(), nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/v1/system/dashboard", nil)
	req.Header.Set(HeaderAdminJWTVerified, "true")
	req.Header.Set(HeaderAdminJWTSubject, "bridge-user")
	req.Header.Set(HeaderAdminRoleID, "9")
	req.Header.Set(HeaderAdminJWTClaimsVersion, "v1")
	req.Header.Set(HeaderAdminJWTSource, "gateway")
	req.Header.Set(HeaderAdminJWTSourceChain, "edge-auth,gateway")
	rr := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected dashboard aggregate status 200, got %d", rr.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal dashboard aggregate response failed: %v", err)
	}
	jwtBootstrap, ok := payload["jwt_session_bootstrap"].(map[string]any)
	if !ok {
		t.Fatalf("expected jwt_session_bootstrap object, got %v", payload["jwt_session_bootstrap"])
	}
	if got, ok := jwtBootstrap["claims_trusted"].(bool); !ok || !got {
		t.Fatalf("expected claims_trusted=true, got %v", jwtBootstrap["claims_trusted"])
	}
	if got, ok := jwtBootstrap["verification_state"].(string); !ok || got != "verified" {
		t.Fatalf("expected verification_state=verified, got %v", jwtBootstrap["verification_state"])
	}
	if got, ok := jwtBootstrap["trust_level"].(string); !ok || got != "trusted" {
		t.Fatalf("expected trust_level=trusted, got %v", jwtBootstrap["trust_level"])
	}
	bridge, ok := jwtBootstrap["middleware_bridge"].(map[string]any)
	if !ok {
		t.Fatalf("expected middleware_bridge object, got %v", jwtBootstrap["middleware_bridge"])
	}
	if got, ok := bridge["source"].(string); !ok || got != "gateway" {
		t.Fatalf("expected middleware_bridge.source=gateway, got %v", bridge["source"])
	}
	if got, ok := bridge["source_provenance"].([]any); !ok || len(got) != 2 {
		t.Fatalf("expected middleware_bridge.source_provenance with 2 items, got %v", bridge["source_provenance"])
	}
	if got, ok := bridge["subject"].(string); !ok || got != "bridge-user" {
		t.Fatalf("expected middleware_bridge.subject=bridge-user, got %v", bridge["subject"])
	}
	if got, ok := bridge["role_source"].(string); !ok || got != "x-admin-role-id" {
		t.Fatalf("expected middleware_bridge.role_source=x-admin-role-id, got %v", bridge["role_source"])
	}
	auditExport, ok := jwtBootstrap["provenance_audit_export"].(map[string]any)
	if !ok {
		t.Fatalf("expected provenance_audit_export object, got %v", jwtBootstrap["provenance_audit_export"])
	}
	if got, ok := auditExport["enabled"].(bool); !ok || !got {
		t.Fatalf("expected provenance_audit_export.enabled=true, got %v", auditExport["enabled"])
	}
	if got, ok := auditExport["source_path"].(string); !ok || got != "edge-auth>gateway" {
		t.Fatalf("expected provenance_audit_export.source_path=edge-auth>gateway, got %v", auditExport["source_path"])
	}
	if got, ok := auditExport["verification_state"].(string); !ok || got != "verified" {
		t.Fatalf("expected provenance_audit_export.verification_state=verified, got %v", auditExport["verification_state"])
	}
	opsMetrics, ok := auditExport["operational_metrics"].(map[string]any)
	if !ok {
		t.Fatalf("expected provenance_audit_export.operational_metrics object, got %v", auditExport["operational_metrics"])
	}
	if got, ok := opsMetrics["enabled_total"].(float64); !ok || got < 1 {
		t.Fatalf("expected operational_metrics.enabled_total >= 1, got %v", opsMetrics["enabled_total"])
	}
	if got, ok := opsMetrics["verified_total"].(float64); !ok || got < 1 {
		t.Fatalf("expected operational_metrics.verified_total >= 1, got %v", opsMetrics["verified_total"])
	}
	slo, ok := auditExport["slo_dashboard"].(map[string]any)
	if !ok {
		t.Fatalf("expected provenance_audit_export.slo_dashboard object, got %v", auditExport["slo_dashboard"])
	}
	if got, ok := slo["status"].(string); !ok || got == "" {
		t.Fatalf("expected slo_dashboard.status, got %v", slo["status"])
	}
	budget, ok := auditExport["error_budget_policy"].(map[string]any)
	if !ok {
		t.Fatalf("expected provenance_audit_export.error_budget_policy object, got %v", auditExport["error_budget_policy"])
	}
	if _, ok := budget["action"].(string); !ok {
		t.Fatalf("expected error_budget_policy.action string, got %v", budget["action"])
	}
}

func TestCollectDashboardJWTSessionBootstrap_InvalidBearer(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/admin/v1/system/dashboard", nil)
	req.Header.Set("Authorization", "Bearer not-a-jwt")

	jwtBootstrap := collectDashboardJWTSessionBootstrap(req, time.Now().UTC())
	if jwtBootstrap.TokenFormat != "bearer-non-jwt" {
		t.Fatalf("expected token_format=bearer-non-jwt, got %s", jwtBootstrap.TokenFormat)
	}
	if jwtBootstrap.ParseError == "" {
		t.Fatalf("expected parse_error for invalid bearer token")
	}
	if jwtBootstrap.VerificationState != "invalid" {
		t.Fatalf("expected verification_state=invalid, got %s", jwtBootstrap.VerificationState)
	}
	if !jwtBootstrap.RefreshRecommended {
		t.Fatalf("expected refresh_recommended=true for invalid bearer token")
	}
}

func TestCollectDashboardJWTSessionBootstrap_ExpiringTokenNeedsRefresh(t *testing.T) {
	now := time.Now().UTC()
	expiresAt := now.Add(2 * time.Minute).Unix()
	token := testUnsignedJWT(map[string]any{
		"sub": "user-expiring",
		"exp": expiresAt,
	})
	req := httptest.NewRequest(http.MethodGet, "/admin/v1/system/dashboard", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	jwtBootstrap := collectDashboardJWTSessionBootstrap(req, now)
	if jwtBootstrap.SessionState != "expiring" {
		t.Fatalf("expected session_state=expiring, got %s", jwtBootstrap.SessionState)
	}
	if !jwtBootstrap.RefreshRecommended {
		t.Fatalf("expected refresh_recommended=true for expiring token")
	}
	if jwtBootstrap.RefreshReason != "token_expiring_soon" {
		t.Fatalf("expected refresh_reason=token_expiring_soon, got %s", jwtBootstrap.RefreshReason)
	}
	if jwtBootstrap.VerificationState != "unverified" {
		t.Fatalf("expected verification_state=unverified, got %s", jwtBootstrap.VerificationState)
	}
}

func TestCollectDashboardJWTSessionBootstrap_ExpiredTokenNeedsRefresh(t *testing.T) {
	now := time.Now().UTC()
	expiresAt := now.Add(-1 * time.Minute).Unix()
	token := testUnsignedJWT(map[string]any{
		"sub": "user-expired",
		"exp": expiresAt,
	})
	req := httptest.NewRequest(http.MethodGet, "/admin/v1/system/dashboard", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	jwtBootstrap := collectDashboardJWTSessionBootstrap(req, now)
	if jwtBootstrap.SessionState != "expired" {
		t.Fatalf("expected session_state=expired, got %s", jwtBootstrap.SessionState)
	}
	if !jwtBootstrap.Expired {
		t.Fatalf("expected expired=true")
	}
	if !jwtBootstrap.RefreshRecommended {
		t.Fatalf("expected refresh_recommended=true for expired token")
	}
	if jwtBootstrap.VerificationState != "unverified" {
		t.Fatalf("expected verification_state=unverified, got %s", jwtBootstrap.VerificationState)
	}
}

func TestDeriveJWTVerificationHints_UnverifiedBearerJWT(t *testing.T) {
	state, hint, trustLevel, message := deriveJWTVerificationHints("bearer-jwt", "active", false, "")
	if state != "unverified" {
		t.Fatalf("expected verification state unverified, got %s", state)
	}
	if trustLevel != "low" {
		t.Fatalf("expected trust level low, got %s", trustLevel)
	}
	if hint == "" || message == "" {
		t.Fatalf("expected non-empty hint and message")
	}
}

func TestDeriveJWTVerificationHints_InvalidToken(t *testing.T) {
	state, _, trustLevel, _ := deriveJWTVerificationHints("unsupported", "invalid", false, "bad auth header")
	if state != "invalid" {
		t.Fatalf("expected verification state invalid, got %s", state)
	}
	if trustLevel != "untrusted" {
		t.Fatalf("expected trust level untrusted, got %s", trustLevel)
	}
}

func testUnsignedJWT(claims map[string]any) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payloadBytes, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	return header + "." + payload + "."
}

func TestCollectDashboardAuthActionability_NoneModeSuggestsEnablement(t *testing.T) {
	actionability := collectDashboardAuthActionability(
		dashboardAuthSessionContext{AuthModeHint: "none"},
		dashboardAuthObservability{},
	)

	if actionability.Severity != "warning" {
		t.Fatalf("expected severity=warning for none mode, got %s", actionability.Severity)
	}
	if actionability.RecommendedAuthMode != "hmac-sha256" {
		t.Fatalf("expected recommended_auth_mode=hmac-sha256, got %s", actionability.RecommendedAuthMode)
	}
	if len(actionability.NextActions) == 0 {
		t.Fatalf("expected next_actions for none mode")
	}
	if !strings.Contains(strings.Join(actionability.NextActions, " "), "Enable admin auth") {
		t.Fatalf("expected enable admin auth guidance, got %v", actionability.NextActions)
	}
}

func TestCollectDashboardAuthActionability_HighFailurePromotesCritical(t *testing.T) {
	actionability := collectDashboardAuthActionability(
		dashboardAuthSessionContext{AuthModeHint: "hmac-sha256", Authenticated: true, HasRoleBinding: true},
		dashboardAuthObservability{
			HMACSHA256:   dashboardAuthModeCounters{Success: 5, Failure: 15, Reasons: map[string]uint64{"invalid_signature": 10}},
			TotalFailure: 15,
		},
	)

	if actionability.Severity != "critical" {
		t.Fatalf("expected severity=critical for high failure rate, got %s", actionability.Severity)
	}
	if actionability.FailureRate <= 0 {
		t.Fatalf("expected failure_rate > 0, got %f", actionability.FailureRate)
	}
	if len(actionability.TopFailureReasons) == 0 {
		t.Fatalf("expected non-empty top_failure_reasons")
	}
}
