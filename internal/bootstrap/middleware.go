package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	domainrbac "github.com/tinboxw/skoll/internal/domain/rbac"
	"github.com/tinboxw/skoll/internal/domain/shared"
	auditmw "github.com/tinboxw/skoll/internal/handler/middleware"
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

type RoutePermissionPolicy struct {
	Method      string
	PathPattern string
	Resource    string
	Action      string
}

type permissionChecker interface {
	CheckPermission(ctx context.Context, in rbacsvc.CheckPermissionInput) (bool, error)
}

func buildMiddlewareChain(next http.Handler, logger logging.Logger, policy AuthPolicy, apiPrefix, jwtSecret string, checker permissionChecker, auditSink auditmw.AuditEventSink) http.Handler {
	h := recoverMiddleware(logger, auditSink, next)
	h = accessLogMiddleware(logger, h)
	h = authGuardMiddleware(policy, apiPrefix, jwtSecret, checker, auditSink, h)
	return h
}

func authGuardMiddleware(policy AuthPolicy, apiPrefix, jwtSecret string, checker permissionChecker, auditSink auditmw.AuditEventSink, next http.Handler) http.Handler {
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
					appendPermissionDeniedAudit(r, auditSink, claims.Subject, claims.Role, resource, action, "permission_checker_not_configured")
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
					appendPermissionDeniedAudit(r, auditSink, claims.Subject, claims.Role, resource, action, "permission_check_failed")
					apperrors.WriteHTTP(w, apperrors.New("forbidden", "permission check failed", nil))
					return
				}
				if !allowed {
					appendPermissionDeniedAudit(r, auditSink, claims.Subject, claims.Role, resource, action, "permission_denied")
					apperrors.WriteHTTP(w, apperrors.New("forbidden", "permission denied", nil))
					return
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

func appendPermissionDeniedAudit(r *http.Request, sink auditmw.AuditEventSink, actorID, actorName, resource, action, reason string) {
	if r == nil || sink == nil {
		return
	}
	event, err := auditmw.NewPermissionDeniedAuditEvent(auditmw.PermissionDeniedAuditInput{
		ID:         shared.ID("audit-event-" + strconv.FormatInt(time.Now().UTC().UnixNano(), 10)),
		ActorID:    actorID,
		ActorName:  actorName,
		Resource:   resource,
		Action:     action,
		Reason:     reason,
		Request:    r,
		OccurredAt: time.Now().UTC(),
	})
	if err != nil {
		return
	}
	_ = sink.AppendEvent(r.Context(), event)
}

func isRoleBypass(role string) bool {
	return strings.EqualFold(strings.TrimSpace(role), "super_admin")
}

func requiredPermission(method, path, apiPrefix string) (resource, action string, guarded bool) {
	cleanMethod := strings.ToUpper(strings.TrimSpace(method))
	cleanPath := strings.TrimSpace(path)
	normalizedPrefix := config.NormalizeAPIPrefix(apiPrefix)

	for _, policy := range pharmaOACriticalPermissionPolicies(normalizedPrefix) {
		if cleanMethod == policy.Method && routePatternMatches(cleanPath, policy.PathPattern) {
			return policy.Resource, policy.Action, true
		}
	}

	for _, policy := range defaultPermissionPolicies(normalizedPrefix) {
		if strings.HasPrefix(cleanPath, policy.PathPrefix) {
			return policy.Resource, mapAction(cleanMethod, cleanPath), true
		}
	}
	return "", "", false
}

func pharmaOACriticalPermissionPolicies(apiPrefix string) []RoutePermissionPolicy {
	path := func(value string) string { return apiPrefix + value }
	return []RoutePermissionPolicy{
		{Method: http.MethodGet, PathPattern: path("/v1/pharma-oa/employees"), Resource: "pharma_oa.employee", Action: "read"},
		{Method: http.MethodPost, PathPattern: path("/v1/pharma-oa/employees"), Resource: "pharma_oa.employee", Action: "create"},
		{Method: http.MethodPut, PathPattern: path("/v1/pharma-oa/employees/{id}"), Resource: "pharma_oa.employee", Action: "update"},
		{Method: http.MethodPost, PathPattern: path("/v1/pharma-oa/employees/{id}/leave"), Resource: "pharma_oa.employee", Action: "leave"},
		{Method: http.MethodGet, PathPattern: path("/v1/pharma-oa/employees/qualification-reminders"), Resource: "pharma_oa.employee", Action: "reminder"},
		{Method: http.MethodGet, PathPattern: path("/v1/pharma-oa/customers"), Resource: "pharma_oa.customer", Action: "read"},
		{Method: http.MethodPost, PathPattern: path("/v1/pharma-oa/customers"), Resource: "pharma_oa.customer", Action: "create"},
		{Method: http.MethodGet, PathPattern: path("/v1/pharma-oa/customers/qualification-reminders"), Resource: "pharma_oa.customer", Action: "reminder"},
		{Method: http.MethodGet, PathPattern: path("/v1/pharma-oa/customers/{id}"), Resource: "pharma_oa.customer", Action: "read"},
		{Method: http.MethodPut, PathPattern: path("/v1/pharma-oa/customers/{id}"), Resource: "pharma_oa.customer", Action: "update"},
		{Method: http.MethodGet, PathPattern: path("/v1/pharma-oa/customers/{id}/sales-eligibility"), Resource: "pharma_oa.customer", Action: "sales"},
		{Method: http.MethodPost, PathPattern: path("/v1/pharma-oa/customers/{id}/disable"), Resource: "pharma_oa.customer", Action: "disable"},
		{Method: http.MethodPost, PathPattern: path("/v1/pharma-oa/purchase-requests/{id}/approve"), Resource: "pharma_oa.purchase", Action: "approve"},
		{Method: http.MethodPost, PathPattern: path("/v1/pharma-oa/purchase-requests/{id}/reject"), Resource: "pharma_oa.purchase", Action: "reject"},
		{Method: http.MethodGet, PathPattern: path("/v1/pharma-oa/purchase-inbounds"), Resource: "pharma_oa.inbound", Action: "read"},
		{Method: http.MethodPost, PathPattern: path("/v1/pharma-oa/purchase-inbounds"), Resource: "pharma_oa.inbound", Action: "create"},
		{Method: http.MethodGet, PathPattern: path("/v1/pharma-oa/purchase-inbounds/{id}"), Resource: "pharma_oa.inbound", Action: "read"},
		{Method: http.MethodGet, PathPattern: path("/v1/pharma-oa/sales-outbounds"), Resource: "pharma_oa.sales.outbound", Action: "read"},
		{Method: http.MethodPost, PathPattern: path("/v1/pharma-oa/sales-outbounds"), Resource: "pharma_oa.sales.outbound", Action: "create"},
		{Method: http.MethodGet, PathPattern: path("/v1/pharma-oa/sales-outbounds/{id}"), Resource: "pharma_oa.sales.outbound", Action: "read"},
		{Method: http.MethodGet, PathPattern: path("/v1/pharma-oa/stocktakes"), Resource: "pharma_oa.stocktake", Action: "read"},
		{Method: http.MethodPost, PathPattern: path("/v1/pharma-oa/stocktakes"), Resource: "pharma_oa.stocktake", Action: "create"},
		{Method: http.MethodGet, PathPattern: path("/v1/pharma-oa/stocktakes/{id}"), Resource: "pharma_oa.stocktake", Action: "read"},
		{Method: http.MethodPost, PathPattern: path("/v1/pharma-oa/stocktakes/{id}/approve"), Resource: "pharma_oa.stocktake", Action: "approve"},
		{Method: http.MethodPost, PathPattern: path("/v1/pharma-oa/stocktakes/{id}/reject"), Resource: "pharma_oa.stocktake", Action: "reject"},
		{Method: http.MethodGet, PathPattern: path("/v1/pharma-oa/transfers"), Resource: "pharma_oa.transfer", Action: "read"},
		{Method: http.MethodPost, PathPattern: path("/v1/pharma-oa/transfers"), Resource: "pharma_oa.transfer", Action: "create"},
		{Method: http.MethodGet, PathPattern: path("/v1/pharma-oa/transfers/{id}"), Resource: "pharma_oa.transfer", Action: "read"},
		{Method: http.MethodPost, PathPattern: path("/v1/pharma-oa/contracts/{id}/approve"), Resource: "pharma_oa.contract", Action: "approve"},
		{Method: http.MethodPost, PathPattern: path("/v1/pharma-oa/contracts/{id}/reject"), Resource: "pharma_oa.contract", Action: "reject"},
		{Method: http.MethodPost, PathPattern: path("/v1/pharma-oa/quality-complaints/{id}/resolve"), Resource: "pharma_oa.quality_complaint", Action: "resolve"},
		{Method: http.MethodPost, PathPattern: path("/v1/pharma-oa/quality-complaints/{id}/reject"), Resource: "pharma_oa.quality_complaint", Action: "reject"},
		{Method: http.MethodPost, PathPattern: path("/v1/plugins/pharma_oa/api/purchase-requests/approve"), Resource: "pharma_oa.purchase", Action: "approve"},
		{Method: http.MethodPost, PathPattern: path("/v1/plugins/pharma_oa/api/purchase-requests/reject"), Resource: "pharma_oa.purchase", Action: "reject"},
		{Method: http.MethodGet, PathPattern: path("/v1/plugins/pharma_oa/api/purchase-inbounds"), Resource: "pharma_oa.inbound", Action: "read"},
		{Method: http.MethodPost, PathPattern: path("/v1/plugins/pharma_oa/api/purchase-inbounds"), Resource: "pharma_oa.inbound", Action: "create"},
		{Method: http.MethodGet, PathPattern: path("/v1/plugins/pharma_oa/api/purchase-inbounds/detail"), Resource: "pharma_oa.inbound", Action: "read"},
		{Method: http.MethodGet, PathPattern: path("/v1/plugins/pharma_oa/api/sales-outbounds"), Resource: "pharma_oa.sales.outbound", Action: "read"},
		{Method: http.MethodPost, PathPattern: path("/v1/plugins/pharma_oa/api/sales-outbounds"), Resource: "pharma_oa.sales.outbound", Action: "create"},
		{Method: http.MethodGet, PathPattern: path("/v1/plugins/pharma_oa/api/sales-outbounds/detail"), Resource: "pharma_oa.sales.outbound", Action: "read"},
		{Method: http.MethodGet, PathPattern: path("/v1/plugins/pharma_oa/api/stocktakes"), Resource: "pharma_oa.stocktake", Action: "read"},
		{Method: http.MethodPost, PathPattern: path("/v1/plugins/pharma_oa/api/stocktakes"), Resource: "pharma_oa.stocktake", Action: "create"},
		{Method: http.MethodGet, PathPattern: path("/v1/plugins/pharma_oa/api/stocktakes/detail"), Resource: "pharma_oa.stocktake", Action: "read"},
		{Method: http.MethodPost, PathPattern: path("/v1/plugins/pharma_oa/api/stocktakes/approve"), Resource: "pharma_oa.stocktake", Action: "approve"},
		{Method: http.MethodPost, PathPattern: path("/v1/plugins/pharma_oa/api/stocktakes/reject"), Resource: "pharma_oa.stocktake", Action: "reject"},
		{Method: http.MethodGet, PathPattern: path("/v1/plugins/pharma_oa/api/transfers"), Resource: "pharma_oa.transfer", Action: "read"},
		{Method: http.MethodPost, PathPattern: path("/v1/plugins/pharma_oa/api/transfers"), Resource: "pharma_oa.transfer", Action: "create"},
		{Method: http.MethodGet, PathPattern: path("/v1/plugins/pharma_oa/api/transfers/detail"), Resource: "pharma_oa.transfer", Action: "read"},
		{Method: http.MethodPost, PathPattern: path("/v1/plugins/pharma_oa/api/contracts/approve"), Resource: "pharma_oa.contract", Action: "approve"},
		{Method: http.MethodPost, PathPattern: path("/v1/plugins/pharma_oa/api/contracts/reject"), Resource: "pharma_oa.contract", Action: "reject"},
		{Method: http.MethodPost, PathPattern: path("/v1/plugins/pharma_oa/api/quality-complaints/resolve"), Resource: "pharma_oa.quality_complaint", Action: "resolve"},
		{Method: http.MethodPost, PathPattern: path("/v1/plugins/pharma_oa/api/quality-complaints/reject"), Resource: "pharma_oa.quality_complaint", Action: "reject"},
	}
}

func routePatternMatches(path, pattern string) bool {
	pathSegments := strings.Split(strings.Trim(path, "/"), "/")
	patternSegments := strings.Split(strings.Trim(pattern, "/"), "/")
	if len(pathSegments) != len(patternSegments) {
		return false
	}
	for i := range pathSegments {
		if strings.HasPrefix(patternSegments[i], "{") && strings.HasSuffix(patternSegments[i], "}") {
			if pathSegments[i] == "" {
				return false
			}
			continue
		}
		if pathSegments[i] != patternSegments[i] {
			return false
		}
	}
	return true
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

func recoverMiddleware(logger logging.Logger, auditSink auditmw.AuditEventSink, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error("panic recovered", "panic", rec)
				appendErrorAudit(r, auditSink, rec)
				apperrors.WriteHTTP(w, apperrors.New("internal_error", "internal server error", nil))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func appendErrorAudit(r *http.Request, sink auditmw.AuditEventSink, rec any) {
	if r == nil || sink == nil {
		return
	}
	now := time.Now().UTC()
	event, err := auditmw.NewErrorAuditEvent(auditmw.ErrorAuditInput{
		ID:         shared.ID("audit-event-" + strconv.FormatInt(now.UnixNano(), 10)),
		LogID:      shared.ID("error-log-" + strconv.FormatInt(now.UnixNano(), 10)),
		Request:    r,
		StatusCode: http.StatusInternalServerError,
		ErrorCode:  "panic",
		Summary:    "panic recovered",
		Message:    fmt.Sprint(rec),
		Panic:      rec,
		OccurredAt: now,
	})
	if err != nil {
		return
	}
	_ = sink.AppendEvent(r.Context(), event)
}
