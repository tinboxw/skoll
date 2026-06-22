package system

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"

	domainsystem "github.com/tinboxw/skoll/internal/domain/system"
	apiv1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	systemsvc "github.com/tinboxw/skoll/internal/service/system"
	"github.com/tinboxw/skoll/pkg/security"
)

const (
	systemMenuSettingKey       = "skoll.menu.tree"
	systemDictionarySettingKey = "skoll.dictionary.types"
	systemDepartmentSettingKey = "skoll.organization.departments"
	systemPositionSettingKey   = "skoll.organization.positions"
)

type SystemHandler struct {
	service systemsvc.Service
	audit   auditsvc.Service
}

type MenuItem struct {
	ID                  string     `json:"id"`
	Label               string     `json:"label"`
	Path                string     `json:"path"`
	Icon                string     `json:"icon,omitempty"`
	Order               int        `json:"order"`
	Visible             bool       `json:"visible"`
	RequiredRoles       []string   `json:"requiredRoles,omitempty"`
	RequiredPermissions []string   `json:"requiredPermissions,omitempty"`
	Children            []MenuItem `json:"children,omitempty"`
}

type DictionaryType struct {
	ID          string           `json:"id,omitempty"`
	Type        string           `json:"type"`
	Code        string           `json:"code"`
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Status      string           `json:"status"`
	Sort        int              `json:"sort"`
	Order       int              `json:"order"`
	Builtin     bool             `json:"builtin"`
	Items       []DictionaryItem `json:"items"`
}

type DictionaryItem struct {
	ID      string `json:"id,omitempty"`
	Label   string `json:"label"`
	Value   string `json:"value"`
	Status  string `json:"status"`
	Sort    int    `json:"sort"`
	Order   int    `json:"order"`
	Builtin bool   `json:"builtin"`
}

type DepartmentRecord struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ParentID string `json:"parentId,omitempty"`
	Leader   string `json:"leader,omitempty"`
	Status   string `json:"status"`
	Order    int    `json:"order"`
}

type PositionRecord struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"`
	Order       int    `json:"order"`
}

func RegisterSystemRoutes(mux *http.ServeMux, service systemsvc.Service, auditSvc auditsvc.Service) {
	if service == nil {
		return
	}
	h := &SystemHandler{service: service, audit: auditSvc}
	mux.HandleFunc("GET /v1/system/settings", h.list)
	mux.HandleFunc("GET /v1/system/settings/schema", h.getSettingsSchema)
	mux.HandleFunc("GET /v1/system/settings/{key}", h.getByKey)
	mux.HandleFunc("PUT /v1/system/settings/{key}", h.upsert)
	mux.HandleFunc("POST /v1/system/settings/reset", h.reset)
	mux.HandleFunc("GET /v1/system/dictionaries", h.getDictionaries)
	mux.HandleFunc("PUT /v1/system/dictionaries", h.putDictionaries)
	mux.HandleFunc("PUT /v1/system/dictionaries/{type}", h.putDictionaryType)
	mux.HandleFunc("DELETE /v1/system/dictionaries/{type}", h.deleteDictionaryType)
	mux.HandleFunc("GET /v1/system/dictionaries/{type}", h.getDictionaryByType)
	mux.HandleFunc("GET /v1/system/dictionaries/{type}/items", h.getDictionaryItems)
	mux.HandleFunc("PUT /v1/system/dictionaries/{type}/items/{value}", h.putDictionaryItem)
	mux.HandleFunc("DELETE /v1/system/dictionaries/{type}/items/{value}", h.deleteDictionaryItem)
	mux.HandleFunc("GET /v1/system/departments", h.getDepartments)
	mux.HandleFunc("PUT /v1/system/departments", h.putDepartments)
	mux.HandleFunc("GET /v1/system/positions", h.getPositions)
	mux.HandleFunc("PUT /v1/system/positions", h.putPositions)
	mux.HandleFunc("GET /v1/system/menus", h.getMenus)
	mux.HandleFunc("PUT /v1/system/menus", h.putMenus)
}

type SettingSchema struct {
	Title       string               `json:"title,omitempty"`
	TitleZhCN   string               `json:"titleZhCN,omitempty"`
	TitleEnUS   string               `json:"titleEnUS,omitempty"`
	Description string               `json:"description,omitempty"`
	Fields      []SettingSchemaField `json:"fields"`
}

