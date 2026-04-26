package app

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/tinboxw/skoll/internal/module/apiregistry"
	"github.com/tinboxw/skoll/internal/module/rbac"
	"github.com/tinboxw/skoll/internal/module/role"
)

const HeaderAdminRoleID = "X-Admin-Role-ID"

func WithRoleAPIAuthorizer(next http.Handler, roleSvc *role.Service, rbacSvc *rbac.Service) http.Handler {
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

		next.ServeHTTP(w, r)
	})
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
