package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

type Runner struct {
	cfg    RuntimeConfig
	server *http.Server
	c      *Container
}

func NewRunnerFromEnv() (*Runner, error) {
	cfg, err := LoadRuntimeConfigFromEnv()
	if err != nil {
		return nil, err
	}
	if err := ValidateRuntimeConfig(cfg); err != nil {
		return nil, err
	}
	c := NewContainer(cfg)
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	h := Chain(mux, RequestLogMiddleware(c.Logger))
	return &Runner{
		cfg: cfg,
		c:   c,
		server: &http.Server{
			Addr:    cfg.App.Server.Address,
			Handler: h,
		},
	}, nil
}

func (r *Runner) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		r.c.Logger.Info("server starting", "addr", r.server.Addr)
		if err := r.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("listen and serve: %w", err)
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		r.c.Logger.Info("shutdown signal received")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), r.cfg.App.Server.ShutdownTimeout)
		defer cancel()
		if err := r.server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}
		return nil
	case err := <-errCh:
		return err
	}
}
