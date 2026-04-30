package jobs

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	adminauth "github.com/tinboxw/skoll/internal/app/admin/auth"
	"github.com/tinboxw/skoll/internal/module/audit"
	"github.com/tinboxw/skoll/internal/module/jobscheduler"
	"github.com/tinboxw/skoll/internal/module/user"
)

type testAPIRegistry struct {
	entries []string
}

func (r *testAPIRegistry) RegisterMany(entries []string) {
	r.entries = append(r.entries, entries...)
}

func newJobsMux() (*http.ServeMux, *jobscheduler.Service, *audit.Service) {
	mux := http.NewServeMux()
	jobSvc := jobscheduler.NewService()
	auditSvc := audit.NewService()
	NewHandler(jobSvc, auditSvc).Register(mux, nil, &testAPIRegistry{})
	return mux, jobSvc, auditSvc
}

// newJobsAuthMux composes jobs + auth handlers on one mux for cross-domain consistency tests.
func newJobsAuthMux() (*http.ServeMux, *jobscheduler.Service, *audit.Service) {
	mux, jobSvc, auditSvc := newJobsMux()
	userSvc := user.NewService()
	adminauth.NewHandler(userSvc, auditSvc).Register(mux, nil, &testAPIRegistry{})
	return mux, jobSvc, auditSvc
}

