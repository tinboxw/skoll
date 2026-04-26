package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/tinboxw/skoll/internal/app"
	"github.com/tinboxw/skoll/internal/integration/adminauth"
	"github.com/tinboxw/skoll/internal/integration/goadmin"
	"github.com/tinboxw/skoll/pkg/version"
)

func main() {
	shutdownTimeoutDefault, err := durationFromEnv("SKOLL_SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		log.Fatalf("invalid SKOLL_SHUTDOWN_TIMEOUT: %v", err)
	}
	drainTimeDefault, err := durationFromEnv("SKOLL_DRAIN_TIME", 2*time.Second)
	if err != nil {
		log.Fatalf("invalid SKOLL_DRAIN_TIME: %v", err)
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
	flag.Parse()

	nonceStore, closeNonceStore, err := resolveAdminNonceStore(*adminAuthMode, *adminAuthToken, *adminAuthHMACSecret, *adminAuthNonceStore, *adminAuthNonceRedisAddr, *adminAuthNonceRedisPassword, *adminAuthNonceRedisDB, *adminAuthNonceRedisKeyPrefix)
	if err != nil {
		log.Fatalf("invalid admin auth nonce store configuration: %v", err)
	}
	if closeNonceStore != nil {
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := closeNonceStore(ctx); err != nil {
				log.Printf("admin auth nonce store close error: %v", err)
			}
		}()
	}

	adminVerifier, adminAuthEnabled, err := adminauth.ResolveVerifierWithNonceStore(*adminAuthMode, *adminAuthToken, *adminAuthHMACSecret, nonceStore)
	if err != nil {
		log.Fatalf("invalid admin auth configuration: %v", err)
	}
	effectiveAdminAuthMode := "none"
	if adminAuthEnabled {
		effectiveAdminAuthMode = adminVerifier.Mode()
	}

	if err := validateRuntimeConfig(*shutdownTimeout, *drainTime); err != nil {
		log.Fatalf("invalid runtime configuration: %v", err)
	}
	if err := validateAdminAuthPolicy(*goAdminEnabled, *goAdminMode, effectiveAdminAuthMode, *adminAuthAllowStaticTokenInProd); err != nil {
		log.Fatalf("invalid admin auth policy: %v", err)
	}
	log.Print(formatRuntimeConfigLog(*addr, *shutdownTimeout, *drainTime, *goAdminEnabled, *goAdminMode, *adminAuthMode, effectiveAdminAuthMode, adminAuthEnabled, *adminAuthAllowStaticTokenInProd))

	bootstrap, err := goadmin.New(*goAdminEnabled, *goAdminMode)
	if err != nil {
		log.Fatalf("invalid go-admin configuration: %v", err)
	}
	if err := bootstrap.Init(context.Background()); err != nil {
		log.Fatalf("go-admin bootstrap failed: %v", err)
	}
	if bootstrap.Enabled() {
		log.Printf("go-admin minimal bootstrap enabled (mode=%s)", bootstrap.Mode())
	}

	srv := app.New(*addr, version.String())
	srv.AddMetricsCollector(adminauth.MetricsPrometheus)
	srv.SetReady(true)
	if bootstrap.Enabled() {
		adminPingHandler := http.Handler(http.HandlerFunc(bootstrap.AdminPingHandler()))
		if adminAuthEnabled {
			adminPingHandler = adminauth.WithVerifier(adminPingHandler, adminVerifier)
		}
		srv.HandleFunc(goadmin.AdminPingPath, func(w http.ResponseWriter, r *http.Request) {
			adminPingHandler.ServeHTTP(w, r)
		})
	}

	errCh := make(chan error, 1)
	go func() {
		err := srv.Start()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case sig := <-sigCh:
		log.Printf("received signal %s, starting graceful shutdown", sig)
		if err := performGracefulStop(srv, *drainTime, *shutdownTimeout, sigCh); err != nil {
			log.Printf("graceful shutdown error: %v", err)
		}
	case err := <-errCh:
		log.Fatalf("server error: %v", err)
	}
}

type shutdownServer interface {
	SetReady(bool)
	Shutdown(context.Context) error
}

func performGracefulStop(srv shutdownServer, drainTime, shutdownTimeout time.Duration, sigCh <-chan os.Signal) error {
	srv.SetReady(false)
	if drainTime > 0 {
		log.Printf("readiness switched to not_ready, draining for %s", drainTime)
		if interrupted := waitForDrain(drainTime, sigCh); interrupted {
			log.Printf("drain interrupted by a second termination signal")
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func waitForDrain(drainTime time.Duration, sigCh <-chan os.Signal) bool {
	timer := time.NewTimer(drainTime)
	defer timer.Stop()

	select {
	case <-timer.C:
		return false
	case <-sigCh:
		return true
	}
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

func validateAdminAuthPolicy(goAdminEnabled bool, goAdminMode, effectiveAdminAuthMode string, allowStaticTokenInProd bool) error {
	if !goAdminEnabled {
		return nil
	}
	if goAdminMode != "prod" {
		return nil
	}
	if effectiveAdminAuthMode != "static-token" {
		return nil
	}
	if allowStaticTokenInProd {
		return nil
	}
	return fmt.Errorf("static-token admin auth is not allowed in prod mode unless admin-auth-allow-static-token-in-prod=true")
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

func resolveAdminNonceStore(authMode, authToken, hmacSecret, storeMode, redisAddr, redisPassword string, redisDB int, redisKeyPrefix string) (adminauth.ReplayNonceStore, func(context.Context) error, error) {
	if !isHMACFlowEnabled(authMode, authToken, hmacSecret) {
		return nil, nil, nil
	}

	resolvedMode := storeMode
	if resolvedMode == "" {
		resolvedMode = "auto"
	}

	switch resolvedMode {
	case "auto":
		if redisAddr == "" {
			return nil, nil, nil
		}
		cfg := adminauth.RedisNonceStoreConfig{Addr: redisAddr, Password: redisPassword, DB: redisDB, KeyPrefix: redisKeyPrefix}
		return adminauth.NewRedisNonceStoreFromConfig(cfg)
	case "memory":
		return nil, nil, nil
	case "redis":
		cfg := adminauth.RedisNonceStoreConfig{Addr: redisAddr, Password: redisPassword, DB: redisDB, KeyPrefix: redisKeyPrefix}
		return adminauth.NewRedisNonceStoreFromConfig(cfg)
	default:
		return nil, nil, fmt.Errorf("unsupported admin auth nonce store mode %q, valid: auto|memory|redis", resolvedMode)
	}
}

func isHMACFlowEnabled(mode, token, hmacSecret string) bool {
	switch mode {
	case "", "auto":
		return token == "" && hmacSecret != ""
	case "hmac-sha256":
		return hmacSecret != ""
	default:
		return false
	}
}

func formatRuntimeConfigLog(addr string, shutdownTimeout, drainTime time.Duration, goAdminEnabled bool, goAdminMode, adminAuthMode, effectiveAdminAuthMode string, adminAuthEnabled bool, adminAuthAllowStaticTokenInProd bool) string {
	return fmt.Sprintf("runtime config addr=%s shutdown-timeout=%s drain-time=%s go-admin-enabled=%t go-admin-mode=%s admin-auth-mode=%s admin-auth-effective-mode=%s admin-auth-enabled=%t admin-auth-allow-static-token-in-prod=%t", addr, shutdownTimeout, drainTime, goAdminEnabled, goAdminMode, adminAuthMode, effectiveAdminAuthMode, adminAuthEnabled, adminAuthAllowStaticTokenInProd)
}
