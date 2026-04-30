package dashboard

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	admincontracts "github.com/tinboxw/skoll/internal/app/admin/contracts"
	adminhardening "github.com/tinboxw/skoll/internal/app/admin/hardening"
	adminsecurity "github.com/tinboxw/skoll/internal/app/admin/security"
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

type testAPIRegistry struct {
	entries []string
}

func (r *testAPIRegistry) RegisterMany(entries []string) {
	r.entries = append(r.entries, entries...)
}

func TestHandler_RegisterAndEndpoints(t *testing.T) {
	mux := http.NewServeMux()
	registry := &testAPIRegistry{}

	h := NewHandler(
		func() any { return map[string]any{"kind": "status", "ok": true} },
		func() any { return map[string]any{"kind": "runtime", "goroutines": 8} },
		func() any { return map[string]any{"kind": "node", "node_status": "up"} },
		func(_ *http.Request) any { return map[string]any{"kind": "dashboard", "generated_at_unix_sec": 1} },
	)
	h.Register(mux, nil, registry)

	if len(registry.entries) != 4 {
		t.Fatalf("expected 4 routes registered, got %d", len(registry.entries))
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/admin/v1/system/status", nil)
	statusRR := httptest.NewRecorder()
	mux.ServeHTTP(statusRR, statusReq)
	if statusRR.Code != http.StatusOK || !strings.Contains(statusRR.Body.String(), `"kind":"status"`) {
		t.Fatalf("expected status response payload, got code=%d body=%s", statusRR.Code, statusRR.Body.String())
	}

	runtimeReq := httptest.NewRequest(http.MethodGet, "/admin/v1/system/runtime-metrics", nil)
	runtimeRR := httptest.NewRecorder()
	mux.ServeHTTP(runtimeRR, runtimeReq)
	if runtimeRR.Code != http.StatusOK || !strings.Contains(runtimeRR.Body.String(), `"kind":"runtime"`) {
		t.Fatalf("expected runtime response payload, got code=%d body=%s", runtimeRR.Code, runtimeRR.Body.String())
	}

	nodeReq := httptest.NewRequest(http.MethodGet, "/admin/v1/system/node-health", nil)
	nodeRR := httptest.NewRecorder()
	mux.ServeHTTP(nodeRR, nodeReq)
	if nodeRR.Code != http.StatusOK || !strings.Contains(nodeRR.Body.String(), `"kind":"node"`) {
		t.Fatalf("expected node response payload, got code=%d body=%s", nodeRR.Code, nodeRR.Body.String())
	}

	dashboardReq := httptest.NewRequest(http.MethodGet, "/admin/v1/system/dashboard", nil)
	dashboardRR := httptest.NewRecorder()
	mux.ServeHTTP(dashboardRR, dashboardReq)
	if dashboardRR.Code != http.StatusOK || !strings.Contains(dashboardRR.Body.String(), `"kind":"dashboard"`) {
		t.Fatalf("expected dashboard response payload, got code=%d body=%s", dashboardRR.Code, dashboardRR.Body.String())
	}
}

// testUnsignedJWT creates a minimal unsigned JWT (alg=none) for testing collectors.
func testUnsignedJWT(claims map[string]any) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payloadBytes, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	return header + "." + payload + "."
}

// newServicesMux creates a real dashboard handler mux backed by real in-memory services,
// mirroring the closure pattern from admin_modules.go.
func newServicesMux() *http.ServeMux {
	memFileBackend := &memTestFileBackend{items: make(map[string][]byte)}
	services := admincontracts.AdminModuleServices{
		Users:        user.NewService(),
		Roles:        role.NewService(),
		Menus:        menu.NewService(),
		Audit:        audit.NewService(),
		Configs:      config.NewService(),
		Dictionaries: dictionary.NewService(),
		Files:        fileservice.NewService(memFileBackend),
		Jobs:         jobscheduler.NewService(),
		Generator:    modgenerator.NewService(),
		Plugins:      pluginmgr.NewService(),
		RBAC:         rbac.NewService(),
		APIs:         apiregistry.NewService(),
		Releases:     releasegov.NewService(),
	}
	hardeningH := adminhardening.NewHandler(services.Audit)
	mux := http.NewServeMux()
	NewHandler(
		func() any { return CollectSystemStatus(services) },
		func() any { return CollectRuntimeMetrics() },
		func() any { return CollectNodeHealth(services) },
		func(r *http.Request) any {
			authSession := CollectAuthSessionContext(r)
			authObservability := CollectAuthObservability()
			jwtSession := CollectJWTSessionBootstrap(r, time.Now().UTC())
			return AggregateResponse{
				Contract:             CollectContractDescriptor(),
				GeneratedAtUnixSec:   time.Now().UTC().Unix(),
				AuthSession:          authSession,
				AuthObservability:    authObservability,
				AuthActionability:    CollectAuthActionability(authSession, authObservability),
				JWTSessionBootstrap:  jwtSession,
				Status:               CollectSystemStatus(services),
				RuntimeMetrics:       CollectRuntimeMetrics(),
				NodeHealth:           CollectNodeHealth(services),
				SchedulerReliability: services.Jobs.ReliabilitySnapshot(time.Now().UTC()),
				HardeningPosture:     hardeningH.Snapshot(),
			}
		},
	).Register(mux, nil, &testAPIRegistry{})
	return mux
}

