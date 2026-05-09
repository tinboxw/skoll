package bootstrap

import "testing"

func TestLoadAuthPolicyFromEnvDefaults(t *testing.T) {
	t.Setenv("SKOLL_AUTH_ENABLED", "")
	t.Setenv("SKOLL_AUTH_SKIP_PATHS", "")

	policy := loadAuthPolicyFromEnv()
	if !policy.Enabled {
		t.Fatalf("expected auth policy enabled by default")
	}

	for _, path := range []string{"/health", "/ready", "/v1/plugins", "/v1/auth"} {
		if _, ok := policy.SkipPaths[path]; !ok {
			t.Fatalf("expected default skip path %s", path)
		}
		if policy.ShouldAuthenticate(path) {
			t.Fatalf("expected %s to bypass auth", path)
		}
	}

	if !policy.ShouldAuthenticate("/v1/users") {
		t.Fatalf("expected /v1/users to require auth")
	}
}

func TestLoadAuthPolicyFromEnvCustomPaths(t *testing.T) {
	t.Setenv("SKOLL_AUTH_ENABLED", "true")
	t.Setenv("SKOLL_AUTH_SKIP_PATHS", "api/public, /api/open")

	policy := loadAuthPolicyFromEnv()
	if policy.ShouldAuthenticate("/api/public/ping") {
		t.Fatalf("expected /api/public to bypass auth")
	}
	if policy.ShouldAuthenticate("/api/open/status") {
		t.Fatalf("expected /api/open to bypass auth")
	}
}
