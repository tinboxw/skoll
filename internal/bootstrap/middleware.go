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
	"github.com/tinboxw/skoll/internal/plugin"
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

func buildMiddlewareChain(next http.Handler, logger logging.Logger, policy AuthPolicy, apiPrefix, jwtSecret string, checker permissionChecker, routeResolver plugin.RoutePermissionResolver, auditSink auditmw.AuditEventSink) http.Handler {
	h := recoverMiddleware(logger, auditSink, next)
	h = accessLogMiddleware(logger, h)
	h = authGuardMiddleware(policy, apiPrefix, jwtSecret, checker, routeResolver, auditSink, h)
	return h
}

func authGuardMiddleware(policy AuthPolicy, apiPrefix, jwtSecret string, checker permissionChecker, routeResolver plugin.RoutePermissionResolver, auditSink auditmw.AuditEventSink, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pluginPath, pluginBusinessRoute := pluginBusinessRoutePath(r.URL.Path, apiPrefix)
		if !pluginBusinessRoute && !policy.ShouldAuthenticate(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		token, err := parseBearerToken(r.Header.Get("Authorization"))
		if err != nil {
			apperrors.WriteHTTP(w, apperrors.New("unauthorized", err.Error(), nil))
			return
		}

		claims, err := security.ParseJWT(jwtSecret, token)
		if err != nil || claims == nil {
			apperrors.WriteHTTP(w, apperrors.New("unauthorized", "invalid access token", nil))
			return
		}
		r = r.WithContext(security.WithJWTClaimsContext(r.Context(), claims))

		resource, action, guarded := requiredPermission(r.Method, r.URL.Path, apiPrefix)
		var routeDescriptor plugin.RoutePermissionDescriptor
		if pluginBusinessRoute {
			var reason string
			routeDescriptor, resource, action, reason = resolvedPluginRoutePermission(r.Method, pluginPath, routeResolver)
			if reason != "" {
				appendPermissionDeniedAudit(r, auditSink, claims.Subject, claims.Role, resource, action, reason)
				writePluginPermissionDenied(w, r, reason)
				return
			}
			guarded = true
		}

		if !claims.HasRole("super_admin") && guarded {
			if checker == nil {
				appendPermissionDeniedAudit(r, auditSink, claims.Subject, claims.Role, resource, action, "permission_checker_not_configured")
				writePermissionDenied(w, r, pluginBusinessRoute, "permission denied")
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
				writePermissionDenied(w, r, pluginBusinessRoute, "permission check failed")
				return
			}
			if !allowed {
				appendPermissionDeniedAudit(r, auditSink, claims.Subject, claims.Role, resource, action, "permission_denied")
				writePermissionDenied(w, r, pluginBusinessRoute, "permission denied")
				return
			}
		}

		if pluginBusinessRoute && routeDescriptor.AuditAction != "" {
			recorder := &pluginRouteAuditResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(recorder, r)
			appendPluginRouteAudit(r, auditSink, claims.Subject, claims.Role, routeDescriptor, recorder.statusCode)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func pluginBusinessRoutePath(path, apiPrefix string) (string, bool) {
	cleanPath := strings.TrimSpace(path)
	prefix := config.NormalizeAPIPrefix(apiPrefix)
	if !strings.HasPrefix(cleanPath, prefix+"/") {
		return "", false
	}
	manifestPath := strings.TrimPrefix(cleanPath, prefix)
	segments := strings.Split(strings.Trim(manifestPath, "/"), "/")
	if len(segments) < 5 || segments[0] != "v1" || segments[1] != "plugins" || segments[2] == "" || segments[3] != "api" {
		return "", false
	}
	return manifestPath, true
}

func resolvedPluginRoutePermission(method, path string, resolver plugin.RoutePermissionResolver) (descriptor plugin.RoutePermissionDescriptor, resource, action, reason string) {
	if resolver == nil {
		return descriptor, "plugin.route", "resolve", "plugin_route_resolver_not_configured"
	}
	descriptor, ok := resolver.ResolveRoutePermission(method, path)
	if !ok {
		return descriptor, "plugin.route", "resolve", "plugin_route_permission_not_declared"
	}
	resource, action, ok = splitPermissionKey(descriptor.Permission)
	if !ok {
		return descriptor, "plugin.route", "resolve", "plugin_route_permission_invalid"
	}
	return descriptor, resource, action, ""
}

func splitPermissionKey(permission string) (resource, action string, ok bool) {
	permission = strings.TrimSpace(strings.ToLower(permission))
	separator := strings.LastIndexAny(permission, ".:")
	if separator <= 0 || separator == len(permission)-1 {
		return "", "", false
	}
	resource = strings.TrimSpace(permission[:separator])
	action = strings.TrimSpace(permission[separator+1:])
	return resource, action, resource != "" && action != ""
}

func writePluginPermissionDenied(w http.ResponseWriter, r *http.Request, reason string) {
	message := "插件路由权限不可用"
	if reason == "plugin_route_permission_not_declared" {
		message = "插件路由权限未声明"
	}
	if requestPrefersEnglish(r) {
		message = "plugin route permission is unavailable"
		if reason == "plugin_route_permission_not_declared" {
			message = "plugin route permission is not declared"
		}
	}
	apperrors.WriteHTTP(w, apperrors.New("forbidden", message, nil))
}

func writePermissionDenied(w http.ResponseWriter, r *http.Request, pluginRoute bool, fallback string) {
	if !pluginRoute {
		apperrors.WriteHTTP(w, apperrors.New("forbidden", fallback, nil))
		return
	}
	message := "插件路由权限不足"
	if requestPrefersEnglish(r) {
		message = "plugin route permission denied"
	}
	apperrors.WriteHTTP(w, apperrors.New("forbidden", message, nil))
}

func requestPrefersEnglish(r *http.Request) bool {
	if r == nil {
		return false
	}
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(r.Header.Get("Accept-Language"))), "en")
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

type pluginRouteAuditResponseWriter struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func (w *pluginRouteAuditResponseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	w.statusCode = statusCode
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *pluginRouteAuditResponseWriter) Write(data []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(data)
}

func (w *pluginRouteAuditResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func appendPluginRouteAudit(r *http.Request, sink auditmw.AuditEventSink, actorID, actorName string, descriptor plugin.RoutePermissionDescriptor, statusCode int) {
	if r == nil || sink == nil || descriptor.AuditAction == "" {
		return
	}
	now := time.Now().UTC()
	event, err := auditmw.NewPluginRouteAuditEvent(auditmw.PluginRouteAuditInput{
		ID:          shared.ID("audit-event-" + strconv.FormatInt(now.UnixNano(), 10)),
		ActorID:     actorID,
		ActorName:   actorName,
		Source:      descriptor.Source,
		Permission:  descriptor.Permission,
		AuditAction: descriptor.AuditAction,
		StatusCode:  statusCode,
		Request:     r,
		OccurredAt:  now,
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
	if cleanMethod == http.MethodGet && cleanPath == normalizedPrefix+"/v1/rbac/data-scope" {
		return "", "", false
	}

	for _, policy := range defaultPermissionPolicies(normalizedPrefix) {
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
