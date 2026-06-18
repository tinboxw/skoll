package menu

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	domainmenu "github.com/tinboxw/skoll/internal/domain/menu"
	apiv1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	menusvc "github.com/tinboxw/skoll/internal/service/menu"
)

type Handler struct {
	service menusvc.Service
}

func RegisterMenuRoutes(mux *http.ServeMux, service menusvc.Service) {
	if service == nil {
		return
	}
	h := &Handler{service: service}
	mux.HandleFunc("GET /v1/menus/tree", h.tree)
	mux.HandleFunc("PUT /v1/menus", h.save)
	mux.HandleFunc("POST /v1/menus/reorder", h.reorder)
	mux.HandleFunc("PATCH /v1/menus/visibility", h.visibility)
}

type menuRecord struct {
	Key                 string   `json:"key"`
	ParentKey           string   `json:"parentKey,omitempty"`
	Source              string   `json:"source"`
	Name                string   `json:"name"`
	Path                string   `json:"path"`
	Component           string   `json:"component,omitempty"`
	Icon                string   `json:"icon,omitempty"`
	Sort                int      `json:"sort"`
	Visible             bool     `json:"visible"`
	RequiredRoles       []string `json:"requiredRoles,omitempty"`
	RequiredPermissions []string `json:"requiredPermissions,omitempty"`
}

func (h *Handler) tree(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	var visible *bool
	if raw := strings.TrimSpace(query.Get("visible")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			apiv1.WriteError(w, http.StatusBadRequest, err)
			return
		}
		visible = &value
	}
	nodes, err := h.service.Tree(r.Context(), menusvc.TreeInput{
		Source:  strings.TrimSpace(query.Get("source")),
		Visible: visible,
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	if query.Get("roles") != "" || query.Get("permissions") != "" {
		nodes, err = h.service.Filter(r.Context(), menusvc.FilterInput{
			Nodes:       nodes,
			Roles:       splitCSV(query.Get("roles")),
			Permissions: splitCSV(query.Get("permissions")),
		})
		if err != nil {
			apiv1.WriteError(w, http.StatusForbidden, err)
			return
		}
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": menuRecords(nodes)})
}

func (h *Handler) save(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Items []menuRecord `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	nodes, err := menuRecordsToDomain(req.Items)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	merged, err := h.service.MergeNodes(r.Context(), menusvc.MergeNodesInput{Nodes: nodes})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": menuRecords(merged)})
}

func (h *Handler) reorder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ParentKey   string   `json:"parentKey"`
		OrderedKeys []string `json:"orderedKeys"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.service.Reorder(r.Context(), menusvc.ReorderInput{
		ParentKey:   req.ParentKey,
		OrderedKeys: req.OrderedKeys,
	}); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"parentKey": req.ParentKey, "orderedKeys": req.OrderedKeys})
}

func (h *Handler) visibility(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key     string `json:"key"`
		Visible bool   `json:"visible"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	key := strings.TrimSpace(strings.ToLower(req.Key))
	if key == "" {
		apiv1.WriteError(w, http.StatusBadRequest, fmt.Errorf("menu key is required"))
		return
	}
	nodes, err := h.service.Tree(r.Context(), menusvc.TreeInput{})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	for _, node := range nodes {
		if node.Key() != key {
			continue
		}
		node.Visible = req.Visible
		if _, err := h.service.MergeNodes(r.Context(), menusvc.MergeNodesInput{Nodes: []domainmenu.MenuNode{node}}); err != nil {
			apiv1.WriteError(w, http.StatusBadRequest, err)
			return
		}
		apiv1.WriteJSON(w, http.StatusOK, map[string]any{"key": key, "visible": req.Visible})
		return
	}
	apiv1.WriteMessage(w, http.StatusNotFound, "not_found", "menu node not found")
}

func menuRecords(nodes []domainmenu.MenuNode) []menuRecord {
	out := make([]menuRecord, 0, len(nodes))
	for _, node := range nodes {
		out = append(out, menuRecordFromDomain(node))
	}
	return out
}

func menuRecordFromDomain(node domainmenu.MenuNode) menuRecord {
	return menuRecord{
		Key:                 node.Key(),
		ParentKey:           node.ParentKey(),
		Source:              node.Source(),
		Name:                node.Name(),
		Path:                node.Path(),
		Component:           node.Component(),
		Icon:                node.Icon(),
		Sort:                node.Sort,
		Visible:             node.Visible,
		RequiredRoles:       append([]string(nil), node.RequiredRoles...),
		RequiredPermissions: append([]string(nil), node.RequiredPermissions...),
	}
}

func menuRecordsToDomain(records []menuRecord) ([]domainmenu.MenuNode, error) {
	out := make([]domainmenu.MenuNode, 0, len(records))
	for _, record := range records {
		node, err := domainmenu.NewNode(domainmenu.NodeIdentity{
			Key:       record.Key,
			ParentKey: record.ParentKey,
			Source:    record.Source,
		}, domainmenu.NodeView{
			Name:      record.Name,
			Path:      record.Path,
			Component: record.Component,
			Icon:      record.Icon,
		}, record.Sort)
		if err != nil {
			return nil, err
		}
		node.Visible = record.Visible
		node.RequiredRoles = append([]string(nil), record.RequiredRoles...)
		node.RequiredPermissions = append([]string(nil), record.RequiredPermissions...)
		out = append(out, node)
	}
	return out, nil
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
