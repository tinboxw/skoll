package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	srv := New(":0", "test-version")
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	srv.httpServer.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}

	if body["status"] != "ok" {
		t.Fatalf("expected status ok, got %q", body["status"])
	}
	if body["version"] != "test-version" {
		t.Fatalf("expected version test-version, got %q", body["version"])
	}
}

func TestReadyEndpoint(t *testing.T) {
	srv := New(":0", "test-version")

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rr := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503 when not ready, got %d", rr.Code)
	}

	srv.SetReady(true)
	rr = httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 when ready, got %d", rr.Code)
	}
}

func BenchmarkHealthEndpoint(b *testing.B) {
	srv := New(":0", "bench")
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		srv.httpServer.Handler.ServeHTTP(rr, req)
	}
}

func TestMetricsEndpoint(t *testing.T) {
	srv := New(":0", "test-version")

	healthReq := httptest.NewRequest(http.MethodGet, "/health", nil)
	healthRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(healthRR, healthReq)

	readyReq := httptest.NewRequest(http.MethodGet, "/ready", nil)
	readyRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(readyRR, readyReq)

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(metricsRR, metricsReq)

	if metricsRR.Code != http.StatusOK {
		t.Fatalf("expected status 200 for metrics, got %d", metricsRR.Code)
	}

	body := metricsRR.Body.String()
	if !strings.Contains(body, "skoll_http_requests_total 2") {
		t.Fatalf("expected requests total metric in body, got %q", body)
	}
	if !strings.Contains(body, "skoll_http_requests_path_total{path=\"/health\"} 1") {
		t.Fatalf("expected /health path metric in body, got %q", body)
	}
	if !strings.Contains(body, "skoll_http_requests_path_total{path=\"/ready\"} 1") {
		t.Fatalf("expected /ready path metric in body, got %q", body)
	}
	if strings.Contains(body, "skoll_http_requests_path_total{path=\"/metrics\"}") {
		t.Fatalf("did not expect /metrics path metric in the same /metrics response, got %q", body)
	}
	if !strings.Contains(body, "skoll_http_responses_status_total{code=\"503\"} 1") {
		t.Fatalf("expected 503 status metric in body, got %q", body)
	}
	if !strings.Contains(body, "skoll_http_responses_status_total{code=\"200\"} 1") {
		t.Fatalf("expected 200 status metric in body, got %q", body)
	}
}

func TestMetricsEndpointIncludesExternalCollector(t *testing.T) {
	srv := New(":0", "test-version")
	srv.AddMetricsCollector(func() string {
		return "# HELP skoll_external_test_metric test metric\n# TYPE skoll_external_test_metric counter\nskoll_external_test_metric 1\n"
	})

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(metricsRR, metricsReq)

	if metricsRR.Code != http.StatusOK {
		t.Fatalf("expected status 200 for metrics, got %d", metricsRR.Code)
	}
	if !strings.Contains(metricsRR.Body.String(), "skoll_external_test_metric 1") {
		t.Fatalf("expected external collector metric in body, got %q", metricsRR.Body.String())
	}
}
