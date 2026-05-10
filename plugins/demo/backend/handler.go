package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

// registerRoutes mounts separated demo plugin endpoints.
//
// Exposed routes:
// - GET /demo-separated/health
// - GET /demo-separated/overview?plugin_id=demo
// - GET /demo-separated/recommendations?active_users=120&error_count=1&rpm=240
func registerRoutes(mux *http.ServeMux) {
	service := NewDemoService()

	mux.HandleFunc("/demo-separated/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("demo-separated:ok"))
	})

	mux.HandleFunc("/demo-separated/overview", func(w http.ResponseWriter, r *http.Request) {
		pluginID := r.URL.Query().Get("plugin_id")
		payload := service.BuildOverview(pluginID, time.Now().UTC())
		writeJSON(w, http.StatusOK, payload)
	})

	mux.HandleFunc("/demo-separated/recommendations", func(w http.ResponseWriter, r *http.Request) {
		ctx := DashboardContext{
			ActiveUsers:       parseIntParam(r, "active_users", 0),
			ErrorCount:        parseIntParam(r, "error_count", 0),
			RequestsPerMinute: parseFloatParam(r, "rpm", 0),
		}
		writeJSON(w, http.StatusOK, service.RecommendActions(ctx))
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
