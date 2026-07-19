package rbac

import (
	"encoding/json"
	"net/http"
	"strings"

	domainrbac "github.com/tinboxw/skoll/internal/domain/rbac"
	apiv1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	rbacsvc "github.com/tinboxw/skoll/internal/service/rbac"
	"github.com/tinboxw/skoll/pkg/security"
)

type RBACHandler struct {
	service rbacsvc.Service
	audit   auditsvc.Service
}

func RegisterRBACRoutes(mux *http.ServeMux, service rbacsvc.Service, auditSvc auditsvc.Service) {
	if service == nil {
		return
	}
	h := &RBACHandler{service: service, audit: auditSvc}
	mux.HandleFunc("POST /v1/rbac/bind", h.bind)
	mux.HandleFunc("GET /v1/rbac/bindings", h.listBindings)
	mux.HandleFunc("DELETE /v1/rbac/bindings/{id}", h.unbind)
	mux.HandleFunc("PUT /v1/rbac/policies/{roleId}", h.setPolicies)
	mux.HandleFunc("POST /v1/rbac/check", h.check)
	mux.HandleFunc("GET /v1/rbac/data-scope", h.dataScope)
}

func (h *RBACHandler) bind(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SubjectType string `json:"subjectType"`
		SubjectID   string `json:"subjectId"`
		RoleID      string `json:"roleId"`
		Scope       string `json:"scope"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	entity, err := h.service.BindRole(r.Context(), rbacsvc.BindRoleInput{
		SubjectType: domainrbac.SubjectType(req.SubjectType),
		SubjectID:   req.SubjectID,
		RoleID:      req.RoleID,
		Scope:       domainrbac.DataScope(req.Scope),
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	h.appendAudit(r, "bind", "rbac", req.RoleID, map[string]any{"subjectType": req.SubjectType, "subjectId": req.SubjectID, "scope": req.Scope})
	apiv1.WriteJSON(w, http.StatusCreated, entity)
}

func (h *RBACHandler) setPolicies(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Rules []domainrbac.PolicyRule `json:"rules"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.service.SetRolePolicies(r.Context(), rbacsvc.SetRolePoliciesInput{RoleID: r.PathValue("roleId"), Rules: req.Rules}); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	h.appendAudit(r, "set_policies", "rbac", r.PathValue("roleId"), map[string]any{"ruleCount": len(req.Rules)})
	apiv1.WriteMessage(w, http.StatusOK, "ok", "updated")
}

func (h *RBACHandler) check(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SubjectType string `json:"subjectType"`
		SubjectID   string `json:"subjectId"`
		Resource    string `json:"resource"`
		Action      string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	decision, err := h.service.ResolvePermission(r.Context(), rbacsvc.CheckPermissionInput{
		SubjectType: domainrbac.SubjectType(req.SubjectType),
		SubjectID:   req.SubjectID,
		Resource:    req.Resource,
		Action:      req.Action,
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, decision)
}

func (h *RBACHandler) dataScope(w http.ResponseWriter, r *http.Request) {
	resource := strings.TrimSpace(r.URL.Query().Get("resource"))
	action := strings.TrimSpace(r.URL.Query().Get("action"))
	if resource == "" || action == "" {
		apiv1.WriteMessage(w, http.StatusBadRequest, "invalid_request", "resource and action are required")
		return
	}
	decision, err := h.service.ResolveDataScope(r.Context(), rbacsvc.ResolveDataScopeInput{
		Resource: resource,
		Action:   action,
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusForbidden, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, decision)
}

func (h *RBACHandler) listBindings(w http.ResponseWriter, r *http.Request) {
	subjectType := domainrbac.SubjectType(strings.TrimSpace(r.URL.Query().Get("subjectType")))
	subjectID := strings.TrimSpace(r.URL.Query().Get("subjectId"))
	items, err := h.service.ListBindings(r.Context(), subjectType, subjectID)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, items)
}

func (h *RBACHandler) unbind(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		apiv1.WriteMessage(w, http.StatusBadRequest, "invalid_request", "binding id is required")
		return
	}
	if err := h.service.UnbindBinding(r.Context(), id); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	h.appendAudit(r, "unbind", "rbac", id, nil)
	apiv1.WriteMessage(w, http.StatusOK, "ok", "unbound")
}

func (h *RBACHandler) appendAudit(r *http.Request, action, resource, resourceID string, detail map[string]any) {
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