// memTestFileBackend is a minimal in-memory file backend for service construction.
type memTestFileBackend struct{ items map[string][]byte }

func (b *memTestFileBackend) Save(_ string, content []byte) (string, error) {
	key := "f-test"
	b.items[key] = append([]byte(nil), content...)
	return key, nil
}
func (b *memTestFileBackend) Open(key string) ([]byte, error) {
	v, ok := b.items[key]
	if !ok {
		return nil, nil
	}
	return append([]byte(nil), v...), nil
}

func TestRuntimeMetricsRoute_ReturnsSnapshot(t *testing.T) {
	mux := newServicesMux()

	req := httptest.NewRequest(http.MethodGet, "/admin/v1/system/runtime-metrics", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
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
	mux := newServicesMux()

	req := httptest.NewRequest(http.MethodGet, "/admin/v1/system/node-health", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
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

func TestDashboardAggregateRoute_AuthSessionAlignmentWithHeaders(t *testing.T) {
	mux := newServicesMux()

	req := httptest.NewRequest(http.MethodGet, "/admin/v1/system/dashboard", nil)
	req.Header.Set("X-Admin-Token", "secret")
	req.Header.Set(adminsecurity.HeaderAdminRoleID, "1")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
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
}

func TestDashboardAggregateRoute_JWTSessionBootstrapFromBearerToken(t *testing.T) {
	mux := newServicesMux()

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
	mux.ServeHTTP(rr, req)
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
	if got, ok := jwtBootstrap["session_state"].(string); !ok || got != "active" {
		t.Fatalf("expected jwt_session_bootstrap.session_state=active, got %v", jwtBootstrap["session_state"])
	}
	if got, ok := jwtBootstrap["verification_state"].(string); !ok || got != "unverified" {
		t.Fatalf("expected jwt_session_bootstrap.verification_state=unverified, got %v", jwtBootstrap["verification_state"])
	}
	if got, ok := jwtBootstrap["trust_level"].(string); !ok || got != "low" {
		t.Fatalf("expected jwt_session_bootstrap.trust_level=low, got %v", jwtBootstrap["trust_level"])
	}
}

func TestDashboardAggregateRoute_JWTMiddlewareBridgePromotesVerifiedTrust(t *testing.T) {
	ResetJWTProvenanceMetricsForTest()
	mux := newServicesMux()

	req := httptest.NewRequest(http.MethodGet, "/admin/v1/system/dashboard", nil)
	req.Header.Set(adminsecurity.HeaderAdminJWTVerified, "true")
	req.Header.Set(adminsecurity.HeaderAdminJWTSubject, "bridge-user")
	req.Header.Set(adminsecurity.HeaderAdminRoleID, "9")
	req.Header.Set(adminsecurity.HeaderAdminJWTClaimsVersion, "v1")
	req.Header.Set(adminsecurity.HeaderAdminJWTSource, "gateway")
	req.Header.Set(adminsecurity.HeaderAdminJWTSourceChain, "edge-auth,gateway")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
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
}

func TestCollectDashboardJWTSessionBootstrap_InvalidBearer(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/admin/v1/system/dashboard", nil)
	req.Header.Set("Authorization", "Bearer not-a-jwt")

	jwtBootstrap := CollectJWTSessionBootstrap(req, time.Now().UTC())
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

	jwtBootstrap := CollectJWTSessionBootstrap(req, now)
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

	jwtBootstrap := CollectJWTSessionBootstrap(req, now)
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
	state, hint, trustLevel, message := DeriveJWTVerificationHints("bearer-jwt", "active", false, "")
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
	state, _, trustLevel, _ := DeriveJWTVerificationHints("unsupported", "invalid", false, "bad auth header")
	if state != "invalid" {
		t.Fatalf("expected verification state invalid, got %s", state)
	}
	if trustLevel != "untrusted" {
		t.Fatalf("expected trust level untrusted, got %s", trustLevel)
	}
}

func TestCollectDashboardAuthActionability_NoneModeSuggestsEnablement(t *testing.T) {
	actionability := CollectAuthActionability(
		AuthSessionContext{AuthModeHint: "none"},
		AuthObservability{},
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
	actionability := CollectAuthActionability(
		AuthSessionContext{AuthModeHint: "hmac-sha256", Authenticated: true, HasRoleBinding: true},
		AuthObservability{
			HMACSHA256:   AuthModeCounters{Success: 5, Failure: 15, Reasons: map[string]uint64{"invalid_signature": 10}},
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
