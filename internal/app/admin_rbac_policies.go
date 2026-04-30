package app

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

func createRoleHandler(svc RoleService) http.HandlerFunc {
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

		respondJSON(w, http.StatusCreated, svc.Create(req.Name, req.Permissions))
	}
}

func listRolesHandler(svc RoleService) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, http.StatusOK, svc.List())
	}
}

func getRoleHandler(svc RoleService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		out, err := svc.Get(id)
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

func setRoleMenusHandler(roleSvc RoleService, menuSvc MenuService, rbacSvc RBACService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if _, err := roleSvc.Get(roleID); err != nil {
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
			if _, err := menuSvc.Get(menuID); err != nil {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("menu id %d not found", menuID)})
				return
			}
		}

		respondJSON(w, http.StatusOK, roleMenusResponse{RoleID: roleID, MenuIDs: rbacSvc.SetRoleMenus(roleID, req.MenuIDs)})
	}
}

func getRoleMenusHandler(roleSvc RoleService, rbacSvc RBACService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if _, err := roleSvc.Get(roleID); err != nil {
			if err == role.ErrRoleNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		respondJSON(w, http.StatusOK, roleMenusResponse{RoleID: roleID, MenuIDs: rbacSvc.GetRoleMenus(roleID)})
	}
}

func setRoleAPIsHandler(roleSvc RoleService, rbacSvc RBACService, apiSvc APIRegistryService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if _, err := roleSvc.Get(roleID); err != nil {
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
			if !apiSvc.Exists(n) {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("api not registered: %s", n)})
				return
			}
			normalized = append(normalized, n)
		}

		respondJSON(w, http.StatusOK, roleAPIsResponse{RoleID: roleID, APIs: rbacSvc.SetRoleAPIs(roleID, normalized)})
	}
}

func getRoleAPIsHandler(roleSvc RoleService, rbacSvc RBACService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if _, err := roleSvc.Get(roleID); err != nil {
			if err == role.ErrRoleNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		respondJSON(w, http.StatusOK, roleAPIsResponse{RoleID: roleID, APIs: rbacSvc.GetRoleAPIs(roleID)})
	}
}

