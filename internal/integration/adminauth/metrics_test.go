package adminauth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMetricsPrometheusStaticTokenReasons(t *testing.T) {
	resetMetricsForTest()

	v, enabled, err := ResolveVerifier("static-token", "secret", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !enabled {
		t.Fatalf("expected enabled verifier")
	}

	missing := httptest.NewRequest(http.MethodGet, "/admin/ping", nil)
	_ = v.Verify(missing)

	invalid := httptest.NewRequest(http.MethodGet, "/admin/ping", nil)
	invalid.Header.Set(HeaderToken, "bad")
	_ = v.Verify(invalid)

	valid := httptest.NewRequest(http.MethodGet, "/admin/ping", nil)
	valid.Header.Set(HeaderToken, "secret")
	_ = v.Verify(valid)

	metrics := MetricsPrometheus()
	checks := []string{
		"skoll_admin_auth_verifications_total{mode=\"static-token\",result=\"success\"} 1",
		"skoll_admin_auth_verifications_total{mode=\"static-token\",result=\"failure\"} 2",
		"skoll_admin_auth_failures_total{mode=\"static-token\",reason=\"missing_token\"} 1",
		"skoll_admin_auth_failures_total{mode=\"static-token\",reason=\"invalid_token\"} 1",
	}
	for _, check := range checks {
		if !strings.Contains(metrics, check) {
			t.Fatalf("expected metrics to contain %q, got\n%s", check, metrics)
		}
	}
}

func TestMetricsPrometheusHMACReasons(t *testing.T) {
	resetMetricsForTest()

	v, enabled, err := ResolveVerifier("hmac-sha256", "", "secret")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !enabled {
		t.Fatalf("expected enabled verifier")
	}

	hv, ok := v.(*hmacSHA256Verifier)
	if !ok {
		t.Fatalf("expected hmac verifier")
	}
	fixedNow := time.Unix(1710000000, 0)
	hv.now = func() time.Time { return fixedNow }

	missing := httptest.NewRequest(http.MethodGet, "/admin/ping", nil)
	_ = hv.Verify(missing)

	stale := httptest.NewRequest(http.MethodGet, "/admin/ping", nil)
	stale.Header.Set(HeaderTimestamp, "1")
	stale.Header.Set(HeaderNonce, "n1")
	stale.Header.Set(HeaderSignature, "deadbeef")
	_ = hv.Verify(stale)

	metrics := MetricsPrometheus()
	checks := []string{
		"skoll_admin_auth_verifications_total{mode=\"hmac-sha256\",result=\"failure\"} 2",
		"skoll_admin_auth_failures_total{mode=\"hmac-sha256\",reason=\"missing_headers\"} 1",
		"skoll_admin_auth_failures_total{mode=\"hmac-sha256\",reason=\"timestamp_skew\"} 1",
	}
	for _, check := range checks {
		if !strings.Contains(metrics, check) {
			t.Fatalf("expected metrics to contain %q, got\n%s", check, metrics)
		}
	}
}

func TestSnapshotReturnsStructuredCounters(t *testing.T) {
	resetMetricsForTest()

	v, enabled, err := ResolveVerifier("static-token", "secret", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !enabled {
		t.Fatalf("expected enabled verifier")
	}

	missing := httptest.NewRequest(http.MethodGet, "/admin/ping", nil)
	_ = v.Verify(missing)

	valid := httptest.NewRequest(http.MethodGet, "/admin/ping", nil)
	valid.Header.Set(HeaderToken, "secret")
	_ = v.Verify(valid)

	snapshot := Snapshot()
	if snapshot.StaticToken.Success != 1 {
		t.Fatalf("expected static success=1, got %d", snapshot.StaticToken.Success)
	}
	if snapshot.StaticToken.Failure != 1 {
		t.Fatalf("expected static failure=1, got %d", snapshot.StaticToken.Failure)
	}
	if snapshot.StaticToken.Reasons[reasonMissingToken] != 1 {
		t.Fatalf("expected missing_token=1, got %d", snapshot.StaticToken.Reasons[reasonMissingToken])
	}
}
