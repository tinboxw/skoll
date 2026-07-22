package http

import (
	"context"
	_ "embed"
	"net/http"
	"strings"
	"time"

	audithttp "github.com/tinboxw/skoll/internal/handler/http/v1/audit"
	filehttp "github.com/tinboxw/skoll/internal/handler/http/v1/file"
	menuhttp "github.com/tinboxw/skoll/internal/handler/http/v1/menu"
	permissionhttp "github.com/tinboxw/skoll/internal/handler/http/v1/permission"
	pluginhttp "github.com/tinboxw/skoll/internal/handler/http/v1/plugin"
	rbachttp "github.com/tinboxw/skoll/internal/handler/http/v1/rbac"
	rolehttp "github.com/tinboxw/skoll/internal/handler/http/v1/role"
	systemhttp "github.com/tinboxw/skoll/internal/handler/http/v1/system"
	userhttp "github.com/tinboxw/skoll/internal/handler/http/v1/user"
	workflowhttp "github.com/tinboxw/skoll/internal/handler/http/v1/workflow"
	"github.com/tinboxw/skoll/internal/plugin"
	"github.com/tinboxw/skoll/internal/service/audit"
	filesvc "github.com/tinboxw/skoll/internal/service/file"
	"github.com/tinboxw/skoll/internal/service/menu"
	"github.com/tinboxw/skoll/internal/service/permission"
	"github.com/tinboxw/skoll/internal/service/rbac"
	"github.com/tinboxw/skoll/internal/service/role"
	"github.com/tinboxw/skoll/internal/service/system"
	"github.com/tinboxw/skoll/internal/service/user"
	"github.com/tinboxw/skoll/internal/service/workflow"
	"github.com/tinboxw/skoll/pkg/config"
)

const defaultAPIPrefix = config.DefaultAPIBasePrefix

type Dependencies struct {
	UserService       user.Service
	RoleService       role.Service
	RBACService       rbac.Service
	AuditService      audit.Service
	AuditEventService audit.EventService
	FileService       filesvc.Service
	SystemService     system.Service
	PermissionService permission.Service
	MenuService       menu.Service
	WorkflowService   workflow.Service
	PluginManager     plugin.Manager
	APIPrefix         string
	LogLevel          string
	LogDir            string
	LogFile           string
	LogPluginPerFile  bool
	DevPortalEnabled  bool
	DevPortalRoot     string
	DevPortalRoots    []string
}

type Middleware func(http.Handler) http.Handler

type pluginExtensionSnapshotProvider interface {
	GetExtensionSnapshot(pluginID string) (plugin.RegistrySnapshot, bool)
}

type pluginRouteExecutor interface {
	HandlePluginRoute(pluginID, method, path string, w http.ResponseWriter, r *http.Request) bool
}

type pluginReadinessProvider interface {
	PluginReadiness(ctx context.Context) plugin.ReadinessReport
}

func NewRouter(deps Dependencies, middleware ...Middleware) http.Handler {
	apiPrefix := normalizeAPIPrefix(deps.APIPrefix)
	apiMux := http.NewServeMux()

	apiMux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		WriteMessage(w, http.StatusOK, "ok", "ok")
	})
	apiMux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		report := plugin.ReadinessReport{Ready: true, CheckedAt: time.Now().UTC(), Plugins: []plugin.HealthReport{}}
		if provider, ok := deps.PluginManager.(pluginReadinessProvider); ok {
			report = provider.PluginReadiness(r.Context())
		}
		if !report.Ready {
			WriteResponse(w, http.StatusServiceUnavailable, "not_ready", "插件运行时未就绪", report)
			return
		}
		WriteResponse(w, http.StatusOK, "ready", "ready", report)
	})
	registerDocumentationRoutes(apiMux, apiPrefix, deps.PluginManager)

	userhttp.RegisterUserRoutes(apiMux, deps.UserService, deps.RBACService, deps.AuditService)
	rolehttp.RegisterRoleRoutes(apiMux, deps.RoleService, deps.UserService, deps.RBACService, deps.AuditService)
	rbachttp.RegisterRBACRoutes(apiMux, deps.RBACService, deps.AuditService)
	audithttp.RegisterAuditRoutes(apiMux, deps.AuditService, deps.AuditEventService)
	filehttp.RegisterFileRoutes(apiMux, deps.FileService)
	systemhttp.RegisterSystemRoutes(apiMux, deps.SystemService, deps.AuditService)
	permissionhttp.RegisterPermissionRoutes(apiMux, deps.PermissionService)
	menuhttp.RegisterMenuRoutes(apiMux, deps.MenuService)
	workflowhttp.RegisterWorkflowRoutes(apiMux, deps.WorkflowService)
	_ = workflowhttp.RegisterWorkflowPermissions(deps.PermissionService)
	pluginhttp.RegisterPluginRoutes(
		apiMux,
		deps.PluginManager,
		pluginhttp.WithPluginLogTarget(deps.LogLevel, deps.LogDir, deps.LogFile, deps.LogPluginPerFile),
		pluginhttp.WithPluginAuditService(deps.AuditService),
		pluginhttp.WithPluginAuditEventSink(deps.AuditEventService),
		pluginhttp.WithPluginRoleCatalogProvider(deps.RoleService),
		pluginhttp.WithPluginDevPortal(deps.DevPortalEnabled, deps.DevPortalRoot, deps.DevPortalRoots),
	)
	registerPluginExtensionRoutes(apiMux, deps.PluginManager)

	rootMux := http.NewServeMux()
	stripped := http.StripPrefix(apiPrefix, apiMux)
	rootMux.Handle(apiPrefix+"/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set("X-Skoll-Api-Prefix", apiPrefix)
		stripped.ServeHTTP(w, r)
	}))

	var h http.Handler = rootMux
	for i := len(middleware) - 1; i >= 0; i-- {
		h = middleware[i](h)
	}
	return h
}

