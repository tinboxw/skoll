package user

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	domainrbac "github.com/tinboxw/skoll/internal/domain/rbac"
	apiv1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	rbacsvc "github.com/tinboxw/skoll/internal/service/rbac"
	usersvc "github.com/tinboxw/skoll/internal/service/user"
	"github.com/tinboxw/skoll/pkg/security"
)

type UserHandler struct {
	service     usersvc.Service
	rbacService rbacsvc.Service
	audit       auditsvc.Service
}

type batchCreateUsersRequest struct {
	Items  []usersvc.CreateUserInput `json:"items"`
	Atomic bool                      `json:"atomic"`
}

func RegisterUserRoutes(mux *http.ServeMux, service usersvc.Service, rbacService rbacsvc.Service, auditSvc auditsvc.Service) {
	if service == nil {
		return
	}
	h := &UserHandler{service: service, rbacService: rbacService, audit: auditSvc}
	mux.HandleFunc("POST /v1/users", h.create)
	mux.HandleFunc("POST /v1/users/batch", h.createBatch)
	mux.HandleFunc("GET /v1/users", h.list)
	mux.HandleFunc("GET /v1/users/{id}", h.get)
	mux.HandleFunc("PUT /v1/users/{id}", h.update)
	mux.HandleFunc("POST /v1/users/{id}/roles", h.assignRole)
	mux.HandleFunc("PATCH /v1/users/{id}/email", h.updateEmail)
	mux.HandleFunc("DELETE /v1/users/{id}", h.delete)
	mux.HandleFunc("POST /v1/users/{id}/disable", h.disable)
}

func (h *UserHandler) create(w http.ResponseWriter, r *http.Request) {
	var req usersvc.CreateUserInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	entity, err := h.service.Create(r.Context(), req)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	h.appendAudit(r, actorIDFromRequest(r.Context(), ""), "create", "user", entity.ID.String(), map[string]any{"account": entity.Account})
	apiv1.WriteJSON(w, http.StatusCreated, entity)
}

func (h *UserHandler) createBatch(w http.ResponseWriter, r *http.Request) {
	var req batchCreateUsersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	if len(req.Items) == 0 {
		apiv1.WriteMessage(w, http.StatusBadRequest, "invalid_request", "items are required")
		return
	}

	results, err := h.service.CreateBatch(r.Context(), usersvc.BatchCreateInput{Items: req.Items, Atomic: req.Atomic})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	successCount := 0
	for _, item := range results {
		if item.Success {
			successCount++
		}
	}

	apiv1.WriteJSON(w, http.StatusOK, map[string]any{
		"successCount": successCount,
		"failureCount": len(req.Items) - successCount,
		"results":      results,
	})
	h.appendAudit(r, actorIDFromRequest(r.Context(), ""), "batch_create", "user", "", map[string]any{"count": len(req.Items), "successCount": successCount, "atomic": req.Atomic})
}

func (h *UserHandler) list(w http.ResponseWriter, r *http.Request) {
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.service.List(r.Context(), usersvc.ListInput{Offset: offset, Limit: limit})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, items)
}

func (h *UserHandler) get(w http.ResponseWriter, r *http.Request) {
	entity, err := h.service.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	if entity == nil {
		apiv1.WriteMessage(w, http.StatusNotFound, "not_found", "user not found")
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, entity)
}