type SettingSchemaField struct {
	Key         string                `json:"key"`
	Label       string                `json:"label,omitempty"`
	LabelZhCN   string                `json:"labelZhCN,omitempty"`
	LabelEnUS   string                `json:"labelEnUS,omitempty"`
	Type        string                `json:"type,omitempty"`
	Required    bool                  `json:"required,omitempty"`
	Default     string                `json:"default,omitempty"`
	Placeholder string                `json:"placeholder,omitempty"`
	Help        string                `json:"help,omitempty"`
	Min         *float64              `json:"min,omitempty"`
	Max         *float64              `json:"max,omitempty"`
	MinLength   *int                  `json:"minLength,omitempty"`
	MaxLength   *int                  `json:"maxLength,omitempty"`
	Pattern     string                `json:"pattern,omitempty"`
	Options     []SettingSchemaOption `json:"options,omitempty"`
}

type SettingSchemaOption struct {
	Label     string `json:"label,omitempty"`
	LabelZhCN string `json:"labelZhCN,omitempty"`
	LabelEnUS string `json:"labelEnUS,omitempty"`
	Value     string `json:"value"`
}

func (h *SystemHandler) list(w http.ResponseWriter, r *http.Request) {
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.service.List(r.Context(), systemsvc.ListInput{Offset: offset, Limit: limit})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, items)
}

func (h *SystemHandler) getSettingsSchema(w http.ResponseWriter, _ *http.Request) {
	apiv1.WriteJSON(w, http.StatusOK, defaultSettingsSchema())
}

func (h *SystemHandler) getByKey(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.GetByKey(r.Context(), r.PathValue("key"))
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	if item == nil {
		apiv1.WriteMessage(w, http.StatusNotFound, "not_found", "setting not found")
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, item)
}

