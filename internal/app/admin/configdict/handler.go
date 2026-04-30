package configdict

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/tinboxw/skoll/internal/module/audit"
	"github.com/tinboxw/skoll/internal/module/config"
	"github.com/tinboxw/skoll/internal/module/dictionary"
)

const auditQueryMaxSize = 200

type APIRegistry interface {
	RegisterMany(entries []string)
}

type ConfigService interface {
	Set(key, value, description string) config.Entry
	Get(key string) (config.Entry, error)
	List() []config.Entry
}

type DictionaryService interface {
	Create(itemType, label, value string, sortOrder int, enabled bool) dictionary.Item
	Get(id int64) (dictionary.Item, error)
	List() []dictionary.Item
	ListByType(itemType string) []dictionary.Item
}

type AuditService interface {
	Append(actor, action, target string) audit.Record
}

type Handler struct {
	configs      ConfigService
	dictionaries DictionaryService
	audit        AuditService
}

func NewHandler(configs ConfigService, dictionaries DictionaryService, auditSvc AuditService) *Handler {
	return &Handler{configs: configs, dictionaries: dictionaries, audit: auditSvc}
}

func (h *Handler) Register(mux *http.ServeMux, wrapper func(http.Handler) http.Handler, apis APIRegistry) {
	if mux == nil || h == nil || h.configs == nil || h.dictionaries == nil {
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

	handle("POST /admin/v1/configs", h.upsertConfig)
	handle("POST /admin/v1/configs/bulk", h.upsertConfigsBulk)
	handle("GET /admin/v1/configs", h.listConfigs)
	handle("GET /admin/v1/configs/query", h.listConfigsQuery)
	handle("GET /admin/v1/configs/{key}", h.getConfig)
	handle("POST /admin/v1/dictionaries", h.createDictionary)
	handle("POST /admin/v1/dictionaries/bulk", h.createDictionariesBulk)
	handle("GET /admin/v1/dictionaries", h.listDictionaries)
	handle("GET /admin/v1/dictionaries/query", h.listDictionariesQuery)
	handle("GET /admin/v1/dictionaries/{id}", h.getDictionary)
}

func (h *Handler) upsertConfig(w http.ResponseWriter, r *http.Request) {
	var req upsertConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	req.Key = strings.TrimSpace(req.Key)
	req.Value = strings.TrimSpace(req.Value)
	req.Description = strings.TrimSpace(req.Description)
	if req.Key == "" || req.Value == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "key and value are required"})
		return
	}

	respondJSON(w, http.StatusCreated, h.configs.Set(req.Key, req.Value, req.Description))
}

func (h *Handler) upsertConfigsBulk(w http.ResponseWriter, r *http.Request) {
	var req upsertConfigsBulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	if len(req.Items) == 0 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "items are required"})
		return
	}
	for i := range req.Items {
		req.Items[i].Key = strings.TrimSpace(req.Items[i].Key)
		req.Items[i].Value = strings.TrimSpace(req.Items[i].Value)
		req.Items[i].Description = strings.TrimSpace(req.Items[i].Description)
		if req.Items[i].Key == "" || req.Items[i].Value == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid item at index %d", i)})
			return
		}
	}

	out := make([]config.Entry, 0, len(req.Items))
	for _, item := range req.Items {
		out = append(out, h.configs.Set(item.Key, item.Value, item.Description))
	}
	h.appendAudit("admin", "configs_bulk_upsert", fmt.Sprintf("count:%d", len(out)))
	respondJSON(w, http.StatusCreated, upsertConfigsBulkResponse{Atomic: true, Count: len(out), Items: out})
}

func (h *Handler) listConfigs(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, h.configs.List())
}

