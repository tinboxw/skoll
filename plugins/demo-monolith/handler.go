package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

// registerRoutes exposes monolith backend + frontend metadata endpoints.
//
// Routes:
// - GET /demo-monolith/health
// - GET /demo-monolith/widgets?visitors=42&error_rate=1.8
// - GET /demo-monolith/manifest
func registerRoutes(mux *http.ServeMux) {
	service := NewService()

	mux.HandleFunc("/demo-monolith/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(service.Ping(time.Now().UTC())))
	})

	mux.HandleFunc("/demo-monolith/widgets", func(w http.ResponseWriter, r *http.Request) {
		visitors := parseIntParam(r, "visitors", 36)
		errorRate := parseFloatParam(r, "error_rate", 0.8)
		writeJSON(w, http.StatusOK, service.BuildWidgets(visitors, errorRate))
	})

	mux.HandleFunc("/demo-monolith/manifest", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, service.BuildManifest())
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func parseIntParam(r *http.Request, key string, fallback int) int {
	if r == nil {
		return fallback
	}
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}

func parseFloatParam(r *http.Request, key string, fallback float64) float64 {
	if r == nil {
		return fallback
	}
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}
	return v
}
