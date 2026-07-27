package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"
)

type closeable interface {
	Close() error
}

// Runner holds process-level runtime dependencies and startup configuration.
type Runner struct {
	config RuntimeConfig
	deps   *dependencies
}

// NewRunnerFromEnv builds a runner from environment configuration.
func NewRunnerFromEnv() (*Runner, error) {
	runtimeCfg, err := loadRuntimeConfigFromEnv()
	if err != nil {
		return nil, err
	}
	return NewRunner(runtimeCfg)
}

// NewRunner builds a runner from provided runtime config.
func NewRunner(cfg RuntimeConfig) (*Runner, error) {
	if err := validateRuntimeConfig(cfg); err != nil {
		return nil, err
	}

	deps, err := buildDependencies(cfg)
	if err != nil {
		return nil, err
	}

	return &Runner{config: cfg, deps: deps}, nil
}

// Run starts the runtime and blocks until context cancellation.
func (r *Runner) Run(ctx context.Context) (runErr error) {
	if r == nil || r.deps == nil || r.deps.server == nil {
		return errors.New("runner is not initialized")
	}
	defer func() {
		if r.deps.pluginRuntime != nil {
			runErr = errors.Join(runErr, r.deps.pluginRuntime.Close())
		}
		if bus, ok := r.deps.eventBus.(closeable); ok && bus != nil {
			runErr = errors.Join(runErr, bus.Close())
		}
	}()
	workerCtx, stopWorker := context.WithCancel(ctx)
	defer stopWorker()
	if r.deps.eventDispatcher != nil {
		workerID := fmt.Sprintf("runtime-%d", os.Getpid())
		go func() {
			_ = r.deps.eventDispatcher.Run(workerCtx, workerID, 0, 50)
		}()
	}
	if r.deps.workflowService != nil {
		workerID := fmt.Sprintf("workflow-runtime-%d", os.Getpid())
		go func() {
			_ = r.deps.workflowService.RunTimerWorker(workerCtx, workerID, time.Second, 50)
		}()
	}

	errCh := make(chan error, 1)
	go func() {
		err := r.deps.server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("http server failed: %w", err)
		}
		return nil
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), r.config.AppConfig.Server.ShutdownTimeout)
		defer cancel()
		if err := r.deps.server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}
		if err := <-errCh; err != nil {
			return fmt.Errorf("http server failed: %w", err)
		}
		return nil
	}
}
