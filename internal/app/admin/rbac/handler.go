package rbac

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/module/apiregistry"
	"github.com/tinboxw/skoll/internal/module/audit"
	"github.com/tinboxw/skoll/internal/module/menu"
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

type RoleService interface {
	Create(name string, permissions []string) role.Role
	Get(id int64) (role.Role, error)
	List() []role.Role
}

type MenuService interface {
	Get(id int64) (menu.Item, error)
}

type AuditService interface {
	Append(actor, action, target string) audit.Record
}

type RBACService interface {
	SetRoleMenus(roleID int64, menuIDs []int64) []int64
	GetRoleMenus(roleID int64) []int64
	SetRoleAPIs(roleID int64, apis []string) []string
	GetRoleAPIs(roleID int64) []string
	SetRolePolicies(roleID int64, rules []rbac.PolicyRule) []rbac.PolicyRule
	GetRolePolicies(roleID int64) []rbac.PolicyRule
	CreateRolePolicySnapshot(roleID int64) rbac.PolicySnapshot
	ListRolePolicySnapshots(roleID int64) []rbac.PolicySnapshot
	RollbackRolePolicies(roleID int64, version string) ([]rbac.PolicyRule, error)
	PermissionBundle(roleID int64) rbac.PermissionBundle
	SetRoleDataScope(roleID int64, scope rbac.DataScope) rbac.DataScope
	GetRoleDataScope(roleID int64) rbac.DataScope
	SetRoleRoutePermissions(roleID int64, version string, items []rbac.RoutePermissionItem) rbac.RoutePermissionContract
	GetRoleRoutePermissions(roleID int64) rbac.RoutePermissionContract
	CheckRoleRoutePermissionConsistency(roleID int64) rbac.RoutePermissionConsistency
}

type APIRegistryService interface {
	Exists(entry string) bool
}

type APIRegistry interface {
	RegisterMany(entries []string)
}

type Handler struct {
	roles RoleService
	menus MenuService
	rbac  RBACService
	apis  APIRegistryService
	audit AuditService
}

func NewHandler(roles RoleService, menus MenuService, rbacSvc RBACService, apiSvc APIRegistryService, auditSvc AuditService) *Handler {
	return &Handler{roles: roles, menus: menus, rbac: rbacSvc, apis: apiSvc, audit: auditSvc}
}

func (h *Handler) Register(mux *http.ServeMux, wrapper func(http.Handler) http.Handler, registry APIRegistry) {
	if mux == nil || h == nil || h.roles == nil || h.menus == nil || h.rbac == nil || h.apis == nil {
		return
	}
	handle := func(pattern string, next http.HandlerFunc) {
		if registry != nil {
			registry.RegisterMany([]string{pattern})
		}
		hd := http.Handler(next)
		if wrapper != nil {
			hd = wrapper(hd)
		}
		mux.Handle(pattern, hd)
	}

	handle("POST /admin/v1/roles", createRoleHandler(h.roles))
	handle("GET /admin/v1/roles", listRolesHandler(h.roles))
	handle("GET /admin/v1/roles/{id}", getRoleHandler(h.roles))
	handle("PUT /admin/v1/roles/{id}/menus", setRoleMenusHandler(h.roles, h.menus, h.rbac))
	handle("GET /admin/v1/roles/{id}/menus", getRoleMenusHandler(h.roles, h.rbac))
	handle("PUT /admin/v1/roles/{id}/apis", setRoleAPIsHandler(h.roles, h.rbac, h.apis))
	handle("GET /admin/v1/roles/{id}/apis", getRoleAPIsHandler(h.roles, h.rbac))
	handle("PUT /admin/v1/roles/{id}/policies", setRolePoliciesHandler(h.roles, h.rbac, h.apis))
	handle("GET /admin/v1/roles/{id}/policies", getRolePoliciesHandler(h.roles, h.rbac))
	handle("POST /admin/v1/roles/{id}/policies/snapshots", createRolePolicySnapshotHandler(h.roles, h.rbac, h.audit))
	handle("GET /admin/v1/roles/{id}/policies/snapshots", listRolePolicySnapshotsHandler(h.roles, h.rbac))
	handle("POST /admin/v1/roles/{id}/permissions/diff", diffRolePermissionsHandler(h.roles, h.rbac))
	handle("POST /admin/v1/roles/{id}/permissions/check", checkRolePermissionsHandler(h.roles, h.rbac))
	handle("GET /admin/v1/roles/{id}/policies/persistence/export", exportRolePolicyPersistenceHandler(h.roles, h.rbac))
	handle("POST /admin/v1/roles/{id}/policies/persistence/import", importRolePolicyPersistenceHandler(h.roles, h.menus, h.rbac, h.apis, h.audit))
	handle("POST /admin/v1/roles/{id}/policies/rollback", rollbackRolePoliciesHandler(h.roles, h.rbac, h.audit))
	handle("PUT /admin/v1/roles/{id}/data-scope", setRoleDataScopeHandler(h.roles, h.rbac))
	handle("GET /admin/v1/roles/{id}/data-scope", getRoleDataScopeHandler(h.roles, h.rbac))
	handle("PUT /admin/v1/roles/{id}/permission-contract", setRolePermissionContractHandler(h.roles, h.rbac, h.menus))
	handle("GET /admin/v1/roles/{id}/permission-contract", getRolePermissionContractHandler(h.roles, h.rbac))
	handle("POST /admin/v1/roles/{id}/permission-contract/consistency-check", checkRolePermissionContractConsistencyHandler(h.roles, h.rbac))
}

