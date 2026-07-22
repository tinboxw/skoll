package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tinboxw/skoll/internal/event"
	httpHandler "github.com/tinboxw/skoll/internal/handler/http"
	"github.com/tinboxw/skoll/internal/handler/middleware"
	"github.com/tinboxw/skoll/internal/plugin"
	builtinAuth "github.com/tinboxw/skoll/internal/plugin/builtin/auth"
	builtinDashboard "github.com/tinboxw/skoll/internal/plugin/builtin/dashboard"
	builtinLogger "github.com/tinboxw/skoll/internal/plugin/builtin/logger"
	"github.com/tinboxw/skoll/internal/plugin/datastore"
	"github.com/tinboxw/skoll/internal/plugin/hostservice"
	organizationrepo "github.com/tinboxw/skoll/internal/repository/organization"
	pluginrepo "github.com/tinboxw/skoll/internal/repository/plugin"
	rbacrepo "github.com/tinboxw/skoll/internal/repository/rbac"
	rolerepo "github.com/tinboxw/skoll/internal/repository/role"
	userrepo "github.com/tinboxw/skoll/internal/repository/user"
	"github.com/tinboxw/skoll/internal/service/audit"
	documentnumbersvc "github.com/tinboxw/skoll/internal/service/documentnumber"
	filesvc "github.com/tinboxw/skoll/internal/service/file"
	jobsvc "github.com/tinboxw/skoll/internal/service/job"
	menusvc "github.com/tinboxw/skoll/internal/service/menu"
	permissionsvc "github.com/tinboxw/skoll/internal/service/permission"
	"github.com/tinboxw/skoll/internal/service/rbac"
	"github.com/tinboxw/skoll/internal/service/role"
	"github.com/tinboxw/skoll/internal/service/system"
	"github.com/tinboxw/skoll/internal/service/user"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
	"github.com/tinboxw/skoll/internal/store"
	objectstore "github.com/tinboxw/skoll/internal/store/object"
	"github.com/tinboxw/skoll/internal/store/sql/gormrepo"
	"github.com/tinboxw/skoll/pkg/config"
	"github.com/tinboxw/skoll/pkg/logging"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"github.com/tinboxw/skoll/pkg/security"
)

type dependencies struct {
	logger           logging.Logger
	handler          http.Handler
	server           *http.Server
	eventBus         event.Bus
	businessEventBus *event.BusinessEventBus
	pluginRuntime    closeable
	jobService       *jobsvc.Service
}

func buildDependencies(cfg RuntimeConfig) (*dependencies, error) {
	logging.SetDefaultOutput(cfg.AppConfig.Log.Dir, cfg.AppConfig.Log.File)
	logger := logging.NewWithOptions(logging.Options{
		Level: cfg.AppConfig.Log.Level,
		Dir:   cfg.AppConfig.Log.Dir,
		File:  cfg.AppConfig.Log.File,
	})

	bundle, err := store.NewBundle(store.Options{
		Mode:          store.Mode(cfg.AppConfig.Store.Mode),
		PrimaryDSN:    cfg.AppConfig.Store.DSN,
		ClickHouseDSN: "clickhouse://local",
	})
	if err != nil {
		return nil, err
	}

	auditService := audit.NewService(bundle.Audit)
	var auditEventService audit.EventService
	if bundle.AuditEvents != nil {
		auditEventService = audit.NewEventService(bundle.AuditEvents)
	}
	bus, err := buildEventBus(cfg.AppConfig.Event)
	if err != nil {
		return nil, err
	}
	_ = event.NewPublisher(bus)
	_ = event.NewSubscriber(bus)
	businessEventBus := event.NewBusinessEventBus(nil)
	roleService := role.NewService(bundle.Roles)
	rbacService := rbac.NewServiceWithOrganization(bundle.RBAC, bundle.Organization)
	userService := user.NewServiceWithDataScope(bundle.Users, bundle.Audit, bundle.UnitOfWork, rbacService)
	systemService := system.NewService(bundle.System)
	permissionService := permissionsvc.NewService(bundle.Permissions)
	menuService := menusvc.NewService(bundle.Menus)
	workflowService := workflowsvc.NewService(bundle.Workflow)
	objectStore, err := objectstore.NewLocalStore(filepath.Join("data", "objects"))
	if err != nil {
		return nil, err
	}
	fileService := filesvc.NewService(bundle.Files, objectStore, filesvc.Options{
		Permission: rbacService,
		Audit:      auditEventService,
	})
	jobService := jobsvc.NewService(bundle.Jobs, nil)
	documentNumberService := documentnumbersvc.NewService(gormrepo.NewDocumentNumberStore(bundle.PluginDataDB))
	documentWorkflowStore := gormrepo.NewDocumentWorkflowStore(bundle.PluginDataDB)
	transactionService, err := hostservice.NewTransactionService(bundle.UnitOfWork)
	if err != nil {
		return nil, err
	}
	dataScopeService, err := hostservice.NewDataScopeService(rbacService, bundle.Organization)
	if err != nil {
		return nil, err
	}
	pluginDataDialect, err := datastore.ParseSQLDialect(bundle.PluginDataDialect)
	if err != nil {
		return nil, err
	}
	pluginDataRegistry := datastore.NewSchemaRegistry()
	pluginDataLifecycle, err := datastore.NewLifecycle(bundle.PluginDataDB, pluginDataRegistry)
	if err != nil {
		return nil, err
	}
	dataStoreFactory := func(pluginID string, scopes pluginsdk.DataScopeService, audit pluginsdk.AuditService) (pluginsdk.DataStoreService, error) {
		return datastore.NewService(bundle.PluginDataDB, bundle.UnitOfWork, pluginDataRegistry, scopes, audit, pluginDataDialect, pluginID)
	}
	pluginManager, err := newPluginManager(
		logger, cfg.AppConfig.Security.JWTSecret, bundle.Users, bundle.Roles, bundle.RBAC, bundle.Organization,
		bundle.Plugins, bundle.PluginMigrations, auditService, auditEventService, businessEventBus, pluginDataLifecycle,
		hostservice.HostServicesDependencies{
			Transactions: transactionService, DataScopes: dataScopeService, DataStore: dataStoreFactory, Files: fileService, Audit: auditService,
			DocumentNumbers: documentNumberService, DocumentWorkflows: documentWorkflowStore, System: systemService, MasterSecret: cfg.AppConfig.Security.JWTSecret, Workflow: workflowService, Jobs: jobService,
		},
	)
	if err != nil {
		return nil, err
	}
	router := httpHandler.NewRouter(httpHandler.Dependencies{
		UserService:       userService,
		RoleService:       roleService,
		RBACService:       rbacService,
		AuditService:      auditService,
		AuditEventService: auditEventService,
		JobService:        jobService,
		FileService:       fileService,
		SystemService:     systemService,
		PermissionService: permissionService,
		MenuService:       menuService,
		WorkflowService:   workflowService,
		PluginManager:     pluginManager,
		APIPrefix:         cfg.AppConfig.Server.APIPrefix,
		LogLevel:          cfg.AppConfig.Log.Level,
		LogDir:            cfg.AppConfig.Log.Dir,
		LogFile:           cfg.AppConfig.Log.File,
		LogPluginPerFile:  cfg.AppConfig.Log.PluginPerFile,
		DevPortalEnabled:  cfg.AppConfig.Dev.PortalEnabled,
		DevPortalRoot:     cfg.AppConfig.Dev.PluginsRoot,
		DevPortalRoots:    append([]string(nil), cfg.AppConfig.Dev.PluginsRoots...),
	},
		middleware.Logger(),
		middleware.RateLimit(100, 100),
	)

	routePermissionResolver, _ := pluginManager.(plugin.RoutePermissionResolver)
	h := buildMiddlewareChain(router, logger, cfg.AuthPolicy, cfg.AppConfig.Server.APIPrefix, cfg.AppConfig.Security.JWTSecret, rbacService, routePermissionResolver, auditEventService)
	server := &http.Server{
		Addr:              cfg.AppConfig.Server.Address,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
	}

	if server.Addr == "" {
		return nil, fmt.Errorf("server address is empty")
	}

	ensureBuiltinAuthData(context.Background(), logger, bundle.Users, bundle.Roles, bundle.RBAC)
	ensureSystemPermissionCatalog(context.Background(), logger, permissionService)

	pluginRuntime, _ := pluginManager.(closeable)
	return &dependencies{logger: logger, handler: h, server: server, eventBus: bus, businessEventBus: businessEventBus, pluginRuntime: pluginRuntime, jobService: jobService}, nil
}

