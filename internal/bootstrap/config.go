package bootstrap

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/module/storageadapter"
)

type runOptions struct {
	addr                            string
	shutdownTimeout                 time.Duration
	drainTime                       time.Duration
	goAdminEnabled                  bool
	goAdminMode                     string
	adminAuthMode                   string
	adminAuthToken                  string
	adminAuthHMACSecret             string
	adminAuthNonceStore             string
	adminAuthNonceRedisAddr         string
	adminAuthNonceRedisPassword     string
	adminAuthNonceRedisDB           int
	adminAuthNonceRedisKeyPrefix    string
	adminAuthAllowStaticTokenInProd bool
	storageAdapterMode              string
}

func parseRunOptions() (runOptions, error) {
	shutdownTimeoutDefault, err := durationFromEnv("SKOLL_SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return runOptions{}, fmt.Errorf("invalid SKOLL_SHUTDOWN_TIMEOUT: %w", err)
	}
	drainTimeDefault, err := durationFromEnv("SKOLL_DRAIN_TIME", 2*time.Second)
	if err != nil {
		return runOptions{}, fmt.Errorf("invalid SKOLL_DRAIN_TIME: %w", err)
	}

	addr := flag.String("addr", ":8080", "HTTP listen address")
	shutdownTimeout := flag.Duration("shutdown-timeout", shutdownTimeoutDefault, "graceful shutdown timeout (env: SKOLL_SHUTDOWN_TIMEOUT)")
	drainTime := flag.Duration("drain-time", drainTimeDefault, "readiness drain duration before shutdown (0 to disable, env: SKOLL_DRAIN_TIME)")
	goAdminEnabled := flag.Bool("go-admin-enabled", false, "enable go-admin minimal bootstrap integration")
	goAdminMode := flag.String("go-admin-mode", "dev", "go-admin mode: dev|test|prod")
	adminAuthMode := flag.String("admin-auth-mode", os.Getenv("SKOLL_ADMIN_AUTH_MODE"), "admin auth mode: auto|none|static-token|hmac-sha256 (env: SKOLL_ADMIN_AUTH_MODE)")
	adminAuthToken := flag.String("admin-auth-token", os.Getenv("SKOLL_ADMIN_AUTH_TOKEN"), "optional admin auth token for /admin routes (env: SKOLL_ADMIN_AUTH_TOKEN)")
	adminAuthHMACSecret := flag.String("admin-auth-hmac-secret", os.Getenv("SKOLL_ADMIN_AUTH_HMAC_SECRET"), "optional admin auth hmac secret for hmac-sha256 mode (env: SKOLL_ADMIN_AUTH_HMAC_SECRET)")
	adminAuthNonceStore := flag.String("admin-auth-nonce-store", envOrDefault("SKOLL_ADMIN_AUTH_NONCE_STORE", "auto"), "admin auth nonce store mode: auto|memory|redis (env: SKOLL_ADMIN_AUTH_NONCE_STORE)")
	adminAuthNonceRedisAddr := flag.String("admin-auth-nonce-redis-addr", os.Getenv("SKOLL_ADMIN_AUTH_NONCE_REDIS_ADDR"), "redis address for shared admin auth nonce store (env: SKOLL_ADMIN_AUTH_NONCE_REDIS_ADDR)")
	adminAuthNonceRedisPassword := flag.String("admin-auth-nonce-redis-password", os.Getenv("SKOLL_ADMIN_AUTH_NONCE_REDIS_PASSWORD"), "redis password for shared admin auth nonce store (env: SKOLL_ADMIN_AUTH_NONCE_REDIS_PASSWORD)")
	adminAuthNonceRedisDB := flag.Int("admin-auth-nonce-redis-db", intFromEnv("SKOLL_ADMIN_AUTH_NONCE_REDIS_DB", 0), "redis DB index for shared admin auth nonce store (env: SKOLL_ADMIN_AUTH_NONCE_REDIS_DB)")
	adminAuthNonceRedisKeyPrefix := flag.String("admin-auth-nonce-redis-key-prefix", envOrDefault("SKOLL_ADMIN_AUTH_NONCE_REDIS_KEY_PREFIX", ""), "redis key prefix for shared admin auth nonce store (env: SKOLL_ADMIN_AUTH_NONCE_REDIS_KEY_PREFIX)")
	adminAuthAllowStaticTokenInProd := flag.Bool("admin-auth-allow-static-token-in-prod", boolFromEnv("SKOLL_ADMIN_AUTH_ALLOW_STATIC_TOKEN_IN_PROD", false), "allow static-token admin auth in prod mode (env: SKOLL_ADMIN_AUTH_ALLOW_STATIC_TOKEN_IN_PROD)")
	storageAdapterMode := flag.String("storage-adapter", envOrDefault("SKOLL_STORAGE_ADAPTER", storageadapter.ModeMemory), "storage adapter mode: memory|mysql|postgres (env: SKOLL_STORAGE_ADAPTER)")
	flag.Parse()

	return runOptions{
		addr:                            *addr,
		shutdownTimeout:                 *shutdownTimeout,
		drainTime:                       *drainTime,
		goAdminEnabled:                  *goAdminEnabled,
		goAdminMode:                     *goAdminMode,
		adminAuthMode:                   *adminAuthMode,
		adminAuthToken:                  *adminAuthToken,
		adminAuthHMACSecret:             *adminAuthHMACSecret,
		adminAuthNonceStore:             *adminAuthNonceStore,
		adminAuthNonceRedisAddr:         *adminAuthNonceRedisAddr,
		adminAuthNonceRedisPassword:     *adminAuthNonceRedisPassword,
		adminAuthNonceRedisDB:           *adminAuthNonceRedisDB,
		adminAuthNonceRedisKeyPrefix:    *adminAuthNonceRedisKeyPrefix,
		adminAuthAllowStaticTokenInProd: *adminAuthAllowStaticTokenInProd,
		storageAdapterMode:              *storageAdapterMode,
	}, nil
}