func parsePathInt64(r *http.Request, key string) (int64, error) {
	raw := strings.TrimSpace(r.PathValue(key))
	if raw == "" {
		return 0, fmt.Errorf("%s is required", key)
	}
	var id int64
	for _, ch := range raw {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("%s must be a positive integer", key)
		}
		id = id*10 + int64(ch-'0')
	}
	if id <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}
	return id, nil
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

type createRoleRequest struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

type setRoleMenusRequest struct {
	MenuIDs []int64 `json:"menu_ids"`
}

type roleMenusResponse struct {
	RoleID  int64   `json:"role_id"`
	MenuIDs []int64 `json:"menu_ids"`
}

type setRoleAPIsRequest struct {
	APIs []string `json:"apis"`
}

type roleAPIsResponse struct {
	RoleID int64    `json:"role_id"`
	APIs   []string `json:"apis"`
}

type rolePolicyRuleItem struct {
	API                  string `json:"api"`
	Effect               string `json:"effect"`
	RequireVerified      bool   `json:"require_verified"`
	RequireClaimsVersion string `json:"require_claims_version,omitempty"`
}

type setRolePoliciesRequest struct {
	Rules []rolePolicyRuleItem `json:"rules"`
}

type rolePoliciesResponse struct {
	RoleID int64                `json:"role_id"`
	Rules  []rolePolicyRuleItem `json:"rules"`
}

type rolePolicySnapshotResponse struct {
	RoleID  int64                `json:"role_id"`
	Version string               `json:"version"`
	Rules   []rolePolicyRuleItem `json:"rules"`
}

type rolePolicySnapshotListResponse struct {
	RoleID    int64                        `json:"role_id"`
	Snapshots []rolePolicySnapshotResponse `json:"snapshots"`
}

type rollbackRolePoliciesRequest struct {
	SnapshotVersion string `json:"snapshot_version"`
	Approver        string `json:"approver"`
}

type setRoleDataScopeRequest struct {
	TenantIDs             []string `json:"tenant_ids"`
	RequireOwnerMatch     bool     `json:"require_owner_match"`
	CrossTenantAdminAllow []string `json:"cross_tenant_admin_allow"`
}

type roleDataScopeResponse struct {
	RoleID                int64    `json:"role_id"`
	TenantIDs             []string `json:"tenant_ids"`
	RequireOwnerMatch     bool     `json:"require_owner_match"`
	CrossTenantAdminAllow []string `json:"cross_tenant_admin_allow"`
}

type rolePermissionContractItem struct {
	MenuID  int64    `json:"menu_id"`
	Route   string   `json:"route"`
	Buttons []string `json:"buttons"`
}

type setRolePermissionContractRequest struct {
	Version string                       `json:"version"`
	Items   []rolePermissionContractItem `json:"items"`
}

type rolePermissionContractResponse struct {
	RoleID  int64                        `json:"role_id"`
	Version string                       `json:"version"`
	Items   []rolePermissionContractItem `json:"items"`
}

type rolePermissionContractConsistencyResponse struct {
	RoleID   int64    `json:"role_id"`
	Passed   bool     `json:"passed"`
	Problems []string `json:"problems,omitempty"`
}

type rolePermissionDiffRequest struct {
	MenuIDs            []int64                          `json:"menu_ids"`
	APIs               []string                         `json:"apis"`
	Rules              []rolePolicyRuleItem             `json:"rules"`
	DataScope          setRoleDataScopeRequest          `json:"data_scope"`
	PermissionContract setRolePermissionContractRequest `json:"permission_contract"`
}

type rolePermissionDiffResponse struct {
	RoleID           int64                `json:"role_id"`
	AddedMenus       []int64              `json:"added_menus"`
	RemovedMenus     []int64              `json:"removed_menus"`
	AddedAPIs        []string             `json:"added_apis"`
	RemovedAPIs      []string             `json:"removed_apis"`
	AddedRules       []rolePolicyRuleItem `json:"added_rules"`
	RemovedRules     []rolePolicyRuleItem `json:"removed_rules"`
	DataScopeChanged bool                 `json:"data_scope_changed"`
	RouteChanged     bool                 `json:"route_changed"`
}

type rolePermissionCheckResponse struct {
	RoleID   int64                      `json:"role_id"`
	Pass     bool                       `json:"pass"`
	Blocking bool                       `json:"blocking"`
	Reasons  []string                   `json:"reasons"`
	Diff     rolePermissionDiffResponse `json:"diff"`
}

type rolePolicyPersistenceBundle struct {
	MenuIDs            []int64                          `json:"menu_ids"`
	APIs               []string                         `json:"apis"`
	Rules              []rolePolicyRuleItem             `json:"rules"`
	DataScope          setRoleDataScopeRequest          `json:"data_scope"`
	PermissionContract setRolePermissionContractRequest `json:"permission_contract"`
	Snapshots          []rolePolicySnapshotResponse     `json:"snapshots,omitempty"`
}

type rolePolicyPersistenceResponse struct {
	RoleID         int64                       `json:"role_id"`
	ExportedAtUnix int64                       `json:"exported_at_unix_sec"`
	Bundle         rolePolicyPersistenceBundle `json:"bundle"`
}

type importRolePolicyPersistenceRequest struct {
	Operator string                      `json:"operator"`
	Bundle   rolePolicyPersistenceBundle `json:"bundle"`
}
