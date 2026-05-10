package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBackendPluginMetricsRoute(t *testing.T) {
	mux := http.NewServeMux()
	registerRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/demo-backend/metrics", nil)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	var got MetricsSnapshot
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode metrics snapshot failed: %v", err)
	}
	if got.TotalRequests != 152 {
		t.Fatalf("expected total requests 152, got %d", got.TotalRequests)
	}
}

func TestBackendPluginAuditRoute(t *testing.T) {
	mux := http.NewServeMux()
	registerRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/demo-backend/audit/report?events=user.create,user.create,role.bind&limit=1", nil)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	var got AuditDigest
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode audit digest failed: %v", err)
	}
	if len(got.TopCategories) != 1 || got.TopCategories[0].Category != "user.create" {
		t.Fatalf("unexpected top categories: %+v", got.TopCategories)
	}
}
