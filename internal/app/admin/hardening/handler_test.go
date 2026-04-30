package hardening

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tinboxw/skoll/internal/module/audit"
)

type testAPIRegistry struct {
	entries []string
}

func (r *testAPIRegistry) RegisterMany(entries []string) {
	r.entries = append(r.entries, entries...)
}

func newHardeningMux() *http.ServeMux {
	mux := http.NewServeMux()
	NewHandler(audit.NewService()).Register(mux, nil, &testAPIRegistry{})
	return mux
}

func TestSystemEndpointGuardrailsRoutes(t *testing.T) {
	mux := newHardeningMux()

	invalidReq := httptest.NewRequest(http.MethodPut, "/admin/v1/system/hardening/endpoint-guardrails", strings.NewReader(`{"profiles":[{"endpoint":"/admin/v1/jobs","rate_limit_rpm":0,"timeout_millis":1200,"circuit_error_threshold":5,"circuit_open_window_sec":60,"enabled":true}]}`))
	invalidReq.Header.Set("Content-Type", "application/json")
	invalidRR := httptest.NewRecorder()
	mux.ServeHTTP(invalidRR, invalidReq)
	if invalidRR.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid guardrail status 400, got %d body=%s", invalidRR.Code, invalidRR.Body.String())
	}

	putReq := httptest.NewRequest(http.MethodPut, "/admin/v1/system/hardening/endpoint-guardrails", strings.NewReader(`{"profiles":[{"endpoint":"/admin/v1/jobs","rate_limit_rpm":120,"timeout_millis":1200,"circuit_error_threshold":5,"circuit_open_window_sec":60,"enabled":true},{"endpoint":"/admin/v1/system/dashboard","rate_limit_rpm":90,"timeout_millis":1000,"circuit_error_threshold":3,"circuit_open_window_sec":45,"enabled":true}]}`))
	putReq.Header.Set("Content-Type", "application/json")
	putRR := httptest.NewRecorder()
	mux.ServeHTTP(putRR, putReq)
	if putRR.Code != http.StatusOK {
		t.Fatalf("expected set endpoint guardrails status 200, got %d body=%s", putRR.Code, putRR.Body.String())
	}
	if !strings.Contains(putRR.Body.String(), `"endpoint":"/admin/v1/jobs"`) || !strings.Contains(putRR.Body.String(), `"rate_limit_rpm":120`) {
		t.Fatalf("expected guardrail payload in put response, got %s", putRR.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/admin/v1/system/hardening/endpoint-guardrails", nil)
	listRR := httptest.NewRecorder()
	mux.ServeHTTP(listRR, listReq)
	if listRR.Code != http.StatusOK {
		t.Fatalf("expected list endpoint guardrails status 200, got %d body=%s", listRR.Code, listRR.Body.String())
	}
	if !strings.Contains(listRR.Body.String(), `"endpoint":"/admin/v1/system/dashboard"`) || !strings.Contains(listRR.Body.String(), `"timeout_millis":1000`) {
		t.Fatalf("expected endpoint guardrails list payload, got %s", listRR.Body.String())
	}
}

func TestSystemAlertProfilesRoutes(t *testing.T) {
	mux := newHardeningMux()

	invalidReq := httptest.NewRequest(http.MethodPut, "/admin/v1/system/hardening/alert-profiles", strings.NewReader(`{"profiles":[{"metric":"admin_request_latency_p95_ms","warn_threshold":120,"critical_threshold":80,"window_seconds":300,"runbook":"docs/runbooks/admin-latency.md","owner":"sre-oncall","enabled":true}]}`))
	invalidReq.Header.Set("Content-Type", "application/json")
	invalidRR := httptest.NewRecorder()
	mux.ServeHTTP(invalidRR, invalidReq)
	if invalidRR.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid alert profile status 400, got %d body=%s", invalidRR.Code, invalidRR.Body.String())
	}

	putReq := httptest.NewRequest(http.MethodPut, "/admin/v1/system/hardening/alert-profiles", strings.NewReader(`{"profiles":[{"metric":"admin_request_latency_p95_ms","warn_threshold":120,"critical_threshold":250,"window_seconds":300,"runbook":"docs/runbooks/admin-latency.md","owner":"sre-oncall","enabled":true},{"metric":"scheduler_dead_letter_count","warn_threshold":5,"critical_threshold":10,"window_seconds":600,"runbook":"docs/runbooks/scheduler-dead-letter.md","owner":"platform-ops","enabled":true}]}`))
	putReq.Header.Set("Content-Type", "application/json")
	putRR := httptest.NewRecorder()
	mux.ServeHTTP(putRR, putReq)
	if putRR.Code != http.StatusOK {
		t.Fatalf("expected set alert profiles status 200, got %d body=%s", putRR.Code, putRR.Body.String())
	}
	if !strings.Contains(putRR.Body.String(), `"metric":"admin_request_latency_p95_ms"`) || !strings.Contains(putRR.Body.String(), `"critical_threshold":250`) {
		t.Fatalf("expected alert profile payload in put response, got %s", putRR.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/admin/v1/system/hardening/alert-profiles", nil)
	listRR := httptest.NewRecorder()
	mux.ServeHTTP(listRR, listReq)
	if listRR.Code != http.StatusOK {
		t.Fatalf("expected list alert profiles status 200, got %d body=%s", listRR.Code, listRR.Body.String())
	}
	if !strings.Contains(listRR.Body.String(), `"metric":"scheduler_dead_letter_count"`) || !strings.Contains(listRR.Body.String(), `"runbook":"docs/runbooks/scheduler-dead-letter.md"`) {
		t.Fatalf("expected alert profiles list payload, got %s", listRR.Body.String())
	}
}

func TestSystemIncidentRunbooksAndFaultDrillsRoutes(t *testing.T) {
	mux := newHardeningMux()

	invalidRunbookReq := httptest.NewRequest(http.MethodPut, "/admin/v1/system/hardening/incident-runbooks", strings.NewReader(`{"profiles":[{"domain":"scheduler","severity":"p1","runbook":"docs/runbooks/scheduler.md","owner":"platform-ops","escalation":"pagerduty:platform","mitigation_sla_seconds":0}]}`))
	invalidRunbookReq.Header.Set("Content-Type", "application/json")
	invalidRunbookRR := httptest.NewRecorder()
	mux.ServeHTTP(invalidRunbookRR, invalidRunbookReq)
	if invalidRunbookRR.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid runbook status 400, got %d body=%s", invalidRunbookRR.Code, invalidRunbookRR.Body.String())
	}

	runbookReq := httptest.NewRequest(http.MethodPut, "/admin/v1/system/hardening/incident-runbooks", strings.NewReader(`{"profiles":[{"domain":"auth","severity":"p1","runbook":"docs/runbooks/auth-outage.md","owner":"security-ops","escalation":"pagerduty:security","mitigation_sla_seconds":600},{"domain":"scheduler","severity":"p2","runbook":"docs/runbooks/scheduler.md","owner":"platform-ops","escalation":"pagerduty:platform","mitigation_sla_seconds":900}]}`))
	runbookReq.Header.Set("Content-Type", "application/json")
	runbookRR := httptest.NewRecorder()
	mux.ServeHTTP(runbookRR, runbookReq)
	if runbookRR.Code != http.StatusOK {
		t.Fatalf("expected set runbooks status 200, got %d body=%s", runbookRR.Code, runbookRR.Body.String())
	}
	if !strings.Contains(runbookRR.Body.String(), `"domain":"auth"`) || !strings.Contains(runbookRR.Body.String(), `"mitigation_sla_seconds":600`) {
		t.Fatalf("expected runbook payload in response, got %s", runbookRR.Body.String())
	}

	listRunbookReq := httptest.NewRequest(http.MethodGet, "/admin/v1/system/hardening/incident-runbooks", nil)
	listRunbookRR := httptest.NewRecorder()
	mux.ServeHTTP(listRunbookRR, listRunbookReq)
	if listRunbookRR.Code != http.StatusOK {
		t.Fatalf("expected list runbooks status 200, got %d body=%s", listRunbookRR.Code, listRunbookRR.Body.String())
	}
	if !strings.Contains(listRunbookRR.Body.String(), `"domain":"scheduler"`) {
		t.Fatalf("expected scheduler runbook payload, got %s", listRunbookRR.Body.String())
	}

	invalidDrillReq := httptest.NewRequest(http.MethodPost, "/admin/v1/system/hardening/fault-drills", strings.NewReader(`{"scenario":"","domain":"scheduler","injector":"chaos-mesh","mitigation_evidence":"rolled back worker","residual_risk":"low"}`))
	invalidDrillReq.Header.Set("Content-Type", "application/json")
	invalidDrillRR := httptest.NewRecorder()
	mux.ServeHTTP(invalidDrillRR, invalidDrillReq)
	if invalidDrillRR.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid fault drill status 400, got %d body=%s", invalidDrillRR.Code, invalidDrillRR.Body.String())
	}

	createDrillReq := httptest.NewRequest(http.MethodPost, "/admin/v1/system/hardening/fault-drills", strings.NewReader(`{"scenario":"scheduler worker timeout storm","domain":"scheduler","injector":"chaos-mesh","mitigation_evidence":"retry policy tightened and unhealthy worker drained","residual_risk":"pending queue burst under peak load"}`))
	createDrillReq.Header.Set("Content-Type", "application/json")
	createDrillRR := httptest.NewRecorder()
	mux.ServeHTTP(createDrillRR, createDrillReq)
	if createDrillRR.Code != http.StatusCreated {
		t.Fatalf("expected create fault drill status 201, got %d body=%s", createDrillRR.Code, createDrillRR.Body.String())
	}
	if !strings.Contains(createDrillRR.Body.String(), `"drill_id":"drill-`) || !strings.Contains(createDrillRR.Body.String(), `"domain":"scheduler"`) {
		t.Fatalf("expected fault drill response payload, got %s", createDrillRR.Body.String())
	}

	listDrillReq := httptest.NewRequest(http.MethodGet, "/admin/v1/system/hardening/fault-drills?limit=5", nil)
	listDrillRR := httptest.NewRecorder()
	mux.ServeHTTP(listDrillRR, listDrillReq)
	if listDrillRR.Code != http.StatusOK {
		t.Fatalf("expected list fault drills status 200, got %d body=%s", listDrillRR.Code, listDrillRR.Body.String())
	}
	if !strings.Contains(listDrillRR.Body.String(), `"scenario":"scheduler worker timeout storm"`) || !strings.Contains(listDrillRR.Body.String(), `"mitigation_evidence":"retry policy tightened and unhealthy worker drained"`) {
		t.Fatalf("expected fault drill list payload, got %s", listDrillRR.Body.String())
	}
}