func buildEventBus(cfg config.EventConfig) (event.Bus, error) {
	mode := strings.ToLower(strings.TrimSpace(cfg.Mode))
	switch mode {
	case "", "memory":
		return event.NewInMemoryBus(), nil
	case "redis":
		return event.NewRedisBus(cfg.RedisAddr, cfg.ChannelPrefix)
	default:
		return nil, fmt.Errorf("unsupported event mode: %q", cfg.Mode)
	}
}

func newPluginManager(logger logging.Logger, jwtSecret string, usersRepo userrepo.UserRepository, rolesRepo rolerepo.RoleRepository, rbacRepo rbacrepo.RBACRepository, organizationRepo organizationrepo.OrganizationRepository, pluginsRepo pluginrepo.PluginRepository, migrationStore plugin.MigrationStore, auditSvc audit.Service, auditEventSvc audit.EventService, businessEvents *event.BusinessEventBus, dataLifecycle *datastore.Lifecycle, hostDeps hostservice.HostServicesDependencies) (plugin.Manager, error) {
	if businessEvents == nil {
		businessEvents = event.NewBusinessEventBus(nil)
	}
	runtimeManager := plugin.NewRuntimeManager(plugin.NewFileLoader(), plugin.NewTopologicalResolver())
	runtimeManager.SetCatalogRegistry(plugin.NewMemoryCatalogRegistry())
	runtimeManager.SetCatalogAuditSink(pluginCatalogAuditSink{auditSvc: auditSvc})
	authHandler := newBuiltinAuthHandler(jwtSecret, usersRepo, rolesRepo, rbacRepo, organizationRepo, auditSvc, auditEventSvc, logger)
	builtinInfos, extensions, handlers := registerBuiltinPluginExtensions(logger, jwtSecret, authHandler)
	healthChecker := plugin.NewHTTPHealthChecker(2 * time.Second)
	m := &pluginManagerWithExtensions{
		Manager:            runtimeManager,
		builtinInfos:       builtinInfos,
		extensions:         extensions,
		routeHandlers:      handlers,
		pluginsRepo:        pluginsRepo,
		routePermissions:   mustEmptyRoutePermissionRegistry(),
		healthChecker:      healthChecker,
		healthCache:        make(map[string]plugin.HealthReport),
		healthTTL:          5 * time.Second,
		inProcessBackends:  plugin.NewInProcessBackendRegistry(),
		businessEvents:     businessEvents,
		eventDelivery:      plugin.NewHTTPEventDeliveryClient(nil, 5*time.Second),
		eventSubscriptions: make(map[string][]func()),
		migrationHook:      plugin.NewPluginMigrationHook(migrationStore, pluginMigrationAuditSink{auditSvc: auditSvc}),
		dataLifecycle:      dataLifecycle,
	}
	dataDirectories, err := plugin.NewPluginDataDirectories(filepath.Join("data", "plugins"))
	if err != nil {
		return nil, err
	}
	m.dataDirectories = dataDirectories
	hostGateway, err := plugin.NewHostGateway(func(pluginID string) (pluginsdk.HostServices, error) {
		deps := hostDeps
		deps.PluginID = pluginID
		deps.ConfigStore = m
		return hostservice.NewHostServices(deps)
	}, jwtSecret, 30*time.Second)
	if err != nil {
		return nil, err
	}
	m.hostGateway = hostGateway
	m.serviceSupervisor = plugin.NewServiceSupervisor(
		plugin.NewManagedProcessLauncher(healthChecker, 5*time.Second, hostGateway, dataDirectories),
		pluginServiceAuditSink{auditSvc: auditSvc}, 5*time.Second, 5*time.Second,
	)

	entries, err := os.ReadDir("plugins")
	if err != nil {
		logger.Warn("load plugins directory failed", "error", err)
		return m, nil
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join("plugins", entry.Name())
		hasManifest, manifestErr := hasPluginManifest(path)
		if manifestErr != nil {
			logger.Warn("inspect plugin directory failed", "path", path, "error", manifestErr)
			continue
		}
		if !hasManifest {
			continue
		}
		info, installErr := m.Install(path)
		if installErr != nil {
			logger.Warn("plugin install failed", "path", path, "error", installErr)
			continue
		}
		if enableErr := m.Enable(info.ID); enableErr != nil {
			logger.Warn("plugin enable failed", "plugin", info.ID, "error", enableErr)
		}
	}
	if err := m.refreshRoutePermissions(); err != nil {
		logger.Error("build plugin route permission registry failed", "error", err)
	}

	m.persistAll(context.Background())

	return m, nil
}

type pluginCatalogAuditSink struct {
	auditSvc audit.Service
}

type pluginServiceAuditSink struct {
	auditSvc audit.Service
}

type pluginMigrationAuditSink struct {
	auditSvc audit.Service
}

