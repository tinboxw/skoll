package app

import "testing"

func TestNormalizePathLabelWhitelistAndFallback(t *testing.T) {
	if got := normalizePathLabel("/health"); got != "/health" {
		t.Fatalf("expected /health, got %q", got)
	}
	if got := normalizePathLabel("/ready"); got != "/ready" {
		t.Fatalf("expected /ready, got %q", got)
	}
	if got := normalizePathLabel("/metrics"); got != "/metrics" {
		t.Fatalf("expected /metrics, got %q", got)
	}
	if got := normalizePathLabel("/admin/ping"); got != "/admin/ping" {
		t.Fatalf("expected /admin/ping, got %q", got)
	}
	if got := normalizePathLabel("/admin/users"); got != "/admin/:path" {
		t.Fatalf("expected /admin/:path, got %q", got)
	}
	if got := normalizePathLabel("/users/123"); got != pathLabelOther {
		t.Fatalf("expected %q for unknown path, got %q", pathLabelOther, got)
	}
}

func TestObserveRequestUsesNormalizedPath(t *testing.T) {
	m := NewMetrics()

	m.ObserveRequest("/dynamic/1", 200)
	m.ObserveRequest("/dynamic/2", 200)
	m.ObserveRequest("/admin/users", 200)
	m.ObserveRequest("/health", 200)

	if got := m.pathOther.Load(); got != 2 {
		t.Fatalf("expected __other__ count 2, got %d", got)
	}
	if got := m.pathHealth.Load(); got != 1 {
		t.Fatalf("expected /health count 1, got %d", got)
	}
	if got := m.pathAdminT.Load(); got != 1 {
		t.Fatalf("expected /admin/:path count 1, got %d", got)
	}
}