func normalizeAPIPrefix(raw string) string {
	v := strings.TrimSpace(raw)
	if v == "" {
		return defaultAPIPrefix
	}
	if !strings.HasPrefix(v, "/") {
		v = "/" + v
	}
	v = strings.TrimRight(v, "/")
	if v == "" {
		return defaultAPIPrefix
	}
	return v
}

func registerPluginExtensionRoutes(mux *http.ServeMux, manager plugin.Manager) {
	if mux == nil || manager == nil {
		return
	}
	mux.HandleFunc("/v1/plugins/{pluginID}/api/{resource...}", pluginRouteHandler(manager))

	provider, hasSnapshots := manager.(pluginExtensionSnapshotProvider)
	executor, canExecute := manager.(pluginRouteExecutor)
	if !hasSnapshots || !canExecute {
		return
	}

	for _, info := range manager.List() {
		if !info.SystemBuiltin || info.State != plugin.StateEnabled {
			continue
		}
		snapshot, ok := provider.GetExtensionSnapshot(info.ID)
		if !ok {
			continue
		}
		for _, route := range snapshot.Routes {
			method := strings.ToUpper(strings.TrimSpace(route.Method))
			path := strings.TrimSpace(route.Path)
			if !isAllowedPluginRoute(method, path) {
				continue
			}
			pluginID := info.ID
			routePath := path
			mux.HandleFunc(method+" "+routePath, func(w http.ResponseWriter, r *http.Request) {
				if executor.HandlePluginRoute(pluginID, method, routePath, w, r) {
					return
				}
				WriteMessage(w, http.StatusBadGateway, "plugin_route_unavailable", "插件路由不可用")
			})
		}
	}
}

func pluginRouteHandler(manager plugin.Manager) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		pluginID := strings.TrimSpace(r.PathValue("pluginID"))
		method := strings.ToUpper(strings.TrimSpace(r.Method))
		path := strings.TrimSpace(r.URL.Path)
		if pluginID == "" || !isAllowedPluginRoute(method, path) {
			WriteMessage(w, http.StatusMethodNotAllowed, "plugin_route_method_not_allowed", "插件路由方法不可用")
			return
		}
		if executor, ok := manager.(pluginRouteExecutor); ok {
			if executor.HandlePluginRoute(pluginID, method, path, w, r) {
				return
			}
		}
		WriteMessage(w, http.StatusBadGateway, "plugin_route_unavailable", "插件路由不可用")
	}
}

func isAllowedPluginRoute(method, path string) bool {
	if !strings.HasPrefix(path, "/") || strings.Contains(path, "{") || strings.Contains(path, "}") {
		return false
	}
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func registerDocumentationRoutes(mux *http.ServeMux, apiPrefix string, manager plugin.Manager) {
	if mux == nil {
		return
	}
	mux.HandleFunc("GET /docs/openapi.yaml", func(w http.ResponseWriter, _ *http.Request) {
		serveOpenAPIYAML(w, apiPrefix, manager)
	})
	mux.HandleFunc("GET /docs/swagger", func(w http.ResponseWriter, r *http.Request) {
		serveSwaggerUI(w, r, apiPrefix)
	})
}

func serveOpenAPIYAML(w http.ResponseWriter, apiPrefix string, manager plugin.Manager) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	doc := strings.ReplaceAll(openAPIYAMLDocument, "__API_BASE_PREFIX__", normalizeAPIPrefix(apiPrefix))
	doc = aggregatePluginOpenAPI(doc, manager)
	_, _ = w.Write([]byte(doc))
}

func serveSwaggerUI(w http.ResponseWriter, _ *http.Request, apiPrefix string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	openAPIURL := apiPrefix + "/docs/openapi.yaml"
	_, _ = w.Write([]byte(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width,initial-scale=1" />
  <title>Skoll Swagger UI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = function () {
      SwaggerUIBundle({
				url: '` + openAPIURL + `',
        dom_id: '#swagger-ui'
      });
    };
  </script>
</body>
</html>`))
}

//go:embed openapi.yaml
var openAPIYAMLDocument string
