package audit

import (
	"encoding/csv"
	"fmt"
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
	events  auditsvc.EventService
}

type auditEventListData struct {
	Items  []auditEventDTO `json:"items"`
	Offset int             `json:"offset"`
	Limit  int             `json:"limit"`
}

type auditEventDTO struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Action     string         `json:"action"`
	Actor      auditRefDTO    `json:"actor"`
	Resource   auditRefDTO    `json:"resource"`
	Result     string         `json:"result"`
	Risk       string         `json:"risk"`
	Trace      auditTraceDTO  `json:"trace"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	OccurredAt string         `json:"occurredAt"`
}

type auditRefDTO struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

type auditTraceDTO struct {
	TraceID   string `json:"traceId,omitempty"`
	RequestID string `json:"requestId,omitempty"`
	Method    string `json:"method,omitempty"`
	Path      string `json:"path,omitempty"`
	IP        string `json:"ip,omitempty"`
	UserAgent string `json:"userAgent,omitempty"`
}

type auditRecordDTO struct {
	ID         string         `json:"id"`
	ActorID    string         `json:"actorId"`
	ActorName  string         `json:"actorName,omitempty"`
	Action     string         `json:"action"`
	Resource   string         `json:"resource"`
	ResourceID string         `json:"resourceId"`
	Detail     map[string]any `json:"detail,omitempty"`
	OccurredAt string         `json:"occurredAt"`
}

func resolveActorName(detail map[string]any) string {
	if len(detail) == 0 {
		return ""
	}
	name, ok := detail["account"].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(name)
}

func toAuditRecordDTO(item *domainaudit.Record) auditRecordDTO {
	if item == nil {
		return auditRecordDTO{}
	}
	detail := item.Detail
	if len(detail) == 0 {
		detail = nil
	}
	return auditRecordDTO{
		ID:         item.ID.String(),
		ActorID:    item.ActorID.String(),
		ActorName:  resolveActorName(detail),
		Action:     item.Action,
		Resource:   item.Resource,
		ResourceID: item.ResourceID,
		Detail:     detail,
		OccurredAt: item.OccurredAt.Format(time.RFC3339Nano),
	}
}

func toAuditRecordDTOs(items []*domainaudit.Record) []auditRecordDTO {
	if len(items) == 0 {
		return []auditRecordDTO{}
	}
	out := make([]auditRecordDTO, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, toAuditRecordDTO(item))
	}
	return out
}

func RegisterAuditRoutes(mux *http.ServeMux, service auditsvc.Service, eventServices ...auditsvc.EventService) {
	var events auditsvc.EventService
	if len(eventServices) > 0 {
		events = eventServices[0]
	}
	if service == nil && events == nil {
		return
	}
	h := &AuditHandler{service: service, events: events}
	mux.HandleFunc("GET /v1/audit", h.list)
	mux.HandleFunc("GET /v1/audit/export", h.export)
	mux.HandleFunc("GET /v1/audit/{id}", h.get)
	mux.HandleFunc("DELETE /v1/audit", h.clear)
	mux.HandleFunc("GET /v1/audit/actors/{actorId}", h.listByActor)
}

func (h *AuditHandler) listByActor(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.service.ListByActor(r.Context(), r.PathValue("actorId"), limit)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, toAuditRecordDTOs(items))
}

func (h *AuditHandler) list(w http.ResponseWriter, r *http.Request) {
	if h.events != nil {
		data, err := h.queryEvents(r)
		if err != nil {
			apiv1.WriteError(w, http.StatusBadRequest, err)
			return
		}
		apiv1.WriteJSON(w, http.StatusOK, data)
		return
	}
	items, err := h.queryRecords(r)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, toAuditRecordDTOs(items))
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
	apiv1.WriteJSON(w, http.StatusOK, toAuditRecordDTO(item))
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
	actorName := strings.TrimSpace(r.URL.Query().Get("actorName"))
	action := strings.TrimSpace(r.URL.Query().Get("action"))
	resource := strings.TrimSpace(r.URL.Query().Get("resource"))
	if actorID != "" && actorName == "" && action == "" && resource == "" && r.URL.Query().Get("from") == "" && r.URL.Query().Get("to") == "" {
		items, err := h.service.ListByActor(r.Context(), actorID, limit)
		if err != nil {
			return nil, err
		}
		return filterRecords(items, actorID, actorName, action, resource, limit), nil
	}
	from, to, err := parseTimeRange(r)
	if err != nil {
		return nil, err
	}
	items, err := h.service.ListByTimeRange(r.Context(), from, to, 5000)
	if err != nil {
		return nil, err
	}
	return filterRecords(items, actorID, actorName, action, resource, limit), nil
}

func (h *AuditHandler) queryEvents(r *http.Request) (auditEventListData, error) {
	offset, err := parseNonNegativeInt(r.URL.Query().Get("offset"), 0)
	if err != nil {
		return auditEventListData{}, err
	}
	limit, err := parseBoundedLimit(r.URL.Query().Get("limit"), 50, 200)
	if err != nil {
		return auditEventListData{}, err
	}
	filter := auditsvc.EventFilter{
		ActorID:      strings.TrimSpace(r.URL.Query().Get("actorId")),
		ResourceType: strings.TrimSpace(r.URL.Query().Get("resourceType")),
		ResourceID:   strings.TrimSpace(r.URL.Query().Get("resourceId")),
		Offset:       offset,
		Limit:        limit,
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("type")); raw != "" {
		eventType, err := domainaudit.ParseEventType(raw)
		if err != nil {
			return auditEventListData{}, err
		}
		filter.Type = eventType
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("action")); raw != "" {
		action, err := domainaudit.ParseAuditAction(raw)
		if err != nil {
			return auditEventListData{}, err
		}
		filter.Action = action
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("result")); raw != "" {
		result := domainaudit.EventResult(strings.ToLower(raw))
		if err := result.Validate(); err != nil {
			return auditEventListData{}, err
		}
		filter.Result = result
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("risk")); raw != "" {
		risk := domainaudit.EventRisk(strings.ToLower(raw))
		if err := risk.Validate(); err != nil {
			return auditEventListData{}, err
		}
		filter.Risk = risk
	}
	fromRaw := strings.TrimSpace(r.URL.Query().Get("from"))
	toRaw := strings.TrimSpace(r.URL.Query().Get("to"))
	if fromRaw != "" || toRaw != "" {
		if fromRaw == "" || toRaw == "" {
			return auditEventListData{}, domainaudit.ErrInvalidTimeRange
		}
		from, err := time.Parse(time.RFC3339, fromRaw)
		if err != nil {
			return auditEventListData{}, err
		}
		to, err := time.Parse(time.RFC3339, toRaw)
		if err != nil {
			return auditEventListData{}, err
		}
		filter.From = from
		filter.To = to
	}
	items, err := h.events.ListEvents(r.Context(), filter)
	if err != nil {
		return auditEventListData{}, err
	}
	return auditEventListData{Items: toAuditEventDTOs(items), Offset: offset, Limit: limit}, nil
}

func toAuditEventDTOs(items []*domainaudit.Event) []auditEventDTO {
	if len(items) == 0 {
		return []auditEventDTO{}
	}
	out := make([]auditEventDTO, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		metadata := item.Metadata
		if len(metadata) == 0 {
			metadata = nil
		}
		out = append(out, auditEventDTO{
			ID:     item.ID.String(),
			Type:   item.Type.String(),
			Action: item.Action.String(),
			Actor: auditRefDTO{
				Type: item.Actor.Type,
				ID:   item.Actor.ID.String(),
				Name: item.Actor.Name,
			},
			Resource: auditRefDTO{
				Type: item.Resource.Type,
				ID:   item.Resource.ID,
				Name: item.Resource.Name,
			},
			Result: string(item.Result),
			Risk:   string(item.Risk),
			Trace: auditTraceDTO{
				TraceID:   item.Trace.TraceID,
				RequestID: item.Trace.RequestID,
				Method:    item.Trace.Method,
				Path:      item.Trace.Path,
				IP:        item.Trace.IP,
				UserAgent: item.Trace.UserAgent,
			},
			Metadata:   metadata,
			OccurredAt: item.OccurredAt.Format(time.RFC3339Nano),
		})
	}
	return out
}

func parseNonNegativeInt(raw string, defaultValue int) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return defaultValue, nil
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	if n < 0 {
		return 0, fmt.Errorf("value must be >= 0")
	}
	return n, nil
}

func parseBoundedLimit(raw string, defaultValue, maxValue int) (int, error) {
	limit, err := parseNonNegativeInt(raw, defaultValue)
	if err != nil {
		return 0, err
	}
	if limit <= 0 {
		limit = defaultValue
	}
	if limit > maxValue {
		limit = maxValue
	}
	return limit, nil
}

func filterRecords(items []*domainaudit.Record, actorID, actorName, action, resource string, limit int) []*domainaudit.Record {
	if len(items) == 0 {
		return []*domainaudit.Record{}
	}
	wantActor := strings.TrimSpace(actorID)
	wantActorName := strings.TrimSpace(actorName)
	wantAction := strings.TrimSpace(action)
	wantResource := strings.TrimSpace(resource)
	max := limit
	if max <= 0 {
		max = len(items)
	}
	result := make([]*domainaudit.Record, 0, min(max, len(items)))
	for _, item := range items {
		if item == nil {
			continue
		}
		if wantActor != "" && item.ActorID.String() != wantActor {
			continue
		}
		if wantActorName != "" && !strings.EqualFold(resolveActorName(item.Detail), wantActorName) {
			continue
		}
		if wantAction != "" && !strings.EqualFold(strings.TrimSpace(item.Action), wantAction) {
			continue
		}
		if wantResource != "" && !strings.EqualFold(strings.TrimSpace(item.Resource), wantResource) {
			continue
		}
		result = append(result, item)
		if len(result) >= max {
			break
		}
	}
	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
