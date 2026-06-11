package system

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	apiv1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	systemsvc "github.com/tinboxw/skoll/internal/service/system"
	"github.com/tinboxw/skoll/pkg/security"
)

const systemMenuSettingKey = "skoll.menu.tree"

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
		{ID: "audit", Label: "Audit", Path: "/skoll/audit", Icon: "audit", Order: 50, Visible: true, RequiredPermissions: []string{"audit.read"}},
		{ID: "plugins", Label: "Plugins", Path: "/skoll/plugin", Icon: "plugins", Order: 60, Visible: true, RequiredPermissions: []string{"plugin.read"}},
		{ID: "settings", Label: "Settings", Path: "/skoll/setting", Icon: "settings", Order: 70, Visible: true, RequiredPermissions: []string{"system.manage"}},
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
