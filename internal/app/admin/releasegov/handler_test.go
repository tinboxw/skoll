package releasegov

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tinboxw/skoll/internal/module/audit"
	releasegov "github.com/tinboxw/skoll/internal/module/releasegov"
)

type testAPIRegistry struct {
	entries []string
}

func (r *testAPIRegistry) RegisterMany(entries []string) {
	r.entries = append(r.entries, entries...)
}

func newReleaseGovMux() *http.ServeMux {
	mux := http.NewServeMux()
	NewHandler(releasegov.NewService(), audit.NewService()).Register(mux, nil, &testAPIRegistry{})
	return mux
}

func TestReleaseGovernanceRoutes(t *testing.T) {
	mux := newReleaseGovMux()

	evidenceReq := httptest.NewRequest(http.MethodPost, "/admin/v1/release-governance/evidence", strings.NewReader(`{"milestone":"E8-step1","go_test_passed":true,"go_race_passed":true,"readme_synced":true,"benchmark_ns_per_op":3300,"baseline_ns_per_op":3000,"benchmark_command":"go test -bench=BenchmarkAdminUsersListEndpoint -benchmem ./internal/app"}`))
	evidenceReq.Header.Set("Content-Type", "application/json")
	evidenceRR := httptest.NewRecorder()
	mux.ServeHTTP(evidenceRR, evidenceReq)
	if evidenceRR.Code != http.StatusCreated {
		t.Fatalf("expected evidence submit status 201, got %d", evidenceRR.Code)
	}

	scoreReq := httptest.NewRequest(http.MethodGet, "/admin/v1/release-governance/scorecard/E8-step1?allowed_regression=0.15", nil)
	scoreRR := httptest.NewRecorder()
	mux.ServeHTTP(scoreRR, scoreReq)
	if scoreRR.Code != http.StatusOK {
		t.Fatalf("expected release scorecard status 200, got %d", scoreRR.Code)
	}
	if !strings.Contains(scoreRR.Body.String(), `"release_ready":true`) {
		t.Fatalf("expected release_ready=true in scorecard, got %s", scoreRR.Body.String())
	}

	policyReq := httptest.NewRequest(http.MethodPut, "/admin/v1/release-governance/blocking-policy", strings.NewReader(`{"allowed_regression_ratio":0.05,"block_on_go_test_failure":true,"block_on_go_race_failure":true,"block_on_readme_not_synced":true,"block_on_missing_evidence":true}`))
	policyReq.Header.Set("Content-Type", "application/json")
	policyRR := httptest.NewRecorder()
	mux.ServeHTTP(policyRR, policyReq)
	if policyRR.Code != http.StatusOK {
		t.Fatalf("expected release blocking policy status 200, got %d body=%s", policyRR.Code, policyRR.Body.String())
	}
	if !strings.Contains(policyRR.Body.String(), `"allowed_regression_ratio":0.05`) {
		t.Fatalf("expected allowed_regression_ratio=0.05 in policy response, got %s", policyRR.Body.String())
	}

	policyGetReq := httptest.NewRequest(http.MethodGet, "/admin/v1/release-governance/blocking-policy", nil)
	policyGetRR := httptest.NewRecorder()
	mux.ServeHTTP(policyGetRR, policyGetReq)
	if policyGetRR.Code != http.StatusOK {
		t.Fatalf("expected get release blocking policy status 200, got %d body=%s", policyGetRR.Code, policyGetRR.Body.String())
	}

	blockDecisionReq := httptest.NewRequest(http.MethodGet, "/admin/v1/release-governance/block-decision/E8-step1", nil)
	blockDecisionRR := httptest.NewRecorder()
	mux.ServeHTTP(blockDecisionRR, blockDecisionReq)
	if blockDecisionRR.Code != http.StatusOK {
		t.Fatalf("expected block decision status 200, got %d body=%s", blockDecisionRR.Code, blockDecisionRR.Body.String())
	}
	if !strings.Contains(blockDecisionRR.Body.String(), `"blocked":true`) || !strings.Contains(blockDecisionRR.Body.String(), `"performance_regression_exceeded"`) {
		t.Fatalf("expected blocked decision with regression reason, got %s", blockDecisionRR.Body.String())
	}

	invalidCheckpointReq := httptest.NewRequest(http.MethodPut, "/admin/v1/release-governance/parity-closure/checkpoints", strings.NewReader(`{"items":[{"reference_project":"gin-vue-admin","capability":"rbac_policy_governance","status":"completed","evidence_links":[],"owner":"platform-team"}]}`))
	invalidCheckpointReq.Header.Set("Content-Type", "application/json")
	invalidCheckpointRR := httptest.NewRecorder()
	mux.ServeHTTP(invalidCheckpointRR, invalidCheckpointReq)
	if invalidCheckpointRR.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid parity checkpoint status 400, got %d body=%s", invalidCheckpointRR.Code, invalidCheckpointRR.Body.String())
	}

	checkpointReq := httptest.NewRequest(http.MethodPut, "/admin/v1/release-governance/parity-closure/checkpoints", strings.NewReader(`{"items":[{"reference_project":"gin-vue-admin","capability":"rbac_policy_governance","status":"completed","evidence_links":["docs/milestones/E11-step3-policy-persistence-governance-baseline.md"],"owner":"platform-team"},{"reference_project":"hisiphp","capability":"plugin_upgrade_governance","status":"partial","evidence_links":["docs/milestones/E14-step3-upgrade-transaction-checkpoints-and-provenance.md"],"known_gap":"pending plugin marketplace compatibility matrix sample","owner":"platform-team"}]}`))
	checkpointReq.Header.Set("Content-Type", "application/json")
	checkpointRR := httptest.NewRecorder()
	mux.ServeHTTP(checkpointRR, checkpointReq)
	if checkpointRR.Code != http.StatusOK {
		t.Fatalf("expected parity checkpoint set status 200, got %d body=%s", checkpointRR.Code, checkpointRR.Body.String())
	}
	if !strings.Contains(checkpointRR.Body.String(), `"reference_project":"gin-vue-admin"`) || !strings.Contains(checkpointRR.Body.String(), `"capability":"plugin_upgrade_governance"`) {
		t.Fatalf("expected parity checkpoint payload, got %s", checkpointRR.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/admin/v1/release-governance/parity-closure/checkpoints", nil)
	listRR := httptest.NewRecorder()
	mux.ServeHTTP(listRR, listReq)
	if listRR.Code != http.StatusOK {
		t.Fatalf("expected parity checkpoint list status 200, got %d body=%s", listRR.Code, listRR.Body.String())
	}

	reportReq := httptest.NewRequest(http.MethodGet, "/admin/v1/release-governance/parity-closure/report", nil)
	reportRR := httptest.NewRecorder()
	mux.ServeHTTP(reportRR, reportReq)
	if reportRR.Code != http.StatusOK {
		t.Fatalf("expected parity closure report status 200, got %d body=%s", reportRR.Code, reportRR.Body.String())
	}
	if !strings.Contains(reportRR.Body.String(), `"total_checkpoints":2`) || !strings.Contains(reportRR.Body.String(), `"known_gap_checkpoints":1`) {
		t.Fatalf("expected parity closure report totals, got %s", reportRR.Body.String())
	}
}
