package main

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// registerRoutes mounts analytics demo endpoints focused on backend signals.
//
// Routes:
// - GET /demo-backend/metrics?api=120&jobs=32&errors=2
// - GET /demo-backend/audit/report?events=user.create,user.update,user.create&limit=2
func registerRoutes(mux *http.ServeMux) {
	service := NewMetricsService()

	mux.HandleFunc("/demo-backend/metrics", func(w http.ResponseWriter, r *http.Request) {
		routeHit := map[string]int{
			"api":  parseIntParam(r, "api", 120),
			"jobs": parseIntParam(r, "jobs", 32),
		}
		cacheHits := parseIntParam(r, "cache", 0)
		if cacheHits > 0 {
			routeHit["cache"] = cacheHits
		}
		writeJSON(w, http.StatusOK, service.Snapshot(routeHit, parseIntParam(r, "errors", 2)))
	})

	mux.HandleFunc("/demo-backend/audit/report", func(w http.ResponseWriter, r *http.Request) {
		events := parseCSV(r, "events", []string{"user.create", "user.update", "user.create", "role.bind"})
		limit := parseIntParam(r, "limit", 3)
		writeJSON(w, http.StatusOK, service.BuildAuditDigest(events, limit))
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

func parseCSV(r *http.Request, key string, fallback []string) []string {
	if r == nil {
		return fallback
	}
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return fallback
	}
	items := make([]string, 0, len(raw))
	start := 0
	for i := 0; i <= len(raw); i++ {
		if i != len(raw) && raw[i] != ',' {
			continue
		}
		if i > start {
			items = append(items, raw[start:i])
		}
		start = i + 1
	}
	if len(items) == 0 {
		return fallback
	}
	return items
}
