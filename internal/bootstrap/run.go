package bootstrap

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
	"github.com/tinboxw/skoll/internal/module/storageadapter"
	"github.com/tinboxw/skoll/pkg/version"
)

func Run() error {
	shutdownTimeoutDefault, err := durationFromEnv("SKOLL_SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return fmt.Errorf("invalid SKOLL_SHUTDOWN_TIMEOUT: %w", err)
	}
	drainTimeDefault, err := durationFromEnv("SKOLL_DRAIN_TIME", 2*time.Second)
	if err != nil {
		return fmt.Errorf("invalid SKOLL_DRAIN_TIME: %w", err)
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

	nonceStore, closeNonceStore, err := resolveAdminNonceStore(*adminAuthMode, *adminAuthToken, *adminAuthHMACSecret, *adminAuthNonceStore, *adminAuthNonceRedisAddr, *adminAuthNonceRedisPassword, *adminAuthNonceRedisDB, *adminAuthNonceRedisKeyPrefix)
	if err != nil {
		return fmt.Errorf("invalid admin auth nonce store configuration: %w", err)
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
		return fmt.Errorf("invalid admin auth configuration: %w", err)
	}
	effectiveAdminAuthMode := "none"
	if adminAuthEnabled {
		effectiveAdminAuthMode = adminVerifier.Mode()
	}

	if err := validateRuntimeConfig(*shutdownTimeout, *drainTime); err != nil {
		return fmt.Errorf("invalid runtime configuration: %w", err)
	}
	if err := validateAdminAuthPolicy(*goAdminEnabled, *goAdminMode, effectiveAdminAuthMode, *adminAuthAllowStaticTokenInProd); err != nil {
		return fmt.Errorf("invalid admin auth policy: %w", err)
	}
	log.Print(formatRuntimeConfigLog(*addr, *shutdownTimeout, *drainTime, *goAdminEnabled, *goAdminMode, *adminAuthMode, effectiveAdminAuthMode, adminAuthEnabled, *adminAuthAllowStaticTokenInProd, *storageAdapterMode))

	goAdminBootstrap, err := goadmin.New(*goAdminEnabled, *goAdminMode)
	if err != nil {
		return fmt.Errorf("invalid go-admin configuration: %w", err)
	}
	if err := goAdminBootstrap.Init(context.Background()); err != nil {
		return fmt.Errorf("go-admin bootstrap failed: %w", err)
	}
	if goAdminBootstrap.Enabled() {
		log.Printf("go-admin minimal bootstrap enabled (mode=%s)", goAdminBootstrap.Mode())
	}

	srv := app.New(*addr, version.String())
	srv.AddMetricsCollector(adminauth.MetricsPrometheus)
	srv.SetReady(true)
	storage, err := storageadapter.NewByMode(*storageAdapterMode)
	if err != nil {
		return fmt.Errorf("invalid storage adapter configuration: %w", err)
	}

	var adminWrapper func(http.Handler) http.Handler
	if adminAuthEnabled {
		adminWrapper = func(next http.Handler) http.Handler {
			return adminauth.WithVerifier(app.WithRoleAPIAuthorizer(next, storage.Roles(), storage.RBAC()), adminVerifier)
		}
	}

	srv.MountAdminModuleRoutes(app.AdminModuleServices{
		Users:        storage.Users(),
		Roles:        storage.Roles(),
		Menus:        storage.Menus(),
		Audit:        storage.Audit(),
		Configs:      storage.Configs(),
		Dictionaries: storage.Dictionaries(),
		Files:        storage.Files(),
		Jobs:         storage.Jobs(),
		Generator:    storage.Generators(),
		Plugins:      storage.Plugins(),
		RBAC:         storage.RBAC(),
		APIs:         storage.APIs(),
		Releases:     storage.Releases(),
	}, adminWrapper)

	if goAdminBootstrap.Enabled() {
		adminPingHandler := http.Handler(http.HandlerFunc(goAdminBootstrap.AdminPingHandler()))
		if adminWrapper != nil {
			adminPingHandler = adminWrapper(adminPingHandler)
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
		return nil
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
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

func formatRuntimeConfigLog(addr string, shutdownTimeout, drainTime time.Duration, goAdminEnabled bool, goAdminMode, adminAuthMode, effectiveAdminAuthMode string, adminAuthEnabled bool, adminAuthAllowStaticTokenInProd bool, storageAdapterMode string) string {
	return fmt.Sprintf("runtime config addr=%s shutdown-timeout=%s drain-time=%s go-admin-enabled=%t go-admin-mode=%s admin-auth-mode=%s admin-auth-effective-mode=%s admin-auth-enabled=%t admin-auth-allow-static-token-in-prod=%t storage-adapter=%s", addr, shutdownTimeout, drainTime, goAdminEnabled, goAdminMode, adminAuthMode, effectiveAdminAuthMode, adminAuthEnabled, adminAuthAllowStaticTokenInProd, storageAdapterMode)
}
