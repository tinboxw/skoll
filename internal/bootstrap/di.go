package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tinboxw/skoll/internal/event"
	events2 "github.com/tinboxw/skoll/internal/event/events"
	httpHandler "github.com/tinboxw/skoll/internal/handler/http"
	"github.com/tinboxw/skoll/internal/handler/middleware"
	"github.com/tinboxw/skoll/internal/plugin"
	builtinAuth "github.com/tinboxw/skoll/internal/plugin/builtin/auth"
	builtinDashboard "github.com/tinboxw/skoll/internal/plugin/builtin/dashboard"
	builtinLogger "github.com/tinboxw/skoll/internal/plugin/builtin/logger"
	"github.com/tinboxw/skoll/internal/repository"
	"github.com/tinboxw/skoll/internal/service/audit"
	"github.com/tinboxw/skoll/internal/service/rbac"
	"github.com/tinboxw/skoll/internal/service/role"
	"github.com/tinboxw/skoll/internal/service/system"
	"github.com/tinboxw/skoll/internal/service/user"
	"github.com/tinboxw/skoll/internal/store"
	"github.com/tinboxw/skoll/pkg/logging"
	"github.com/tinboxw/skoll/pkg/security"
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

	auditService := audit.NewService(bundle.Audit)
	bus := event.NewInMemoryBus()
	_ = event.NewPublisher(bus)
	_ = event.NewSubscriber(bus)
	_ = bus.Publish
	_ = events2.UserCreatedEventName
	userService := user.NewService(bundle.Users, bundle.Audit, bundle.UnitOfWork)
	roleService := role.NewService(bundle.Roles)
	rbacService := rbac.NewService(bundle.RBAC)
	systemService := system.NewService(bundle.System)
	pluginManager := newPluginManager(logger, cfg.AppConfig.Security.JWTSecret, bundle.Users, bundle.Roles, bundle.RBAC, bundle.Plugins)

	router := httpHandler.NewRouter(httpHandler.Dependencies{
		UserService:   userService,
		RoleService:   roleService,
		RBACService:   rbacService,
		AuditService:  auditService,
		SystemService: systemService,
		PluginManager: pluginManager,
	},
		middleware.Logger(),
		middleware.RateLimit(100, 100),
		middleware.Auth(),
	)

	h := buildMiddlewareChain(router, logger, cfg.AuthPolicy, cfg.AppConfig.Security.JWTSecret, rbacService)
	server := &http.Server{
		Addr:              cfg.AppConfig.Server.Address,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
	}

	if server.Addr == "" {
		return nil, fmt.Errorf("server address is empty")
	}

	ensureBuiltinAuthData(context.Background(), logger, bundle.Users, bundle.Roles, bundle.RBAC)

	return &dependencies{logger: logger, handler: h, server: server}, nil
}

func newPluginManager(logger logging.Logger, jwtSecret string, usersRepo repository.UserRepository, rolesRepo repository.RoleRepository, rbacRepo repository.RBACRepository, pluginsRepo repository.PluginRepository) plugin.Manager {
	runtimeManager := plugin.NewRuntimeManager(plugin.NewFileLoader(), plugin.NewTopologicalResolver())
	authHandler := newBuiltinAuthHandler(jwtSecret, usersRepo, rolesRepo, rbacRepo)
	builtinInfos, extensions, handlers := registerBuiltinPluginExtensions(logger, jwtSecret, authHandler)
	m := &pluginManagerWithExtensions{
		Manager:       runtimeManager,
		builtinInfos:  builtinInfos,
		extensions:    extensions,
		routeHandlers: handlers,
		pluginsRepo:   pluginsRepo,
	}

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

	m.persistAll(context.Background())

	return m
}

type pluginManagerWithExtensions struct {
	plugin.Manager
	mu            sync.RWMutex
	builtinInfos  map[string]plugin.Info
	extensions    map[string]plugin.RegistrySnapshot
	routeHandlers map[string]http.HandlerFunc
	pluginsRepo   repository.PluginRepository
}

func (m *pluginManagerWithExtensions) Install(path string) (plugin.Info, error) {
	info, err := m.Manager.Install(path)
	if err != nil {
		return plugin.Info{}, err
	}
	m.persistOne(context.Background(), info.ID)
	return info, nil
}

func (m *pluginManagerWithExtensions) HandlePluginRoute(pluginID, method, path string, w http.ResponseWriter, r *http.Request) bool {
	if m == nil || w == nil || r == nil {
		return false
	}
	key := pluginRouteKey(pluginID, method, path)
	m.mu.RLock()
	h, ok := m.routeHandlers[key]
	m.mu.RUnlock()
	if !ok {
		return false
	}
	h(w, r)
	return true
}

