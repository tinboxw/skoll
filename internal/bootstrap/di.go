package bootstrap

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/tinboxw/skoll/internal/event"
	events2 "github.com/tinboxw/skoll/internal/event/events"
	httpHandler "github.com/tinboxw/skoll/internal/handler/http"
	"github.com/tinboxw/skoll/internal/handler/middleware"
	"github.com/tinboxw/skoll/internal/plugin"
	"github.com/tinboxw/skoll/internal/service/audit"
	"github.com/tinboxw/skoll/internal/service/rbac"
	"github.com/tinboxw/skoll/internal/service/role"
	"github.com/tinboxw/skoll/internal/service/user"
	"github.com/tinboxw/skoll/internal/store"
	"github.com/tinboxw/skoll/pkg/logging"
)

type dependencies struct {
	logger  logging.Logger
	handler http.Handler
	server  *http.Server
}

func buildDependencies(cfg RuntimeConfig) (*dependencies, error) {
	logger := logging.New(cfg.AppConfig.Log.Level)

	bundle, err := store.NewBundle(store.Options{
		Mode:          store.Mode(cfg.AppConfig.Store.Mode),
		PrimaryDSN:    cfg.AppConfig.Store.DSN,
		ClickHouseDSN: "clickhouse://local",
	})
	if err != nil {
		return nil, err
	}

	_ = audit.NewService(bundle.Audit)
	bus := event.NewInMemoryBus()
	_ = event.NewPublisher(bus)
	_ = event.NewSubscriber(bus)
	_ = bus.Publish
	_ = events2.UserCreatedEventName
	userService := user.NewService(bundle.Users, bundle.Audit, bundle.UnitOfWork)
	roleService := role.NewService(bundle.Roles)
	rbacService := rbac.NewService(bundle.RBAC)
	pluginManager := newPluginManager(logger)

	router := httpHandler.NewRouter(httpHandler.Dependencies{
		UserService:   userService,
		RoleService:   roleService,
		RBACService:   rbacService,
		PluginManager: pluginManager,
	},
		middleware.Logger(),
		middleware.RateLimit(100, 100),
		middleware.Auth(),
	)

	h := buildMiddlewareChain(router, logger, cfg.AuthPolicy)
	server := &http.Server{
		Addr:              cfg.AppConfig.Server.Address,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
	}

	if server.Addr == "" {
		return nil, fmt.Errorf("server address is empty")
	}

	return &dependencies{logger: logger, handler: h, server: server}, nil
}

func newPluginManager(logger logging.Logger) plugin.Manager {
	m := plugin.NewRuntimeManager(plugin.NewFileLoader(), plugin.NewTopologicalResolver())

	entries, err := os.ReadDir("plugins")
	if err != nil {
		logger.Warn("load plugins directory failed", "error", err)
		return m
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join("plugins", entry.Name())
		info, installErr := m.Install(path)
		if installErr != nil {
			logger.Warn("plugin install failed", "path", path, "error", installErr)
			continue
		}
		if enableErr := m.Enable(info.ID); enableErr != nil {
			logger.Warn("plugin enable failed", "plugin", info.ID, "error", enableErr)
		}
	}

	return m
}
