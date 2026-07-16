package bootstrap

import "testing"

func TestLoadAuthPolicyFromEnvDefaults(t *testing.T) {
	t.Setenv("SKOLL_AUTH_ENABLED", "")
	t.Setenv("SKOLL_AUTH_SKIP_PATHS", "")

	policy := loadAuthPolicyFromEnv()
	if !policy.Enabled {
		t.Fatalf("expected auth policy enabled by default")
	}

	for _, path := range []string{"/skoll/health", "/skoll/ready", "/skoll/docs", "/skoll/v1/plugins", "/skoll/v1/auth/login"} {
		if _, ok := policy.SkipPaths[path]; !ok {
			t.Fatalf("expected default skip path %s", path)
		}
		if policy.ShouldAuthenticate(path) {
			t.Fatalf("expected %s to bypass auth", path)
		}
	}
	if policy.ShouldAuthenticate("/skoll/docs/swagger") {
		t.Fatalf("expected /skoll/docs/swagger to bypass auth")
	}

	if !policy.ShouldAuthenticate("/skoll/v1/users") {
		t.Fatalf("expected /v1/users to require auth")
	}
	if !policy.ShouldAuthenticate("/skoll/v1/plugins/dev/scaffold") {
		t.Fatalf("expected /v1/plugins/dev/scaffold to require auth")
	}
	for _, path := range []string{
		"/skoll/v1/plugins/install",
		"/skoll/v1/plugins/pharma_oa/api/stocktakes/approve",
	} {
		if !policy.ShouldAuthenticate(path) {
			t.Fatalf("expected %s to require auth", path)
		}
	}
	for _, path := range []string{
		"/skoll/v1/plugins/pharma_oa/page",
		"/skoll/v1/plugins/pharma_oa/assets/app.js",
	} {
		if policy.ShouldAuthenticate(path) {
			t.Fatalf("expected public plugin path %s to bypass auth", path)
		}
	}
}

func TestLoadAuthPolicyFromEnvCustomPaths(t *testing.T) {
	t.Setenv("SKOLL_AUTH_ENABLED", "true")
	t.Setenv("SKOLL_AUTH_SKIP_PATHS", "skoll/public, /skoll/open")

	policy := loadAuthPolicyFromEnv()
	if policy.ShouldAuthenticate("/skoll/public/ping") {
		t.Fatalf("expected /skoll/public to bypass auth")
	}
	if policy.ShouldAuthenticate("/skoll/open/status") {
		t.Fatalf("expected /skoll/open to bypass auth")
	}
}

func TestAuthPolicyWithCustomAPIPrefix(t *testing.T) {
	policy := loadAuthPolicyFromEnv().WithAPIPrefix("/gateway")

	for _, path := range []string{"/gateway/health", "/gateway/docs/swagger", "/gateway/v1/plugins", "/gateway/v1/auth/login"} {
		if policy.ShouldAuthenticate(path) {
			t.Fatalf("expected %s to bypass auth", path)
		}
	}
	if !policy.ShouldAuthenticate("/gateway/v1/users") {
		t.Fatalf("expected /gateway/v1/users to require auth")
	}
	if !policy.ShouldAuthenticate("/gateway/v1/plugins/dev/validate-all") {
		t.Fatalf("expected /gateway/v1/plugins/dev/validate-all to require auth")
	}
	if !policy.ShouldAuthenticate("/gateway/v1/plugins/pharma_oa/api/purchase-requests/approve") {
		t.Fatalf("expected plugin business API to require auth")
	}
	if policy.ShouldAuthenticate("/gateway/v1/plugins/pharma_oa/assets/app.js") {
		t.Fatalf("expected plugin asset to bypass auth")
	}
}
