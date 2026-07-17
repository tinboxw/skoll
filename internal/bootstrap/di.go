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
	httpHandler "github.com/tinboxw/skoll/internal/handler/http"
	"github.com/tinboxw/skoll/internal/handler/middleware"
	"github.com/tinboxw/skoll/internal/plugin"
	builtinAuth "github.com/tinboxw/skoll/internal/plugin/builtin/auth"
	builtinDashboard "github.com/tinboxw/skoll/internal/plugin/builtin/dashboard"
	builtinLogger "github.com/tinboxw/skoll/internal/plugin/builtin/logger"
	pluginrepo "github.com/tinboxw/skoll/internal/repository/plugin"
	rbacrepo "github.com/tinboxw/skoll/internal/repository/rbac"
	rolerepo "github.com/tinboxw/skoll/internal/repository/role"
	userrepo "github.com/tinboxw/skoll/internal/repository/user"
	"github.com/tinboxw/skoll/internal/service/audit"
	filesvc "github.com/tinboxw/skoll/internal/service/file"
	menusvc "github.com/tinboxw/skoll/internal/service/menu"
	notificationsvc "github.com/tinboxw/skoll/internal/service/notification"
	permissionsvc "github.com/tinboxw/skoll/internal/service/permission"
	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
	"github.com/tinboxw/skoll/internal/service/rbac"
	"github.com/tinboxw/skoll/internal/service/role"
	"github.com/tinboxw/skoll/internal/service/system"
	"github.com/tinboxw/skoll/internal/service/user"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
	"github.com/tinboxw/skoll/internal/store"
	objectstore "github.com/tinboxw/skoll/internal/store/object"
	"github.com/tinboxw/skoll/pkg/config"
	"github.com/tinboxw/skoll/pkg/logging"
	"github.com/tinboxw/skoll/pkg/security"
)