func TestJobRoutes_CreateRunHistory(t *testing.T) {
	mux, _, _ := newJobsMux()

	createReq := httptest.NewRequest(http.MethodPost, "/admin/v1/jobs", strings.NewReader(`{"name":"daily-sync","schedule":"0 0 * * *"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	mux.ServeHTTP(createRR, createReq)
	if createRR.Code != http.StatusCreated {
		t.Fatalf("expected create job status 201, got %d", createRR.Code)
	}

	runReq := httptest.NewRequest(http.MethodPost, "/admin/v1/jobs/1/run", nil)
	runRR := httptest.NewRecorder()
	mux.ServeHTTP(runRR, runReq)
	if runRR.Code != http.StatusOK {
		t.Fatalf("expected run job status 200, got %d", runRR.Code)
	}

	historyReq := httptest.NewRequest(http.MethodGet, "/admin/v1/jobs/1/history?limit=10", nil)
	historyRR := httptest.NewRecorder()
	mux.ServeHTTP(historyRR, historyReq)
	if historyRR.Code != http.StatusOK {
		t.Fatalf("expected job history status 200, got %d", historyRR.Code)
	}
	if !strings.Contains(historyRR.Body.String(), `"Status":"success"`) {
		t.Fatalf("expected successful run in history, got %s", historyRR.Body.String())
	}
}

func TestMultiInstanceConsistencyRoutes(t *testing.T) {
	// This test spans auth (session consistency) and jobs (dispatch-claim) domains.
	// Both handlers are composed on a single mux sharing the same audit service.
	mux, _, _ := newJobsAuthMux()

	createJobReq := httptest.NewRequest(http.MethodPost, "/admin/v1/jobs", strings.NewReader(`{"name":"daily-sync","schedule":"0 0 * * *"}`))
	createJobReq.Header.Set("Content-Type", "application/json")
	createJobRR := httptest.NewRecorder()
	mux.ServeHTTP(createJobRR, createJobReq)
	if createJobRR.Code != http.StatusCreated {
		t.Fatalf("expected create job status 201, got %d", createJobRR.Code)
	}

	heartbeatReq := httptest.NewRequest(http.MethodPost, "/admin/v1/sessions/consistency/heartbeat", strings.NewReader(`{"session_id":"sess-1","instance_id":"node-a","version":10}`))
	heartbeatReq.Header.Set("Content-Type", "application/json")
	heartbeatRR := httptest.NewRecorder()
	mux.ServeHTTP(heartbeatRR, heartbeatReq)
	if heartbeatRR.Code != http.StatusOK {
		t.Fatalf("expected heartbeat status 200, got %d", heartbeatRR.Code)
	}

	conflictReq := httptest.NewRequest(http.MethodPost, "/admin/v1/sessions/consistency/heartbeat", strings.NewReader(`{"session_id":"sess-1","instance_id":"node-b","version":10}`))
	conflictReq.Header.Set("Content-Type", "application/json")
	conflictRR := httptest.NewRecorder()
	mux.ServeHTTP(conflictRR, conflictReq)
	if conflictRR.Code != http.StatusOK || !strings.Contains(conflictRR.Body.String(), `"consistent":false`) {
		t.Fatalf("expected writer conflict response, got status=%d body=%s", conflictRR.Code, conflictRR.Body.String())
	}

	sessionStatusReq := httptest.NewRequest(http.MethodGet, "/admin/v1/sessions/sess-1/consistency", nil)
	sessionStatusRR := httptest.NewRecorder()
	mux.ServeHTTP(sessionStatusRR, sessionStatusReq)
	if sessionStatusRR.Code != http.StatusOK {
		t.Fatalf("expected session consistency status 200, got %d", sessionStatusRR.Code)
	}

	claimReq := httptest.NewRequest(http.MethodPost, "/admin/v1/jobs/1/dispatch-claim", strings.NewReader(`{"execution_key":"job-1:20260426T100000Z","instance_id":"node-a"}`))
	claimReq.Header.Set("Content-Type", "application/json")
	claimRR := httptest.NewRecorder()
	mux.ServeHTTP(claimRR, claimReq)
	if claimRR.Code != http.StatusOK || !strings.Contains(claimRR.Body.String(), `"claimed":true`) {
		t.Fatalf("expected dispatch claim accepted, got status=%d body=%s", claimRR.Code, claimRR.Body.String())
	}

	dupReq := httptest.NewRequest(http.MethodPost, "/admin/v1/jobs/1/dispatch-claim", strings.NewReader(`{"execution_key":"job-1:20260426T100000Z","instance_id":"node-b"}`))
	dupReq.Header.Set("Content-Type", "application/json")
	dupRR := httptest.NewRecorder()
	mux.ServeHTTP(dupRR, dupReq)
	if dupRR.Code != http.StatusOK || !strings.Contains(dupRR.Body.String(), `"duplicate_blocked":true`) {
		t.Fatalf("expected duplicate blocked response, got status=%d body=%s", dupRR.Code, dupRR.Body.String())
	}

	claimStatusReq := httptest.NewRequest(http.MethodGet, "/admin/v1/job-dispatch-claims/job-1:20260426T100000Z", nil)
	claimStatusRR := httptest.NewRecorder()
	mux.ServeHTTP(claimStatusRR, claimStatusReq)
	if claimStatusRR.Code != http.StatusOK {
		t.Fatalf("expected claim status 200, got %d", claimStatusRR.Code)
	}

	renewForbiddenReq := httptest.NewRequest(http.MethodPost, "/admin/v1/job-dispatch-claims/job-1:20260426T100000Z/renew", strings.NewReader(`{"instance_id":"node-b","lease_ttl_sec":60}`))
	renewForbiddenReq.Header.Set("Content-Type", "application/json")
	renewForbiddenRR := httptest.NewRecorder()
	mux.ServeHTTP(renewForbiddenRR, renewForbiddenReq)
	if renewForbiddenRR.Code != http.StatusForbidden {
		t.Fatalf("expected renew forbidden status 403, got %d", renewForbiddenRR.Code)
	}

	renewReq := httptest.NewRequest(http.MethodPost, "/admin/v1/job-dispatch-claims/job-1:20260426T100000Z/renew", strings.NewReader(`{"instance_id":"node-a","lease_ttl_sec":60}`))
	renewReq.Header.Set("Content-Type", "application/json")
	renewRR := httptest.NewRecorder()
	mux.ServeHTTP(renewRR, renewReq)
	if renewRR.Code != http.StatusOK {
		t.Fatalf("expected renew status 200, got %d body=%s", renewRR.Code, renewRR.Body.String())
	}
	if !strings.Contains(renewRR.Body.String(), `"lease_renewal_count":1`) {
		t.Fatalf("expected lease renewal payload, got %s", renewRR.Body.String())
	}

	retryPolicyReq := httptest.NewRequest(http.MethodPut, "/admin/v1/jobs/1/retry-policy", strings.NewReader(`{"max_retries":4,"backoff_base_millis":200,"backoff_max_millis":1600,"jitter_percent":25}`))
	retryPolicyReq.Header.Set("Content-Type", "application/json")
	retryPolicyRR := httptest.NewRecorder()
	mux.ServeHTTP(retryPolicyRR, retryPolicyReq)
	if retryPolicyRR.Code != http.StatusOK {
		t.Fatalf("expected retry policy set status 200, got %d body=%s", retryPolicyRR.Code, retryPolicyRR.Body.String())
	}

	retryScheduleReq := httptest.NewRequest(http.MethodPost, "/admin/v1/jobs/1/retries/schedule", strings.NewReader(`{"execution_key":"job-1:20260426T100000Z","attempt":2}`))
	retryScheduleReq.Header.Set("Content-Type", "application/json")
	retryScheduleRR := httptest.NewRecorder()
	mux.ServeHTTP(retryScheduleRR, retryScheduleReq)
	if retryScheduleRR.Code != http.StatusOK {
		t.Fatalf("expected retry schedule status 200, got %d body=%s", retryScheduleRR.Code, retryScheduleRR.Body.String())
	}
	if !strings.Contains(retryScheduleRR.Body.String(), `"attempt":2`) {
		t.Fatalf("expected retry schedule payload, got %s", retryScheduleRR.Body.String())
	}

	markDLQReq := httptest.NewRequest(http.MethodPost, "/admin/v1/jobs/1/dead-letters", strings.NewReader(`{"execution_key":"job-1:20260426T100000Z","reason":"retry exhausted","retry_count":4}`))
	markDLQReq.Header.Set("Content-Type", "application/json")
	markDLQRR := httptest.NewRecorder()
	mux.ServeHTTP(markDLQRR, markDLQReq)
	if markDLQRR.Code != http.StatusOK {
		t.Fatalf("expected mark dead-letter status 200, got %d body=%s", markDLQRR.Code, markDLQRR.Body.String())
	}

	listDLQReq := httptest.NewRequest(http.MethodGet, "/admin/v1/jobs/dead-letters?limit=10", nil)
	listDLQRR := httptest.NewRecorder()
	mux.ServeHTTP(listDLQRR, listDLQReq)
	if listDLQRR.Code != http.StatusOK {
		t.Fatalf("expected list dead-letters status 200, got %d body=%s", listDLQRR.Code, listDLQRR.Body.String())
	}
	if !strings.Contains(listDLQRR.Body.String(), `"execution_key":"job-1:20260426T100000Z"`) {
		t.Fatalf("expected dead-letter entry payload, got %s", listDLQRR.Body.String())
	}

	replayDLQReq := httptest.NewRequest(http.MethodPost, "/admin/v1/jobs/dead-letters/job-1:20260426T100000Z/replay", strings.NewReader(`{"operator":"ops-a"}`))
	replayDLQReq.Header.Set("Content-Type", "application/json")
	replayDLQRR := httptest.NewRecorder()
	mux.ServeHTTP(replayDLQRR, replayDLQReq)
	if replayDLQRR.Code != http.StatusOK {
		t.Fatalf("expected replay dead-letter status 200, got %d body=%s", replayDLQRR.Code, replayDLQRR.Body.String())
	}
	if !strings.Contains(replayDLQRR.Body.String(), `"status":"replayed"`) {
		t.Fatalf("expected replayed dead-letter payload, got %s", replayDLQRR.Body.String())
	}

	reliabilityReq := httptest.NewRequest(http.MethodGet, "/admin/v1/jobs/reliability/metrics", nil)
	reliabilityRR := httptest.NewRecorder()
	mux.ServeHTTP(reliabilityRR, reliabilityReq)
	if reliabilityRR.Code != http.StatusOK {
		t.Fatalf("expected reliability metrics status 200, got %d body=%s", reliabilityRR.Code, reliabilityRR.Body.String())
	}
	if !strings.Contains(reliabilityRR.Body.String(), `"retry_schedule_count":1`) || !strings.Contains(reliabilityRR.Body.String(), `"dead_letter_count":1`) {
		t.Fatalf("expected reliability metrics payload, got %s", reliabilityRR.Body.String())
	}
}
