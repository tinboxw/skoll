package bootstrap

import (
	"fmt"
	"net/http"
	"time"

	"github.com/tinboxw/skoll/pkg/logging"
)

type dependencies struct {
	logger  logging.Logger
	handler http.Handler
	server  *http.Server
}

func buildDependencies(cfg RuntimeConfig) (*dependencies, error) {
	logger := logging.New(cfg.AppConfig.Log.Level)
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	})

	h := buildMiddlewareChain(mux, logger, cfg.AuthPolicy)
	server := &http.Server{
		Addr:              cfg.AppConfig.Server.Address,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
	}

	if server.Addr == "" {
		return nil, fmt.Errorf("server address is empty")
	}

	return &dependencies{logger: logger, handler: h, server: server}, nil
}
