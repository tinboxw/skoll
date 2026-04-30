package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	adminauditlog "github.com/tinboxw/skoll/internal/app/admin/auditlog"
	"github.com/tinboxw/skoll/internal/module/audit"
	"github.com/tinboxw/skoll/internal/module/user"
)

type testAPIRegistry struct {
	entries []string
}

func (r *testAPIRegistry) RegisterMany(entries []string) {
	r.entries = append(r.entries, entries...)
}

func newAuthMux() (*http.ServeMux, *user.Service, *audit.Service) {
	mux := http.NewServeMux()
	userSvc := user.NewService()
	auditSvc := audit.NewService()
	NewHandler(userSvc, auditSvc).Register(mux, nil, &testAPIRegistry{})
	return mux, userSvc, auditSvc
}

// newAuthAuditMux composes auth + auditlog handlers on one mux sharing the same audit service.
func newAuthAuditMux() (*http.ServeMux, *user.Service, *audit.Service) {
	mux, userSvc, auditSvc := newAuthMux()
	adminauditlog.NewHandler(auditSvc).Register(mux, nil, &testAPIRegistry{})
	return mux, userSvc, auditSvc
}

func TestAdminUserRoutes_CreateListGet(t *testing.T) {
	mux, _, _ := newAuthMux()

	createReq := httptest.NewRequest(http.MethodPost, "/admin/v1/users", strings.NewReader(`{"name":"alice","email":"alice@example.com"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	mux.ServeHTTP(createRR, createReq)
	if createRR.Code != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d", createRR.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/admin/v1/users", nil)
	listRR := httptest.NewRecorder()
	mux.ServeHTTP(listRR, listReq)
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
	mux.ServeHTTP(getRR, getReq)
	if getRR.Code != http.StatusOK {
		t.Fatalf("expected get status 200, got %d", getRR.Code)
	}
	if !strings.Contains(getRR.Body.String(), `"Email":"alice@example.com"`) {
		t.Fatalf("expected user email in response, got %s", getRR.Body.String())
	}
}

func TestAdminBulkUserCreateAtomic(t *testing.T) {
	mux, _, _ := newAuthMux()

	bulkReq := httptest.NewRequest(http.MethodPost, "/admin/v1/users/bulk", strings.NewReader(`{"items":[{"name":"alice","email":"alice@example.com"},{"name":"bob","email":"bob@example.com"}]}`))
	bulkReq.Header.Set("Content-Type", "application/json")
	bulkRR := httptest.NewRecorder()
	mux.ServeHTTP(bulkRR, bulkReq)
	if bulkRR.Code != http.StatusCreated {
		t.Fatalf("expected bulk create status 201, got %d body=%s", bulkRR.Code, bulkRR.Body.String())
	}
	if !strings.Contains(bulkRR.Body.String(), `"atomic":true`) {
		t.Fatalf("expected atomic true in bulk create response, got %s", bulkRR.Body.String())
	}

	invalidReq := httptest.NewRequest(http.MethodPost, "/admin/v1/users/bulk", strings.NewReader(`{"items":[{"name":"ok","email":"ok@example.com"},{"name":"","email":"bad@example.com"}]}`))
	invalidReq.Header.Set("Content-Type", "application/json")
	invalidRR := httptest.NewRecorder()
	mux.ServeHTTP(invalidRR, invalidReq)
	if invalidRR.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid bulk create status 400, got %d", invalidRR.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/admin/v1/users", nil)
	listRR := httptest.NewRecorder()
	mux.ServeHTTP(listRR, listReq)
	if !strings.Contains(listRR.Body.String(), `"Email":"alice@example.com"`) || !strings.Contains(listRR.Body.String(), `"Email":"bob@example.com"`) {
		t.Fatalf("expected only valid bulk-created users persisted, got %s", listRR.Body.String())
	}
}

func TestAdminUserSecurityRoutes(t *testing.T) {
	mux, _, _ := newAuthAuditMux()

	createReq := httptest.NewRequest(http.MethodPost, "/admin/v1/users", strings.NewReader(`{"name":"alice","email":"alice@example.com"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	mux.ServeHTTP(createRR, createReq)
	if createRR.Code != http.StatusCreated {
		t.Fatalf("expected user create status 201, got %d", createRR.Code)
	}

	rotateReq := httptest.NewRequest(http.MethodPost, "/admin/v1/users/1/password/rotate", strings.NewReader(`{"min_interval_minutes":0}`))
	rotateReq.Header.Set("Content-Type", "application/json")
	rotateRR := httptest.NewRecorder()
	mux.ServeHTTP(rotateRR, rotateReq)
	if rotateRR.Code != http.StatusOK {
		t.Fatalf("expected password rotate status 200, got %d", rotateRR.Code)
	}

	loginFailureReq := httptest.NewRequest(http.MethodPost, "/admin/v1/users/1/login-failures", strings.NewReader(`{"lock_threshold":1,"lock_duration_minutes":15}`))
	loginFailureReq.Header.Set("Content-Type", "application/json")
	loginFailureRR := httptest.NewRecorder()
	mux.ServeHTTP(loginFailureRR, loginFailureReq)
	if loginFailureRR.Code != http.StatusOK {
		t.Fatalf("expected login failure status 200, got %d", loginFailureRR.Code)
	}
	if !strings.Contains(loginFailureRR.Body.String(), `"locked_until_unix_sec":`) {
		t.Fatalf("expected lock info in response, got %s", loginFailureRR.Body.String())
	}

	mfaReq := httptest.NewRequest(http.MethodPost, "/admin/v1/users/1/mfa", strings.NewReader(`{"enabled":true,"provider":"totp"}`))
	mfaReq.Header.Set("Content-Type", "application/json")
	mfaRR := httptest.NewRecorder()
	mux.ServeHTTP(mfaRR, mfaReq)
	if mfaRR.Code != http.StatusOK {
		t.Fatalf("expected mfa update status 200, got %d", mfaRR.Code)
	}
	if !strings.Contains(mfaRR.Body.String(), `"mfa_enabled":true`) {
		t.Fatalf("expected mfa enabled in response, got %s", mfaRR.Body.String())
	}

	revokeReq := httptest.NewRequest(http.MethodPost, "/admin/v1/sessions/revoke", strings.NewReader(`{"session_id":"sess-1","reason":"manual"}`))
	revokeReq.Header.Set("Content-Type", "application/json")
	revokeRR := httptest.NewRecorder()
	mux.ServeHTTP(revokeRR, revokeReq)
	if revokeRR.Code != http.StatusOK {
		t.Fatalf("expected session revoke status 200, got %d", revokeRR.Code)
	}

	anomalyReq := httptest.NewRequest(http.MethodPost, "/admin/v1/sessions/anomalies", strings.NewReader(`{"session_id":"sess-1","category":"geo_jump","detail":"ip changed"}`))
	anomalyReq.Header.Set("Content-Type", "application/json")
	anomalyRR := httptest.NewRecorder()
	mux.ServeHTTP(anomalyRR, anomalyReq)
	if anomalyRR.Code != http.StatusCreated {
		t.Fatalf("expected session anomaly status 201, got %d body=%s", anomalyRR.Code, anomalyRR.Body.String())
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/admin/v1/sessions/sess-1/status", nil)
	statusRR := httptest.NewRecorder()
	mux.ServeHTTP(statusRR, statusReq)
	if statusRR.Code != http.StatusOK {
		t.Fatalf("expected session status 200, got %d", statusRR.Code)
	}
	if !strings.Contains(statusRR.Body.String(), `"revoked":true`) {
		t.Fatalf("expected revoked session status, got %s", statusRR.Body.String())
	}

	auditReq := httptest.NewRequest(http.MethodGet, "/admin/v1/audit-logs?page=1&size=20&actor=security", nil)
	auditRR := httptest.NewRecorder()
	mux.ServeHTTP(auditRR, auditReq)
	if auditRR.Code != http.StatusOK {
		t.Fatalf("expected audit query status 200, got %d", auditRR.Code)
	}
	if !strings.Contains(auditRR.Body.String(), `"total":5`) {
		t.Fatalf("expected security audit events in response, got %s", auditRR.Body.String())
	}
}

func TestAdminAuthLifecycleRoutes(t *testing.T) {
	mux, _, _ := newAuthMux()

	createReq := httptest.NewRequest(http.MethodPost, "/admin/v1/users", strings.NewReader(`{"name":"alice","email":"alice@example.com"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	mux.ServeHTTP(createRR, createReq)
	if createRR.Code != http.StatusCreated {
		t.Fatalf("expected user create status 201, got %d", createRR.Code)
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/admin/v1/auth/login", strings.NewReader(`{"user_id":1,"role_id":1,"claims_version":"v2"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRR := httptest.NewRecorder()
	mux.ServeHTTP(loginRR, loginReq)
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
	mux.ServeHTTP(refreshRR, refreshReq)
	if refreshRR.Code != http.StatusOK {
		t.Fatalf("expected refresh status 200, got %d body=%s", refreshRR.Code, refreshRR.Body.String())
	}

	sessionReq := httptest.NewRequest(http.MethodGet, "/admin/v1/auth/sessions/"+sessionID, nil)
	sessionRR := httptest.NewRecorder()
	mux.ServeHTTP(sessionRR, sessionReq)
	if sessionRR.Code != http.StatusOK {
		t.Fatalf("expected auth session get status 200, got %d", sessionRR.Code)
	}

	logoutReq := httptest.NewRequest(http.MethodPost, "/admin/v1/auth/logout", strings.NewReader(`{"session_id":"`+sessionID+`","reason":"manual"}`))
	logoutReq.Header.Set("Content-Type", "application/json")
	logoutRR := httptest.NewRecorder()
	mux.ServeHTTP(logoutRR, logoutReq)
	if logoutRR.Code != http.StatusOK {
		t.Fatalf("expected logout status 200, got %d body=%s", logoutRR.Code, logoutRR.Body.String())
	}
	if !strings.Contains(logoutRR.Body.String(), `"revoked":true`) {
		t.Fatalf("expected revoked auth session, got %s", logoutRR.Body.String())
	}
}

func BenchmarkAdminUsersListEndpoint(b *testing.B) {
	mux, userSvc, _ := newAuthMux()
	for i := 0; i < 100; i++ {
		userSvc.Create("user", "user@example.com")
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/v1/users", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
	}
}

func BenchmarkAdminAuthLoginEndpoint(b *testing.B) {
	mux, userSvc, _ := newAuthMux()
	userSvc.Create("alice", "alice@example.com")

	req := httptest.NewRequest(http.MethodPost, "/admin/v1/auth/login", strings.NewReader(`{"user_id":1,"role_id":1,"claims_version":"v2"}`))
	req.Header.Set("Content-Type", "application/json")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
	}
}
