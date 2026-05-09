package v1

import (
	"encoding/json"
	"net/http"

	domainrbac "github.com/tinboxw/skoll/internal/domain/rbac"
	rbacsvc "github.com/tinboxw/skoll/internal/service/rbac"
)

type RBACHandler struct {
	service rbacsvc.Service
}

func RegisterRBACRoutes(mux *http.ServeMux, service rbacsvc.Service) {
	if service == nil {
		return
	}
	h := &RBACHandler{service: service}
	mux.HandleFunc("POST /v1/rbac/bind", h.bind)
	mux.HandleFunc("PUT /v1/rbac/policies/{roleId}", h.setPolicies)
	mux.HandleFunc("POST /v1/rbac/check", h.check)
}

func (h *RBACHandler) bind(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SubjectType string `json:"subjectType"`
		SubjectID   string `json:"subjectId"`
		RoleID      string `json:"roleId"`
		Scope       string `json:"scope"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	entity, err := h.service.BindRole(r.Context(), rbacsvc.BindRoleInput{
		SubjectType: domainrbac.SubjectType(req.SubjectType),
		SubjectID:   req.SubjectID,
		RoleID:      req.RoleID,
		Scope:       domainrbac.DataScope(req.Scope),
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, entity)
}

func (h *RBACHandler) setPolicies(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Rules []domainrbac.PolicyRule `json:"rules"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.service.SetRolePolicies(r.Context(), rbacsvc.SetRolePoliciesInput{RoleID: r.PathValue("roleId"), Rules: req.Rules}); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeMessage(w, http.StatusOK, "ok", "updated")
}

func (h *RBACHandler) check(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SubjectType string `json:"subjectType"`
		SubjectID   string `json:"subjectId"`
		Resource    string `json:"resource"`
		Action      string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	ok, err := h.service.CheckPermission(r.Context(), rbacsvc.CheckPermissionInput{
		SubjectType: domainrbac.SubjectType(req.SubjectType),
		SubjectID:   req.SubjectID,
		Resource:    req.Resource,
		Action:      req.Action,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"allowed": ok})
}