func setRolePoliciesHandler(roleSvc RoleService, rbacSvc RBACService, apiSvc APIRegistryService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if _, err := roleSvc.Get(roleID); err != nil {
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
			if !apiSvc.Exists(n) {
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

		out := rbacSvc.SetRolePolicies(roleID, rules)
		respondJSON(w, http.StatusOK, rolePoliciesResponse{RoleID: roleID, Rules: toRolePolicyRuleItems(out)})
	}
}

func getRolePoliciesHandler(roleSvc RoleService, rbacSvc RBACService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if _, err := roleSvc.Get(roleID); err != nil {
			if err == role.ErrRoleNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		out := rbacSvc.GetRolePolicies(roleID)
		respondJSON(w, http.StatusOK, rolePoliciesResponse{RoleID: roleID, Rules: toRolePolicyRuleItems(out)})
	}
}

func createRolePolicySnapshotHandler(roleSvc RoleService, rbacSvc RBACService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if _, err := roleSvc.Get(roleID); err != nil {
			if err == role.ErrRoleNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		snapshot := rbacSvc.CreateRolePolicySnapshot(roleID)
		auditSvc.Append("rbac", "policy_snapshot_create", fmt.Sprintf("role:%d@%s", roleID, snapshot.Version))
		respondJSON(w, http.StatusCreated, rolePolicySnapshotResponse{RoleID: snapshot.RoleID, Version: snapshot.Version, Rules: toRolePolicyRuleItems(snapshot.Rules)})
	}
}

func listRolePolicySnapshotsHandler(roleSvc RoleService, rbacSvc RBACService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if _, err := roleSvc.Get(roleID); err != nil {
			if err == role.ErrRoleNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		raw := rbacSvc.ListRolePolicySnapshots(roleID)
		snapshots := make([]rolePolicySnapshotResponse, 0, len(raw))
		for _, item := range raw {
			snapshots = append(snapshots, rolePolicySnapshotResponse{RoleID: item.RoleID, Version: item.Version, Rules: toRolePolicyRuleItems(item.Rules)})
		}
		respondJSON(w, http.StatusOK, rolePolicySnapshotListResponse{RoleID: roleID, Snapshots: snapshots})
	}
}

func rollbackRolePoliciesHandler(roleSvc RoleService, rbacSvc RBACService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if _, err := roleSvc.Get(roleID); err != nil {
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

		rules, err := rbacSvc.RollbackRolePolicies(roleID, version)
		if err != nil {
			if err == rbac.ErrPolicySnapshotNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		auditSvc.Append("rbac", "policy_snapshot_rollback", fmt.Sprintf("role:%d@%s approver:%s", roleID, version, approver))
		respondJSON(w, http.StatusOK, rolePoliciesResponse{RoleID: roleID, Rules: toRolePolicyRuleItems(rules)})
	}
}

func diffRolePermissionsHandler(roleSvc RoleService, rbacSvc RBACService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if _, err := roleSvc.Get(roleID); err != nil {
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

		current := rbacSvc.PermissionBundle(roleID)
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

func checkRolePermissionsHandler(roleSvc RoleService, rbacSvc RBACService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if _, err := roleSvc.Get(roleID); err != nil {
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

		current := rbacSvc.PermissionBundle(roleID)
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

func exportRolePolicyPersistenceHandler(roleSvc RoleService, rbacSvc RBACService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if _, err := roleSvc.Get(roleID); err != nil {
			if err == role.ErrRoleNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		bundle := rbacSvc.PermissionBundle(roleID)
		rawSnapshots := rbacSvc.ListRolePolicySnapshots(roleID)
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

func importRolePolicyPersistenceHandler(roleSvc RoleService, menuSvc MenuService, rbacSvc RBACService, apiSvc APIRegistryService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if _, err := roleSvc.Get(roleID); err != nil {
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
			if _, err := menuSvc.Get(menuID); err != nil {
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
			if !apiSvc.Exists(n) {
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
			if !apiSvc.Exists(n) {
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
				if _, err := menuSvc.Get(item.MenuID); err != nil {
					respondJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("menu id %d not found", item.MenuID)})
					return
				}
			}
			contractItems = append(contractItems, rbac.RoutePermissionItem{MenuID: item.MenuID, Route: strings.TrimSpace(item.Route), Buttons: append([]string(nil), item.Buttons...)})
		}

		rbacSvc.SetRoleMenus(roleID, req.Bundle.MenuIDs)
		rbacSvc.SetRoleAPIs(roleID, normalizedAPIs)
		rbacSvc.SetRolePolicies(roleID, rules)
		rbacSvc.SetRoleDataScope(roleID, rbac.DataScope{
			TenantIDs:             append([]string(nil), req.Bundle.DataScope.TenantIDs...),
			RequireOwnerMatch:     req.Bundle.DataScope.RequireOwnerMatch,
			CrossTenantAdminAllow: append([]string(nil), req.Bundle.DataScope.CrossTenantAdminAllow...),
		})
		if req.Bundle.PermissionContract.Version != "" || len(contractItems) > 0 {
			rbacSvc.SetRoleRoutePermissions(roleID, req.Bundle.PermissionContract.Version, contractItems)
		}

		auditSvc.Append("rbac", "policy_persistence_import", fmt.Sprintf("role:%d operator:%s", roleID, operator))

		bundle := rbacSvc.PermissionBundle(roleID)
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

func setRoleDataScopeHandler(roleSvc RoleService, rbacSvc RBACService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if _, err := roleSvc.Get(roleID); err != nil {
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

		out := rbacSvc.SetRoleDataScope(roleID, rbac.DataScope{TenantIDs: tenantIDs, RequireOwnerMatch: req.RequireOwnerMatch, CrossTenantAdminAllow: crossTenantAllow})
		respondJSON(w, http.StatusOK, roleDataScopeResponse{RoleID: roleID, TenantIDs: out.TenantIDs, RequireOwnerMatch: out.RequireOwnerMatch, CrossTenantAdminAllow: out.CrossTenantAdminAllow})
	}
}

func getRoleDataScopeHandler(roleSvc RoleService, rbacSvc RBACService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if _, err := roleSvc.Get(roleID); err != nil {
			if err == role.ErrRoleNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		out := rbacSvc.GetRoleDataScope(roleID)
		respondJSON(w, http.StatusOK, roleDataScopeResponse{RoleID: roleID, TenantIDs: out.TenantIDs, RequireOwnerMatch: out.RequireOwnerMatch, CrossTenantAdminAllow: out.CrossTenantAdminAllow})
	}
}

func setRolePermissionContractHandler(roleSvc RoleService, rbacSvc RBACService, menuSvc MenuService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if _, err := roleSvc.Get(roleID); err != nil {
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
				if _, err := menuSvc.Get(item.MenuID); err != nil {
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

		out := rbacSvc.SetRoleRoutePermissions(roleID, req.Version, items)
		respondJSON(w, http.StatusOK, rolePermissionContractResponse{RoleID: roleID, Version: out.Version, Items: toRolePermissionContractItems(out.Items)})
	}
}

func getRolePermissionContractHandler(roleSvc RoleService, rbacSvc RBACService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if _, err := roleSvc.Get(roleID); err != nil {
			if err == role.ErrRoleNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		out := rbacSvc.GetRoleRoutePermissions(roleID)
		respondJSON(w, http.StatusOK, rolePermissionContractResponse{RoleID: roleID, Version: out.Version, Items: toRolePermissionContractItems(out.Items)})
	}
}

func checkRolePermissionContractConsistencyHandler(roleSvc RoleService, rbacSvc RBACService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if _, err := roleSvc.Get(roleID); err != nil {
			if err == role.ErrRoleNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		out := rbacSvc.CheckRoleRoutePermissionConsistency(roleID)
		respondJSON(w, http.StatusOK, rolePermissionContractConsistencyResponse{RoleID: roleID, Passed: out.Passed, Problems: out.Problems})
	}
}

func toRolePermissionContractItems(items []rbac.RoutePermissionItem) []rolePermissionContractItem {
	if len(items) == 0 {
		return nil
	}
	out := make([]rolePermissionContractItem, len(items))
	for i, item := range items {
		out[i] = rolePermissionContractItem{MenuID: item.MenuID, Route: item.Route, Buttons: append([]string(nil), item.Buttons...)}
	}
	return out
}

func toRolePolicyRuleItems(rules []rbac.PolicyRule) []rolePolicyRuleItem {
	if len(rules) == 0 {
		return nil
	}
	out := make([]rolePolicyRuleItem, len(rules))
	for i, rule := range rules {
		out[i] = rolePolicyRuleItem{
			API:                  rule.API,
			Effect:               rule.Effect,
			RequireVerified:      rule.RequireVerified,
			RequireClaimsVersion: rule.RequireClaimsVersion,
		}
	}
	return out
}

func toRolePolicyRules(items []rolePolicyRuleItem) []rbac.PolicyRule {
	if len(items) == 0 {
		return nil
	}
	out := make([]rbac.PolicyRule, len(items))
	for i, item := range items {
		out[i] = rbac.PolicyRule{
			API:                  strings.TrimSpace(item.API),
			Effect:               strings.TrimSpace(item.Effect),
			RequireVerified:      item.RequireVerified,
			RequireClaimsVersion: strings.TrimSpace(item.RequireClaimsVersion),
		}
	}
	return out
}

func toRoutePermissionItems(items []rolePermissionContractItem) []rbac.RoutePermissionItem {
	if len(items) == 0 {
		return nil
	}
	out := make([]rbac.RoutePermissionItem, len(items))
	for i, item := range items {
		out[i] = rbac.RoutePermissionItem{MenuID: item.MenuID, Route: strings.TrimSpace(item.Route), Buttons: append([]string(nil), item.Buttons...)}
	}
	return out
}

func toRolePermissionDiffResponse(roleID int64, diff rbac.PermissionDiff) rolePermissionDiffResponse {
	return rolePermissionDiffResponse{
		RoleID:           roleID,
		AddedMenus:       append([]int64(nil), diff.AddedMenus...),
		RemovedMenus:     append([]int64(nil), diff.RemovedMenus...),
		AddedAPIs:        append([]string(nil), diff.AddedAPIs...),
		RemovedAPIs:      append([]string(nil), diff.RemovedAPIs...),
		AddedRules:       toRolePolicyRuleItems(diff.AddedPolicies),
		RemovedRules:     toRolePolicyRuleItems(diff.RemovedPolicies),
		DataScopeChanged: diff.DataScopeChanged,
		RouteChanged:     diff.RouteChanged,
	}
}
