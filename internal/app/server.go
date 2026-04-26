package app

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain"
	"github.com/tinboxw/skoll/internal/service"
)

type Server struct {
	httpServer *http.Server
	mux        *http.ServeMux
	svc        *service.SystemService
	metrics    *Metrics
	collectors []func() string
}

type healthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Uptime  string `json:"uptime"`
}

func New(addr, version string) *Server {
	s := &Server{
		svc:     service.NewSystemService(version, nil),
		metrics: NewMetrics(),
	}

	s.mux = http.NewServeMux()
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/ready", s.handleReady)
	s.mux.HandleFunc("/metrics", s.handleMetrics)

	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           s.instrument(s.mux),
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return s
}

func (s *Server) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	s.mux.HandleFunc(pattern, handler)
}

func (s *Server) MountAdminModuleRoutes(services AdminModuleServices, wrapper func(http.Handler) http.Handler) {
	MountAdminModuleRoutes(s.mux, services, wrapper)
}

func (s *Server) AddMetricsCollector(collector func() string) {
	if collector == nil {
		return
	}
	s.collectors = append(s.collectors, collector)
}

func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) SetReady(v bool) {
	s.svc.SetReady(v)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	health := s.svc.Health()

	respondJSON(w, http.StatusOK, healthResponse{
		Status:  health.Status,
		Version: health.Version,
		Uptime:  health.Uptime.String(),
	})
}

func (s *Server) handleReady(w http.ResponseWriter, _ *http.Request) {
	if s.svc.Readiness() != domain.ReadinessReady {
		respondJSON(w, http.StatusServiceUnavailable, map[string]string{"status": string(domain.ReadinessNotReady)})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": string(domain.ReadinessReady)})
}

func (s *Server) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	w.WriteHeader(http.StatusOK)
	payload := s.metrics.Prometheus()
	for _, collector := range s.collectors {
		extra := strings.TrimSpace(collector())
		if extra == "" {
			continue
		}
		if !strings.HasSuffix(payload, "\n") {
			payload += "\n"
		}
		payload += extra
		if !strings.HasSuffix(payload, "\n") {
			payload += "\n"
		}
	}
	_, _ = w.Write([]byte(payload))
}

func (s *Server) instrument(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		s.metrics.ObserveRequest(r.URL.Path, rw.statusCode)

		if rw.statusCode >= http.StatusInternalServerError || duration >= time.Second {
			log.Printf("request method=%s path=%s status=%d duration=%s", r.Method, r.URL.Path, rw.statusCode, duration)
		}
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
