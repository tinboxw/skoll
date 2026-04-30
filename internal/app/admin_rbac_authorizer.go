package app

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/tinboxw/skoll/internal/module/apiregistry"
	"github.com/tinboxw/skoll/internal/module/rbac"
)

const HeaderAdminRoleID = "X-Admin-Role-ID"

const (
	HeaderAdminDataTenantID  = "X-Data-Tenant-ID"
	HeaderAdminResourceOwner = "X-Resource-Owner"
)

func WithRoleAPIAuthorizer(next http.Handler, roleSvc RoleService, rbacSvc RBACService) http.Handler {
	if next == nil || roleSvc == nil || rbacSvc == nil {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawRoleID := resolveAdminVerifiedClaims(r).RoleID
		if rawRoleID == "" {
			writeRBACError(w, http.StatusUnauthorized, "admin role required")
			return
		}

		roleID, err := strconv.ParseInt(rawRoleID, 10, 64)
		if err != nil || roleID <= 0 {
			writeRBACError(w, http.StatusUnauthorized, "invalid admin role id")
			return
		}

		if _, err := roleSvc.Get(roleID); err != nil {
			writeRBACError(w, http.StatusForbidden, "admin role not found")
			return
		}

		apiKey := apiregistry.Normalize(strings.TrimSpace(r.Pattern))
		if apiKey == "" {
			apiKey = apiregistry.Normalize(r.Method + " " + r.URL.Path)
		}
		if apiKey == "" {
			writeRBACError(w, http.StatusForbidden, "api resource not resolvable")
			return
		}

		if !containsString(rbacSvc.GetRoleAPIs(roleID), apiKey) {
			writeRBACError(w, http.StatusForbidden, "api permission denied")
			return
		}

		claims := resolveAdminVerifiedClaims(r)
		if !authorizeByPolicy(rbacSvc.GetRolePolicies(roleID), apiKey, claims) {
			writeRBACError(w, http.StatusForbidden, "policy permission denied")
			return
		}

		if !authorizeByDataScope(rbacSvc.GetRoleDataScope(roleID), claims, r) {
			writeRBACError(w, http.StatusForbidden, "data scope denied")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func authorizeByPolicy(rules []rbac.PolicyRule, apiKey string, claims adminVerifiedClaims) bool {
	if len(rules) == 0 {
		return true
	}

	matched := false
	allowed := false
	for _, rule := range rules {
		if rule.API != apiKey {
			continue
		}
		matched = true
		if rule.Effect == "deny" {
			return false
		}
		if rule.RequireVerified && !claims.Verified {
			continue
		}
		if rule.RequireClaimsVersion != "" && claims.ClaimsVersion != rule.RequireClaimsVersion {
			continue
		}
		allowed = true
	}

	if !matched {
		return true
	}
	return allowed
}

func authorizeByDataScope(scope rbac.DataScope, claims adminVerifiedClaims, r *http.Request) bool {
	if len(scope.TenantIDs) > 0 {
		tenantID := strings.TrimSpace(r.Header.Get(HeaderAdminDataTenantID))
		if tenantID == "" || !containsString(scope.TenantIDs, tenantID) {
			if claims.Subject == "" || !containsString(scope.CrossTenantAdminAllow, claims.Subject) {
				return false
			}
		}
	}

	if scope.RequireOwnerMatch {
		owner := strings.TrimSpace(r.Header.Get(HeaderAdminResourceOwner))
		if owner == "" || claims.Subject == "" || owner != claims.Subject {
			return false
		}
	}

	return true
}

func writeRBACError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
