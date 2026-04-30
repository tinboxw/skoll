package auditlog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/tinboxw/skoll/internal/module/audit"
)

type AuditService interface {
	Append(actor, action, target string) audit.Record
	Recent(limit int) []audit.Record
	Query(q audit.Query) audit.QueryResult
}

type APIRegistry interface {
	RegisterMany(entries []string)
}

const auditQueryMaxSize = 200

type Handler struct {
	audit AuditService
}

func NewHandler(auditSvc AuditService) *Handler {
	return &Handler{audit: auditSvc}
}

func (h *Handler) Register(mux *http.ServeMux, wrapper func(http.Handler) http.Handler, apis APIRegistry) {
	if mux == nil || h.audit == nil {
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

	handle("POST /admin/v1/audit-logs", h.appendAuditLog)
	handle("GET /admin/v1/audit-logs", h.recentAuditLogs)
	handle("GET /admin/v1/audit-logs/profile", h.auditQueryProfile)
	handle("GET /admin/v1/admin-ops/control-profile", h.adminOpsControlProfile)
}

func (h *Handler) appendAuditLog(w http.ResponseWriter, r *http.Request) {
	var req appendAuditLogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	req.Actor = strings.TrimSpace(req.Actor)
	req.Action = strings.TrimSpace(req.Action)
	req.Target = strings.TrimSpace(req.Target)
	if req.Actor == "" || req.Action == "" || req.Target == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "actor, action and target are required"})
		return
	}
	respondJSON(w, http.StatusCreated, h.audit.Append(req.Actor, req.Action, req.Target))
}

func (h *Handler) recentAuditLogs(w http.ResponseWriter, r *http.Request) {
	query, err := parseAuditQuery(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, h.audit.Query(query))
}

func (h *Handler) auditQueryProfile(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, auditQueryProfileResponse{
		DefaultPage:     audit.DefaultPage,
		DefaultSize:     audit.DefaultSize,
		MaxSize:         auditQueryMaxSize,
		TargetP95Millis: 100,
		SupportedFilters: []string{
			"actor",
			"action",
			"target",
			"q",
			"page",
			"size",
			"limit",
		},
	})
}

func (h *Handler) adminOpsControlProfile(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, adminOpsControlProfileResponse{
		AuditRetentionDays: 180,
		AuditArchiveDays:   30,
		ArchiveBatchSize:   1000,
		RateGuardHints: []string{
			"actor:security action:policy_snapshot_rollback burst<=2/min",
			"actor:admin action:users_bulk_create burst<=10/min",
			"actor:admin action:configs_bulk_upsert burst<=15/min",
		},
	})
}

func parseAuditQuery(r *http.Request) (audit.Query, error) {
	q := audit.Query{
		Page:   audit.DefaultPage,
		Size:   audit.DefaultSize,
		Actor:  strings.TrimSpace(r.URL.Query().Get("actor")),
		Action: strings.TrimSpace(r.URL.Query().Get("action")),
		Target: strings.TrimSpace(r.URL.Query().Get("target")),
		Q:      strings.TrimSpace(r.URL.Query().Get("q")),
	}

	if rawPage := strings.TrimSpace(r.URL.Query().Get("page")); rawPage != "" {
		parsed, err := strconv.Atoi(rawPage)
		if err != nil || parsed <= 0 {
			return audit.Query{}, fmt.Errorf("page must be a positive integer")
		}
		q.Page = parsed
	}

	if rawSize := strings.TrimSpace(r.URL.Query().Get("size")); rawSize != "" {
		parsed, err := strconv.Atoi(rawSize)
		if err != nil || parsed <= 0 {
			return audit.Query{}, fmt.Errorf("size must be a positive integer")
		}
		if parsed > auditQueryMaxSize {
			return audit.Query{}, fmt.Errorf("size must be <= %d", auditQueryMaxSize)
		}
		q.Size = parsed
	}

	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 {
			return audit.Query{}, fmt.Errorf("limit must be a positive integer")
		}
		if parsed > auditQueryMaxSize {
			return audit.Query{}, fmt.Errorf("limit must be <= %d", auditQueryMaxSize)
		}
		q.Page = 1
		q.Size = parsed
	}

	return q, nil
}

type appendAuditLogRequest struct {
	Actor  string `json:"actor"`
	Action string `json:"action"`
	Target string `json:"target"`
}

type auditQueryProfileResponse struct {
	DefaultPage      int      `json:"default_page"`
	DefaultSize      int      `json:"default_size"`
	MaxSize          int      `json:"max_size"`
	TargetP95Millis  int      `json:"target_p95_millis"`
	SupportedFilters []string `json:"supported_filters"`
}

type adminOpsControlProfileResponse struct {
	AuditRetentionDays int      `json:"audit_retention_days"`
	AuditArchiveDays   int      `json:"audit_archive_days"`
	ArchiveBatchSize   int      `json:"archive_batch_size"`
	RateGuardHints     []string `json:"rate_guard_hints"`
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
