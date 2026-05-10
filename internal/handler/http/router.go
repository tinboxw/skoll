package http

import (
	_ "embed"
	"net/http"
	"strings"

	v1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	"github.com/tinboxw/skoll/internal/plugin"
	"github.com/tinboxw/skoll/internal/service/audit"
	"github.com/tinboxw/skoll/internal/service/rbac"
	"github.com/tinboxw/skoll/internal/service/role"
	"github.com/tinboxw/skoll/internal/service/system"
	"github.com/tinboxw/skoll/internal/service/user"
)

type Dependencies struct {
	UserService   user.Service
	RoleService   role.Service
	RBACService   rbac.Service
	AuditService  audit.Service
	SystemService system.Service
	PluginManager plugin.Manager
	LogDir        string
	LogFile       string
}

type Middleware func(http.Handler) http.Handler

type pluginExtensionSnapshotProvider interface {
	GetExtensionSnapshot(pluginID string) (plugin.RegistrySnapshot, bool)
}

type pluginRouteExecutor interface {
	HandlePluginRoute(pluginID, method, path string, w http.ResponseWriter, r *http.Request) bool
}

func NewRouter(deps Dependencies, middleware ...Middleware) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		WriteMessage(w, http.StatusOK, "ok", "ok")
	})
	registerDocumentationRoutes(mux)

	v1.RegisterUserRoutes(mux, deps.UserService)
	v1.RegisterRoleRoutes(mux, deps.RoleService)
	v1.RegisterRBACRoutes(mux, deps.RBACService)
	v1.RegisterAuditRoutes(mux, deps.AuditService)
	v1.RegisterSystemRoutes(mux, deps.SystemService)
	v1.RegisterPluginRoutes(mux, deps.PluginManager, v1.WithPluginLogTarget(deps.LogDir, deps.LogFile))
	registerPluginExtensionRoutes(mux, deps.PluginManager)

	var h http.Handler = mux
	for i := len(middleware) - 1; i >= 0; i-- {
		h = middleware[i](h)
	}
	return h
}

func registerPluginExtensionRoutes(mux *http.ServeMux, manager plugin.Manager) {
	if mux == nil || manager == nil {
		return
	}

	provider, ok := manager.(pluginExtensionSnapshotProvider)
	if !ok {
		return
	}

	registered := map[string]struct{}{}
	for _, item := range manager.List() {
		if item.State != plugin.StateEnabled {
			continue
		}
		snapshot, exists := provider.GetExtensionSnapshot(item.ID)
		if !exists {
			continue
		}
		for _, route := range snapshot.Routes {
			method := strings.ToUpper(strings.TrimSpace(route.Method))
			path := strings.TrimSpace(route.Path)
			if !isAllowedPluginRoute(method, path) {
				continue
			}

			pattern := method + " " + path
			if _, seen := registered[pattern]; seen {
				continue
			}
			if !safeHandleFunc(mux, pattern, pluginRouteHandler(manager, item.ID, method, path)) {
				continue
			}
			registered[pattern] = struct{}{}
		}
	}
}

func pluginRouteHandler(manager plugin.Manager, pluginID, method, path string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if executor, ok := manager.(pluginRouteExecutor); ok {
			if executor.HandlePluginRoute(pluginID, method, path, w, r) {
				return
			}
		}
		WriteJSON(w, http.StatusOK, map[string]string{
			"plugin": pluginID,
			"method": method,
			"route":  path,
			"status": "registered",
		})
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

func safeHandleFunc(mux *http.ServeMux, pattern string, handler func(http.ResponseWriter, *http.Request)) (ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	mux.HandleFunc(pattern, handler)
	return true
}

func registerDocumentationRoutes(mux *http.ServeMux) {
	if mux == nil {
		return
	}
	mux.HandleFunc("GET /docs/openapi.yaml", serveOpenAPIYAML)
	mux.HandleFunc("GET /docs/swagger", serveSwaggerUI)
}

func serveOpenAPIYAML(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(openAPIYAMLDocument))
}

func serveSwaggerUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
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
        url: '/docs/openapi.yaml',
        dom_id: '#swagger-ui'
      });
    };
  </script>
</body>
</html>`))
}

//go:embed openapi.yaml
var openAPIYAMLDocument string
