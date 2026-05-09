package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tinboxw/skoll/internal/app"
	admincontracts "github.com/tinboxw/skoll/internal/app/admin/contracts"
	adminsecurity "github.com/tinboxw/skoll/internal/app/admin/security"
	"github.com/tinboxw/skoll/internal/integration/adminauth"
	"github.com/tinboxw/skoll/internal/integration/goadmin"
	"github.com/tinboxw/skoll/internal/module/storageadapter"
	storagecontracts "github.com/tinboxw/skoll/internal/module/storageadapter/contracts"
	"github.com/tinboxw/skoll/pkg/version"
)

// Runner owns bootstrap startup and shutdown workflow.
type Runner struct{}

func NewRunner() *Runner {
	return &Runner{}
}

func Run() error {
	return NewRunner().Run()
}

func (r *Runner) Run() error {
	opts, err := parseRunOptions()
	if err != nil {
		return err
	}

	adminVerifier, adminAuthEnabled, effectiveAdminAuthMode, closeNonceStore, err := r.resolveAdminAuth(opts)
	if err != nil {
		return err
	}
	if closeNonceStore != nil {
		defer r.closeNonceStore(closeNonceStore)
	}

	if err := validateRuntimeConfig(opts.shutdownTimeout, opts.drainTime); err != nil {
		return fmt.Errorf("invalid runtime configuration: %w", err)
	}
	if err := validateAdminAuthPolicy(opts.goAdminEnabled, opts.goAdminMode, effectiveAdminAuthMode, opts.adminAuthAllowStaticTokenInProd); err != nil {
		return fmt.Errorf("invalid admin auth policy: %w", err)
	}
	log.Print(formatRuntimeConfigLog(
		opts.addr,
		opts.shutdownTimeout,
		opts.drainTime,
		opts.goAdminEnabled,
		opts.goAdminMode,
		opts.adminAuthMode,
		effectiveAdminAuthMode,
		adminAuthEnabled,
		opts.adminAuthAllowStaticTokenInProd,
		opts.storageAdapterMode,
	))

	goAdminBootstrap, err := r.initGoAdminBootstrap(opts)
	if err != nil {
		return err
	}

	srv, storage, err := r.buildServer(opts, adminVerifier, adminAuthEnabled, goAdminBootstrap)
	if err != nil {
		return err
	}
	defer r.closeStorageAdapter(storage)

	return r.serveUntilShutdown(srv, opts.drainTime, opts.shutdownTimeout)
}

func (r *Runner) resolveAdminAuth(opts runOptions) (adminauth.Verifier, bool, string, func(context.Context) error, error) {
	nonceStore, closeNonceStore, err := resolveAdminNonceStore(
		opts.adminAuthMode,
		opts.adminAuthToken,
		opts.adminAuthHMACSecret,
		opts.adminAuthNonceStore,
		opts.adminAuthNonceRedisAddr,
		opts.adminAuthNonceRedisPassword,
		opts.adminAuthNonceRedisDB,
		opts.adminAuthNonceRedisKeyPrefix,
	)
	if err != nil {
		return nil, false, "", nil, fmt.Errorf("invalid admin auth nonce store configuration: %w", err)
	}

	adminVerifier, adminAuthEnabled, err := adminauth.ResolveVerifierWithNonceStore(
		opts.adminAuthMode,
		opts.adminAuthToken,
		opts.adminAuthHMACSecret,
		nonceStore,
	)
	if err != nil {
		return nil, false, "", closeNonceStore, fmt.Errorf("invalid admin auth configuration: %w", err)
	}

	effectiveAdminAuthMode := "none"
	if adminAuthEnabled {
		effectiveAdminAuthMode = adminVerifier.Mode()
	}

	return adminVerifier, adminAuthEnabled, effectiveAdminAuthMode, closeNonceStore, nil
}

func (r *Runner) closeNonceStore(closeNonceStore func(context.Context) error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := closeNonceStore(ctx); err != nil {
		log.Printf("admin auth nonce store close error: %v", err)
	}
}

func (r *Runner) initGoAdminBootstrap(opts runOptions) (*goadmin.Bootstrap, error) {
	goAdminBootstrap, err := goadmin.New(opts.goAdminEnabled, opts.goAdminMode)
	if err != nil {
		return nil, fmt.Errorf("invalid go-admin configuration: %w", err)
	}
	if err := goAdminBootstrap.Init(context.Background()); err != nil {
		return nil, fmt.Errorf("go-admin bootstrap failed: %w", err)
	}
	if goAdminBootstrap.Enabled() {
		log.Printf("go-admin minimal bootstrap enabled (mode=%s)", goAdminBootstrap.Mode())
	}
	return goAdminBootstrap, nil
}

func (r *Runner) buildServer(opts runOptions, adminVerifier adminauth.Verifier, adminAuthEnabled bool, goAdminBootstrap *goadmin.Bootstrap) (*app.Server, storagecontracts.Adapter, error) {
	srv := app.New(opts.addr, version.String())
	srv.AddMetricsCollector(adminauth.MetricsPrometheus)
	srv.SetReady(true)

	storage, err := storageadapter.NewByMode(opts.storageAdapterMode)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid storage adapter configuration: %w", err)
	}
	if opts.storageAdapterMode == storageadapter.ModeMemory && opts.goAdminMode == "prod" {
		log.Printf("WARNING: storage-adapter=memory selected in go-admin-mode=prod; data is volatile and will be lost on restart. Set SKOLL_STORAGE_ADAPTER=mysql|postgres for production deployments.")
	}

	var adminWrapper func(http.Handler) http.Handler
	if adminAuthEnabled {
		adminWrapper = func(next http.Handler) http.Handler {
			return adminauth.WithVerifier(adminsecurity.WithRoleAPIAuthorizer(next, storage.Roles(), storage.RBAC()), adminVerifier)
		}
	}

	r.mountAdminModules(srv, storage, adminWrapper)
	r.mountGoAdminPing(srv, goAdminBootstrap, adminWrapper)

	return srv, storage, nil
}

func (r *Runner) closeStorageAdapter(storage storagecontracts.Adapter) {
	lifecycle, ok := storage.(storagecontracts.Lifecycle)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := lifecycle.Close(ctx); err != nil {
		log.Printf("storage adapter close error: %v", err)
	}
}

func (r *Runner) mountAdminModules(srv *app.Server, storage storagecontracts.Adapter, adminWrapper func(http.Handler) http.Handler) {
	srv.MountAdminModuleRoutes(admincontracts.AdminModuleServices{
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
}

func (r *Runner) mountGoAdminPing(srv *app.Server, goAdminBootstrap *goadmin.Bootstrap, adminWrapper func(http.Handler) http.Handler) {
	if !goAdminBootstrap.Enabled() {
		return
	}

	adminPingHandler := http.Handler(http.HandlerFunc(goAdminBootstrap.AdminPingHandler()))
	if adminWrapper != nil {
		adminPingHandler = adminWrapper(adminPingHandler)
	}

	srv.HandleFunc(goadmin.AdminPingPath, func(w http.ResponseWriter, req *http.Request) {
		adminPingHandler.ServeHTTP(w, req)
	})
}

func (r *Runner) serveUntilShutdown(srv *app.Server, drainTime, shutdownTimeout time.Duration) error {
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
		if err := performGracefulStop(srv, drainTime, shutdownTimeout, sigCh); err != nil {
			log.Printf("graceful shutdown error: %v", err)
		}
		return nil
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	}
}
