package audit

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	apiv1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
)

type AuditHandler struct {
	service auditsvc.Service
}

func RegisterAuditRoutes(mux *http.ServeMux, service auditsvc.Service) {
	if service == nil {
		return
	}
	h := &AuditHandler{service: service}
	mux.HandleFunc("GET /v1/audit/logs", h.list)
	mux.HandleFunc("GET /v1/audit/logs/export", h.export)
	mux.HandleFunc("GET /v1/audit/logs/{id}", h.get)
	mux.HandleFunc("DELETE /v1/audit/logs", h.clear)
	mux.HandleFunc("GET /v1/audit/actors/{actorId}", h.listByActor)
}

func (h *AuditHandler) listByActor(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.service.ListByActor(r.Context(), r.PathValue("actorId"), limit)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, items)
}

func (h *AuditHandler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.queryRecords(r)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, items)
}

func (h *AuditHandler) get(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		apiv1.WriteMessage(w, http.StatusBadRequest, "invalid_request", "id is required")
		return
	}
	item, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	if item == nil {
		apiv1.WriteMessage(w, http.StatusNotFound, "not_found", "audit record not found")
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, item)
}

func (h *AuditHandler) export(w http.ResponseWriter, r *http.Request) {
	items, err := h.queryRecords(r)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=audit_logs.csv")
	w.WriteHeader(http.StatusOK)
	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"id", "actorId", "action", "resource", "resourceId", "occurredAt"})
	for _, item := range items {
		_ = writer.Write([]string{
			item.ID.String(),
			item.ActorID.String(),
			item.Action,
			item.Resource,
			item.ResourceID,
			item.OccurredAt.Format(time.RFC3339),
		})
	}
	writer.Flush()
}

func (h *AuditHandler) clear(w http.ResponseWriter, r *http.Request) {
	from, to, err := parseTimeRange(r)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	deleted, err := h.service.ClearByTimeRange(r.Context(), from, to)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]int{"deleted": deleted})
}

func (h *AuditHandler) queryRecords(r *http.Request) ([]*domainaudit.Record, error) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	actorID := strings.TrimSpace(r.URL.Query().Get("actorId"))
	if actorID != "" {
		return h.service.ListByActor(r.Context(), actorID, limit)
	}
	from, to, err := parseTimeRange(r)
	if err != nil {
		return nil, err
	}
	return h.service.ListByTimeRange(r.Context(), from, to, limit)
}

func parseTimeRange(r *http.Request) (time.Time, time.Time, error) {
	const allStart = "1970-01-01T00:00:00Z"
	const allEnd = "2999-12-31T23:59:59Z"
	fromRaw := strings.TrimSpace(r.URL.Query().Get("from"))
	toRaw := strings.TrimSpace(r.URL.Query().Get("to"))
	if fromRaw == "" {
		fromRaw = allStart
	}
	if toRaw == "" {
		toRaw = allEnd
	}
	from, err := time.Parse(time.RFC3339, fromRaw)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	to, err := time.Parse(time.RFC3339, toRaw)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if to.Before(from) {
		return time.Time{}, time.Time{}, domainaudit.ErrInvalidTimeRange
	}
	return from, to, nil
}