func (m *pluginManagerWithExtensions) GetExtensionSnapshot(pluginID string) (plugin.RegistrySnapshot, bool) {
	if m == nil {
		return plugin.RegistrySnapshot{}, false
	}
	snapshot, ok := m.extensions[pluginID]
	return snapshot, ok
}

func (m *pluginManagerWithExtensions) List() []plugin.Info {
	if m != nil && m.pluginsRepo != nil {
		items, err := m.pluginsRepo.List(context.Background())
		if err == nil && len(items) > 0 {
			sort.Slice(items, func(i, j int) bool {
				return items[i].ID < items[j].ID
			})
			return items
		}
	}

	base := m.Manager.List()
	merged := make(map[string]plugin.Info, len(base)+len(m.builtinInfos))
	for _, item := range base {
		merged[item.ID] = item
	}

	m.mu.RLock()
	for id, item := range m.builtinInfos {
		if _, ok := merged[id]; ok {
			continue
		}
		merged[id] = item
	}
	m.mu.RUnlock()

	items := make([]plugin.Info, 0, len(merged))
	for _, item := range merged {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
	return items
}

func (m *pluginManagerWithExtensions) Get(pluginID string) (plugin.Info, error) {
	if m != nil && m.pluginsRepo != nil {
		item, err := m.pluginsRepo.Get(context.Background(), pluginID)
		if err == nil && item != nil {
			return *item, nil
		}
	}

	item, err := m.Manager.Get(pluginID)
	if err == nil {
		return item, nil
	}
	if !errors.Is(err, plugin.ErrPluginNotFound) {
		return plugin.Info{}, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	item, ok := m.builtinInfos[pluginID]
	if !ok {
		return plugin.Info{}, plugin.ErrPluginNotFound
	}
	return item, nil
}

func (m *pluginManagerWithExtensions) Enable(pluginID string) error {
	if err := m.Manager.Enable(pluginID); err == nil {
		m.persistOne(context.Background(), pluginID)
		return nil
	} else if !errors.Is(err, plugin.ErrPluginNotFound) {
		return err
	}

	if m.pluginsRepo != nil {
		stored, storedErr := m.pluginsRepo.Get(context.Background(), pluginID)
		if storedErr == nil && stored != nil {
			now := time.Now().UTC()
			stored.State = plugin.StateEnabled
			stored.EnabledAt = &now
			_ = m.pluginsRepo.Save(context.Background(), *stored)
			return nil
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	item, ok := m.builtinInfos[pluginID]
	if !ok {
		return plugin.ErrPluginNotFound
	}
	now := time.Now().UTC()
	item.State = plugin.StateEnabled
	item.EnabledAt = &now
	m.builtinInfos[pluginID] = item
	m.persistOne(context.Background(), pluginID)
	return nil
}

func (m *pluginManagerWithExtensions) Disable(pluginID string) error {
	if err := m.Manager.Disable(pluginID); err == nil {
		m.persistOne(context.Background(), pluginID)
		return nil
	} else if !errors.Is(err, plugin.ErrPluginNotFound) {
		return err
	}

	if m.pluginsRepo != nil {
		stored, storedErr := m.pluginsRepo.Get(context.Background(), pluginID)
		if storedErr == nil && stored != nil {
			if stored.SystemBuiltin {
				return plugin.ErrPluginSystemProtected
			}
			stored.State = plugin.StateDisabled
			stored.EnabledAt = nil
			_ = m.pluginsRepo.Save(context.Background(), *stored)
			return nil
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.builtinInfos[pluginID]
	if !ok {
		return plugin.ErrPluginNotFound
	}
	return plugin.ErrPluginSystemProtected
}

func (m *pluginManagerWithExtensions) Uninstall(pluginID string) error {
	if err := m.Manager.Uninstall(pluginID); err == nil {
		if m.pluginsRepo != nil {
			_ = m.pluginsRepo.Delete(context.Background(), pluginID)
		}
		return nil
	} else if !errors.Is(err, plugin.ErrPluginNotFound) {
		return err
	}

	if m.pluginsRepo != nil {
		stored, storedErr := m.pluginsRepo.Get(context.Background(), pluginID)
		if storedErr == nil && stored != nil {
			if stored.SystemBuiltin {
				return plugin.ErrPluginSystemProtected
			}
			_ = m.pluginsRepo.Delete(context.Background(), pluginID)
			return nil
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.builtinInfos[pluginID]
	if !ok {
		return plugin.ErrPluginNotFound
	}
	return plugin.ErrPluginSystemProtected
}

func (m *pluginManagerWithExtensions) persistAll(ctx context.Context) {
	if m == nil || m.pluginsRepo == nil {
		return
	}
	for _, item := range m.collectCurrentItems() {
		_ = m.pluginsRepo.Save(ctx, item)
	}
}

func (m *pluginManagerWithExtensions) RegisterExternalPlugin(info plugin.Info) error {
	if m == nil || m.pluginsRepo == nil {
		return errors.New("plugin persistence is not configured")
	}
	if strings.TrimSpace(info.ID) == "" || strings.TrimSpace(info.Name) == "" {
		return errors.New("plugin id and name are required")
	}
	now := time.Now().UTC()
	if info.InstalledAt.IsZero() {
		info.InstalledAt = now
	}
	if info.State == "" {
		info.State = plugin.StateEnabled
	}
	if info.State == plugin.StateEnabled && info.EnabledAt == nil {
		info.EnabledAt = &now
	}
	if info.UIMode == "" {
		info.UIMode = plugin.UIModeSeparated
	}
	return m.pluginsRepo.Save(context.Background(), info)
}

func (m *pluginManagerWithExtensions) persistOne(ctx context.Context, pluginID string) {
	if m == nil || m.pluginsRepo == nil {
		return
	}
	item, err := m.getCurrent(pluginID)
	if err != nil {
		return
	}
	_ = m.pluginsRepo.Save(ctx, item)
}

func (m *pluginManagerWithExtensions) collectCurrentItems() []plugin.Info {
	base := m.Manager.List()
	merged := make(map[string]plugin.Info, len(base)+len(m.builtinInfos))
	for _, item := range base {
		if strings.TrimSpace(item.ID) == "" {
			continue
		}
		merged[item.ID] = item
	}

	m.mu.RLock()
	for id, item := range m.builtinInfos {
		if _, ok := merged[id]; ok {
			continue
		}
		if strings.TrimSpace(item.ID) == "" {
			continue
		}
		merged[id] = item
	}
	m.mu.RUnlock()

	items := make([]plugin.Info, 0, len(merged))
	for _, item := range merged {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
	return items
}

func (m *pluginManagerWithExtensions) getCurrent(pluginID string) (plugin.Info, error) {
	item, err := m.Manager.Get(pluginID)
	if err == nil {
		return item, nil
	}
	if !errors.Is(err, plugin.ErrPluginNotFound) {
		return plugin.Info{}, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	item, ok := m.builtinInfos[pluginID]
	if !ok {
		return plugin.Info{}, plugin.ErrPluginNotFound
	}
	return item, nil
}

func registerBuiltinPluginExtensions(logger logging.Logger, jwtSecret string, authHandler *builtinAuthHandler) (map[string]plugin.Info, map[string]plugin.RegistrySnapshot, map[string]http.HandlerFunc) {
	builtinPlugins := []interface {
		ID() string
		Name() string
		Version() string
		Register(registry plugin.ExtensionRegistry) error
	}{
		builtinAuth.New(),
		builtinLogger.New(),
		builtinDashboard.New(),
	}

	infos := make(map[string]plugin.Info, len(builtinPlugins))
	snapshots := make(map[string]plugin.RegistrySnapshot, len(builtinPlugins))
	handlers := make(map[string]http.HandlerFunc, len(builtinPlugins))
	for _, p := range builtinPlugins {
		now := time.Now().UTC()
		uiMode := plugin.UIModeBackendOnly
		frontendEntry := ""
		if p.ID() == "builtin-auth" {
			uiMode = plugin.UIModeSeparated
			frontendEntry = "/plugins/auth"
		}
		infos[p.ID()] = plugin.Info{
			ID:            p.ID(),
			Name:          p.Name(),
			Version:       p.Version(),
			State:         plugin.StateEnabled,
			InstalledAt:   now,
			EnabledAt:     &now,
			Source:        "builtin",
			UIMode:        uiMode,
			FrontendEntry: frontendEntry,
			SystemBuiltin: true,
		}

		registry := plugin.NewMemoryRegistry()
		if err := p.Register(registry); err != nil {
			logger.Warn("register builtin plugin extension failed", "plugin", p.ID(), "error", err)
			continue
		}
		snapshot := registry.Snapshot()
		snapshots[p.ID()] = snapshot
		registerBuiltinRouteHandlers(p.ID(), snapshot.Routes, handlers, jwtSecret, authHandler)
	}

	return infos, snapshots, handlers
}

func registerBuiltinRouteHandlers(pluginID string, routes []plugin.RouteExtension, handlers map[string]http.HandlerFunc, jwtSecret string, authHandler *builtinAuthHandler) {
	for _, route := range routes {
		method := strings.ToUpper(strings.TrimSpace(route.Method))
		path := strings.TrimSpace(route.Path)
		key := pluginRouteKey(pluginID, method, path)
		if h, ok := selectBuiltinRouteHandler(pluginID, method, path, jwtSecret, authHandler); ok {
			handlers[key] = h
			continue
		}
		handlers[key] = defaultBuiltinRouteHandler(pluginID, method, path)
	}
}

func selectBuiltinRouteHandler(pluginID, method, path, jwtSecret string, authHandler *builtinAuthHandler) (http.HandlerFunc, bool) {
	switch pluginRouteKey(pluginID, method, path) {
	case pluginRouteKey("builtin-auth", http.MethodPost, "/v1/auth/login"):
		if authHandler != nil {
			return authHandler.handleLogin, true
		}
		return handleBuiltinAuthLogin(jwtSecret), true
	case pluginRouteKey("builtin-auth", http.MethodPost, "/v1/auth/logout"):
		if authHandler != nil {
			return authHandler.handleLogout, true
		}
		return nil, false
	case pluginRouteKey("builtin-auth", http.MethodGet, "/v1/auth/me"):
		if authHandler != nil {
			return authHandler.handleMe, true
		}
		return nil, false
	case pluginRouteKey("builtin-auth", http.MethodPut, "/v1/auth/me/profile"):
		if authHandler != nil {
			return authHandler.handleUpdateProfile, true
		}
		return nil, false
	case pluginRouteKey("builtin-auth", http.MethodPatch, "/v1/auth/me/password"):
		if authHandler != nil {
			return authHandler.handleUpdatePassword, true
		}
		return nil, false
	case pluginRouteKey("builtin-logger", http.MethodGet, "/v1/logs"):
		return handleBuiltinLoggerLogs, true
	case pluginRouteKey("builtin-dashboard", http.MethodGet, "/v1/dashboard/widgets"):
		return handleBuiltinDashboardWidgets, true
	default:
		return nil, false
	}
}

func defaultBuiltinRouteHandler(pluginID, method, path string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		httpHandler.WriteJSON(w, http.StatusOK, map[string]string{
			"plugin": pluginID,
			"method": method,
			"route":  path,
			"status": "ok",
		})
	}
}

func handleBuiltinAuthLogin(jwtSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Account  string `json:"account"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpHandler.WriteError(w, http.StatusBadRequest, err)
			return
		}
		req.Account = strings.TrimSpace(req.Account)
		req.Password = strings.TrimSpace(req.Password)
		if req.Account == "" || req.Password == "" {
			httpHandler.WriteMessage(w, http.StatusBadRequest, "invalid_credentials", "account and password are required")
			return
		}
		token, err := security.SignJWT(jwtSecret, req.Account, "super_admin", time.Hour, time.Now().UTC())
		if err != nil {
			httpHandler.WriteError(w, http.StatusInternalServerError, err)
			return
		}

		httpHandler.WriteJSON(w, http.StatusOK, map[string]any{
			"token":       token,
			"tokenType":   "Bearer",
			"expiresIn":   3600,
			"permissions": []string{"menu.read", "role.manage", "permission.manage"},
			"user": map[string]string{
				"account": req.Account,
				"name":    req.Account,
				"role":    "super_admin",
			},
		})
	}
}

func handleBuiltinLoggerLogs(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 {
			httpHandler.WriteMessage(w, http.StatusBadRequest, "invalid_limit", "limit must be a positive integer")
			return
		}
		if v > 100 {
			v = 100
		}
		limit = v
	}

	entries := make([]map[string]string, 0, limit)
	now := time.Now().UTC()
	for i := 0; i < limit; i++ {
		entries = append(entries, map[string]string{
			"time":    now.Add(-time.Duration(i) * time.Minute).Format(time.RFC3339),
			"level":   "info",
			"message": "builtin logger heartbeat",
		})
	}

	httpHandler.WriteJSON(w, http.StatusOK, map[string]any{
		"total":   len(entries),
		"entries": entries,
	})
}

func handleBuiltinDashboardWidgets(w http.ResponseWriter, _ *http.Request) {
	httpHandler.WriteJSON(w, http.StatusOK, []map[string]any{
		{"id": "system-health-widget", "title": "System Health", "value": "ok", "trend": "stable"},
		{"id": "active-plugin-widget", "title": "Active Plugins", "value": 3, "trend": "up"},
	})
}

func pluginRouteKey(pluginID, method, path string) string {
	return strings.TrimSpace(pluginID) + "|" + strings.ToUpper(strings.TrimSpace(method)) + "|" + strings.TrimSpace(path)
}