func (h *Handler) listConfigsQuery(w http.ResponseWriter, r *http.Request) {
	page, size, err := parsePageSizeQuery(r, audit.DefaultPage, audit.DefaultSize, auditQueryMaxSize)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	prefix := strings.TrimSpace(r.URL.Query().Get("key_prefix"))
	keyword := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	sortMode := strings.TrimSpace(r.URL.Query().Get("sort"))

	items := h.configs.List()
	filtered := make([]config.Entry, 0, len(items))
	for _, item := range items {
		if prefix != "" && !strings.HasPrefix(item.Key, prefix) {
			continue
		}
		if keyword != "" {
			blob := strings.ToLower(item.Key + " " + item.Value + " " + item.Description)
			if !strings.Contains(blob, keyword) {
				continue
			}
		}
		filtered = append(filtered, item)
	}

	if sortMode == "key_desc" {
		sort.Slice(filtered, func(i, j int) bool { return filtered[i].Key > filtered[j].Key })
	}

	pageItems, total := paginateSlice(filtered, page, size)
	respondJSON(w, http.StatusOK, configQueryResponse{
		Items:   pageItems,
		Page:    page,
		Size:    size,
		Total:   total,
		HasNext: page*size < total,
	})
}

func (h *Handler) getConfig(w http.ResponseWriter, r *http.Request) {
	key, err := parsePathString(r, "key")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	item, err := h.configs.Get(key)
	if err != nil {
		if err == config.ErrConfigNotFound {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	respondJSON(w, http.StatusOK, item)
}

func (h *Handler) createDictionary(w http.ResponseWriter, r *http.Request) {
	var req createDictionaryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	req.Type = strings.TrimSpace(req.Type)
	req.Label = strings.TrimSpace(req.Label)
	req.Value = strings.TrimSpace(req.Value)
	if req.Type == "" || req.Label == "" || req.Value == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "type, label and value are required"})
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	respondJSON(w, http.StatusCreated, h.dictionaries.Create(req.Type, req.Label, req.Value, req.Sort, enabled))
}

func (h *Handler) createDictionariesBulk(w http.ResponseWriter, r *http.Request) {
	var req createDictionariesBulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	if len(req.Items) == 0 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "items are required"})
		return
	}
	for i := range req.Items {
		req.Items[i].Type = strings.TrimSpace(req.Items[i].Type)
		req.Items[i].Label = strings.TrimSpace(req.Items[i].Label)
		req.Items[i].Value = strings.TrimSpace(req.Items[i].Value)
		if req.Items[i].Type == "" || req.Items[i].Label == "" || req.Items[i].Value == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid item at index %d", i)})
			return
		}
	}

	out := make([]dictionary.Item, 0, len(req.Items))
	for _, item := range req.Items {
		enabled := true
		if item.Enabled != nil {
			enabled = *item.Enabled
		}
		out = append(out, h.dictionaries.Create(item.Type, item.Label, item.Value, item.Sort, enabled))
	}
	h.appendAudit("admin", "dictionaries_bulk_create", fmt.Sprintf("count:%d", len(out)))
	respondJSON(w, http.StatusCreated, createDictionariesBulkResponse{Atomic: true, Count: len(out), Items: out})
}

func (h *Handler) listDictionaries(w http.ResponseWriter, r *http.Request) {
	itemType := strings.TrimSpace(r.URL.Query().Get("type"))
	if itemType == "" {
		respondJSON(w, http.StatusOK, h.dictionaries.List())
		return
	}
	respondJSON(w, http.StatusOK, h.dictionaries.ListByType(itemType))
}

func (h *Handler) listDictionariesQuery(w http.ResponseWriter, r *http.Request) {
	page, size, err := parsePageSizeQuery(r, audit.DefaultPage, audit.DefaultSize, auditQueryMaxSize)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	itemType := strings.TrimSpace(r.URL.Query().Get("type"))
	keyword := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	enabledRaw := strings.TrimSpace(r.URL.Query().Get("enabled"))
	enabledFilter := false
	enabledValue := false
	if enabledRaw != "" {
		enabledFilter = true
		parsed, parseErr := strconv.ParseBool(enabledRaw)
		if parseErr != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "enabled must be true or false"})
			return
		}
		enabledValue = parsed
	}

	items := h.dictionaries.List()
	filtered := make([]dictionary.Item, 0, len(items))
	for _, item := range items {
		if itemType != "" && item.Type != itemType {
			continue
		}
		if enabledFilter && item.Enabled != enabledValue {
			continue
		}
		if keyword != "" {
			blob := strings.ToLower(item.Type + " " + item.Label + " " + item.Value)
			if !strings.Contains(blob, keyword) {
				continue
			}
		}
		filtered = append(filtered, item)
	}

	pageItems, total := paginateSlice(filtered, page, size)
	respondJSON(w, http.StatusOK, dictionaryQueryResponse{
		Items:   pageItems,
		Page:    page,
		Size:    size,
		Total:   total,
		HasNext: page*size < total,
	})
}