func (s pluginMigrationAuditSink) RecordPluginMigration(event plugin.PluginMigrationEvent) error {
	if s.auditSvc == nil {
		return nil
	}
	_, err := s.auditSvc.Append(context.Background(), "system", "plugin_migration_"+string(event.Action), "plugin", event.PluginID, map[string]any{
		"status":      event.Status,
		"fromVersion": event.FromVersion,
		"toVersion":   event.ToVersion,
		"steps":       len(event.Steps),
		"error":       event.Error,
	})
	return err
}

func (s pluginServiceAuditSink) RecordPluginServiceEvent(event plugin.ServiceLifecycleEvent) error {
	if s.auditSvc == nil {
		return nil
	}
	_, err := s.auditSvc.Append(context.Background(), "system", "plugin_service."+string(event.State), "plugin", event.PluginID, map[string]any{
		"code": event.Code,
	})
	return err
}

func (s pluginCatalogAuditSink) RecordPluginCatalogEvent(event plugin.CatalogAuditEvent) error {
	if s.auditSvc == nil {
		return nil
	}
	_, err := s.auditSvc.Append(context.Background(), "system", event.Action, "plugin", event.PluginID, map[string]any{
		"result":      event.Result,
		"permissions": event.Permissions,
		"menus":       event.Menus,
	})
	return err
}

func hasPluginManifest(path string) (bool, error) {
	manifestPath := filepath.Join(path, "plugin.yaml")
	if _, err := os.Stat(manifestPath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

type pluginManagerWithExtensions struct {
	plugin.Manager
	lifecycleMu        sync.RWMutex
	mu                 sync.RWMutex
	builtinInfos       map[string]plugin.Info
	extensions         map[string]plugin.RegistrySnapshot
	routeHandlers      map[string]http.HandlerFunc
	pluginsRepo        pluginrepo.PluginRepository
	routePermissions   *plugin.RoutePermissionRegistry
	healthChecker      plugin.HealthChecker
	healthMu           sync.RWMutex
	healthCache        map[string]plugin.HealthReport
	healthTTL          time.Duration
	inProcessBackends  *plugin.InProcessBackendRegistry
	serviceSupervisor  *plugin.ServiceSupervisor
	hostGateway        *plugin.HostGateway
	migrationHook      *plugin.PluginMigrationHook
	dataLifecycle      *datastore.Lifecycle
	dataDirectories    *plugin.PluginDataDirectories
	businessEvents     *event.BusinessEventBus
	eventDelivery      plugin.EventDeliveryClient
	eventSubscriptions map[string][]func()
}

var _ plugin.RoutePermissionResolver = (*pluginManagerWithExtensions)(nil)
var _ plugin.HealthProvider = (*pluginManagerWithExtensions)(nil)

func (m *pluginManagerWithExtensions) ReloadPluginMetadata(pluginID string) error {
	if m == nil {
		return plugin.ErrPluginNotFound
	}
	pluginID = strings.TrimSpace(pluginID)
	if pluginID == "" {
		return plugin.ErrPluginNotFound
	}
	m.lifecycleMu.Lock()
	defer m.lifecycleMu.Unlock()
	if m.inProcessBackends != nil {
		if err := m.inProcessBackends.Reset(pluginID); err != nil {
			return err
		}
	}
	current, currentErr := m.getCurrent(pluginID)
	serviceWasRunning := currentErr == nil && current.State == plugin.StateEnabled && strings.TrimSpace(current.ServiceBaseURL) != "" && m.serviceSupervisor != nil
	if serviceWasRunning {
		if err := m.serviceSupervisor.Stop(context.Background(), pluginID); err != nil {
			return err
		}
	}
	reloader, ok := m.Manager.(interface {
		ReloadPluginMetadata(pluginID string) error
	})
	if !ok {
		return nil
	}
	if err := reloader.ReloadPluginMetadata(pluginID); err != nil {
		if serviceWasRunning {
			_ = m.Manager.Disable(pluginID)
			m.removePluginEventSubscriptions(pluginID)
			_ = m.refreshRoutePermissions()
			m.clearHealthCache()
			m.persistOne(context.Background(), pluginID)
		}
		return err
	}
	next, err := m.getCurrent(pluginID)
	if err != nil {
		return err
	}
	if err := m.runPluginDataLifecycle(context.Background(), next, plugin.PluginMigrationUpgrade); err != nil {
		if current.State == plugin.StateEnabled {
			_ = m.Manager.Disable(pluginID)
			m.removePluginEventSubscriptions(pluginID)
			_ = m.refreshRoutePermissions()
			m.clearHealthCache()
			m.persistOne(context.Background(), pluginID)
		}
		return err
	}
	if err := m.refreshRoutePermissions(); err != nil {
		if current.State == plugin.StateEnabled {
			_ = m.Manager.Disable(pluginID)
			m.removePluginEventSubscriptions(pluginID)
			_ = m.refreshRoutePermissions()
			m.clearHealthCache()
			m.persistOne(context.Background(), pluginID)
		}
		return err
	}
	if serviceWasRunning {
		if err := m.serviceSupervisor.Start(context.Background(), next); err != nil {
			_ = m.Manager.Disable(pluginID)
			m.removePluginEventSubscriptions(pluginID)
			_ = m.refreshRoutePermissions()
			m.clearHealthCache()
			m.persistOne(context.Background(), pluginID)
			return err
		}
	}
	m.removePluginEventSubscriptions(pluginID)
	if next.State == plugin.StateEnabled {
		if err := m.registerPluginEventSubscriptions(next); err != nil {
			_ = m.Manager.Disable(pluginID)
			m.removePluginEventSubscriptions(pluginID)
			_ = m.refreshRoutePermissions()
			_ = m.stopPluginService(pluginID)
			m.persistOne(context.Background(), pluginID)
			return err
		}
	}
	m.clearHealthCache()
	m.persistOne(context.Background(), pluginID)
	return nil
}

func (m *pluginManagerWithExtensions) Install(path string) (plugin.Info, error) {
	info, err := m.Manager.Install(path)
	if err != nil {
		return plugin.Info{}, err
	}
	if _, err = m.preparePluginDataStore(info); err != nil {
		_ = m.Manager.Uninstall(info.ID)
		return plugin.Info{}, err
	}
	m.persistOne(context.Background(), info.ID)
	return info, nil
}

func (m *pluginManagerWithExtensions) HandlePluginRoute(pluginID, method, path string, w http.ResponseWriter, r *http.Request) bool {
	if m == nil || w == nil || r == nil {
		return false
	}
	m.lifecycleMu.RLock()
	defer m.lifecycleMu.RUnlock()
	key := pluginRouteKey(pluginID, method, path)
	m.mu.RLock()
	h, ok := m.routeHandlers[key]
	m.mu.RUnlock()
	if ok {
		h(w, r)
		return true
	}

	if m.Manager == nil {
		return false
	}
	info, err := m.Manager.Get(strings.TrimSpace(pluginID))
	if err != nil {
		return false
	}
	declared, err := pluginRouteDeclared(info, method, path)
	if err != nil {
		httpHandler.WriteMessage(w, http.StatusServiceUnavailable, "plugin_manifest_invalid", "插件清单无效")
		return true
	}
	if !declared {
		return false
	}
	if info.State != plugin.StateEnabled {
		httpHandler.WriteMessage(w, http.StatusServiceUnavailable, "plugin_not_enabled", "插件未启用")
		return true
	}

	if m.inProcessBackends != nil {
		handled, backendErr := m.inProcessBackends.ServeHTTP(pluginID, w, r)
		if handled {
			if backendErr != nil {
				httpHandler.WriteMessage(w, http.StatusServiceUnavailable, "plugin_backend_start_failed", "plugin backend failed to start")
			}
			return true
		}
	}

	target, err := parsePluginServiceURL(info.ServiceBaseURL)
	if err != nil {
		httpHandler.WriteMessage(w, http.StatusServiceUnavailable, "plugin_backend_not_configured", "插件后端未配置")
		return true
	}
	health := m.checkPluginHealth(r.Context(), info, false)
	if !health.Ready() {
		httpHandler.WriteResponse(w, http.StatusServiceUnavailable, "plugin_unhealthy", "插件后端未就绪", health)
		return true
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	director := proxy.Director
	proxy.Director = func(req *http.Request) {
		director(req)
		req.Host = target.Host
	}
	proxy.ErrorHandler = func(response http.ResponseWriter, _ *http.Request, _ error) {
		httpHandler.WriteMessage(response, http.StatusBadGateway, "plugin_backend_unavailable", "插件后端不可用")
	}
	proxy.ServeHTTP(w, r)
	return true
}

func (m *pluginManagerWithExtensions) CheckPluginHealth(ctx context.Context, pluginID string) (plugin.HealthReport, error) {
	if m == nil {
		return plugin.HealthReport{}, plugin.ErrPluginNotFound
	}
	pluginID = strings.TrimSpace(pluginID)
	if pluginID == "" {
		return plugin.HealthReport{}, plugin.ErrPluginNotFound
	}
	if m.Manager != nil {
		if info, err := m.Manager.Get(pluginID); err == nil {
			return m.checkPluginHealth(ctx, info, true), nil
		} else if !errors.Is(err, plugin.ErrPluginNotFound) {
			return plugin.HealthReport{}, err
		}
	}
	m.mu.RLock()
	info, ok := m.builtinInfos[pluginID]
	m.mu.RUnlock()
	if !ok {
		return plugin.HealthReport{}, plugin.ErrPluginNotFound
	}
	return m.checkPluginHealth(ctx, info, true), nil
}

func (m *pluginManagerWithExtensions) PluginReadiness(ctx context.Context) plugin.ReadinessReport {
	checkedAt := time.Now().UTC()
	if m == nil || m.Manager == nil {
		return plugin.ReadinessReport{Ready: false, CheckedAt: checkedAt, Plugins: []plugin.HealthReport{}}
	}
	items := m.Manager.List()
	servicePlugins := make([]plugin.Info, 0, len(items))
	for _, item := range items {
		if item.State != plugin.StateEnabled {
			continue
		}
		if strings.TrimSpace(item.ServiceBaseURL) == "" && strings.TrimSpace(item.ServiceHealthURL) == "" {
			continue
		}
		servicePlugins = append(servicePlugins, item)
	}

	reports := make([]plugin.HealthReport, len(servicePlugins))
	var wg sync.WaitGroup
	for i := range servicePlugins {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			reports[index] = m.checkPluginHealth(ctx, servicePlugins[index], true)
		}(i)
	}
	wg.Wait()
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].PluginID < reports[j].PluginID
	})
	ready := true
	for _, report := range reports {
		if !report.Ready() {
			ready = false
			break
		}
	}
	return plugin.ReadinessReport{Ready: ready, CheckedAt: checkedAt, Plugins: reports}
}

