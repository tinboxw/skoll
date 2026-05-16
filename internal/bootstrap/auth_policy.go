package bootstrap

import (
	"os"
	"strings"
)

// AuthPolicy defines route-level auth decisions for bootstrap middleware.
type AuthPolicy struct {
	Enabled   bool
	SkipPaths map[string]struct{}
}

func (p AuthPolicy) WithAPIPrefix(prefix string) AuthPolicy {
	normalizedPrefix := normalizePolicyPrefix(prefix)
	if normalizedPrefix == "/api" {
		return p
	}
	next := AuthPolicy{Enabled: p.Enabled, SkipPaths: map[string]struct{}{}}
	for path := range p.SkipPaths {
		next.SkipPaths[rewriteAPIPrefix(path, normalizedPrefix)] = struct{}{}
	}
	return next
}

func loadAuthPolicyFromEnv() AuthPolicy {
	policy := AuthPolicy{
		Enabled:   parseBoolEnv("SKOLL_AUTH_ENABLED", true),
		SkipPaths: map[string]struct{}{"/api/health": {}, "/api/ready": {}, "/api/v1/plugins": {}, "/api/v1/auth/login": {}},
	}

	for _, p := range strings.Split(os.Getenv("SKOLL_AUTH_SKIP_PATHS"), ",") {
		path := strings.TrimSpace(p)
		if path == "" {
			continue
		}
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		policy.SkipPaths[path] = struct{}{}
	}

	return policy
}

func (p AuthPolicy) ShouldAuthenticate(path string) bool {
	if !p.Enabled {
		return false
	}
	for skipPath := range p.SkipPaths {
		if path == skipPath || strings.HasPrefix(path, skipPath+"/") {
			return false
		}
	}
	return true
}

func parseBoolEnv(key string, fallback bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return fallback
	}
	switch v {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func normalizePolicyPrefix(prefix string) string {
	v := strings.TrimSpace(prefix)
	if v == "" {
		return "/api"
	}
	if !strings.HasPrefix(v, "/") {
		v = "/" + v
	}
	v = strings.TrimRight(v, "/")
	if v == "" {
		return "/api"
	}
	return v
}

func rewriteAPIPrefix(path, prefix string) string {
	clean := strings.TrimSpace(path)
	if clean == "" {
		return prefix
	}
	if !strings.HasPrefix(clean, "/") {
		clean = "/" + clean
	}
	if clean == "/api" {
		return prefix
	}
	if strings.HasPrefix(clean, "/api/") {
		return prefix + strings.TrimPrefix(clean, "/api")
	}
	return clean
}
