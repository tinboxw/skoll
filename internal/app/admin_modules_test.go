package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	admincontracts "github.com/tinboxw/skoll/internal/app/admin/contracts"
	admindashboard "github.com/tinboxw/skoll/internal/app/admin/dashboard"
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

func testAdminModuleServices() admincontracts.AdminModuleServices {
	return admincontracts.AdminModuleServices{
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

func TestDashboardAggregateRoute_ReturnsUnifiedSnapshot(t *testing.T) {
	admindashboard.ResetJWTProvenanceMetricsForTest()
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
	if !ok || len(sections) < 9 {
		t.Fatalf("expected contract.required_sections with 9 entries, got %v", contract["required_sections"])
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
	schedulerReliability, ok := payload["scheduler_reliability"].(map[string]any)
	if !ok {
		t.Fatalf("expected scheduler_reliability object, got %v", payload["scheduler_reliability"])
	}
	if _, ok := schedulerReliability["retry_schedule_count"].(float64); !ok {
		t.Fatalf("expected scheduler_reliability.retry_schedule_count field, got %v", schedulerReliability["retry_schedule_count"])
	}
	hardeningPosture, ok := payload["hardening_posture"].(map[string]any)
	if !ok {
		t.Fatalf("expected hardening_posture object, got %v", payload["hardening_posture"])
	}
	if _, ok := hardeningPosture["endpoint_guardrails_enabled"].(float64); !ok {
		t.Fatalf("expected hardening_posture.endpoint_guardrails_enabled field, got %v", hardeningPosture["endpoint_guardrails_enabled"])
	}
}