func (h *UserHandler) updateEmail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email   string `json:"email"`
		ActorID string `json:"actorId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	before, _ := h.service.Get(r.Context(), r.PathValue("id"))
	actorID := actorIDFromRequest(r.Context(), req.ActorID)
	entity, err := h.service.UpdateEmail(r.Context(), usersvc.UpdateEmailInput{ID: r.PathValue("id"), Email: req.Email, ActorID: actorID})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	targetID := strings.TrimSpace(r.PathValue("id"))
	detail := map[string]any{"after": map[string]any{"email": req.Email}}
	if entity != nil {
		targetID = entity.ID.String()
		detail["after"] = map[string]any{"email": entity.Email}
	}
	if before != nil {
		detail["before"] = map[string]any{"email": before.Email}
	}
	h.appendAudit(r, actorID, "update_email", "user", targetID, detail)
	apiv1.WriteJSON(w, http.StatusOK, entity)
}

func (h *UserHandler) update(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name   string `json:"name"`
		Email  string `json:"email"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	before, _ := h.service.Get(r.Context(), r.PathValue("id"))
	entity, err := h.service.Update(r.Context(), usersvc.UpdateUserInput{
		ID:     r.PathValue("id"),
		Name:   req.Name,
		Email:  req.Email,
		Status: req.Status,
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	targetID := strings.TrimSpace(r.PathValue("id"))
	detail := map[string]any{"after": map[string]any{"name": req.Name, "email": req.Email, "status": req.Status}}
	if entity != nil {
		targetID = entity.ID.String()
		detail["after"] = map[string]any{"name": entity.Name, "email": entity.Email, "status": entity.Status}
	}
	if before != nil {
		detail["before"] = map[string]any{"name": before.Name, "email": before.Email, "status": before.Status}
	}
	h.appendAudit(r, actorIDFromRequest(r.Context(), ""), "update", "user", targetID, detail)
	apiv1.WriteJSON(w, http.StatusOK, entity)
}

func (h *UserHandler) delete(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if err := h.service.Delete(r.Context(), id); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	h.appendAudit(r, actorIDFromRequest(r.Context(), ""), "delete", "user", id, nil)
	apiv1.WriteMessage(w, http.StatusOK, "ok", "deleted")
}

func (h *UserHandler) disable(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ActorID string `json:"actorId"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	actorID := actorIDFromRequest(r.Context(), req.ActorID)
	id := strings.TrimSpace(r.PathValue("id"))
	if err := h.service.Disable(r.Context(), id, actorID); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	h.appendAudit(r, actorID, "disable", "user", id, nil)
	apiv1.WriteMessage(w, http.StatusOK, "ok", "disabled")
}

func (h *UserHandler) assignRole(w http.ResponseWriter, r *http.Request) {
	if h.rbacService == nil {
		apiv1.WriteError(w, http.StatusServiceUnavailable, errors.New("rbac service is not configured"))
		return
	}

	var req struct {
		RoleID string `json:"roleId"`
		Scope  string `json:"scope"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	roleID := strings.TrimSpace(req.RoleID)
	if roleID == "" {
		apiv1.WriteMessage(w, http.StatusBadRequest, "invalid_request", "roleId is required")
		return
	}
	scope := domainrbac.DataScope(strings.TrimSpace(req.Scope))
	if scope == "" {
		scope = domainrbac.DataScopeSelf
	}

	binding, err := h.rbacService.BindRole(r.Context(), rbacsvc.BindRoleInput{
		SubjectType: domainrbac.SubjectUser,
		SubjectID:   strings.TrimSpace(r.PathValue("id")),
		RoleID:      roleID,
		Scope:       scope,
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	h.appendAudit(r, actorIDFromRequest(r.Context(), ""), "assign_role", "user", strings.TrimSpace(r.PathValue("id")), map[string]any{"roleId": roleID, "scope": string(scope)})
	apiv1.WriteJSON(w, http.StatusOK, binding)
}

func (h *UserHandler) appendAudit(r *http.Request, actorID, action, resource, resourceID string, detail map[string]any) {
	if h == nil || h.audit == nil || r == nil {
		return
	}
	actor := strings.TrimSpace(actorID)
	if actor == "" {
		actor = "system"
	}
	_, _ = h.audit.Append(r.Context(), actor, action, resource, strings.TrimSpace(resourceID), detail)
}

func actorIDFromRequest(ctx context.Context, raw string) string {
	if actorID := strings.TrimSpace(raw); actorID != "" {
		return actorID
	}
	claims, ok := security.JWTClaimsFromContext(ctx)
	if !ok {
		return ""
	}
	return strings.TrimSpace(claims.Subject)
}
