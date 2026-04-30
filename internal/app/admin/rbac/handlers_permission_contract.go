package rbac

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/tinboxw/skoll/internal/module/rbac"
	"github.com/tinboxw/skoll/internal/module/role"
)

func (h *Handler) diffRolePermissionsHandler() http.HandlerFunc {
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

		var req rolePermissionDiffRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}

		current := h.rbac.PermissionBundle(roleID)
		target := rbac.PermissionBundle{
			MenuIDs:  req.MenuIDs,
			APIs:     req.APIs,
			Policies: toRolePolicyRules(req.Rules),
			DataScope: rbac.DataScope{
				TenantIDs:             req.DataScope.TenantIDs,
				RequireOwnerMatch:     req.DataScope.RequireOwnerMatch,
				CrossTenantAdminAllow: req.DataScope.CrossTenantAdminAllow,
			},
			RoutePermission: rbac.RoutePermissionContract{Version: req.PermissionContract.Version, Items: toRoutePermissionItems(req.PermissionContract.Items)},
		}
		diff := rbac.BuildPermissionDiff(current, target)
		respondJSON(w, http.StatusOK, toRolePermissionDiffResponse(roleID, diff))
	}
}

func (h *Handler) checkRolePermissionsHandler() http.HandlerFunc {
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

		var req rolePermissionDiffRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}

		current := h.rbac.PermissionBundle(roleID)
		target := rbac.PermissionBundle{
			MenuIDs:  req.MenuIDs,
			APIs:     req.APIs,
			Policies: toRolePolicyRules(req.Rules),
			DataScope: rbac.DataScope{
				TenantIDs:             req.DataScope.TenantIDs,
				RequireOwnerMatch:     req.DataScope.RequireOwnerMatch,
				CrossTenantAdminAllow: req.DataScope.CrossTenantAdminAllow,
			},
			RoutePermission: rbac.RoutePermissionContract{Version: req.PermissionContract.Version, Items: toRoutePermissionItems(req.PermissionContract.Items)},
		}
		diff := rbac.BuildPermissionDiff(current, target)
		check := rbac.EvaluatePermissionCheck(diff)
		respondJSON(w, http.StatusOK, rolePermissionCheckResponse{
			RoleID:   roleID,
			Pass:     check.Pass,
			Blocking: check.Blocking,
			Reasons:  check.Reasons,
			Diff:     toRolePermissionDiffResponse(roleID, check.Diff),
		})
	}
}

func (h *Handler) setRoleDataScopeHandler() http.HandlerFunc {
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

		var req setRoleDataScopeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}

		tenantIDs := make([]string, 0, len(req.TenantIDs))
		for _, tenantID := range req.TenantIDs {
			tenantID = strings.TrimSpace(tenantID)
			if tenantID == "" {
				continue
			}
			tenantIDs = append(tenantIDs, tenantID)
		}

		crossTenantAllow := make([]string, 0, len(req.CrossTenantAdminAllow))
		for _, subject := range req.CrossTenantAdminAllow {
			subject = strings.TrimSpace(subject)
			if subject == "" {
				continue
			}
			crossTenantAllow = append(crossTenantAllow, subject)
		}

		out := h.rbac.SetRoleDataScope(roleID, rbac.DataScope{TenantIDs: tenantIDs, RequireOwnerMatch: req.RequireOwnerMatch, CrossTenantAdminAllow: crossTenantAllow})
		respondJSON(w, http.StatusOK, roleDataScopeResponse{RoleID: roleID, TenantIDs: out.TenantIDs, RequireOwnerMatch: out.RequireOwnerMatch, CrossTenantAdminAllow: out.CrossTenantAdminAllow})
	}
}

func (h *Handler) getRoleDataScopeHandler() http.HandlerFunc {
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

		out := h.rbac.GetRoleDataScope(roleID)
		respondJSON(w, http.StatusOK, roleDataScopeResponse{RoleID: roleID, TenantIDs: out.TenantIDs, RequireOwnerMatch: out.RequireOwnerMatch, CrossTenantAdminAllow: out.CrossTenantAdminAllow})
	}
}

func (h *Handler) setRolePermissionContractHandler() http.HandlerFunc {
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

		var req setRolePermissionContractRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}

		items := make([]rbac.RoutePermissionItem, 0, len(req.Items))
		for _, item := range req.Items {
			if item.MenuID > 0 {
				if _, err := h.menus.Get(item.MenuID); err != nil {
					respondJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("menu id %d not found", item.MenuID)})
					return
				}
			}
			buttons := make([]string, 0, len(item.Buttons))
			for _, button := range item.Buttons {
				button = strings.TrimSpace(button)
				if button == "" {
					continue
				}
				buttons = append(buttons, button)
			}
			items = append(items, rbac.RoutePermissionItem{MenuID: item.MenuID, Route: strings.TrimSpace(item.Route), Buttons: buttons})
		}

		out := h.rbac.SetRoleRoutePermissions(roleID, req.Version, items)
		respondJSON(w, http.StatusOK, rolePermissionContractResponse{RoleID: roleID, Version: out.Version, Items: toRolePermissionContractItems(out.Items)})
	}
}

func (h *Handler) getRolePermissionContractHandler() http.HandlerFunc {
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

		out := h.rbac.GetRoleRoutePermissions(roleID)
		respondJSON(w, http.StatusOK, rolePermissionContractResponse{RoleID: roleID, Version: out.Version, Items: toRolePermissionContractItems(out.Items)})
	}
}

func (h *Handler) checkRolePermissionContractConsistencyHandler() http.HandlerFunc {
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

		out := h.rbac.CheckRoleRoutePermissionConsistency(roleID)
		respondJSON(w, http.StatusOK, rolePermissionContractConsistencyResponse{RoleID: roleID, Passed: out.Passed, Problems: out.Problems})
	}
}
