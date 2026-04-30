package rbac

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/module/apiregistry"
	"github.com/tinboxw/skoll/internal/module/rbac"
	"github.com/tinboxw/skoll/internal/module/role"
)

func (h *Handler) createRolePolicySnapshotHandler() http.HandlerFunc {
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

		snapshot := h.rbac.CreateRolePolicySnapshot(roleID)
		h.audit.Append("rbac", "policy_snapshot_create", fmt.Sprintf("role:%d@%s", roleID, snapshot.Version))
		respondJSON(w, http.StatusCreated, rolePolicySnapshotResponse{RoleID: snapshot.RoleID, Version: snapshot.Version, Rules: toRolePolicyRuleItems(snapshot.Rules)})
	}
}

func (h *Handler) listRolePolicySnapshotsHandler() http.HandlerFunc {
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

		raw := h.rbac.ListRolePolicySnapshots(roleID)
		snapshots := make([]rolePolicySnapshotResponse, 0, len(raw))
		for _, item := range raw {
			snapshots = append(snapshots, rolePolicySnapshotResponse{RoleID: item.RoleID, Version: item.Version, Rules: toRolePolicyRuleItems(item.Rules)})
		}
		respondJSON(w, http.StatusOK, rolePolicySnapshotListResponse{RoleID: roleID, Snapshots: snapshots})
	}
}

func (h *Handler) rollbackRolePoliciesHandler() http.HandlerFunc {
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

		var req rollbackRolePoliciesRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		version := strings.ToLower(strings.TrimSpace(req.SnapshotVersion))
		approver := strings.TrimSpace(req.Approver)
		if version == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "snapshot_version is required"})
			return
		}
		if approver == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "approver is required"})
			return
		}

		rules, err := h.rbac.RollbackRolePolicies(roleID, version)
		if err != nil {
			if err == rbac.ErrPolicySnapshotNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		h.audit.Append("rbac", "policy_snapshot_rollback", fmt.Sprintf("role:%d@%s approver:%s", roleID, version, approver))
		respondJSON(w, http.StatusOK, rolePoliciesResponse{RoleID: roleID, Rules: toRolePolicyRuleItems(rules)})
	}
}

func (h *Handler) exportRolePolicyPersistenceHandler() http.HandlerFunc {
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

		bundle := h.rbac.PermissionBundle(roleID)
		rawSnapshots := h.rbac.ListRolePolicySnapshots(roleID)
		snapshots := make([]rolePolicySnapshotResponse, 0, len(rawSnapshots))
		for _, item := range rawSnapshots {
			snapshots = append(snapshots, rolePolicySnapshotResponse{RoleID: item.RoleID, Version: item.Version, Rules: toRolePolicyRuleItems(item.Rules)})
		}

		respondJSON(w, http.StatusOK, rolePolicyPersistenceResponse{
			RoleID:         roleID,
			ExportedAtUnix: time.Now().UTC().Unix(),
			Bundle: rolePolicyPersistenceBundle{
				MenuIDs: append([]int64(nil), bundle.MenuIDs...),
				APIs:    append([]string(nil), bundle.APIs...),
				Rules:   toRolePolicyRuleItems(bundle.Policies),
				DataScope: setRoleDataScopeRequest{
					TenantIDs:             append([]string(nil), bundle.DataScope.TenantIDs...),
					RequireOwnerMatch:     bundle.DataScope.RequireOwnerMatch,
					CrossTenantAdminAllow: append([]string(nil), bundle.DataScope.CrossTenantAdminAllow...),
				},
				PermissionContract: setRolePermissionContractRequest{Version: bundle.RoutePermission.Version, Items: toRolePermissionContractItems(bundle.RoutePermission.Items)},
				Snapshots:          snapshots,
			},
		})
	}
}

func (h *Handler) importRolePolicyPersistenceHandler() http.HandlerFunc {
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

		var req importRolePolicyPersistenceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		operator := strings.TrimSpace(req.Operator)
		if operator == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "operator is required"})
			return
		}

		for _, menuID := range req.Bundle.MenuIDs {
			if menuID <= 0 {
				continue
			}
			if _, err := h.menus.Get(menuID); err != nil {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("menu id %d not found", menuID)})
				return
			}
		}

		normalizedAPIs := make([]string, 0, len(req.Bundle.APIs))
		for _, item := range req.Bundle.APIs {
			n := apiregistry.Normalize(item)
			if n == "" {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid api format: %q", item)})
				return
			}
			if !h.apis.Exists(n) {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("api not registered: %s", n)})
				return
			}
			normalizedAPIs = append(normalizedAPIs, n)
		}

		rules := make([]rbac.PolicyRule, 0, len(req.Bundle.Rules))
		for _, item := range req.Bundle.Rules {
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

		contractItems := make([]rbac.RoutePermissionItem, 0, len(req.Bundle.PermissionContract.Items))
		for _, item := range req.Bundle.PermissionContract.Items {
			if item.MenuID > 0 {
				if _, err := h.menus.Get(item.MenuID); err != nil {
					respondJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("menu id %d not found", item.MenuID)})
					return
				}
			}
			contractItems = append(contractItems, rbac.RoutePermissionItem{MenuID: item.MenuID, Route: strings.TrimSpace(item.Route), Buttons: append([]string(nil), item.Buttons...)})
		}

		h.rbac.SetRoleMenus(roleID, req.Bundle.MenuIDs)
		h.rbac.SetRoleAPIs(roleID, normalizedAPIs)
		h.rbac.SetRolePolicies(roleID, rules)
		h.rbac.SetRoleDataScope(roleID, rbac.DataScope{
			TenantIDs:             append([]string(nil), req.Bundle.DataScope.TenantIDs...),
			RequireOwnerMatch:     req.Bundle.DataScope.RequireOwnerMatch,
			CrossTenantAdminAllow: append([]string(nil), req.Bundle.DataScope.CrossTenantAdminAllow...),
		})
		if req.Bundle.PermissionContract.Version != "" || len(contractItems) > 0 {
			h.rbac.SetRoleRoutePermissions(roleID, req.Bundle.PermissionContract.Version, contractItems)
		}

		h.audit.Append("rbac", "policy_persistence_import", fmt.Sprintf("role:%d operator:%s", roleID, operator))

		bundle := h.rbac.PermissionBundle(roleID)
		respondJSON(w, http.StatusOK, rolePolicyPersistenceResponse{
			RoleID:         roleID,
			ExportedAtUnix: time.Now().UTC().Unix(),
			Bundle: rolePolicyPersistenceBundle{
				MenuIDs: append([]int64(nil), bundle.MenuIDs...),
				APIs:    append([]string(nil), bundle.APIs...),
				Rules:   toRolePolicyRuleItems(bundle.Policies),
				DataScope: setRoleDataScopeRequest{
					TenantIDs:             append([]string(nil), bundle.DataScope.TenantIDs...),
					RequireOwnerMatch:     bundle.DataScope.RequireOwnerMatch,
					CrossTenantAdminAllow: append([]string(nil), bundle.DataScope.CrossTenantAdminAllow...),
				},
				PermissionContract: setRolePermissionContractRequest{Version: bundle.RoutePermission.Version, Items: toRolePermissionContractItems(bundle.RoutePermission.Items)},
			},
		})
	}
}
