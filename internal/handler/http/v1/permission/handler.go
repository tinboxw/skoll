package permission

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	domainpermission "github.com/tinboxw/skoll/internal/domain/permission"
	apiv1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	permissionsvc "github.com/tinboxw/skoll/internal/service/permission"
)

type Handler struct {
	service permissionsvc.Service
}

func RegisterPermissionRoutes(mux *http.ServeMux, service permissionsvc.Service) {
	if service == nil {
		return
	}
	h := &Handler{service: service}
	mux.HandleFunc("GET /v1/permissions", h.list)
	mux.HandleFunc("GET /v1/permissions/{key}", h.get)
	mux.HandleFunc("POST /v1/permissions/{key}/enable", h.enable)
	mux.HandleFunc("POST /v1/permissions/{key}/disable", h.disable)
	mux.HandleFunc("POST /v1/permissions/diff", h.diff)
}

type resourceRecord struct {
	Key      string            `json:"key"`
	Type     string            `json:"type"`
	Module   string            `json:"module"`
	Source   string            `json:"source"`
	Name     string            `json:"name"`
	Risk     string            `json:"risk"`
	Metadata map[string]string `json:"metadata"`
	Enabled  bool              `json:"enabled"`
}

type resourceInput struct {
	Key      string            `json:"key"`
	Type     string            `json:"type"`
	Module   string            `json:"module"`
	Source   string            `json:"source"`
	Name     string            `json:"name"`
	Risk     string            `json:"risk"`
	Metadata map[string]string `json:"metadata"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	offset, _ := strconv.Atoi(query.Get("offset"))
	limit, _ := strconv.Atoi(query.Get("limit"))
	var enabled *bool
	if raw := strings.TrimSpace(query.Get("enabled")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			apiv1.WriteError(w, http.StatusBadRequest, err)
			return
		}
		enabled = &value
	}
	items, err := h.service.ListResources(r.Context(), permissionsvc.ListResourcesInput{
		Type:    domainpermission.ResourceType(strings.TrimSpace(query.Get("type"))),
		Source:  strings.TrimSpace(query.Get("source")),
		Enabled: enabled,
		Offset:  offset,
		Limit:   limit,
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{
		"items":  resourceRecords(items),
		"offset": offset,
		"limit":  limit,
	})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.GetResource(r.Context(), r.PathValue("key"))
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	if item == nil {
		apiv1.WriteMessage(w, http.StatusNotFound, "not_found", "permission resource not found")
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": resourceRecordFromDomain(*item)})
}

func (h *Handler) enable(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if err := h.service.EnableResource(r.Context(), key); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"key": strings.TrimSpace(key), "enabled": true})
}

func (h *Handler) disable(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if err := h.service.DisableResource(r.Context(), key); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"key": strings.TrimSpace(key), "enabled": false})
}

func (h *Handler) diff(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Source  string          `json:"source"`
		Desired []resourceInput `json:"desired"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	desired := make([]permissionsvc.RegisterResourceInput, 0, len(req.Desired))
	for _, item := range req.Desired {
		desired = append(desired, permissionsvc.RegisterResourceInput{
			Key:      item.Key,
			Type:     domainpermission.ResourceType(item.Type),
			Module:   item.Module,
			Source:   item.Source,
			Name:     item.Name,
			Risk:     domainpermission.RiskLevel(item.Risk),
			Metadata: item.Metadata,
		})
	}
	result, err := h.service.DiffResources(r.Context(), permissionsvc.DiffResourcesInput{
		Source:  req.Source,
		Desired: desired,
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{
		"added":   resourceRecords(result.Added),
		"updated": resourceRecords(result.Updated),
		"removed": resourceRecords(result.Removed),
	})
}

func resourceRecords(items []domainpermission.PermissionResource) []resourceRecord {
	out := make([]resourceRecord, 0, len(items))
	for _, item := range items {
		out = append(out, resourceRecordFromDomain(item))
	}
	return out
}

func resourceRecordFromDomain(item domainpermission.PermissionResource) resourceRecord {
	return resourceRecord{
		Key:      item.Key(),
		Type:     string(item.Type()),
		Module:   item.Module(),
		Source:   item.Source(),
		Name:     item.Name,
		Risk:     string(item.Risk),
		Metadata: domainpermission.NormalizeMetadata(item.Metadata),
		Enabled:  item.Enabled,
	}
}
