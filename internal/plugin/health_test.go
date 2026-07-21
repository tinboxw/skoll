package plugin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPHealthCheckerReportsHealthyWithoutLeakingURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method=%s", r.Method)
		}
		if r.Header.Get("User-Agent") != "Skoll-Plugin-Health/1" {
			t.Errorf("user-agent=%s", r.Header.Get("User-Agent"))
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	checker := NewHTTPHealthChecker(time.Second)
	report := checker.Check(context.Background(), Info{
		ID:               "reports",
		State:            StateEnabled,
		ServiceBaseURL:   server.URL,
		ServiceHealthURL: server.URL + "/health?token=secret",
	})
	if !report.Ready() || report.Status != HealthStatusHealthy || report.Code != "health_ok" || report.HTTPStatus != http.StatusNoContent {
		t.Fatalf("unexpected report: %+v", report)
	}
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	if strings.Contains(string(raw), server.URL) || strings.Contains(string(raw), "secret") {
		t.Fatalf("health report leaked endpoint data: %s", raw)
	}
}

func TestHTTPHealthCheckerFailureCodes(t *testing.T) {
	unhealthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer unhealthy.Close()

	tests := []struct {
		name string
		info Info
		code string
	}{
		{name: "disabled", info: Info{ID: "reports", State: StateDisabled}, code: "plugin_not_enabled"},
		{name: "not applicable", info: Info{ID: "frontend", State: StateEnabled}, code: "health_not_applicable"},
		{name: "not configured", info: Info{ID: "reports", State: StateEnabled, ServiceBaseURL: unhealthy.URL}, code: "health_not_configured"},
		{name: "http status", info: Info{ID: "reports", State: StateEnabled, ServiceBaseURL: unhealthy.URL, ServiceHealthURL: unhealthy.URL}, code: "health_http_status"},
		{name: "unreachable", info: Info{ID: "reports", State: StateEnabled, ServiceBaseURL: "http://127.0.0.1:1", ServiceHealthURL: "http://127.0.0.1:1/health"}, code: "health_unreachable"},
	}

	checker := NewHTTPHealthChecker(250 * time.Millisecond)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := checker.Check(context.Background(), tt.info)
			if report.Code != tt.code {
				t.Fatalf("code=%s want=%s report=%+v", report.Code, tt.code, report)
			}
			if tt.name == "not applicable" {
				if !report.Ready() || report.Status != HealthStatusNotApplicable {
					t.Fatalf("not-applicable report should be ready: %+v", report)
				}
				return
			}
			if report.Ready() || report.Status != HealthStatusUnhealthy {
				t.Fatalf("failure report should be unhealthy: %+v", report)
			}
		})
	}
}