func (h *SystemHandler) upsert(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	var req struct {
		Value     string `json:"value"`
		Encrypted bool   `json:"encrypted"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	before, _ := h.service.GetByKey(r.Context(), key)
	item, err := h.service.Upsert(r.Context(), systemsvc.UpsertInput{
		Key:       key,
		Value:     req.Value,
		Encrypted: req.Encrypted,
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	detail := map[string]any{"after": map[string]any{"value": item.Value, "encrypted": item.Encrypted}}
	if before != nil {
		detail["before"] = map[string]any{"value": before.Value, "encrypted": before.Encrypted}
	}
	h.appendAudit(r, "upsert", "system_setting", key, detail)
	apiv1.WriteJSON(w, http.StatusOK, item)
}

func (h *SystemHandler) reset(w http.ResponseWriter, r *http.Request) {
	count, err := h.service.Reset(r.Context())
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	h.appendAudit(r, "reset", "system_setting", "", map[string]any{"count": count})
	apiv1.WriteJSON(w, http.StatusOK, map[string]int{"reset": count})
}

func (h *SystemHandler) getMenus(w http.ResponseWriter, r *http.Request) {
	items, customized, err := h.loadMenus(r)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{
		"items":      items,
		"customized": customized,
	})
}

func (h *SystemHandler) putMenus(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Items []MenuItem `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	items := normalizeMenuItems(req.Items)
	raw, err := json.Marshal(items)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	before, _ := h.service.GetByKey(r.Context(), systemMenuSettingKey)
	_, err = h.service.Upsert(r.Context(), systemsvc.UpsertInput{
		Key:   systemMenuSettingKey,
		Value: string(raw),
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	detail := map[string]any{"after": len(items)}
	if before != nil {
		detail["before"] = len(parseStoredMenus(before.Value))
	}
	h.appendAudit(r, "upsert_menus", "system_menu", systemMenuSettingKey, detail)
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{
		"items":      items,
		"customized": true,
	})
}

func (h *SystemHandler) getDictionaries(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	offset, _ := strconv.Atoi(query.Get("offset"))
	limit, _ := strconv.Atoi(query.Get("limit"))
	status := normalizeDictionaryStatusFilter(query.Get("status"))
	search := strings.ToLower(strings.TrimSpace(query.Get("search")))

	items, err := h.service.ListDictionaryTypes(r.Context(), systemsvc.DictionaryTypeListInput{Offset: 0, Limit: 0})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	records := make([]DictionaryType, 0, len(items))
	for _, item := range items {
		if status != "" && string(item.Status) != status {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(item.Code+" "+item.Name+" "+item.Description), search) {
			continue
		}
		dictItems, err := h.service.ListDictionaryItems(r.Context(), systemsvc.DictionaryItemListInput{TypeCode: item.Code})
		if err != nil {
			apiv1.WriteError(w, http.StatusBadRequest, err)
			return
		}
		records = append(records, dictionaryTypeRecord(item, dictItems))
	}
	records = paginateDictionaryTypeRecords(records, offset, limit)
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{
		"items":  records,
		"offset": offset,
		"limit":  limit,
	})
}

func (h *SystemHandler) getDictionaryByType(w http.ResponseWriter, r *http.Request) {
	dictType := strings.TrimSpace(r.PathValue("type"))
	item, err := h.service.GetDictionaryTypeByCode(r.Context(), dictType)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	if item == nil {
		apiv1.WriteMessage(w, http.StatusNotFound, "not_found", "dictionary type not found")
		return
	}
	items, err := h.service.ListDictionaryItems(r.Context(), systemsvc.DictionaryItemListInput{TypeCode: item.Code})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": dictionaryTypeRecord(*item, items)})
}

func (h *SystemHandler) putDictionaries(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Items []DictionaryType `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	items := normalizeDictionaryTypes(req.Items)
	out := make([]DictionaryType, 0, len(items))
	for _, item := range items {
		saved, err := h.saveDictionaryTypeRecord(r, item)
		if err != nil {
			apiv1.WriteError(w, http.StatusBadRequest, err)
			return
		}
		out = append(out, saved)
	}
	h.appendAudit(r, "upsert_dictionaries", "system_dictionary", "", map[string]any{"after": len(out)})
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{
		"items": out,
	})
}

func (h *SystemHandler) putDictionaryType(w http.ResponseWriter, r *http.Request) {
	var req DictionaryType
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	pathType := strings.TrimSpace(r.PathValue("type"))
	if req.Code == "" && req.Type == "" {
		req.Code = pathType
		req.Type = pathType
	}
	normalized := normalizeDictionaryTypes([]DictionaryType{req})
	if len(normalized) == 0 {
		apiv1.WriteMessage(w, http.StatusBadRequest, "error", "dictionary type is required")
		return
	}
	saved, err := h.saveDictionaryTypeRecord(r, normalized[0])
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	h.appendAudit(r, "upsert_dictionary_type", "system_dictionary", saved.Code, map[string]any{"code": saved.Code})
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": saved})
}

func (h *SystemHandler) deleteDictionaryType(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(r.PathValue("type"))
	item, err := h.service.GetDictionaryTypeByCode(r.Context(), code)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	if item == nil {
		apiv1.WriteMessage(w, http.StatusNotFound, "not_found", "dictionary type not found")
		return
	}
	if err := h.service.DeleteDictionaryType(r.Context(), item.ID.String()); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	h.appendAudit(r, "delete_dictionary_type", "system_dictionary", item.Code, map[string]any{"code": item.Code})
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"code": item.Code, "deleted": true})
}

func (h *SystemHandler) getDictionaryItems(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	offset, _ := strconv.Atoi(query.Get("offset"))
	limit, _ := strconv.Atoi(query.Get("limit"))
	status := normalizeDictionaryStatusFilter(query.Get("status"))
	search := strings.ToLower(strings.TrimSpace(query.Get("search")))
	typeCode := strings.TrimSpace(r.PathValue("type"))
	items, err := h.service.ListDictionaryItems(r.Context(), systemsvc.DictionaryItemListInput{TypeCode: typeCode})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	records := make([]DictionaryItem, 0, len(items))
	for _, item := range items {
		if status != "" && string(item.Status) != status {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(item.Label+" "+item.Value), search) {
			continue
		}
		records = append(records, dictionaryItemRecord(item))
	}
	records = paginateDictionaryItemRecords(records, offset, limit)
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": records, "offset": offset, "limit": limit})
}

func (h *SystemHandler) putDictionaryItem(w http.ResponseWriter, r *http.Request) {
	var req DictionaryItem
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	typeCode := strings.TrimSpace(r.PathValue("type"))
	if strings.TrimSpace(req.Value) == "" {
		req.Value = strings.TrimSpace(r.PathValue("value"))
	}
	saved, err := h.service.SaveDictionaryItem(r.Context(), systemsvc.DictionaryItemInput{
		ID:       req.ID,
		TypeCode: typeCode,
		Label:    req.Label,
		Value:    req.Value,
		Status:   req.Status,
		Sort:     dictionarySort(req.Sort, req.Order),
		Builtin:  req.Builtin,
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	h.appendAudit(r, "upsert_dictionary_item", "system_dictionary_item", saved.TypeCode+":"+saved.Value, map[string]any{"type": saved.TypeCode, "value": saved.Value})
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": dictionaryItemRecord(*saved)})
}

func (h *SystemHandler) deleteDictionaryItem(w http.ResponseWriter, r *http.Request) {
	typeCode := strings.TrimSpace(r.PathValue("type"))
	value := strings.TrimSpace(r.PathValue("value"))
	item, err := h.service.GetDictionaryItemByTypeAndValue(r.Context(), typeCode, value)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	if item == nil {
		apiv1.WriteMessage(w, http.StatusNotFound, "not_found", "dictionary item not found")
		return
	}
	if err := h.service.DeleteDictionaryItem(r.Context(), item.ID.String()); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	h.appendAudit(r, "delete_dictionary_item", "system_dictionary_item", item.TypeCode+":"+item.Value, map[string]any{"type": item.TypeCode, "value": item.Value})
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"type": item.TypeCode, "value": item.Value, "deleted": true})
}

func (h *SystemHandler) saveDictionaryTypeRecord(r *http.Request, item DictionaryType) (DictionaryType, error) {
	code := strings.TrimSpace(item.Code)
	if code == "" {
		code = strings.TrimSpace(item.Type)
	}
	saved, err := h.service.SaveDictionaryType(r.Context(), systemsvc.DictionaryTypeInput{
		ID:          item.ID,
		Code:        code,
		Name:        item.Name,
		Description: item.Description,
		Status:      item.Status,
		Sort:        dictionarySort(item.Sort, item.Order),
		Builtin:     item.Builtin,
	})
	if err != nil {
		return DictionaryType{}, err
	}
	outItems := make([]domainsystem.DictionaryItem, 0, len(item.Items))
	for _, child := range item.Items {
		savedChild, err := h.service.SaveDictionaryItem(r.Context(), systemsvc.DictionaryItemInput{
			ID:       child.ID,
			TypeCode: saved.Code,
			Label:    child.Label,
			Value:    child.Value,
			Status:   child.Status,
			Sort:     dictionarySort(child.Sort, child.Order),
			Builtin:  child.Builtin,
		})
		if err != nil {
			return DictionaryType{}, err
		}
		outItems = append(outItems, *savedChild)
	}
	return dictionaryTypeRecord(*saved, outItems), nil
}

func (h *SystemHandler) getDepartments(w http.ResponseWriter, r *http.Request) {
	items, customized, err := h.loadDepartments(r)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "customized": customized})
}

func (h *SystemHandler) putDepartments(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Items []DepartmentRecord `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	items := normalizeDepartments(req.Items)
	if err := h.saveSystemList(r, systemDepartmentSettingKey, items, "upsert_departments", "system_department"); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "customized": true})
}