func (m *pluginManagerWithExtensions) checkPluginHealth(ctx context.Context, info plugin.Info, force bool) plugin.HealthReport {
	if ctx == nil {
		ctx = context.Background()
	}
	pluginID := strings.TrimSpace(info.ID)
	if !force && m != nil && m.healthTTL > 0 {
		m.healthMu.RLock()
		cached, ok := m.healthCache[pluginID]
		m.healthMu.RUnlock()
		age := time.Since(cached.CheckedAt)
		if ok && age >= 0 && age <= m.healthTTL {
			return cached
		}
	}

	var report plugin.HealthReport
	if m == nil || m.healthChecker == nil {
		report = plugin.HealthReport{
			PluginID:  pluginID,
			Status:    plugin.HealthStatusUnhealthy,
			Code:      "health_checker_unavailable",
			CheckedAt: time.Now().UTC(),
		}
	} else {
		report = m.healthChecker.Check(ctx, info)
	}
	if m != nil {
		m.healthMu.Lock()
		if m.healthCache == nil {
			m.healthCache = make(map[string]plugin.HealthReport)
		}
		m.healthCache[pluginID] = report
		m.healthMu.Unlock()
	}
	return report
}

func pluginRouteDeclared(info plugin.Info, method, path string) (bool, error) {
	routes, err := info.RouteExtensions()
	if err != nil {
		return false, err
	}
	wantMethod := strings.ToUpper(strings.TrimSpace(method))
	wantPath := plugin.NormalizeEntryPath(path)
	for _, route := range routes {
		if strings.ToUpper(strings.TrimSpace(route.Method)) == wantMethod && plugin.MatchRoutePath(route.Path, wantPath) {
			return true, nil
		}
	}
	return false, nil
}

func parsePluginServiceURL(raw string) (*url.URL, error) {
	target, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || target == nil || target.Host == "" {
		return nil, plugin.ErrPluginManifestBroken
	}
	if target.Scheme != "http" && target.Scheme != "https" {
		return nil, plugin.ErrPluginManifestBroken
	}
	return target, nil
}

func (m *pluginManagerWithExtensions) GetExtensionSnapshot(pluginID string) (plugin.RegistrySnapshot, bool) {
	if m == nil {
		return plugin.RegistrySnapshot{}, false
	}
	m.lifecycleMu.RLock()
	defer m.lifecycleMu.RUnlock()
	m.mu.RLock()
	snapshot, ok := m.extensions[pluginID]
	m.mu.RUnlock()
	if ok {
		return snapshot, true
	}
	if m.Manager == nil {
		return plugin.RegistrySnapshot{}, false
	}
	info, err := m.Manager.Get(pluginID)
	if err != nil || info.State != plugin.StateEnabled {
		return plugin.RegistrySnapshot{}, false
	}
	routes, err := info.RouteExtensions()
	if err != nil {
		return plugin.RegistrySnapshot{}, false
	}
	snapshot.Routes = routes
	for _, subscription := range info.EventSubscriptions() {
		snapshot.Events = append(snapshot.Events, subscription.Name)
	}
	return snapshot, true
}

