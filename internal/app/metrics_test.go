package app

import (
	"strings"
	"testing"
)

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

func TestDeriveDashboardJWTProvenanceAlertingHints(t *testing.T) {
	hints := deriveDashboardJWTProvenanceAlertingHints(dashboardJWTProvenanceAuditExport{
		Enabled:           true,
		Verified:          false,
		VerificationState: "unverified",
		SourceProvenance:  []string{"edge", "gateway", "admin"},
	})

	joined := strings.Join(hints, ",")
	if !strings.Contains(joined, provenanceHintVerificationUnverified) {
		t.Fatalf("expected unverified hint, got %v", hints)
	}
	if !strings.Contains(joined, provenanceHintClaimsNotVerified) {
		t.Fatalf("expected claims_not_verified hint, got %v", hints)
	}
	if !strings.Contains(joined, provenanceHintClaimsVersionMissing) {
		t.Fatalf("expected claims_version_missing hint, got %v", hints)
	}
	if !strings.Contains(joined, provenanceHintChainDepthHigh) {
		t.Fatalf("expected chain depth high hint, got %v", hints)
	}
}

func TestMetricsPrometheusIncludesDashboardJWTProvenanceMetrics(t *testing.T) {
	resetDashboardJWTProvenanceMetricsForTest()
	m := NewMetrics()

	observeDashboardJWTProvenanceAuditExport(
		dashboardJWTProvenanceAuditExport{Enabled: true, Verified: false, VerificationState: "invalid"},
		[]string{provenanceHintVerificationInvalid},
	)

	out := m.Prometheus()
	if !strings.Contains(out, "skoll_dashboard_jwt_provenance_exports_total{enabled=\"true\"} 1") {
		t.Fatalf("expected enabled provenance export metric, got %q", out)
	}
	if !strings.Contains(out, "skoll_dashboard_jwt_provenance_verification_states_total{state=\"invalid\"} 1") {
		t.Fatalf("expected invalid verification state metric, got %q", out)
	}
	if !strings.Contains(out, "skoll_dashboard_jwt_provenance_alert_hints_total{hint=\"verification_state_invalid\"} 1") {
		t.Fatalf("expected provenance alert hint metric, got %q", out)
	}
}