func validateRuntimeConfig(shutdownTimeout, drainTime time.Duration) error {
	const maxDrainTime = 30 * time.Second

	if shutdownTimeout <= 0 {
		return fmt.Errorf("shutdown-timeout must be > 0, got %s", shutdownTimeout)
	}
	if drainTime < 0 {
		return fmt.Errorf("drain-time must be >= 0, got %s", drainTime)
	}
	if drainTime > maxDrainTime {
		return fmt.Errorf("drain-time must be <= %s, got %s", maxDrainTime, drainTime)
	}

	return nil
}

func durationFromEnv(key string, fallback time.Duration) (time.Duration, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s=%q parse failed: %w", key, raw, err)
	}
	return parsed, nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func intFromEnv(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return parsed
}

func boolFromEnv(key string, fallback bool) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	switch raw {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	case "":
		return fallback
	default:
		return fallback
	}
}

func formatRuntimeConfigLog(addr string, shutdownTimeout, drainTime time.Duration, goAdminEnabled bool, goAdminMode, adminAuthMode, effectiveAdminAuthMode string, adminAuthEnabled bool, adminAuthAllowStaticTokenInProd bool, storageAdapterMode string) string {
	return fmt.Sprintf("runtime config addr=%s shutdown-timeout=%s drain-time=%s go-admin-enabled=%t go-admin-mode=%s admin-auth-mode=%s admin-auth-effective-mode=%s admin-auth-enabled=%t admin-auth-allow-static-token-in-prod=%t storage-adapter=%s", addr, shutdownTimeout, drainTime, goAdminEnabled, goAdminMode, adminAuthMode, effectiveAdminAuthMode, adminAuthEnabled, adminAuthAllowStaticTokenInProd, storageAdapterMode)
}
