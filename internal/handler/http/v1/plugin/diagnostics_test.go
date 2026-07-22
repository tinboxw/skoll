package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	pluginruntime "github.com/tinboxw/skoll/internal/plugin"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	jobsvc "github.com/tinboxw/skoll/internal/service/job"
	"github.com/tinboxw/skoll/internal/store/clickhouse"
)

type diagnosticsProviderStub struct {
	query       pluginruntime.DiagnosticQuery
	retryPlugin string
	retryJob    string
}

func (s *diagnosticsProviderStub) Inspect(_ context.Context, pluginID string, query pluginruntime.DiagnosticQuery) (pluginruntime.DiagnosticSnapshot, error) {
	s.query = query
	return pluginruntime.DiagnosticSnapshot{
		PluginID: pluginID, CapturedAt: time.Now().UTC(),
		Jobs: []pluginruntime.DiagnosticJob{{ID: "scan", Kind: "expiry_scan", Status: string(jobsvc.StatusDeadLetter), CanRetry: true}},
	}, nil
}

func (s *diagnosticsProviderStub) RetryDeadLetter(_ context.Context, pluginID, jobID string) (pluginruntime.DiagnosticRetryResult, error) {
	s.retryPlugin, s.retryJob = pluginID, jobID
	return pluginruntime.DiagnosticRetryResult{
		OperationID: "retry-op-1", CompletedAt: time.Now().UTC(), SourceJobID: jobID,
		RetryJob: pluginruntime.DiagnosticJob{ID: jobID + "-retry-1", Status: string(jobsvc.StatusScheduled)},
	}, nil
}

func TestPluginDiagnosticsForwardsFilters(t *testing.T) {
	provider := &diagnosticsProviderStub{}
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, &fakePluginManager{items: map[string]pluginruntime.Info{"pharma_oa": {ID: "pharma_oa"}}}, WithPluginDiagnosticsProvider(provider))
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/plugins/pharma_oa/diagnostics?jobStatus=dead_letter&auditResult=failure&correlation=trace-1&limit=25", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if provider.query.JobStatus != jobsvc.StatusDeadLetter || provider.query.AuditResult != "failure" || provider.query.Correlation != "trace-1" || provider.query.Limit != 25 {
		t.Fatalf("unexpected diagnostic query: %+v", provider.query)
	}
}

func TestPluginDeadLetterRetryRequiresSuperAdminAndExactConfirmation(t *testing.T) {
	provider := &diagnosticsProviderStub{}
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, &fakePluginManager{items: map[string]pluginruntime.Info{"pharma_oa": {ID: "pharma_oa"}}}, WithPluginDiagnosticsProvider(provider))

	for name, tc := range map[string]struct {
		role string
		body string
		want int
	}{
		"role":         {role: "admin", body: `{"confirmPluginId":"pharma_oa","confirmJobId":"scan"}`, want: http.StatusForbidden},
		"confirmation": {role: "super_admin", body: `{"confirmPluginId":"pharma_oa","confirmJobId":"other"}`, want: http.StatusBadRequest},
	} {
		t.Run(name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/v1/plugins/pharma_oa/jobs/scan/retry", bytes.NewBufferString(tc.body))
			mux.ServeHTTP(recorder, withRole(request, tc.role))
			if recorder.Code != tc.want {
				t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestPluginDeadLetterRetryIsAudited(t *testing.T) {
	provider := &diagnosticsProviderStub{}
	auditService := auditsvc.NewService(clickhouse.NewAuditStore())
	mux := http.NewServeMux()
	RegisterPluginRoutes(
		mux,
		&fakePluginManager{items: map[string]pluginruntime.Info{"pharma_oa": {ID: "pharma_oa"}}},
		WithPluginDiagnosticsProvider(provider), WithPluginAuditService(auditService),
	)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/plugins/pharma_oa/jobs/scan/retry", bytes.NewBufferString(`{"confirmPluginId":"pharma_oa","confirmJobId":"scan"}`))
	mux.ServeHTTP(recorder, withRole(request, "super_admin"))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Data pluginruntime.DiagnosticRetryResult `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if provider.retryPlugin != "pharma_oa" || provider.retryJob != "scan" || response.Data.RetryJob.ID == "" {
		t.Fatalf("unexpected retry result: %+v", response.Data)
	}
	records, err := auditService.ListByActor(context.Background(), "u-1", 10)
	if err != nil || len(records) != 1 {
		t.Fatalf("retry audit missing: records=%d err=%v", len(records), err)
	}
	if records[0].Action != "retry_dead_letter" || records[0].Resource != "plugin_job" || records[0].Detail["retryJobId"] != response.Data.RetryJob.ID {
		t.Fatalf("unexpected retry audit: %+v", records[0])
	}
}
