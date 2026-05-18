package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	domainrbac "github.com/tinboxw/skoll/internal/domain/rbac"
	rbacsvc "github.com/tinboxw/skoll/internal/service/rbac"
	"github.com/tinboxw/skoll/pkg/config"
	apperrors "github.com/tinboxw/skoll/pkg/errors"
	"github.com/tinboxw/skoll/pkg/logging"
	"github.com/tinboxw/skoll/pkg/security"
)

type PermissionPolicy struct {
	PathPrefix string
	Resource   string
}

type permissionChecker interface {
	CheckPermission(ctx context.Context, in rbacsvc.CheckPermissionInput) (bool, error)
}

func buildMiddlewareChain(next http.Handler, logger logging.Logger, policy AuthPolicy, apiPrefix, jwtSecret string, checker permissionChecker) http.Handler {
	h := recoverMiddleware(logger, next)
	h = accessLogMiddleware(logger, h)
	h = authGuardMiddleware(policy, apiPrefix, jwtSecret, checker, h)
	return h
}

func authGuardMiddleware(policy AuthPolicy, apiPrefix, jwtSecret string, checker permissionChecker, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !policy.ShouldAuthenticate(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		token, err := parseBearerToken(r.Header.Get("Authorization"))
		if err != nil {
			apperrors.WriteHTTP(w, apperrors.New("unauthorized", err.Error(), nil))
			return
		}

		claims, err := security.ParseJWT(jwtSecret, token)
		if err != nil {
			apperrors.WriteHTTP(w, apperrors.New("unauthorized", "invalid access token", nil))
			return
		}
		r = r.WithContext(security.WithJWTClaimsContext(r.Context(), claims))

		if claims != nil && !isRoleBypass(claims.Role) {
			resource, action, guarded := requiredPermission(r.Method, r.URL.Path, apiPrefix)
			if guarded {
				if checker == nil {
					apperrors.WriteHTTP(w, apperrors.New("forbidden", "permission denied", nil))
					return
				}
				allowed, err := checker.CheckPermission(r.Context(), rbacsvc.CheckPermissionInput{
					SubjectType: domainrbac.SubjectUser,
					SubjectID:   claims.Subject,
					Resource:    resource,
					Action:      action,
				})
				if err != nil {
					apperrors.WriteHTTP(w, apperrors.New("forbidden", "permission check failed", nil))
					return
				}
				if !allowed {
					apperrors.WriteHTTP(w, apperrors.New("forbidden", "permission denied", nil))
					return
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

func isRoleBypass(role string) bool {
	return strings.EqualFold(strings.TrimSpace(role), "super_admin")
}

func requiredPermission(method, path, apiPrefix string) (resource, action string, guarded bool) {
	cleanMethod := strings.ToUpper(strings.TrimSpace(method))
	cleanPath := strings.TrimSpace(path)

	for _, policy := range defaultPermissionPolicies(config.NormalizeAPIPrefix(apiPrefix)) {
		if strings.HasPrefix(cleanPath, policy.PathPrefix) {
			return policy.Resource, mapAction(cleanMethod, cleanPath), true
		}
	}
	return "", "", false
}

func defaultPermissionPolicies(apiPrefix string) []PermissionPolicy {
	return []PermissionPolicy{
		{PathPrefix: apiPrefix + "/v1/users", Resource: "user"},
		{PathPrefix: apiPrefix + "/v1/roles", Resource: "role"},
		{PathPrefix: apiPrefix + "/v1/rbac", Resource: "permission"},
	}
}

func mapAction(method, path string) string {
	if strings.Contains(path, "/check") {
		return "check"
	}
	switch method {
	case http.MethodGet:
		return "read"
	case http.MethodPost:
		if strings.Contains(path, "/disable") || strings.Contains(path, "/enable") {
			return "update"
		}
		return "create"
	case http.MethodPut, http.MethodPatch:
		return "update"
	case http.MethodDelete:
		return "delete"
	default:
		return "read"
	}
}

func parseBearerToken(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", errors.New("missing authorization header")
	}
	parts := strings.SplitN(value, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("invalid authorization scheme")
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", errors.New("missing bearer token")
	}
	return token, nil
}

func accessLogMiddleware(logger logging.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("http_request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start).String())
	})
}

func recoverMiddleware(logger logging.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error("panic recovered", "panic", rec)
				apperrors.WriteHTTP(w, apperrors.New("internal_error", "internal server error", nil))
			}
		}()

		next.ServeHTTP(w, r)
	})
}