func (h *Handler) getDictionary(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathInt64(r, "id")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	item, err := h.dictionaries.Get(id)
	if err != nil {
		if err == dictionary.ErrItemNotFound {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	respondJSON(w, http.StatusOK, item)
}

func (h *Handler) appendAudit(actor, action, target string) {
	if h.audit == nil {
		return
	}
	h.audit.Append(actor, action, target)
}

func parsePathString(r *http.Request, key string) (string, error) {
	raw := strings.TrimSpace(r.PathValue(key))
	if raw == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return raw, nil
}

func parsePathInt64(r *http.Request, key string) (int64, error) {
	raw := strings.TrimSpace(r.PathValue(key))
	if raw == "" {
		return 0, fmt.Errorf("%s is required", key)
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}
	return id, nil
}

func parsePageSizeQuery(r *http.Request, defaultPage, defaultSize, maxSize int) (int, int, error) {
	page := defaultPage
	size := defaultSize
	if rawPage := strings.TrimSpace(r.URL.Query().Get("page")); rawPage != "" {
		parsed, err := strconv.Atoi(rawPage)
		if err != nil || parsed <= 0 {
			return 0, 0, fmt.Errorf("page must be a positive integer")
		}
		page = parsed
	}
	if rawSize := strings.TrimSpace(r.URL.Query().Get("size")); rawSize != "" {
		parsed, err := strconv.Atoi(rawSize)
		if err != nil || parsed <= 0 {
			return 0, 0, fmt.Errorf("size must be a positive integer")
		}
		if parsed > maxSize {
			return 0, 0, fmt.Errorf("size must be <= %d", maxSize)
		}
		size = parsed
	}
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 {
			return 0, 0, fmt.Errorf("limit must be a positive integer")
		}
		if parsed > maxSize {
			return 0, 0, fmt.Errorf("limit must be <= %d", maxSize)
		}
		page = 1
		size = parsed
	}
	return page, size, nil
}

func paginateSlice[T any](items []T, page, size int) ([]T, int) {
	total := len(items)
	start := (page - 1) * size
	if start >= total {
		return []T{}, total
	}
	end := start + size
	if end > total {
		end = total
	}
	return items[start:end], total
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

type upsertConfigRequest struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

type upsertConfigsBulkRequest struct {
	Items []upsertConfigRequest `json:"items"`
}

type upsertConfigsBulkResponse struct {
	Atomic bool           `json:"atomic"`
	Count  int            `json:"count"`
	Items  []config.Entry `json:"items"`
}

type configQueryResponse struct {
	Items   []config.Entry `json:"items"`
	Page    int            `json:"page"`
	Size    int            `json:"size"`
	Total   int            `json:"total"`
	HasNext bool           `json:"has_next"`
}

type createDictionaryRequest struct {
	Type    string `json:"type"`
	Label   string `json:"label"`
	Value   string `json:"value"`
	Sort    int    `json:"sort"`
	Enabled *bool  `json:"enabled"`
}

type createDictionariesBulkRequest struct {
	Items []createDictionaryRequest `json:"items"`
}

type createDictionariesBulkResponse struct {
	Atomic bool              `json:"atomic"`
	Count  int               `json:"count"`
	Items  []dictionary.Item `json:"items"`
}

type dictionaryQueryResponse struct {
	Items   []dictionary.Item `json:"items"`
	Page    int               `json:"page"`
	Size    int               `json:"size"`
	Total   int               `json:"total"`
	HasNext bool              `json:"has_next"`
}
