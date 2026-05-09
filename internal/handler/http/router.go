package http

import (
	"net/http"

	v1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	"github.com/tinboxw/skoll/internal/plugin"
	"github.com/tinboxw/skoll/internal/service/rbac"
	"github.com/tinboxw/skoll/internal/service/role"
	"github.com/tinboxw/skoll/internal/service/user"
)

type Dependencies struct {
	UserService   user.Service
	RoleService   role.Service
	RBACService   rbac.Service
	PluginManager plugin.Manager
}

type Middleware func(http.Handler) http.Handler

func NewRouter(deps Dependencies, middleware ...Middleware) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		WriteMessage(w, http.StatusOK, "ok", "ok")
	})

	v1.RegisterUserRoutes(mux, deps.UserService)
	v1.RegisterRoleRoutes(mux, deps.RoleService)
	v1.RegisterRBACRoutes(mux, deps.RBACService)
	v1.RegisterPluginRoutes(mux, deps.PluginManager)

	var h http.Handler = mux
	for i := len(middleware) - 1; i >= 0; i-- {
		h = middleware[i](h)
	}
	return h
}