type dependencies struct {
	logger   logging.Logger
	handler  http.Handler
	server   *http.Server
	eventBus event.Bus
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
	userService := user.NewService(bundle.Users, bundle.Audit, bundle.UnitOfWork)
	roleService := role.NewService(bundle.Roles)
	rbacService := rbac.NewService(bundle.RBAC)
	systemService := system.NewService(bundle.System)
	permissionService := permissionsvc.NewService(bundle.Permissions)
	menuService := menusvc.NewService(bundle.Menus)
	pharmaEmployeeService := pharmaoasvc.NewEmployeeService(auditService, bundle.PharmaEmployees)
	pharmaProductService := pharmaoasvc.NewProductService(auditService, bundle.PharmaProducts)
	pharmaSupplierService := pharmaoasvc.NewSupplierService(auditService, bundle.PharmaSuppliers)
	pharmaCustomerService := pharmaoasvc.NewCustomerService(auditService, bundle.PharmaCustomers)
	pharmaCustomerFollowUpService := pharmaoasvc.NewCustomerFollowUpService(pharmaCustomerService, auditService)
	pharmaSalesOpportunityService := pharmaoasvc.NewSalesOpportunityService(pharmaCustomerService, pharmaProductService, auditService)
	pharmaWarehouseService := pharmaoasvc.NewWarehouseService(auditService, bundle.PharmaWarehouses)
	pharmaMasterDataExchangeService := pharmaoasvc.NewMasterDataExchangeService(pharmaEmployeeService, pharmaProductService, pharmaSupplierService, pharmaCustomerService)
	workflowService := workflowsvc.NewService(workflowsvc.NewMemoryRepository())
	objectStore, err := objectstore.NewLocalStore(filepath.Join("data", "objects"))
	if err != nil {
		return nil, err
	}
	fileService := filesvc.NewService(bundle.Files, objectStore, filesvc.Options{
		Permission: rbacService,
		Audit:      auditEventService,
	})
	pharmaPurchaseService := pharmaoasvc.NewPurchaseService(pharmaSupplierService, workflowService, auditService, bundle.PharmaPurchases)
	pharmaInventoryService := pharmaoasvc.NewInventoryService(auditService, bundle.PharmaInventory)
	pharmaPurchaseInboundService := pharmaoasvc.NewPurchaseInboundService(pharmaPurchaseService, pharmaWarehouseService, pharmaInventoryService, auditService, bundle.PharmaInbounds)
	pharmaSalesService := pharmaoasvc.NewSalesService(pharmaCustomerService, pharmaWarehouseService, pharmaInventoryService, auditService, bundle.PharmaSales)
	pharmaInventoryOperationService := pharmaoasvc.NewInventoryOperationService(pharmaInventoryService, pharmaWarehouseService, workflowService, auditService, pharmaoasvc.InventoryOperationRepositories{Stocktakes: bundle.PharmaStocktakes, Transfers: bundle.PharmaTransfers})
	notificationService := notificationsvc.NewService(nil, nil)
	pharmaPaymentInvoiceService := pharmaoasvc.NewPaymentInvoiceService(pharmaSalesService, notificationService, auditService)
	pharmaInventoryAlertService := pharmaoasvc.NewInventoryAlertService(pharmaInventoryService, notificationService, auditService)
	pharmaAnnouncementService := pharmaoasvc.NewAnnouncementService(auditService)
	pharmaContractService := pharmaoasvc.NewContractService(pharmaSupplierService, pharmaCustomerService, workflowService, fileService, notificationService, auditService)
	pharmaQualificationService := pharmaoasvc.NewQualificationService(pharmaEmployeeService, pharmaSupplierService, pharmaCustomerService, notificationService, auditService)
	pharmaQualityComplaintService := pharmaoasvc.NewQualityComplaintService(pharmaCustomerService, pharmaProductService, pharmaInventoryService, workflowService, fileService, auditService)
	pharmaDrugRecallService := pharmaoasvc.NewDrugRecallService(pharmaSalesService, pharmaInventoryService, pharmaProductService, pharmaCustomerService, pharmaQualityComplaintService, auditService)
	pharmaColdChainService := pharmaoasvc.NewColdChainService(pharmaInventoryService, pharmaWarehouseService, notificationService, auditService)
	pharmaComplianceDashboardService := pharmaoasvc.NewComplianceDashboardService(pharmaQualificationService, pharmaQualityComplaintService, pharmaDrugRecallService, pharmaColdChainService, auditService)
	pharmaBusinessMetricsService := pharmaoasvc.NewBusinessMetricsService(pharmaInventoryAlertService, pharmaQualificationService, pharmaPurchaseService, pharmaCustomerFollowUpService, pharmaSalesService, auditService)
	pharmaReportExportService := pharmaoasvc.NewReportExportService(pharmaBusinessMetricsService, fileService, auditService)
	pharmaDemoSeedService := pharmaoasvc.NewDemoSeedService(pharmaoasvc.DemoSeedDependencies{
		Employees: pharmaEmployeeService, Products: pharmaProductService, Suppliers: pharmaSupplierService, Customers: pharmaCustomerService,
		Warehouses: pharmaWarehouseService, Purchases: pharmaPurchaseService, Inbounds: pharmaPurchaseInboundService, Sales: pharmaSalesService,
		Inventory: pharmaInventoryService, FollowUps: pharmaCustomerFollowUpService, Audit: auditService,
	})
	pluginManager := newPluginManager(logger, cfg.AppConfig.Security.JWTSecret, bundle.Users, bundle.Roles, bundle.RBAC, bundle.Plugins, auditService, auditEventService)

	router := httpHandler.NewRouter(httpHandler.Dependencies{
		UserService:                      userService,
		RoleService:                      roleService,
		RBACService:                      rbacService,
		AuditService:                     auditService,
		AuditEventService:                auditEventService,
		FileService:                      fileService,
		SystemService:                    systemService,
		PermissionService:                permissionService,
		PharmaEmployeeService:            pharmaEmployeeService,
		PharmaProductService:             pharmaProductService,
		PharmaSupplierService:            pharmaSupplierService,
		PharmaCustomerService:            pharmaCustomerService,
		PharmaCustomerFollowUpService:    pharmaCustomerFollowUpService,
		PharmaSalesOpportunityService:    pharmaSalesOpportunityService,
		PharmaWarehouseService:           pharmaWarehouseService,
		PharmaMasterDataExchangeService:  pharmaMasterDataExchangeService,
		PharmaPurchaseService:            pharmaPurchaseService,
		PharmaPurchaseInboundService:     pharmaPurchaseInboundService,
		PharmaSalesService:               pharmaSalesService,
		PharmaPaymentInvoiceService:      pharmaPaymentInvoiceService,
		PharmaInventoryOperationService:  pharmaInventoryOperationService,
		PharmaInventoryAlertService:      pharmaInventoryAlertService,
		PharmaAnnouncementService:        pharmaAnnouncementService,
		PharmaContractService:            pharmaContractService,
		PharmaQualificationService:       pharmaQualificationService,
		PharmaQualityComplaintService:    pharmaQualityComplaintService,
		PharmaDrugRecallService:          pharmaDrugRecallService,
		PharmaColdChainService:           pharmaColdChainService,
		PharmaComplianceDashboardService: pharmaComplianceDashboardService,
		PharmaBusinessMetricsService:     pharmaBusinessMetricsService,
		PharmaReportExportService:        pharmaReportExportService,
		PharmaDemoSeedService:            pharmaDemoSeedService,
		MenuService:                      menuService,
		WorkflowService:                  workflowService,
		PluginManager:                    pluginManager,
		APIPrefix:                        cfg.AppConfig.Server.APIPrefix,
		LogLevel:                         cfg.AppConfig.Log.Level,
		LogDir:                           cfg.AppConfig.Log.Dir,
		LogFile:                          cfg.AppConfig.Log.File,
		LogPluginPerFile:                 cfg.AppConfig.Log.PluginPerFile,
		DevPortalEnabled:                 cfg.AppConfig.Dev.PortalEnabled,
		DevPortalRoot:                    cfg.AppConfig.Dev.PluginsRoot,
		DevPortalRoots:                   append([]string(nil), cfg.AppConfig.Dev.PluginsRoots...),
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

	return &dependencies{logger: logger, handler: h, server: server, eventBus: bus}, nil
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

func newPluginManager(logger logging.Logger, jwtSecret string, usersRepo userrepo.UserRepository, rolesRepo rolerepo.RoleRepository, rbacRepo rbacrepo.RBACRepository, pluginsRepo pluginrepo.PluginRepository, auditSvc audit.Service, auditEventSvc audit.EventService) plugin.Manager {
	runtimeManager := plugin.NewRuntimeManager(plugin.NewFileLoader(), plugin.NewTopologicalResolver())
	runtimeManager.SetCatalogRegistry(plugin.NewMemoryCatalogRegistry())
	runtimeManager.SetCatalogAuditSink(pluginCatalogAuditSink{auditSvc: auditSvc})
	authHandler := newBuiltinAuthHandler(jwtSecret, usersRepo, rolesRepo, rbacRepo, auditSvc, auditEventSvc, logger)
	builtinInfos, extensions, handlers := registerBuiltinPluginExtensions(logger, jwtSecret, authHandler)
	m := &pluginManagerWithExtensions{
		Manager:          runtimeManager,
		builtinInfos:     builtinInfos,
		extensions:       extensions,
		routeHandlers:    handlers,
		pluginsRepo:      pluginsRepo,
		routePermissions: mustEmptyRoutePermissionRegistry(),
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

	return m
}

type pluginCatalogAuditSink struct {
	auditSvc audit.Service
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
	mu               sync.RWMutex
	builtinInfos     map[string]plugin.Info
	extensions       map[string]plugin.RegistrySnapshot
	routeHandlers    map[string]http.HandlerFunc
	pluginsRepo      pluginrepo.PluginRepository
	routePermissions *plugin.RoutePermissionRegistry
}

var _ plugin.RoutePermissionResolver = (*pluginManagerWithExtensions)(nil)

func (m *pluginManagerWithExtensions) ReloadPluginMetadata(pluginID string) error {
	if m == nil {
		return plugin.ErrPluginNotFound
	}
	pluginID = strings.TrimSpace(pluginID)
	if pluginID == "" {
		return plugin.ErrPluginNotFound
	}
	reloader, ok := m.Manager.(interface {
		ReloadPluginMetadata(pluginID string) error
	})
	if !ok {
		return nil
	}
	if err := reloader.ReloadPluginMetadata(pluginID); err != nil {
		return err
	}
	if err := m.refreshRoutePermissions(); err != nil {
		return err
	}
	m.persistOne(context.Background(), pluginID)
	return nil
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
	if err := m.Manager.Enable(pluginID); err == nil {
		if err := m.refreshRoutePermissions(); err != nil {
			return err
		}
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
		if err := m.refreshRoutePermissions(); err != nil {
			return err
		}
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
		if err := m.refreshRoutePermissions(); err != nil {
			return err
		}
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
