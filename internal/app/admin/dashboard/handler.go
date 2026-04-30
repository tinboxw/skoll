package dashboard

import (
	"encoding/json"
	"net/http"
)

type APIRegistry interface {
	RegisterMany(entries []string)
}

type Handler struct {
	systemStatus       func() any
	runtimeMetrics     func() any
	nodeHealth         func() any
	dashboardAggregate func(r *http.Request) any
}

func NewHandler(systemStatus, runtimeMetrics, nodeHealth func() any, dashboardAggregate func(r *http.Request) any) *Handler {
	return &Handler{
		systemStatus:       systemStatus,
		runtimeMetrics:     runtimeMetrics,
		nodeHealth:         nodeHealth,
		dashboardAggregate: dashboardAggregate,
	}
}

func (h *Handler) Register(mux *http.ServeMux, wrapper func(http.Handler) http.Handler, apis APIRegistry) {
	if mux == nil || h == nil || h.systemStatus == nil || h.runtimeMetrics == nil || h.nodeHealth == nil || h.dashboardAggregate == nil {
		return
	}

	handle := func(pattern string, next http.HandlerFunc) {
		if apis != nil {
			apis.RegisterMany([]string{pattern})
		}
		hd := http.Handler(next)
		if wrapper != nil {
			hd = wrapper(hd)
		}
		mux.Handle(pattern, hd)
	}

	handle("GET /admin/v1/system/status", h.systemStatusHandler)
	handle("GET /admin/v1/system/runtime-metrics", h.runtimeMetricsHandler)
	handle("GET /admin/v1/system/node-health", h.nodeHealthHandler)
	handle("GET /admin/v1/system/dashboard", h.dashboardAggregateHandler)
}

func (h *Handler) systemStatusHandler(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, h.systemStatus())
}

func (h *Handler) runtimeMetricsHandler(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, h.runtimeMetrics())
}

func (h *Handler) nodeHealthHandler(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, h.nodeHealth())
}

func (h *Handler) dashboardAggregateHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, h.dashboardAggregate(r))
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
