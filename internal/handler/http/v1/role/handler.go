package role

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	apiv1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	rbacsvc "github.com/tinboxw/skoll/internal/service/rbac"
	rolesvc "github.com/tinboxw/skoll/internal/service/role"
	usersvc "github.com/tinboxw/skoll/internal/service/user"
	"github.com/tinboxw/skoll/pkg/security"
)

type RoleHandler struct {
	service     rolesvc.Service
	userService usersvc.Service
	rbacService rbacsvc.Service
	audit       auditsvc.Service
}

func RegisterRoleRoutes(mux *http.ServeMux, service rolesvc.Service, userService usersvc.Service, rbacService rbacsvc.Service, auditSvc auditsvc.Service) {
	if service == nil {
		return
	}
	h := &RoleHandler{service: service, userService: userService, rbacService: rbacService, audit: auditSvc}
	mux.HandleFunc("POST /v1/roles", h.create)
	mux.HandleFunc("GET /v1/roles", h.list)
	mux.HandleFunc("GET /v1/roles/{id}", h.get)
	mux.HandleFunc("GET /v1/roles/{id}/users", h.usersByRole)
	mux.HandleFunc("PUT /v1/roles/{id}", h.update)
	mux.HandleFunc("DELETE /v1/roles/{id}", h.delete)
	mux.HandleFunc("POST /v1/roles/{id}/grant", h.grant)
	mux.HandleFunc("POST /v1/roles/{id}/revoke", h.revoke)
}

func (h *RoleHandler) create(w http.ResponseWriter, r *http.Request) {
	var req rolesvc.CreateRoleInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	entity, err := h.service.Create(r.Context(), req)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	h.appendAudit(r, "create", "role", entity.ID.String(), map[string]any{"key": entity.Key})
	apiv1.WriteJSON(w, http.StatusCreated, entity)
}

func (h *RoleHandler) list(w http.ResponseWriter, r *http.Request) {
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.service.List(r.Context(), rolesvc.ListInput{Offset: offset, Limit: limit})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, items)
}

func (h *RoleHandler) get(w http.ResponseWriter, r *http.Request) {
	entity, err := h.service.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	if entity == nil {
		apiv1.WriteMessage(w, http.StatusNotFound, "not_found", "role not found")
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, entity)
}

func (h *RoleHandler) usersByRole(w http.ResponseWriter, r *http.Request) {
	if h.userService == nil || h.rbacService == nil {
		apiv1.WriteError(w, http.StatusServiceUnavailable, errors.New("role users dependencies are not configured"))
		return
	}

	roleID := strings.TrimSpace(r.PathValue("id"))
	if roleID == "" {
		apiv1.WriteMessage(w, http.StatusBadRequest, "invalid_request", "role id is required")
		return
	}

	entity, err := h.service.Get(r.Context(), roleID)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	if entity == nil {
		apiv1.WriteMessage(w, http.StatusNotFound, "not_found", "role not found")
		return
	}

	const batchSize = 200
	offset := 0
	users := make([]any, 0)
	for {
		items, err := h.userService.List(r.Context(), usersvc.ListInput{Offset: offset, Limit: batchSize})
		if err != nil {
			apiv1.WriteError(w, http.StatusBadRequest, err)
			return
		}
		if len(items) == 0 {
			break
		}
		for _, u := range items {
			if u == nil {
				continue
			}
			bindings, err := h.rbacService.ListBindingsByUser(r.Context(), u.ID.String())
			if err != nil {
				apiv1.WriteError(w, http.StatusBadRequest, err)
				return
			}
			for _, b := range bindings {
				if b != nil && b.RoleID.String() == roleID {
					users = append(users, u)
					break
				}
			}
		}
		if len(items) < batchSize {
			break
		}
		offset += batchSize
	}

	apiv1.WriteJSON(w, http.StatusOK, users)
}

func (h *RoleHandler) update(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string   `json:"name"`
		Key         string   `json:"key"`
		Description string   `json:"description"`
		Permissions []string `json:"permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	before, _ := h.service.Get(r.Context(), r.PathValue("id"))
	entity, err := h.service.Update(r.Context(), rolesvc.UpdateRoleInput{
		ID:          r.PathValue("id"),
		Name:        req.Name,
		Key:         req.Key,
		Description: req.Description,
		Permissions: req.Permissions,
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	detail := map[string]any{"after": map[string]any{"name": entity.Name, "key": entity.Key, "permissions": len(entity.Permissions)}}
	if before != nil {
		detail["before"] = map[string]any{"name": before.Name, "key": before.Key, "permissions": len(before.Permissions)}
	}
	h.appendAudit(r, "update", "role", entity.ID.String(), detail)
	apiv1.WriteJSON(w, http.StatusOK, entity)
}

func (h *RoleHandler) delete(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if err := h.service.Delete(r.Context(), id); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	h.appendAudit(r, "delete", "role", id, nil)
	apiv1.WriteMessage(w, http.StatusOK, "ok", "deleted")
}

func (h *RoleHandler) grant(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Permission string `json:"permission"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	entity, err := h.service.Grant(r.Context(), r.PathValue("id"), req.Permission)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	h.appendAudit(r, "grant", "role", entity.ID.String(), map[string]any{"permission": req.Permission})
	apiv1.WriteJSON(w, http.StatusOK, entity)
}

func (h *RoleHandler) revoke(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Permission string `json:"permission"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	entity, err := h.service.Revoke(r.Context(), r.PathValue("id"), req.Permission)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	h.appendAudit(r, "revoke", "role", entity.ID.String(), map[string]any{"permission": req.Permission})
	apiv1.WriteJSON(w, http.StatusOK, entity)
}

func (h *RoleHandler) appendAudit(r *http.Request, action, resource, resourceID string, detail map[string]any) {
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