func (m *pluginManagerWithExtensions) ResolveRoutePermission(method, path string) (plugin.RoutePermissionDescriptor, bool) {
	if m == nil {
		return plugin.RoutePermissionDescriptor{}, false
	}
	m.lifecycleMu.RLock()
	defer m.lifecycleMu.RUnlock()
	m.mu.RLock()
	registry := m.routePermissions
	m.mu.RUnlock()
	return registry.ResolveRoutePermission(method, path)
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
	if m == nil {
		return plugin.ErrPluginNotFound
	}
	m.lifecycleMu.Lock()
	defer m.lifecycleMu.Unlock()
	if err := m.runEnableMigrations(context.Background(), pluginID); err != nil {
		return err
	}
	serviceStarted := false
	if info, err := m.getCurrent(pluginID); err == nil && info.State != plugin.StateEnabled && strings.TrimSpace(info.ServiceBaseURL) != "" && m.serviceSupervisor != nil {
		probeInfo := info
		probeInfo.State = plugin.StateEnabled
		if err := m.serviceSupervisor.Start(context.Background(), probeInfo); err != nil {
			return err
		}
		serviceStarted = true
	}
	if err := m.Manager.Enable(pluginID); err == nil {
		if err := m.refreshRoutePermissions(); err != nil {
			_ = m.Manager.Disable(pluginID)
			_ = m.refreshRoutePermissions()
			if serviceStarted {
				_ = m.serviceSupervisor.Stop(context.Background(), pluginID)
			}
			return err
		}
		if err := m.registerEnabledPluginEventSubscriptions(); err != nil {
			_ = m.Manager.Disable(pluginID)
			m.removePluginEventSubscriptions(pluginID)
			_ = m.refreshRoutePermissions()
			if serviceStarted {
				_ = m.serviceSupervisor.Stop(context.Background(), pluginID)
			}
			return err
		}
		m.clearHealthCache()
		m.persistOne(context.Background(), pluginID)
		return nil
	} else if !errors.Is(err, plugin.ErrPluginNotFound) {
		if serviceStarted {
			_ = m.serviceSupervisor.Stop(context.Background(), pluginID)
		}
		return err
	}

	if m.pluginsRepo != nil {
		stored, storedErr := m.pluginsRepo.Get(context.Background(), pluginID)
		if storedErr == nil && stored != nil {
			now := time.Now().UTC()
			stored.State = plugin.StateEnabled
			stored.EnabledAt = &now
			_ = m.pluginsRepo.Save(context.Background(), *stored)
			if err := m.registerPluginEventSubscriptions(*stored); err != nil {
				stored.State = plugin.StateDisabled
				stored.EnabledAt = nil
				_ = m.pluginsRepo.Save(context.Background(), *stored)
				_ = m.stopPluginService(pluginID)
				return err
			}
			m.clearHealthCache()
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
	if err := m.registerPluginEventSubscriptions(item); err != nil {
		item.State = plugin.StateDisabled
		item.EnabledAt = nil
		m.builtinInfos[pluginID] = item
		return err
	}
	m.clearHealthCache()
	m.persistOne(context.Background(), pluginID)
	return nil
}

func (m *pluginManagerWithExtensions) Disable(pluginID string) error {
	if m == nil {
		return plugin.ErrPluginNotFound
	}
	m.lifecycleMu.Lock()
	defer m.lifecycleMu.Unlock()
	if err := m.Manager.Disable(pluginID); err == nil {
		m.removePluginEventSubscriptions(pluginID)
		if err := m.refreshRoutePermissions(); err != nil {
			_ = m.stopPluginService(pluginID)
			return err
		}
		m.clearHealthCache()
		m.persistOne(context.Background(), pluginID)
		if err := m.stopPluginService(pluginID); err != nil {
			return err
		}
		return m.stopInProcessBackend(pluginID)
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
			m.removePluginEventSubscriptions(pluginID)
			m.clearHealthCache()
			if err := m.stopPluginService(pluginID); err != nil {
				return err
			}
			return m.stopInProcessBackend(pluginID)
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
	if m == nil {
		return plugin.ErrPluginNotFound
	}
	m.lifecycleMu.Lock()
	defer m.lifecycleMu.Unlock()
	info, err := m.getCurrent(pluginID)
	if err != nil {
		return err
	}
	if info.SystemBuiltin {
		return plugin.ErrPluginSystemProtected
	}

	managedByRuntime := false
	if _, runtimeErr := m.Manager.Get(pluginID); runtimeErr == nil {
		managedByRuntime = true
		if info.State == plugin.StateEnabled {
			if err := m.Manager.Disable(pluginID); err != nil {
				return err
			}
			m.removePluginEventSubscriptions(pluginID)
			if err := m.refreshRoutePermissions(); err != nil {
				return err
			}
		}
	} else if !errors.Is(runtimeErr, plugin.ErrPluginNotFound) {
		return runtimeErr
	} else if info.State == plugin.StateEnabled && m.pluginsRepo != nil {
		info.State = plugin.StateDisabled
		info.EnabledAt = nil
		if err := m.pluginsRepo.Save(context.Background(), info); err != nil {
			return err
		}
		m.removePluginEventSubscriptions(pluginID)
	}
	if err := m.stopPluginService(pluginID); err != nil {
		return err
	}
	if err := m.stopInProcessBackend(pluginID); err != nil {
		return err
	}
	candidate, err := m.preparePluginDataStore(info)
	if err != nil {
		return err
	}
	transformSQL, err := m.pluginMigrationTransformer(candidate)
	if err != nil {
		return err
	}
	if err := m.runPluginMigrationsTransformed(context.Background(), info, plugin.PluginMigrationUninstall, transformSQL); err != nil {
		return err
	}
	if m.dataLifecycle != nil && info.DataManifest != nil {
		if err := m.dataLifecycle.Uninstall(context.Background(), pluginID, info.DataManifest.UninstallPolicy); err != nil {
			return err
		}
	}
	if info.DataManifest != nil {
		if m.dataDirectories == nil {
			return errors.New("plugin data directories are not configured")
		}
		if err := m.dataDirectories.Uninstall(pluginID, info.DataManifest.UninstallPolicy); err != nil {
			return err
		}
	}
	if managedByRuntime {
		if err := m.Manager.Uninstall(pluginID); err != nil {
			return err
		}
		if err := m.refreshRoutePermissions(); err != nil {
			return err
		}
	}
	if m.pluginsRepo != nil {
		if err := m.pluginsRepo.Delete(context.Background(), pluginID); err != nil {
			return err
		}
	}
	m.clearHealthCache()
	return nil
}

func (m *pluginManagerWithExtensions) PublishBusinessEvent(ctx context.Context, businessEvent event.BusinessEvent) error {
	if m == nil || m.businessEvents == nil {
		return errors.New("business event bus is not configured")
	}
	return m.businessEvents.Publish(ctx, businessEvent)
}

func (m *pluginManagerWithExtensions) RetryBusinessEvents(ctx context.Context, now time.Time) (int, error) {
	if m == nil || m.businessEvents == nil {
		return 0, errors.New("business event bus is not configured")
	}
	return m.businessEvents.RetryDue(ctx, now)
}

func (m *pluginManagerWithExtensions) BusinessEventRetryRecords() []event.BusinessRetryRecord {
	if m == nil || m.businessEvents == nil {
		return nil
	}
	return m.businessEvents.RetryRecords()
}

func (m *pluginManagerWithExtensions) BusinessEventDeliveryRecords() []event.BusinessDeliveryRecord {
	if m == nil || m.businessEvents == nil {
		return nil
	}
	return m.businessEvents.DeliveryRecords()
}

func (m *pluginManagerWithExtensions) registerEnabledPluginEventSubscriptions() error {
	if m == nil || m.Manager == nil {
		return nil
	}
	for _, info := range m.Manager.List() {
		if info.State != plugin.StateEnabled {
			continue
		}
		if err := m.registerPluginEventSubscriptions(info); err != nil {
			return err
		}
	}
	return nil
}

func (m *pluginManagerWithExtensions) registerPluginEventSubscriptions(info plugin.Info) error {
	if m == nil {
		return errors.New("plugin manager is not configured")
	}
	pluginID := strings.TrimSpace(info.ID)
	if pluginID == "" {
		return plugin.ErrPluginManifestBroken
	}
	subscriptions := info.EventSubscriptions()
	if len(subscriptions) == 0 {
		return nil
	}
	if m.businessEvents == nil {
		m.businessEvents = event.NewBusinessEventBus(nil)
	}
	if m.eventDelivery == nil {
		m.eventDelivery = plugin.NewHTTPEventDeliveryClient(nil, 5*time.Second)
	}
	if m.eventSubscriptions == nil {
		m.eventSubscriptions = make(map[string][]func())
	}
	if _, exists := m.eventSubscriptions[pluginID]; exists {
		return nil
	}
	unsubscribers := make([]func(), 0, len(subscriptions))
	for _, declared := range subscriptions {
		subscription := declared
		policy, err := event.ParseBusinessRetryPolicy(subscription.RetryPolicy)
		if err != nil {
			for _, unsubscribe := range unsubscribers {
				unsubscribe()
			}
			return plugin.ErrPluginManifestBroken
		}
		handlerName := pluginID + ":" + subscription.Handler
		unsubscribe, err := m.businessEvents.SubscribeWithPolicy(subscription.Name, handlerName, policy, func(ctx context.Context, businessEvent event.BusinessEvent) error {
			return m.deliverPluginBusinessEvent(ctx, pluginID, subscription, businessEvent)
		})
		if err != nil {
			for _, registered := range unsubscribers {
				registered()
			}
			return err
		}
		unsubscribers = append(unsubscribers, unsubscribe)
	}
	m.eventSubscriptions[pluginID] = unsubscribers
	return nil
}

func (m *pluginManagerWithExtensions) removePluginEventSubscriptions(pluginID string) {
	if m == nil {
		return
	}
	pluginID = strings.TrimSpace(pluginID)
	for _, unsubscribe := range m.eventSubscriptions[pluginID] {
		if unsubscribe != nil {
			unsubscribe()
		}
	}
	delete(m.eventSubscriptions, pluginID)
}

func (m *pluginManagerWithExtensions) deliverPluginBusinessEvent(ctx context.Context, pluginID string, subscription plugin.EventSubscription, businessEvent event.BusinessEvent) error {
	if m == nil || m.eventDelivery == nil {
		return errors.New("plugin event delivery is not configured")
	}
	m.lifecycleMu.RLock()
	defer m.lifecycleMu.RUnlock()
	info, err := m.getCurrent(pluginID)
	if err != nil || info.State != plugin.StateEnabled || !pluginEventSubscriptionDeclared(info, subscription) {
		return nil
	}
	return m.eventDelivery.Deliver(ctx, info, subscription, businessEvent)
}

func pluginEventSubscriptionDeclared(info plugin.Info, expected plugin.EventSubscription) bool {
	for _, subscription := range info.EventSubscriptions() {
		if subscription.Name == expected.Name && subscription.Handler == expected.Handler {
			return true
		}
	}
	return false
}

func (m *pluginManagerWithExtensions) runEnableMigrations(ctx context.Context, pluginID string) error {
	items := m.Manager.List()
	byID := make(map[string]plugin.Info, len(items))
	for _, item := range items {
		byID[item.ID] = item
	}
	if _, ok := byID[pluginID]; !ok {
		item, err := m.getCurrent(pluginID)
		if err != nil {
			return err
		}
		return m.runPluginDataLifecycle(ctx, item, plugin.PluginMigrationInstall)
	}
	order, err := plugin.NewTopologicalResolver().ResolveEnableOrder(pluginID, byID)
	if err != nil {
		return err
	}
	for _, id := range order {
		item := byID[id]
		if item.State == plugin.StateEnabled {
			continue
		}
		action := plugin.PluginMigrationUpgrade
		if item.State == plugin.StateInstalled {
			action = plugin.PluginMigrationInstall
		}
		if err := m.runPluginDataLifecycle(ctx, item, action); err != nil {
			return err
		}
	}
	return nil
}

func (m *pluginManagerWithExtensions) runPluginMigrations(ctx context.Context, info plugin.Info, action plugin.PluginMigrationAction) error {
	return m.runPluginMigrationsTransformed(ctx, info, action, nil)
}

func (m *pluginManagerWithExtensions) runPluginMigrationsTransformed(ctx context.Context, info plugin.Info, action plugin.PluginMigrationAction, transformSQL func(string) (string, error)) error {
	if info.DataManifest == nil || strings.TrimSpace(info.DataManifest.MigrationVersion) == "" {
		return nil
	}
	if m.migrationHook == nil {
		return fmt.Errorf("plugin %s requires a transactional migration store", info.ID)
	}
	_, err := m.migrationHook.Run(ctx, plugin.PluginMigrationHookInput{
		PluginID:           info.ID,
		PluginDir:          info.Source,
		MigrationDirectory: info.DataManifest.MigrationDirectory,
		Action:             action,
		ToVersion:          info.DataManifest.MigrationVersion,
		UninstallPolicy:    info.DataManifest.UninstallPolicy,
		RollbackPolicy:     info.DataManifest.RollbackPolicy,
		TransformSQL:       transformSQL,
	})
	return err
}

func (m *pluginManagerWithExtensions) preparePluginDataStore(info plugin.Info) (datastore.SchemaCandidate, error) {
	if m == nil || m.dataLifecycle == nil {
		return datastore.SchemaCandidate{}, nil
	}
	candidate, err := m.dataLifecycle.Prepare(info.ID, info.Source)
	if err != nil {
		return datastore.SchemaCandidate{}, fmt.Errorf("prepare plugin datastore schema: %w", err)
	}
	if candidate.Present && (info.DataManifest == nil || strings.TrimSpace(info.DataManifest.MigrationVersion) == "") {
		return datastore.SchemaCandidate{}, fmt.Errorf("plugin %s datastore schema requires migration metadata", info.ID)
	}
	return candidate, nil
}

func (m *pluginManagerWithExtensions) runPluginDataLifecycle(ctx context.Context, info plugin.Info, action plugin.PluginMigrationAction) error {
	candidate, err := m.preparePluginDataStore(info)
	if err != nil {
		return err
	}
	transformSQL, err := m.pluginMigrationTransformer(candidate)
	if err != nil {
		return err
	}
	if err = m.runPluginMigrationsTransformed(ctx, info, action, transformSQL); err != nil {
		return err
	}
	if m.dataLifecycle == nil {
		return nil
	}
	if err = m.dataLifecycle.ValidateStorage(ctx, candidate); err != nil {
		return fmt.Errorf("validate plugin datastore storage: %w", err)
	}
	if err = m.dataLifecycle.Activate(candidate); err != nil {
		return fmt.Errorf("activate plugin datastore schema: %w", err)
	}
	return nil
}

func (m *pluginManagerWithExtensions) pluginMigrationTransformer(candidate datastore.SchemaCandidate) (func(string) (string, error), error) {
	if m == nil || m.dataLifecycle == nil {
		return nil, nil
	}
	transformer, err := m.dataLifecycle.MigrationTransformer(candidate)
	if err != nil {
		return nil, fmt.Errorf("prepare plugin datastore migration bindings: %w", err)
	}
	return transformer, nil
}

func (m *pluginManagerWithExtensions) RollbackPluginData(pluginID string, limit int) error {
	if m == nil || m.migrationHook == nil || limit <= 0 {
		return errors.New("plugin datastore rollback requires a positive migration limit")
	}
	m.lifecycleMu.Lock()
	defer m.lifecycleMu.Unlock()
	info, err := m.getCurrent(strings.TrimSpace(pluginID))
	if err != nil {
		return err
	}
	if info.State == plugin.StateEnabled {
		return errors.New("plugin must be disabled before datastore rollback")
	}
	if info.DataManifest == nil || strings.TrimSpace(info.DataManifest.MigrationVersion) == "" {
		return errors.New("plugin does not declare datastore migrations")
	}
	candidate, err := m.preparePluginDataStore(info)
	if err != nil {
		return err
	}
	transformSQL, err := m.pluginMigrationTransformer(candidate)
	if err != nil {
		return err
	}
	_, err = m.migrationHook.Run(context.Background(), plugin.PluginMigrationHookInput{
		PluginID: info.ID, PluginDir: info.Source, MigrationDirectory: info.DataManifest.MigrationDirectory,
		Action: plugin.PluginMigrationDowngrade, FromVersion: info.DataManifest.MigrationVersion,
		RollbackPolicy: info.DataManifest.RollbackPolicy, Limit: limit,
		TransformSQL: transformSQL,
	})
	if err != nil {
		return err
	}
	if m.dataLifecycle != nil {
		return m.dataLifecycle.Uninstall(context.Background(), info.ID, plugin.DataUninstallRetain)
	}
	return nil
}

func (m *pluginManagerWithExtensions) PluginDataControl(ctx context.Context, pluginID string) (plugin.DataControlSnapshot, error) {
	if m == nil {
		return plugin.DataControlSnapshot{}, plugin.ErrPluginNotFound
	}
	m.lifecycleMu.Lock()
	defer m.lifecycleMu.Unlock()
	info, err := m.getCurrent(strings.TrimSpace(pluginID))
	if err != nil {
		return plugin.DataControlSnapshot{}, err
	}
	snapshot := plugin.DataControlSnapshot{
		PluginID: info.ID, CapturedAt: time.Now().UTC(), State: string(info.State),
		Schema:    plugin.DataControlSchema{Tables: []plugin.DataControlTable{}},
		Migration: plugin.DataControlMigration{Applied: []plugin.DataControlMigrationStep{}, Pending: []plugin.DataControlMigrationStep{}},
		Policy:    plugin.DataControlPolicy{Effect: "no_plugin_data"},
		Actions:   plugin.DataControlActions{BlockedReason: "migrations_not_declared"},
	}
	if info.DataManifest == nil {
		return snapshot, nil
	}
	manifest := info.DataManifest
	snapshot.Schema.Namespace = strings.TrimSpace(manifest.Namespace)
	snapshot.Migration.DeclaredVersion = strings.TrimSpace(manifest.MigrationVersion)
	snapshot.Policy = plugin.DataControlPolicy{
		Uninstall: string(manifest.UninstallPolicy), Rollback: string(manifest.RollbackPolicy),
		Effect: dataPolicyEffect(manifest.UninstallPolicy),
	}
	if m.dataLifecycle != nil {
		storage, exists, storageErr := m.dataLifecycle.InspectStorage(ctx, info.ID)
		if storageErr != nil {
			return plugin.DataControlSnapshot{}, storageErr
		}
		snapshot.Schema.Registered = exists
		if !exists {
			candidate, candidateErr := m.preparePluginDataStore(info)
			if candidateErr != nil {
				return plugin.DataControlSnapshot{}, candidateErr
			}
			storage, exists, storageErr = m.dataLifecycle.InspectCandidateStorage(ctx, candidate)
			if storageErr != nil {
				return plugin.DataControlSnapshot{}, storageErr
			}
		}
		if exists {
			snapshot.Schema.Available = true
			snapshot.Schema.Namespace = storage.Namespace
			snapshot.Schema.TotalSizeBytes = storage.TotalSizeBytes
			snapshot.Schema.SizeKnown = storage.SizeKnown
			for _, table := range storage.Tables {
				snapshot.Schema.Tables = append(snapshot.Schema.Tables, plugin.DataControlTable{
					LogicalName: table.LogicalName, PhysicalName: table.PhysicalName, Fields: append([]string(nil), table.Fields...),
					PrimaryKey: append([]string(nil), table.PrimaryKey...), IndexCount: table.IndexCount,
					Exists: table.Exists, SizeBytes: table.SizeBytes, SizeKnown: table.SizeKnown,
				})
			}
		}
	}
	if snapshot.Migration.DeclaredVersion == "" {
		return snapshot, nil
	}
	if m.migrationHook == nil {
		snapshot.Migration.Error = "plugin migration store is not configured"
		return snapshot, nil
	}
	candidate, err := m.preparePluginDataStore(info)
	if err != nil {
		snapshot.Migration.Error = err.Error()
		return snapshot, nil
	}
	transformSQL, err := m.pluginMigrationTransformer(candidate)
	if err != nil {
		snapshot.Migration.Error = err.Error()
		return snapshot, nil
	}
	plan, records, inspectErr := m.migrationHook.Inspect(ctx, plugin.PluginMigrationHookInput{
		PluginID: info.ID, PluginDir: info.Source, MigrationDirectory: manifest.MigrationDirectory, TransformSQL: transformSQL,
	})
	if inspectErr != nil {
		snapshot.Migration.Error = inspectErr.Error()
		return snapshot, nil
	}
	recordsByVersion := make(map[int]plugin.MigrationRecord, len(records))
	for _, record := range records {
		recordsByVersion[record.Version] = record
		if record.Version > snapshot.Migration.CurrentVersion {
			snapshot.Migration.CurrentVersion = record.Version
		}
	}
	for _, step := range plan.Applied {
		item := dataControlMigrationStep(step)
		if record, exists := recordsByVersion[step.Version]; exists {
			appliedAt := record.AppliedAt
			item.AppliedAt = &appliedAt
		}
		snapshot.Migration.Applied = append(snapshot.Migration.Applied, item)
	}
	for _, step := range plan.Pending {
		snapshot.Migration.Pending = append(snapshot.Migration.Pending, dataControlMigrationStep(step))
	}
	snapshot.Actions.RollbackMaxSteps = len(snapshot.Migration.Applied)
	switch {
	case info.State == plugin.StateEnabled:
		snapshot.Actions.BlockedReason = "plugin_must_be_disabled"
	case manifest.RollbackPolicy != plugin.DataRollbackAutomatic:
		snapshot.Actions.BlockedReason = "rollback_policy_blocked"
	case len(snapshot.Migration.Applied) == 0:
		snapshot.Actions.BlockedReason = "no_applied_migrations"
	default:
		snapshot.Actions.CanRollback = true
		snapshot.Actions.BlockedReason = ""
	}
	return snapshot, nil
}

func dataControlMigrationStep(step plugin.MigrationStep) plugin.DataControlMigrationStep {
	return plugin.DataControlMigrationStep{Version: step.Version, Name: step.Name, Checksum: step.Checksum}
}

func dataPolicyEffect(policy plugin.DataUninstallPolicy) string {
	switch policy {
	case plugin.DataUninstallRetain:
		return "retain_data"
	case plugin.DataUninstallArchive:
		return "archive_data"
	case plugin.DataUninstallDrop:
		return "drop_data"
	default:
		return "policy_not_declared"
	}
}

func (m *pluginManagerWithExtensions) refreshRoutePermissions() error {
	if m == nil || m.Manager == nil {
		return nil
	}
	routes := make([]plugin.RouteExtension, 0)
	for _, info := range m.Manager.List() {
		if info.State != plugin.StateEnabled {
			continue
		}
		extensions, err := info.RouteExtensions()
		if err != nil {
			return err
		}
		routes = append(routes, extensions...)
	}
	registry, err := plugin.NewRoutePermissionRegistry(routes)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.routePermissions = registry
	m.mu.Unlock()
	return nil
}

func mustEmptyRoutePermissionRegistry() *plugin.RoutePermissionRegistry {
	registry, err := plugin.NewRoutePermissionRegistry(nil)
	if err != nil {
		panic(err)
	}
	return registry
}

func (m *pluginManagerWithExtensions) clearHealthCache() {
	if m == nil {
		return
	}
	m.healthMu.Lock()
	clear(m.healthCache)
	m.healthMu.Unlock()
}

func (m *pluginManagerWithExtensions) stopPluginService(pluginID string) error {
	if m == nil || m.serviceSupervisor == nil {
		return nil
	}
	return m.serviceSupervisor.Stop(context.Background(), pluginID)
}

func (m *pluginManagerWithExtensions) stopInProcessBackend(pluginID string) error {
	if m == nil || m.inProcessBackends == nil {
		return nil
	}
	return m.inProcessBackends.Reset(pluginID)
}

func (m *pluginManagerWithExtensions) Close() error {
	if m == nil {
		return nil
	}
	m.lifecycleMu.Lock()
	for pluginID := range m.eventSubscriptions {
		m.removePluginEventSubscriptions(pluginID)
	}
	m.lifecycleMu.Unlock()
	if m.inProcessBackends != nil {
		if err := m.inProcessBackends.Close(); err != nil {
			return err
		}
	}
	var closeErr error
	if m.serviceSupervisor != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		closeErr = errors.Join(closeErr, m.serviceSupervisor.Shutdown(ctx))
		cancel()
	}
	if m.hostGateway != nil {
		closeErr = errors.Join(closeErr, m.hostGateway.Close())
	}
	return closeErr
}

func (m *pluginManagerWithExtensions) RegisterInProcessBackend(pluginID string, factory plugin.InProcessBackendFactory) error {
	if m == nil || m.inProcessBackends == nil {
		return errors.New("in-process plugin backends are not configured")
	}
	return m.inProcessBackends.Register(pluginID, factory)
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

func (m *pluginManagerWithExtensions) SavePluginConfig(pluginID string, config map[string]any) error {
	if m == nil || m.pluginsRepo == nil {
		return errors.New("plugin persistence is not configured")
	}
	item, err := m.getCurrent(strings.TrimSpace(pluginID))
	if err != nil {
		return err
	}
	raw, err := json.Marshal(config)
	if err != nil {
		return err
	}
	item.ConfigJSON = string(raw)
	return m.pluginsRepo.Save(context.Background(), item)
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
	if m.pluginsRepo != nil {
		item, repoErr := m.pluginsRepo.Get(context.Background(), pluginID)
		if repoErr == nil && item != nil {
			return *item, nil
		}
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
			frontendEntry = "/skoll/plugins/auth"
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
			Level:         plugin.LevelSystem,
			MountPolicy:   plugin.MountPolicyAdmin,
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
		token, err := security.SignJWT(jwtSecret, security.JWTIdentity{Subject: req.Account, Role: "super_admin", Roles: []string{"super_admin"}}, time.Hour, time.Now().UTC())
		if err != nil {
			httpHandler.WriteError(w, http.StatusInternalServerError, err)
			return
		}

		httpHandler.WriteJSON(w, http.StatusOK, map[string]any{
			"token":       token,
			"tokenType":   "Bearer",
			"expiresIn":   3600,
			"permissions": []string{"menu.read", "role.manage", "permission.manage"},
			"user": map[string]any{
				"id":               req.Account,
				"account":          req.Account,
				"name":             req.Account,
				"email":            "",
				"role":             "super_admin",
				"roles":            []string{"super_admin"},
				"organizationId":   "",
				"organizationPath": []string{},
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
