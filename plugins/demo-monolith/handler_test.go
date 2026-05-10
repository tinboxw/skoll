package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMonolithPluginHealthRoute(t *testing.T) {
	mux := http.NewServeMux()
	registerRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/demo-monolith/health", nil)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	if !strings.HasPrefix(strings.TrimSpace(resp.Body.String()), "pong@") {
		t.Fatalf("unexpected health body: %q", resp.Body.String())
	}
}

func TestMonolithPluginManifestRoute(t *testing.T) {
	mux := http.NewServeMux()
	registerRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/demo-monolith/manifest", nil)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	var got PageManifest
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode manifest failed: %v", err)
	}
	if got.EntryPath != "/plugins/demo-monolith" {
		t.Fatalf("unexpected entry path: %q", got.EntryPath)
	}
}