func (h *SystemHandler) getPositions(w http.ResponseWriter, r *http.Request) {
	items, customized, err := h.loadPositions(r)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "customized": customized})
}

func (h *SystemHandler) putPositions(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Items []PositionRecord `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	items := normalizePositions(req.Items)
	if err := h.saveSystemList(r, systemPositionSettingKey, items, "upsert_positions", "system_position"); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "customized": true})
}

func (h *SystemHandler) saveSystemList(r *http.Request, key string, items any, action, resource string) error {
	raw, err := json.Marshal(items)
	if err != nil {
		return err
	}
	before, _ := h.service.GetByKey(r.Context(), key)
	_, err = h.service.Upsert(r.Context(), systemsvc.UpsertInput{
		Key:   key,
		Value: string(raw),
	})
	if err != nil {
		return err
	}
	detail := map[string]any{"key": key}
	if before != nil {
		detail["before"] = before.Value != ""
	}
	h.appendAudit(r, action, resource, key, detail)
	return nil
}

func (h *SystemHandler) loadDepartments(r *http.Request) ([]DepartmentRecord, bool, error) {
	item, err := h.service.GetByKey(r.Context(), systemDepartmentSettingKey)
	if err != nil {
		return nil, false, err
	}
	if item == nil || strings.TrimSpace(item.Value) == "" {
		return defaultDepartments(), false, nil
	}
	items := parseStoredDepartments(item.Value)
	if len(items) == 0 {
		return defaultDepartments(), false, nil
	}
	return items, true, nil
}

func (h *SystemHandler) loadPositions(r *http.Request) ([]PositionRecord, bool, error) {
	item, err := h.service.GetByKey(r.Context(), systemPositionSettingKey)
	if err != nil {
		return nil, false, err
	}
	if item == nil || strings.TrimSpace(item.Value) == "" {
		return defaultPositions(), false, nil
	}
	items := parseStoredPositions(item.Value)
	if len(items) == 0 {
		return defaultPositions(), false, nil
	}
	return items, true, nil
}

func (h *SystemHandler) loadMenus(r *http.Request) ([]MenuItem, bool, error) {
	item, err := h.service.GetByKey(r.Context(), systemMenuSettingKey)
	if err != nil {
		return nil, false, err
	}
	if item == nil || strings.TrimSpace(item.Value) == "" {
		return defaultSystemMenus(), false, nil
	}
	menus := parseStoredMenus(item.Value)
	if len(menus) == 0 {
		return defaultSystemMenus(), false, nil
	}
	return menus, true, nil
}

func parseStoredMenus(raw string) []MenuItem {
	var menus []MenuItem
	if err := json.Unmarshal([]byte(raw), &menus); err != nil {
		return nil
	}
	return normalizeMenuItems(menus)
}

func parseStoredDictionaries(raw string) []DictionaryType {
	var items []DictionaryType
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	return normalizeDictionaryTypes(items)
}

func parseStoredDepartments(raw string) []DepartmentRecord {
	var items []DepartmentRecord
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	return normalizeDepartments(items)
}

func parseStoredPositions(raw string) []PositionRecord {
	var items []PositionRecord
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	return normalizePositions(items)
}

func normalizeDepartments(items []DepartmentRecord) []DepartmentRecord {
	seen := map[string]struct{}{}
	out := make([]DepartmentRecord, 0, len(items))
	for _, item := range items {
		id := strings.TrimSpace(item.ID)
		name := strings.TrimSpace(item.Name)
		if id == "" || name == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		item.ID = id
		item.Name = name
		item.ParentID = strings.TrimSpace(item.ParentID)
		item.Leader = strings.TrimSpace(item.Leader)
		item.Status = normalizeDictionaryStatus(item.Status)
		out = append(out, item)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Order == out[j].Order {
			return out[i].ID < out[j].ID
		}
		return out[i].Order < out[j].Order
	})
	return out
}

func normalizePositions(items []PositionRecord) []PositionRecord {
	seen := map[string]struct{}{}
	out := make([]PositionRecord, 0, len(items))
	for _, item := range items {
		id := strings.TrimSpace(item.ID)
		code := strings.TrimSpace(item.Code)
		name := strings.TrimSpace(item.Name)
		if id == "" || code == "" || name == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		item.ID = id
		item.Code = code
		item.Name = name
		item.Description = strings.TrimSpace(item.Description)
		item.Status = normalizeDictionaryStatus(item.Status)
		out = append(out, item)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Order == out[j].Order {
			return out[i].Code < out[j].Code
		}
		return out[i].Order < out[j].Order
	})
	return out
}

func normalizeDictionaryTypes(items []DictionaryType) []DictionaryType {
	seenTypes := map[string]struct{}{}
	out := make([]DictionaryType, 0, len(items))
	for _, item := range items {
		dictType := strings.TrimSpace(item.Code)
		if dictType == "" {
			dictType = strings.TrimSpace(item.Type)
		}
		name := strings.TrimSpace(item.Name)
		if dictType == "" || name == "" {
			continue
		}
		if _, exists := seenTypes[dictType]; exists {
			continue
		}
		seenTypes[dictType] = struct{}{}
		status := normalizeDictionaryStatus(item.Status)
		item.Type = dictType
		item.Code = dictType
		item.Name = name
		item.Description = strings.TrimSpace(item.Description)
		item.Status = status
		item.Sort = dictionarySort(item.Sort, item.Order)
		item.Order = item.Sort
		item.Items = normalizeDictionaryItems(item.Items)
		out = append(out, item)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Sort == out[j].Sort {
			return out[i].Type < out[j].Type
		}
		return out[i].Sort < out[j].Sort
	})
	return out
}

func normalizeDictionaryItems(items []DictionaryItem) []DictionaryItem {
	seenValues := map[string]struct{}{}
	out := make([]DictionaryItem, 0, len(items))
	for _, item := range items {
		label := strings.TrimSpace(item.Label)
		value := strings.TrimSpace(item.Value)
		if label == "" || value == "" {
			continue
		}
		if _, exists := seenValues[value]; exists {
			continue
		}
		seenValues[value] = struct{}{}
		item.Label = label
		item.Value = value
		item.Status = normalizeDictionaryStatus(item.Status)
		item.Sort = dictionarySort(item.Sort, item.Order)
		item.Order = item.Sort
		out = append(out, item)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Sort == out[j].Sort {
			return out[i].Value < out[j].Value
		}
		return out[i].Sort < out[j].Sort
	})
	return out
}

func dictionaryTypeRecord(item domainsystem.DictionaryType, items []domainsystem.DictionaryItem) DictionaryType {
	out := DictionaryType{
		ID:          item.ID.String(),
		Type:        item.Code,
		Code:        item.Code,
		Name:        item.Name,
		Description: item.Description,
		Status:      string(item.Status),
		Sort:        item.Sort,
		Order:       item.Sort,
		Builtin:     item.Builtin,
		Items:       make([]DictionaryItem, 0, len(items)),
	}
	for _, child := range items {
		out.Items = append(out.Items, dictionaryItemRecord(child))
	}
	return out
}

func dictionaryItemRecord(item domainsystem.DictionaryItem) DictionaryItem {
	return DictionaryItem{
		ID:      item.ID.String(),
		Label:   item.Label,
		Value:   item.Value,
		Status:  string(item.Status),
		Sort:    item.Sort,
		Order:   item.Sort,
		Builtin: item.Builtin,
	}
}

func dictionarySort(sortValue, orderValue int) int {
	if sortValue != 0 {
		return sortValue
	}
	return orderValue
}

func normalizeDictionaryStatusFilter(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", "all":
		return ""
	case "enabled", "disabled":
		return strings.ToLower(strings.TrimSpace(status))
	default:
		return "__invalid__"
	}
}

func paginateDictionaryTypeRecords(items []DictionaryType, offset, limit int) []DictionaryType {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = len(items)
	}
	if offset > len(items) {
		return []DictionaryType{}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return append([]DictionaryType(nil), items[offset:end]...)
}

func paginateDictionaryItemRecords(items []DictionaryItem, offset, limit int) []DictionaryItem {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = len(items)
	}
	if offset > len(items) {
		return []DictionaryItem{}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return append([]DictionaryItem(nil), items[offset:end]...)
}

func normalizeDictionaryStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "disabled":
		return "disabled"
	default:
		return "enabled"
	}
}

func normalizeMenuItems(items []MenuItem) []MenuItem {
	out := make([]MenuItem, 0, len(items))
	for _, item := range items {
		id := strings.TrimSpace(item.ID)
		label := strings.TrimSpace(item.Label)
		path := strings.TrimSpace(item.Path)
		if id == "" || label == "" || path == "" || !strings.HasPrefix(path, "/") {
			continue
		}
		item.ID = id
		item.Label = label
		item.Path = path
		item.Icon = strings.TrimSpace(item.Icon)
		item.RequiredRoles = compactStrings(item.RequiredRoles)
		item.RequiredPermissions = compactStrings(item.RequiredPermissions)
		item.Children = normalizeMenuItems(item.Children)
		out = append(out, item)
	}
	return out
}

func compactStrings(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func defaultSystemMenus() []MenuItem {
	return []MenuItem{
		{ID: "dashboard", Label: "Dashboard", Path: "/skoll/dashboard", Icon: "dashboard", Order: 10, Visible: true},
		{ID: "users", Label: "Users", Path: "/skoll/user", Icon: "users", Order: 20, Visible: true, RequiredPermissions: []string{"user.read"}},
		{ID: "roles", Label: "Roles", Path: "/skoll/role", Icon: "roles", Order: 30, Visible: true, RequiredPermissions: []string{"role.read"}},
		{ID: "permissions", Label: "Permissions", Path: "/skoll/permission", Icon: "permissions", Order: 40, Visible: true, RequiredPermissions: []string{"permission.manage"}},
		{ID: "menus", Label: "Menus", Path: "/skoll/menu", Icon: "menus", Order: 45, Visible: true, RequiredPermissions: []string{"system.manage"}},
		{ID: "dictionaries", Label: "Dictionaries", Path: "/skoll/dictionary", Icon: "settings", Order: 47, Visible: true, RequiredPermissions: []string{"dict.read"}},
		{ID: "organization", Label: "Organization", Path: "/skoll/organization", Icon: "users", Order: 48, Visible: true, RequiredPermissions: []string{"org.read"}},
		{ID: "audit", Label: "Audit", Path: "/skoll/audit", Icon: "audit", Order: 50, Visible: true, RequiredPermissions: []string{"audit.read"}},
		{ID: "plugins", Label: "Plugins", Path: "/skoll/plugin", Icon: "plugins", Order: 60, Visible: true, RequiredPermissions: []string{"plugin.read"}},
		{ID: "settings", Label: "Settings", Path: "/skoll/setting", Icon: "settings", Order: 70, Visible: true, RequiredPermissions: []string{"system.manage"}},
	}
}

func defaultDepartments() []DepartmentRecord {
	return []DepartmentRecord{
		{ID: "dept-root", Name: "Headquarters", Status: "enabled", Order: 10},
	}
}

func defaultPositions() []PositionRecord {
	return []PositionRecord{
		{ID: "pos-admin", Code: "admin", Name: "Administrator", Status: "enabled", Order: 10},
		{ID: "pos-operator", Code: "operator", Name: "Operator", Status: "enabled", Order: 20},
	}
}

func defaultDictionaries() []DictionaryType {
	return []DictionaryType{
		{
			Type:        "system.status",
			Name:        "System Status",
			Description: "Common enabled/disabled status values.",
			Status:      "enabled",
			Order:       10,
			Items: []DictionaryItem{
				{Label: "Enabled", Value: "enabled", Status: "enabled", Order: 10},
				{Label: "Disabled", Value: "disabled", Status: "enabled", Order: 20},
			},
		},
		{
			Type:        "plugin.level",
			Name:        "Plugin Level",
			Description: "Plugin level options used by manifest and admin UI.",
			Status:      "enabled",
			Order:       20,
			Items: []DictionaryItem{
				{Label: "System", Value: "system", Status: "enabled", Order: 10},
				{Label: "Application", Value: "app", Status: "enabled", Order: 20},
			},
		},
	}
}

func defaultSettingsSchema() SettingSchema {
	auditMin := 1.0
	auditMax := 3650.0
	return SettingSchema{
		Title:       "Common Settings",
		TitleZhCN:   "常用配置",
		TitleEnUS:   "Common Settings",
		Description: "System settings rendered by the shared SchemaForm.",
		Fields: []SettingSchemaField{
			{
				Key:       "audit.retention_days",
				LabelZhCN: "审计日志保留天数",
				LabelEnUS: "Audit retention days",
				Type:      "number",
				Default:   "30",
				Min:       &auditMin,
				Max:       &auditMax,
				Help:      "用于后续审计清理策略，当前保存为系统配置项。",
			},
			{
				Key:       "plugin.auto_enable",
				LabelZhCN: "安装后自动启用插件",
				LabelEnUS: "Auto-enable installed plugins",
				Type:      "boolean",
				Default:   "false",
			},
			{
				Key:       "plugin.dev_portal_enabled",
				LabelZhCN: "启用开发者门户",
				LabelEnUS: "Developer portal enabled",
				Type:      "boolean",
				Default:   "true",
			},
			{
				Key:         systemMenuSettingKey,
				LabelZhCN:   "菜单树 JSON",
				LabelEnUS:   "Menu tree JSON",
				Type:        "textarea",
				Default:     "",
				Placeholder: "[]",
			},
		},
	}
}

func (h *SystemHandler) appendAudit(r *http.Request, action, resource, resourceID string, detail map[string]any) {
	if h == nil || h.audit == nil || r == nil {
		return
	}
	actorID := ""
	if claims, ok := security.JWTClaimsFromContext(r.Context()); ok {
		actorID = strings.TrimSpace(claims.Subject)
	}
	if actorID == "" {
		actorID = "system"
	}
	_, _ = h.audit.Append(r.Context(), actorID, action, resource, strings.TrimSpace(resourceID), detail)
}
