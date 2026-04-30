package rbac

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/tinboxw/skoll/internal/module/apiregistry"
	"github.com/tinboxw/skoll/internal/module/rbac"
	"github.com/tinboxw/skoll/internal/module/role"
)

func (h *Handler) createRoleHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createRoleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
			return
		}

		respondJSON(w, http.StatusCreated, h.roles.Create(req.Name, req.Permissions))
	}
}

func (h *Handler) listRolesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, http.StatusOK, h.roles.List())
	}
}

func (h *Handler) getRoleHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		out, err := h.roles.Get(id)
		if err != nil {
			if err == role.ErrRoleNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		respondJSON(w, http.StatusOK, out)
	}
}

func (h *Handler) setRoleMenusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if _, err := h.roles.Get(roleID); err != nil {
			if err == role.ErrRoleNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		var req setRoleMenusRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}

		for _, menuID := range req.MenuIDs {
			if menuID <= 0 {
				continue
			}
			if _, err := h.menus.Get(menuID); err != nil {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("menu id %d not found", menuID)})
				return
			}
		}

		respondJSON(w, http.StatusOK, roleMenusResponse{RoleID: roleID, MenuIDs: h.rbac.SetRoleMenus(roleID, req.MenuIDs)})
	}
}

func (h *Handler) getRoleMenusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if _, err := h.roles.Get(roleID); err != nil {
			if err == role.ErrRoleNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		respondJSON(w, http.StatusOK, roleMenusResponse{RoleID: roleID, MenuIDs: h.rbac.GetRoleMenus(roleID)})
	}
}

func (h *Handler) setRoleAPIsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if _, err := h.roles.Get(roleID); err != nil {
			if err == role.ErrRoleNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		var req setRoleAPIsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}

		normalized := make([]string, 0, len(req.APIs))
		for _, item := range req.APIs {
			n := apiregistry.Normalize(item)
			if n == "" {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid api format: %q", item)})
				return
			}
			if !h.apis.Exists(n) {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("api not registered: %s", n)})
				return
			}
			normalized = append(normalized, n)
		}

		respondJSON(w, http.StatusOK, roleAPIsResponse{RoleID: roleID, APIs: h.rbac.SetRoleAPIs(roleID, normalized)})
	}
}

func (h *Handler) getRoleAPIsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if _, err := h.roles.Get(roleID); err != nil {
			if err == role.ErrRoleNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		respondJSON(w, http.StatusOK, roleAPIsResponse{RoleID: roleID, APIs: h.rbac.GetRoleAPIs(roleID)})
	}
}

func (h *Handler) setRolePoliciesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if _, err := h.roles.Get(roleID); err != nil {
			if err == role.ErrRoleNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		var req setRolePoliciesRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}

		rules := make([]rbac.PolicyRule, 0, len(req.Rules))
		for _, item := range req.Rules {
			n := apiregistry.Normalize(item.API)
			if n == "" {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid api format: %q", item.API)})
				return
			}
			if !h.apis.Exists(n) {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("api not registered: %s", n)})
				return
			}
			effect := strings.ToLower(strings.TrimSpace(item.Effect))
			if effect != "allow" && effect != "deny" {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid policy effect: %q", item.Effect)})
				return
			}
			rules = append(rules, rbac.PolicyRule{
				API:                  n,
				Effect:               effect,
				RequireVerified:      item.RequireVerified,
				RequireClaimsVersion: strings.ToLower(strings.TrimSpace(item.RequireClaimsVersion)),
			})
		}

		out := h.rbac.SetRolePolicies(roleID, rules)
		respondJSON(w, http.StatusOK, rolePoliciesResponse{RoleID: roleID, Rules: toRolePolicyRuleItems(out)})
	}
}

func (h *Handler) getRolePoliciesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if _, err := h.roles.Get(roleID); err != nil {
			if err == role.ErrRoleNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		out := h.rbac.GetRolePolicies(roleID)
		respondJSON(w, http.StatusOK, rolePoliciesResponse{RoleID: roleID, Rules: toRolePolicyRuleItems(out)})
	}
}
