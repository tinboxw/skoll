package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSeparatedPluginHealthRoute(t *testing.T) {
	mux := http.NewServeMux()
	registerRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/demo-separated/health", nil)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	if strings.TrimSpace(resp.Body.String()) != "demo-separated:ok" {
		t.Fatalf("unexpected body: %q", resp.Body.String())
	}
}

func TestSeparatedPluginOverviewRoute(t *testing.T) {
	mux := http.NewServeMux()
	registerRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/demo-separated/overview?plugin_id=demo", nil)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	var got Overview
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode overview failed: %v", err)
	}
	if got.PluginID != "demo" || got.Status != "healthy" {
		t.Fatalf("unexpected overview: %+v", got)
	}
}
