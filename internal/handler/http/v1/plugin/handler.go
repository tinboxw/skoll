package plugin

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tinboxw/skoll/internal/adapter"
	domainrole "github.com/tinboxw/skoll/internal/domain/role"
	apiv1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	auditmw "github.com/tinboxw/skoll/internal/handler/middleware"
	"github.com/tinboxw/skoll/internal/plugin"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	rolesvc "github.com/tinboxw/skoll/internal/service/role"
	"github.com/tinboxw/skoll/pkg/logging"
	"github.com/tinboxw/skoll/pkg/security"
)

type PluginManager interface {
	Install(path string) (plugin.Info, error)
	List() []plugin.Info
	Enable(pluginID string) error
	Disable(pluginID string) error
	Uninstall(pluginID string) error
	Get(pluginID string) (plugin.Info, error)
}

type PluginExtensionSnapshotProvider interface {
	GetExtensionSnapshot(pluginID string) (plugin.RegistrySnapshot, bool)
}

type PluginExternalRegistrar interface {
	RegisterExternalPlugin(info plugin.Info) error
}

type PluginConfigUpdater interface {
	SavePluginConfig(pluginID string, config map[string]any) error
}

type PluginMetadataRefresher interface {
	ReloadPluginMetadata(pluginID string) error
}

type PluginRoleCatalogProvider interface {
	List(ctx context.Context, in rolesvc.ListInput) ([]*domainrole.Role, error)
}

type PluginHandler struct {
	manager            PluginManager
	extensionProvider  PluginExtensionSnapshotProvider
	healthProvider     plugin.HealthProvider
	loader             plugin.MetadataLoader
	logger             logging.Logger
	auditSvc           auditsvc.Service
	auditEventSink     auditmw.AuditEventSink
	roleCatalog        PluginRoleCatalogProvider
	devRolloutExecutor adapter.DevRolloutExecutor
	logLevel           string
	logDir             string
	logFile            string
	pluginPerFile      bool
	devPortalEnabled   bool
	devPortalRoot      string
	devPortalRoots     []string
	devMu              sync.Mutex
}

type PluginRouteOption func(*PluginHandler)

func WithPluginLogTarget(logLevel, logDir, logFile string, pluginPerFile bool) PluginRouteOption {
	return func(h *PluginHandler) {
		h.logLevel = strings.TrimSpace(logLevel)
		h.logDir = strings.TrimSpace(logDir)
		h.logFile = strings.TrimSpace(logFile)
		h.pluginPerFile = pluginPerFile
	}
}

func WithPluginAuditService(auditSvc auditsvc.Service) PluginRouteOption {
	return func(h *PluginHandler) {
		h.auditSvc = auditSvc
	}
}

func WithPluginAuditEventSink(sink auditmw.AuditEventSink) PluginRouteOption {
	return func(h *PluginHandler) {
		h.auditEventSink = sink
	}
}

func WithPluginRoleCatalogProvider(provider PluginRoleCatalogProvider) PluginRouteOption {
	return func(h *PluginHandler) {
		h.roleCatalog = provider
	}
}

func WithPluginDevPortal(enabled bool, pluginsRoot string, pluginsRoots []string) PluginRouteOption {
	return func(h *PluginHandler) {
		h.devPortalEnabled = enabled
		h.devPortalRoot = strings.TrimSpace(pluginsRoot)
		h.devPortalRoots = append([]string(nil), pluginsRoots...)
	}
}

type pluginRecord struct {
	ID            string                    `json:"id"`
	Name          string                    `json:"name"`
	NameZhCN      string                    `json:"nameZhCN,omitempty"`
	NameEnUS      string                    `json:"nameEnUS,omitempty"`
	Version       string                    `json:"version"`
	Enabled       bool                      `json:"enabled"`
	UIMode        string                    `json:"uiMode"`
	Level         string                    `json:"level"`
	AppID         string                    `json:"appId,omitempty"`
	MountPolicy   string                    `json:"mountPolicy"`
	UINavPosition string                    `json:"uiNavPosition,omitempty"`
	UIOpenMode    string                    `json:"uiOpenMode,omitempty"`
	UITabMode     string                    `json:"uiTabMode,omitempty"`
	I18nLocales   []string                  `json:"i18nLocales,omitempty"`
	UIMenu        *pluginMenuRecord         `json:"uiMenu,omitempty"`
	ConfigSchema  *pluginConfigSchemaRecord `json:"configSchema,omitempty"`
	FrontendEntry string                    `json:"frontendEntry,omitempty"`
	SystemBuiltin bool                      `json:"systemBuiltin"`
}

type pluginMenuRecord struct {
	Label               string   `json:"label,omitempty"`
	LabelZhCN           string   `json:"labelZhCN,omitempty"`
	LabelEnUS           string   `json:"labelEnUS,omitempty"`
	Path                string   `json:"path,omitempty"`
	Icon                string   `json:"icon,omitempty"`
	Order               int      `json:"order,omitempty"`
	RequiredRoles       []string `json:"requiredRoles,omitempty"`
	RequiredPermissions []string `json:"requiredPermissions,omitempty"`
}

type pluginConfigSchemaRecord struct {
	Title       string                    `json:"title,omitempty"`
	TitleZhCN   string                    `json:"titleZhCN,omitempty"`
	TitleEnUS   string                    `json:"titleEnUS,omitempty"`
	Description string                    `json:"description,omitempty"`
	Fields      []pluginConfigFieldRecord `json:"fields,omitempty"`
}

type pluginConfigFieldRecord struct {
	Key         string                     `json:"key"`
	Label       string                     `json:"label,omitempty"`
	LabelZhCN   string                     `json:"labelZhCN,omitempty"`
	LabelEnUS   string                     `json:"labelEnUS,omitempty"`
	Type        string                     `json:"type,omitempty"`
	Required    bool                       `json:"required,omitempty"`
	Default     string                     `json:"default,omitempty"`
	Placeholder string                     `json:"placeholder,omitempty"`
	Help        string                     `json:"help,omitempty"`
	Min         *float64                   `json:"min,omitempty"`
	Max         *float64                   `json:"max,omitempty"`
	MinLength   *int                       `json:"minLength,omitempty"`
	MaxLength   *int                       `json:"maxLength,omitempty"`
	Pattern     string                     `json:"pattern,omitempty"`
	Options     []pluginConfigOptionRecord `json:"options,omitempty"`
}

type pluginConfigOptionRecord struct {
	Label     string `json:"label,omitempty"`
	LabelZhCN string `json:"labelZhCN,omitempty"`
	LabelEnUS string `json:"labelEnUS,omitempty"`
	Value     string `json:"value"`
}

type pluginDebugRecord struct {
	ID           string              `json:"id"`
	Name         string              `json:"name"`
	Version      string              `json:"version"`
	Description  string              `json:"description"`
	State        string              `json:"state"`
	Source       string              `json:"source"`
	Permissions  []string            `json:"permissions"`
	Dependencies []plugin.Dependency `json:"dependencies"`
	Extensions   any                 `json:"extensions,omitempty"`
}

type pluginLogsRecord struct {
	PluginID string `json:"pluginId"`
	Content  string `json:"content"`
}

type pluginPathRequest struct {
	Path string `json:"path"`
}

type devScaffoldRequest struct {
	PluginsRoot string `json:"pluginsRoot"`
	PluginID    string `json:"pluginId"`
	PluginName  string `json:"pluginName"`
	AppID       string `json:"appId"`
	Mode        string `json:"mode"`
}

type devValidateAllRequest struct {
	PluginsRoot string `json:"pluginsRoot"`
}

type devProjectActionRequest struct {
	PluginsRoot string `json:"pluginsRoot"`
	PluginID    string `json:"pluginId"`
	OutputDir   string `json:"outputDir"`
	RemoveFiles bool   `json:"removeFiles"`
}

type devManifestRequest struct {
	PluginsRoot string `json:"pluginsRoot"`
	PluginID    string `json:"pluginId"`
	Manifest    string `json:"manifest"`
}

type devPipelineRequest struct {
	PluginsRoot string `json:"pluginsRoot"`
	PluginID    string `json:"pluginId"`
	OutputDir   string `json:"outputDir"`
}

type devConfigResponse struct {
	Enabled      bool     `json:"enabled"`
	DefaultRoot  string   `json:"defaultRoot"`
	AllowedRoots []string `json:"allowedRoots"`
}

type devProjectRecord struct {
	PluginID      string `json:"pluginId"`
	Name          string `json:"name,omitempty"`
	NameZhCN      string `json:"nameZhCN,omitempty"`
	NameEnUS      string `json:"nameEnUS,omitempty"`
	Version       string `json:"version,omitempty"`
	Path          string `json:"path"`
	PluginsRoot   string `json:"pluginsRoot"`
	Status        string `json:"status"`
	Error         string `json:"error,omitempty"`
	Mode          string `json:"mode,omitempty"`
	Installed     bool   `json:"installed"`
	Enabled       bool   `json:"enabled"`
	PreviewURL    string `json:"previewUrl,omitempty"`
	BuildHint     string `json:"buildHint,omitempty"`
	PackageHint   string `json:"packageHint,omitempty"`
	PublishHint   string `json:"publishHint,omitempty"`
	LastUpdatedAt string `json:"lastUpdatedAt,omitempty"`
}

type devListProjectsResponse struct {
	Operation   string             `json:"operation"`
	Status      string             `json:"status"`
	PluginsRoot string             `json:"pluginsRoot"`
	Projects    []devProjectRecord `json:"projects"`
}

type devRemoveProjectResponse struct {
	Operation    string `json:"operation"`
	Status       string `json:"status"`
	PluginsRoot  string `json:"pluginsRoot"`
	PluginID     string `json:"pluginId"`
	PluginDir    string `json:"pluginDir"`
	Uninstalled  bool   `json:"uninstalled"`
	FilesRemoved bool   `json:"filesRemoved"`
}

type devPackageProjectResponse struct {
	Operation    string `json:"operation"`
	Status       string `json:"status"`
	PluginsRoot  string `json:"pluginsRoot"`
	PluginID     string `json:"pluginId"`
	PluginDir    string `json:"pluginDir"`
	ArtifactPath string `json:"artifactPath"`
}

type devManifestResponse struct {
	Operation    string `json:"operation"`
	Status       string `json:"status"`
	PluginsRoot  string `json:"pluginsRoot"`
	PluginID     string `json:"pluginId"`
	PluginDir    string `json:"pluginDir"`
	ManifestPath string `json:"manifestPath"`
	Manifest     string `json:"manifest"`
	Validation   string `json:"validation"`
	Error        string `json:"error,omitempty"`
}

type devPipelineStep struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
	Artifact   string `json:"artifactPath,omitempty"`
	DurationMs int64  `json:"durationMs"`
}

type devPipelineResponse struct {
	Operation   string            `json:"operation"`
	Status      string            `json:"status"`
	PluginsRoot string            `json:"pluginsRoot"`
	PluginID    string            `json:"pluginId"`
	PluginDir   string            `json:"pluginDir"`
	StartedAt   string            `json:"startedAt"`
	FinishedAt  string            `json:"finishedAt"`
	Steps       []devPipelineStep `json:"steps"`
}

type devValidationResult struct {
	Path    string `json:"path"`
	Plugin  string `json:"pluginId,omitempty"`
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
}

type devScaffoldResponse struct {
	Operation   string `json:"operation"`
	Status      string `json:"status"`
	PluginsRoot string `json:"pluginsRoot"`
	PluginID    string `json:"pluginId"`
	PluginName  string `json:"pluginName"`
	PluginDir   string `json:"pluginDir"`
	Mode        string `json:"mode"`
}

const (
	devScaffoldModeWorkspace  = "workspace"
	devScaffoldModeRepository = "repository"
)

type devValidateAllSummary struct {
	Total   int `json:"total"`
	Valid   int `json:"valid"`
	Invalid int `json:"invalid"`
}

type devValidateAllResponse struct {
	Operation   string                `json:"operation"`
	Status      string                `json:"status"`
	PluginsRoot string                `json:"pluginsRoot"`
	Summary     devValidateAllSummary `json:"summary"`
	Results     []devValidationResult `json:"results"`
}

type externalPluginRequest struct {
	PluginID  string `json:"pluginId"`
	Name      string `json:"name"`
	Version   string `json:"version"`
	URL       string `json:"url"`
	OpenMode  string `json:"openMode"`
	ActorID   string `json:"actorId"`
	RoutePath string `json:"routePath"`
}

func RegisterPluginRoutes(mux *http.ServeMux, manager PluginManager, opts ...PluginRouteOption) {
	h := &PluginHandler{manager: manager, loader: plugin.NewFileLoader(), devRolloutExecutor: &adapter.MockDevRolloutExecutor{}}
	for _, opt := range opts {
		if opt != nil {
			opt(h)
		}
	}
	if provider, ok := manager.(PluginExtensionSnapshotProvider); ok {
		h.extensionProvider = provider
	}
	if provider, ok := manager.(plugin.HealthProvider); ok {
		h.healthProvider = provider
	}
	if h.logLevel == "" {
		h.logLevel = "info"
	}
	h.logger = logging.New(h.logLevel)
	mux.HandleFunc("GET /v1/plugins/dev/config", h.devConfig)
	mux.HandleFunc("GET /v1/plugins/dev/permission-catalog", h.devPermissionCatalog)
	mux.HandleFunc("GET /v1/plugins/marketplace/local", h.localMarketplace)
	mux.HandleFunc("GET /v1/plugins", h.list)
	mux.HandleFunc("GET /v1/plugins/{id}", h.get)
	mux.HandleFunc("GET /v1/plugins/{id}/health", h.health)
	mux.HandleFunc("POST /v1/plugins/preflight", h.preflight)
	mux.HandleFunc("POST /v1/plugins/install", h.install)
	mux.HandleFunc("POST /v1/plugins/link", h.createLink)
	mux.HandleFunc("POST /v1/plugins/embed", h.createEmbed)
	mux.HandleFunc("POST /v1/plugins/validate", h.validate)
	mux.HandleFunc("POST /v1/plugins/{id}/enable", h.enable)
	mux.HandleFunc("POST /v1/plugins/{id}/disable", h.disable)
	mux.HandleFunc("DELETE /v1/plugins/{id}", h.uninstall)
	mux.HandleFunc("GET /v1/plugins/{id}/debug", h.debug)
	mux.HandleFunc("GET /v1/plugins/{id}/logs", h.logs)
	mux.HandleFunc("GET /v1/plugins/{id}/config", h.getConfig)
	mux.HandleFunc("PUT /v1/plugins/{id}/config", h.updateConfig)
	mux.HandleFunc("GET /v1/plugins/{id}/page", h.page)
	mux.HandleFunc("GET /v1/plugins/{id}/assets/{asset...}", h.asset)
	if h.devPortalEnabled {
		mux.HandleFunc("GET /v1/plugins/dev/manifest", h.devManifestGet)
		mux.HandleFunc("POST /v1/plugins/dev/manifest/validate", h.devManifestValidate)
		mux.HandleFunc("PUT /v1/plugins/dev/manifest", h.devManifestPut)
		mux.HandleFunc("POST /v1/plugins/dev/projects", h.devProjects)
		mux.HandleFunc("POST /v1/plugins/dev/remove", h.devRemove)
		mux.HandleFunc("POST /v1/plugins/dev/package", h.devPackage)
		mux.HandleFunc("POST /v1/plugins/dev/pipeline", h.devPipeline)
		mux.HandleFunc("POST /v1/plugins/dev/rollout", h.devRollout)
		mux.HandleFunc("POST /v1/plugins/dev/rollback", h.devRollback)
		mux.HandleFunc("GET /v1/plugins/dev/rollout-tasks", h.devListRolloutTasks)
		mux.HandleFunc("GET /v1/plugins/dev/rollout-tasks/{taskId}", h.devGetRolloutTask)
		mux.HandleFunc("GET /v1/plugins/dev/rollout-tasks/{taskId}/logs", h.devGetRolloutTaskLogs)
		mux.HandleFunc("POST /v1/plugins/dev/release-orders", h.devCreateReleaseOrder)
		mux.HandleFunc("GET /v1/plugins/dev/release-orders", h.devListReleaseOrders)
		mux.HandleFunc("POST /v1/plugins/dev/release-orders/{orderId}/approve", h.devApproveReleaseOrder)
		mux.HandleFunc("POST /v1/plugins/dev/release-orders/{orderId}/reject", h.devRejectReleaseOrder)
		mux.HandleFunc("POST /v1/plugins/dev/release-orders/{orderId}/execute", h.devExecuteReleaseOrder)
		mux.HandleFunc("GET /v1/plugins/dev/release-tasks", h.devListReleaseTasks)
		mux.HandleFunc("GET /v1/plugins/dev/release-tasks/{taskId}", h.devGetReleaseTask)
		mux.HandleFunc("GET /v1/plugins/dev/release-tasks/{taskId}/logs", h.devGetReleaseTaskLogs)
		mux.HandleFunc("POST /v1/plugins/dev/scaffold", h.devScaffold)
		mux.HandleFunc("POST /v1/plugins/dev/validate-all", h.devValidateAll)
	}
}

func (h *PluginHandler) list(w http.ResponseWriter, r *http.Request) {
	onlyEnabled := false
	if raw := r.URL.Query().Get("enabled"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			apiv1.WriteError(w, http.StatusBadRequest, err)
			return
		}
		onlyEnabled = v
	}

	records := h.records()
	if onlyEnabled {
		enabled := make([]pluginRecord, 0, len(records))
		for _, item := range records {
			if item.Enabled {
				enabled = append(enabled, item)
			}
		}
		if len(enabled) == 0 {
			records = defaultPluginRecords()
		} else {
			records = enabled
		}
	}

	apiv1.WriteJSON(w, http.StatusOK, records)
}

func (h *PluginHandler) get(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.manager == nil {
		apiv1.WriteError(w, http.StatusServiceUnavailable, errors.New("plugin manager not configured"))
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("plugin id is required"))
		return
	}

	item, err := h.manager.Get(id)
	if err != nil {
		if errors.Is(err, plugin.ErrPluginNotFound) {
			apiv1.WriteMessage(w, http.StatusNotFound, "not_found", "plugin not found")
			return
		}
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	apiv1.WriteJSON(w, http.StatusOK, pluginRecordFromInfo(item))
}

func (h *PluginHandler) health(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.healthProvider == nil {
		apiv1.WriteMessage(w, http.StatusServiceUnavailable, "plugin_health_unavailable", "插件健康检查不可用")
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		apiv1.WriteMessage(w, http.StatusBadRequest, "invalid_plugin_id", "插件 ID 不能为空")
		return
	}
	report, err := h.healthProvider.CheckPluginHealth(r.Context(), id)
	if err != nil {
		if errors.Is(err, plugin.ErrPluginNotFound) {
			apiv1.WriteMessage(w, http.StatusNotFound, "plugin_not_found", "插件不存在")
			return
		}
		apiv1.WriteMessage(w, http.StatusServiceUnavailable, "plugin_health_unavailable", "插件健康检查不可用")
		return
	}
	if !report.Ready() {
		apiv1.WriteResponse(w, http.StatusServiceUnavailable, "plugin_unhealthy", "插件后端未就绪", report)
		return
	}
	apiv1.WriteResponse(w, http.StatusOK, "plugin_healthy", "插件后端已就绪", report)
}

func (h *PluginHandler) localMarketplace(w http.ResponseWriter, r *http.Request) {
	pluginsRoot, err := h.resolveDevPluginsRoot(r.URL.Query().Get("pluginsRoot"))
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	catalog, err := plugin.NewLocalMarketplaceService(h.loader).List(pluginsRoot)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	apiv1.WriteJSON(w, http.StatusOK, catalog)
}

func (h *PluginHandler) records() []pluginRecord {
	if h == nil || h.manager == nil {
		return defaultPluginRecords()
	}

	items := h.manager.List()
	if len(items) == 0 {
		return defaultPluginRecords()
	}

	records := make([]pluginRecord, 0, len(items))
	for _, item := range items {
		if item.State == plugin.StateUninstalled {
			continue
		}
		records = append(records, pluginRecordFromInfo(item))
	}

	if len(records) == 0 {
		return defaultPluginRecords()
	}

	return records
}

func (h *PluginHandler) enable(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.manager == nil {
		apiv1.WriteError(w, http.StatusServiceUnavailable, errors.New("plugin manager not configured"))
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("plugin id is required"))
		return
	}

	if err := h.manager.Enable(id); err != nil {
		h.appendLifecycleAudit(r, "enable", id, false, err)
		if errors.Is(err, plugin.ErrPluginNotFound) {
			apiv1.WriteMessage(w, http.StatusNotFound, "not_found", "plugin not found")
			return
		}
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	h.appendPluginLog(id, "enable", "ok", "enabled")
	h.appendLifecycleAudit(r, "enable", id, true, nil)

	apiv1.WriteMessage(w, http.StatusOK, "ok", "enabled")
}

func (h *PluginHandler) disable(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.manager == nil {
		apiv1.WriteError(w, http.StatusServiceUnavailable, errors.New("plugin manager not configured"))
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("plugin id is required"))
		return
	}

	if err := h.manager.Disable(id); err != nil {
		h.appendLifecycleAudit(r, "disable", id, false, err)
		if errors.Is(err, plugin.ErrPluginNotFound) {
			apiv1.WriteMessage(w, http.StatusNotFound, "not_found", "plugin not found")
			return
		}
		if errors.Is(err, plugin.ErrPluginSystemProtected) {
			apiv1.WriteMessage(w, http.StatusForbidden, "forbidden", "system builtin plugin cannot be disabled")
			return
		}
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	h.appendPluginLog(id, "disable", "ok", "disabled")
	h.appendLifecycleAudit(r, "disable", id, true, nil)

	apiv1.WriteMessage(w, http.StatusOK, "ok", "disabled")
}

func (h *PluginHandler) uninstall(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.manager == nil {
		apiv1.WriteError(w, http.StatusServiceUnavailable, errors.New("plugin manager not configured"))
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("plugin id is required"))
		return
	}

	if err := h.manager.Uninstall(id); err != nil {
		h.appendLifecycleAudit(r, "uninstall", id, false, err)
		if errors.Is(err, plugin.ErrPluginNotFound) {
			apiv1.WriteMessage(w, http.StatusNotFound, "not_found", "plugin not found")
			return
		}
		if errors.Is(err, plugin.ErrPluginSystemProtected) {
			apiv1.WriteMessage(w, http.StatusForbidden, "forbidden", "system builtin plugin cannot be uninstalled")
			return
		}
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	h.appendPluginLog(id, "uninstall", "ok", "uninstalled")
	h.appendLifecycleAudit(r, "uninstall", id, true, nil)

	apiv1.WriteMessage(w, http.StatusOK, "ok", "uninstalled")
}

func (h *PluginHandler) debug(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.manager == nil {
		apiv1.WriteError(w, http.StatusServiceUnavailable, errors.New("plugin manager not configured"))
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("plugin id is required"))
		return
	}

	item, err := h.manager.Get(id)
	if err != nil {
		if errors.Is(err, plugin.ErrPluginNotFound) {
			apiv1.WriteMessage(w, http.StatusNotFound, "not_found", "plugin not found")
			return
		}
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	apiv1.WriteJSON(w, http.StatusOK, pluginDebugRecord{
		ID:           item.ID,
		Name:         item.Name,
		Version:      item.Version,
		Description:  item.Description,
		State:        string(item.State),
		Source:       item.Source,
		Permissions:  append([]string(nil), item.Permissions...),
		Dependencies: append([]plugin.Dependency(nil), item.Dependencies...),
		Extensions:   h.extensions(id),
	})
}

func (h *PluginHandler) extensions(pluginID string) any {
	if h == nil || h.extensionProvider == nil {
		return nil
	}
	snapshot, ok := h.extensionProvider.GetExtensionSnapshot(pluginID)
	if !ok {
		return nil
	}
	return snapshot
}

func (h *PluginHandler) logs(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("plugin id is required"))
		return
	}

	content, err := h.readPluginLogContent(id)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			apiv1.WriteMessage(w, http.StatusNotFound, "not_found", "plugin log not found")
			return
		}
		apiv1.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	apiv1.WriteJSON(w, http.StatusOK, pluginLogsRecord{PluginID: id, Content: strings.TrimSpace(string(content))})
}

func (h *PluginHandler) getConfig(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.manager == nil {
		apiv1.WriteError(w, http.StatusServiceUnavailable, errors.New("plugin manager not configured"))
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("plugin id is required"))
		return
	}

	info, err := h.manager.Get(id)
	if err != nil {
		if errors.Is(err, plugin.ErrPluginNotFound) {
			apiv1.WriteMessage(w, http.StatusNotFound, "not_found", "plugin not found")
			return
		}
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	config := map[string]any{}
	if raw := strings.TrimSpace(info.ConfigJSON); raw != "" {
		if err := json.Unmarshal([]byte(raw), &config); err != nil {
			apiv1.WriteError(w, http.StatusInternalServerError, err)
			return
		}
	}

	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"pluginId": id, "config": config, "configSchema": pluginConfigSchemaRecordFromInfo(pluginConfigSchemaForInfo(info))})
}

func (h *PluginHandler) updateConfig(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.manager == nil {
		apiv1.WriteError(w, http.StatusServiceUnavailable, errors.New("plugin manager not configured"))
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("plugin id is required"))
		return
	}

	var req struct {
		Config map[string]any `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	if req.Config == nil {
		req.Config = map[string]any{}
	}
	beforeConfig := map[string]any{}
	if info, err := h.manager.Get(id); err == nil {
		if raw := strings.TrimSpace(info.ConfigJSON); raw != "" {
			_ = json.Unmarshal([]byte(raw), &beforeConfig)
		}
	}

	updater, ok := h.manager.(PluginConfigUpdater)
	if !ok {
		apiv1.WriteError(w, http.StatusServiceUnavailable, errors.New("plugin config persistence is not configured"))
		return
	}
	if err := updater.SavePluginConfig(id, req.Config); err != nil {
		if errors.Is(err, plugin.ErrPluginNotFound) {
			apiv1.WriteMessage(w, http.StatusNotFound, "not_found", "plugin not found")
			return
		}
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	h.appendPluginLog(id, "update_config", "ok", "config updated")
	h.appendAudit(r, "update_config", "plugin", id, map[string]any{
		"before": map[string]any{"configKeys": len(beforeConfig)},
		"after":  map[string]any{"configKeys": len(req.Config)},
	})
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"pluginId": id, "config": req.Config})
}

func (h *PluginHandler) appendAudit(r *http.Request, action, resource, resourceID string, detail map[string]any) {
	if h == nil || h.auditSvc == nil || r == nil {
		return
	}
	actorID := ""
	if claims, ok := security.JWTClaimsFromContext(r.Context()); ok {
		actorID = strings.TrimSpace(claims.Subject)
	}
	if actorID == "" {
		actorID = "system"
	}
	_, _ = h.auditSvc.Append(r.Context(), actorID, action, resource, strings.TrimSpace(resourceID), detail)
}

func (h *PluginHandler) appendLifecycleAudit(r *http.Request, operation, pluginID string, succeeded bool, operationErr error) {
	if h == nil || h.auditEventSink == nil || r == nil {
		return
	}
	actorID := "system"
	actorName := ""
	if claims, ok := security.JWTClaimsFromContext(r.Context()); ok {
		if value := strings.TrimSpace(claims.Subject); value != "" {
			actorID = value
		}
		actorName = strings.TrimSpace(claims.Role)
	}
	errorMessage := ""
	if operationErr != nil {
		errorMessage = operationErr.Error()
	}
	now := time.Now().UTC()
	event, err := auditmw.NewPluginLifecycleAuditEvent(auditmw.PluginLifecycleAuditInput{
		ID:         auditmw.NewAuditID("audit-event", now),
		ActorID:    actorID,
		ActorName:  actorName,
		Operation:  operation,
		PluginID:   pluginID,
		Succeeded:  succeeded,
		Error:      errorMessage,
		Request:    r,
		OccurredAt: now,
	})
	if err != nil {
		return
	}
	_ = h.auditEventSink.AppendEvent(r.Context(), event)
}

func (h *PluginHandler) page(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("plugin id is required"))
		return
	}

	info, err := h.getPluginInfo(id)
	if err != nil {
		if errors.Is(err, plugin.ErrPluginNotFound) {
			apiv1.WriteMessage(w, http.StatusNotFound, "not_found", "plugin not found")
			return
		}
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	frontendDir, err := resolvePluginFrontendDir(info)
	if err != nil {
		apiv1.WriteError(w, http.StatusNotFound, err)
		return
	}

	indexPath := filepath.Join(frontendDir, "index.html")
	content, err := os.ReadFile(indexPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			apiv1.WriteError(w, http.StatusNotFound, errors.New("plugin frontend index not found"))
			return
		}
		apiv1.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	baseHref := strings.TrimSuffix(strings.TrimSpace(r.URL.Path), "/page") + "/assets/"
	if apiPrefix := strings.TrimRight(strings.TrimSpace(r.Header.Get("X-Skoll-Api-Prefix")), "/"); apiPrefix != "" {
		if strings.HasPrefix(baseHref, "/") {
			baseHref = apiPrefix + baseHref
		} else {
			baseHref = apiPrefix + "/" + baseHref
		}
	}
	baseTag := []byte("<base href=\"" + baseHref + "\">")
	if !bytes.Contains(bytes.ToLower(content), []byte("<base ")) {
		lower := bytes.ToLower(content)
		headPos := bytes.Index(lower, []byte("<head>"))
		if headPos >= 0 {
			insertPos := headPos + len("<head>")
			patched := make([]byte, 0, len(content)+len(baseTag)+1)
			patched = append(patched, content[:insertPos]...)
			patched = append(patched, '\n')
			patched = append(patched, baseTag...)
			patched = append(patched, content[insertPos:]...)
			content = patched
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func (h *PluginHandler) asset(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("plugin id is required"))
		return
	}

	assetPath := strings.TrimSpace(r.PathValue("asset"))
	if assetPath == "" {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("asset path is required"))
		return
	}
	cleanAsset := filepath.Clean(assetPath)
	if cleanAsset == "." || strings.HasPrefix(cleanAsset, "..") || strings.Contains(cleanAsset, "..") {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("invalid asset path"))
		return
	}

	info, err := h.getPluginInfo(id)
	if err != nil {
		if errors.Is(err, plugin.ErrPluginNotFound) {
			apiv1.WriteMessage(w, http.StatusNotFound, "not_found", "plugin not found")
			return
		}
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	frontendDir, err := resolvePluginFrontendDir(info)
	if err != nil {
		apiv1.WriteError(w, http.StatusNotFound, err)
		return
	}

	assetFile := filepath.Join(frontendDir, cleanAsset)
	rel, err := filepath.Rel(frontendDir, assetFile)
	if err != nil || strings.HasPrefix(rel, "..") {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("invalid asset target"))
		return
	}

	fi, err := os.Stat(assetFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			apiv1.WriteMessage(w, http.StatusNotFound, "not_found", "plugin asset not found")
			return
		}
		apiv1.WriteError(w, http.StatusInternalServerError, err)
		return
	}
	if fi.IsDir() {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("asset path points to directory"))
		return
	}

	http.ServeFile(w, r, assetFile)
}

func (h *PluginHandler) getPluginInfo(pluginID string) (plugin.Info, error) {
	if h == nil || h.manager == nil {
		return plugin.Info{}, errors.New("plugin manager not configured")
	}
	return h.manager.Get(pluginID)
}

func resolvePluginFrontendDir(info plugin.Info) (string, error) {
	mode := info.UIMode
	if mode == "" {
		mode = plugin.UIModeBackendOnly
	}
	if mode == plugin.UIModeBackendOnly {
		return "", errors.New("backend-only plugin has no frontend page")
	}

	source := strings.TrimSpace(info.Source)
	if source == "" || strings.EqualFold(source, "builtin") {
		return "", errors.New("frontend page unavailable for builtin plugin")
	}

	candidates := make([]string, 0, 3)
	switch mode {
	case plugin.UIModeSeparated:
		candidates = append(candidates, filepath.Join(source, "frontend", "dist"), filepath.Join(source, "frontend"))
	case plugin.UIModeMonolith, plugin.UIModeFrontendOnly:
		candidates = append(candidates, filepath.Join(source, "static"), source)
	default:
		candidates = append(candidates, filepath.Join(source, "static"), filepath.Join(source, "frontend", "dist"), source)
	}

	for _, dir := range candidates {
		indexFile := filepath.Join(dir, "index.html")
		if fi, err := os.Stat(indexFile); err == nil && !fi.IsDir() {
			return dir, nil
		}
	}

	return "", errors.New("plugin frontend page not found")
}

func (h *PluginHandler) install(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.manager == nil {
		apiv1.WriteError(w, http.StatusServiceUnavailable, errors.New("plugin manager not configured"))
		return
	}

	req, err := decodePluginPathRequest(r)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	info, err := h.manager.Install(req.Path)
	if err != nil {
		h.appendLifecycleAudit(r, "install", "", false, err)
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	h.appendPluginLog(info.ID, "install", "ok", "installed from path")
	h.appendLifecycleAudit(r, "install", info.ID, true, nil)

	apiv1.WriteJSON(w, http.StatusCreated, pluginRecord{
		ID:      info.ID,
		Name:    info.Name,
		Version: info.Version,
		Enabled: info.State == plugin.StateEnabled,
	})
}

func (h *PluginHandler) preflight(w http.ResponseWriter, r *http.Request) {
	req, err := decodePluginPathRequest(r)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	installed := []plugin.Info{}
	if h != nil && h.manager != nil {
		installed = h.manager.List()
	}
	result, err := plugin.NewInstallPreflightService(h.loader).Check(plugin.InstallPreflightInput{
		Path:      req.Path,
		Installed: installed,
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, result)
}

func (h *PluginHandler) createLink(w http.ResponseWriter, r *http.Request) {
	h.createExternal(w, r, "link")
}

func (h *PluginHandler) createEmbed(w http.ResponseWriter, r *http.Request) {
	h.createExternal(w, r, "embed")
}

func (h *PluginHandler) createExternal(w http.ResponseWriter, r *http.Request, pluginType string) {
	registrar, ok := h.manager.(PluginExternalRegistrar)
	if !ok {
		apiv1.WriteError(w, http.StatusServiceUnavailable, errors.New("plugin external registration is not configured"))
		return
	}

	var req externalPluginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	req.PluginID = strings.TrimSpace(req.PluginID)
	req.Name = strings.TrimSpace(req.Name)
	req.Version = strings.TrimSpace(req.Version)
	req.URL = strings.TrimSpace(req.URL)
	req.OpenMode = strings.TrimSpace(req.OpenMode)
	req.RoutePath = strings.TrimSpace(req.RoutePath)
	if req.PluginID == "" || req.Name == "" || req.URL == "" {
		apiv1.WriteMessage(w, http.StatusBadRequest, "invalid_request", "pluginId, name and url are required")
		return
	}
	if req.Version == "" {
		req.Version = "1.0.0"
	}
	if req.OpenMode == "" {
		req.OpenMode = "new_tab"
	}

	now := time.Now().UTC()
	info := plugin.Info{
		ID:            req.PluginID,
		Name:          req.Name,
		Version:       req.Version,
		Description:   pluginType + " external plugin",
		State:         plugin.StateEnabled,
		InstalledAt:   now,
		EnabledAt:     &now,
		Source:        req.URL,
		UIMode:        plugin.UIModeSeparated,
		Level:         plugin.LevelSystem,
		MountPolicy:   plugin.MountPolicyAdmin,
		FrontendEntry: req.URL,
		SystemBuiltin: false,
	}

	if err := registrar.RegisterExternalPlugin(info); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	h.appendPluginLog(req.PluginID, "create_"+pluginType, "ok", "registered external plugin")
	apiv1.WriteJSON(w, http.StatusCreated, map[string]any{
		"id":        req.PluginID,
		"name":      req.Name,
		"version":   req.Version,
		"url":       req.URL,
		"openMode":  req.OpenMode,
		"routePath": req.RoutePath,
		"enabled":   true,
		"type":      pluginType,
	})
}

func (h *PluginHandler) validate(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.loader == nil {
		apiv1.WriteError(w, http.StatusServiceUnavailable, errors.New("plugin loader not configured"))
		return
	}

	req, err := decodePluginPathRequest(r)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	info, err := h.loader.Load(req.Path)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	apiv1.WriteJSON(w, http.StatusOK, map[string]any{
		"id":           info.ID,
		"name":         info.Name,
		"version":      info.Version,
		"dependencies": len(info.Dependencies),
		"permissions":  len(info.Permissions),
	})
}

func (h *PluginHandler) devScaffold(w http.ResponseWriter, r *http.Request) {
	if !h.isSuperAdmin(r) {
		apiv1.WriteMessage(w, http.StatusForbidden, "forbidden", "super_admin role required")
		return
	}

	var req devScaffoldRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	pluginsRoot, err := h.resolveDevPluginsRoot(req.PluginsRoot)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	pluginID := strings.TrimSpace(strings.ToLower(req.PluginID))
	pluginName := strings.TrimSpace(req.PluginName)
	appID := strings.TrimSpace(strings.ToLower(req.AppID))
	scaffoldMode, err := normalizeDevScaffoldMode(req.Mode)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	if pluginID == "" || pluginName == "" {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("pluginId and pluginName are required"))
		return
	}
	if strings.Contains(pluginID, string(filepath.Separator)) || strings.Contains(pluginID, "/") {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("pluginId cannot contain path separators"))
		return
	}

	info := plugin.Info{
		ID:               pluginID,
		Name:             pluginName,
		Version:          "0.1.0",
		APIVersion:       "v1",
		MigrationVersion: "v0.1.0",
		UIMode:           plugin.UIModeSeparated,
		MountPolicy:      plugin.MountPolicyAdmin,
		UINavPosition:    plugin.UINavPositionNone,
		UIOpenMode:       plugin.UIOpenModeIntegrated,
		UITabMode:        plugin.UITabModeOptional,
		I18nLocales:      []string{"zh-CN", "en-US"},
		Level:            plugin.LevelSystem,
		Permissions:      []string{pluginID + ".read"},
	}
	if appID != "" {
		info.Level = plugin.LevelApp
		info.AppID = appID
	}
	if err := info.ValidateManifest(); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	pluginDir := filepath.Join(pluginsRoot, pluginID)
	manifestPath := filepath.Join(pluginDir, "plugin.yaml")
	if _, err := os.Stat(manifestPath); err == nil {
		apiv1.WriteError(w, http.StatusBadRequest, fmt.Errorf("plugin manifest already exists: %s", manifestPath))
		return
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		apiv1.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	if err := createScaffoldLayout(pluginDir, info.ID, scaffoldMode); err != nil {
		apiv1.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	if err := os.WriteFile(manifestPath, []byte(renderScaffoldManifest(info)), 0o644); err != nil {
		apiv1.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	readme := strings.Join([]string{
		"# " + pluginName,
		"",
		"Generated by: plugin scaffold",
		"",
		"## Next Steps",
		"",
		"1. Implement backend APIs in ./backend",
		"2. Build frontend app in ./frontend",
		"3. Add DB migrations in ./migrations",
		"4. Validate with: plugin validate " + pluginDir,
	}, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(pluginDir, "README.md"), []byte(readme), 0o644); err != nil {
		apiv1.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	h.appendAudit(r, "dev_scaffold", "plugin", info.ID, map[string]any{"pluginsRoot": pluginsRoot})
	apiv1.WriteJSON(w, http.StatusCreated, devScaffoldResponse{
		Operation:   "scaffold",
		Status:      "ok",
		PluginsRoot: pluginsRoot,
		PluginID:    info.ID,
		PluginName:  info.Name,
		PluginDir:   pluginDir,
		Mode:        scaffoldMode,
	})
}

func (h *PluginHandler) devConfig(w http.ResponseWriter, r *http.Request) {
	if !h.isSuperAdmin(r) {
		apiv1.WriteMessage(w, http.StatusForbidden, "forbidden", "super_admin role required")
		return
	}
	allowlist, primary := h.devRoots()
	apiv1.WriteJSON(w, http.StatusOK, devConfigResponse{
		Enabled:      h.devPortalEnabled,
		DefaultRoot:  primary,
		AllowedRoots: allowlist,
	})
}

func (h *PluginHandler) devManifestGet(w http.ResponseWriter, r *http.Request) {
	if !h.isSuperAdmin(r) {
		apiv1.WriteMessage(w, http.StatusForbidden, "forbidden", "super_admin role required")
		return
	}

	pluginsRoot, err := h.resolveDevPluginsRoot(r.URL.Query().Get("pluginsRoot"))
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	pluginID := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("pluginId")))
	if pluginID == "" {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("pluginId is required"))
		return
	}

	pluginDir, manifestPath, err := h.resolveManifestPath(pluginsRoot, pluginID)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		apiv1.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	validation := "ok"
	validationErr := ""
	if _, loadErr := h.loader.Load(pluginDir); loadErr != nil {
		validation = "invalid"
		validationErr = loadErr.Error()
	}

	apiv1.WriteJSON(w, http.StatusOK, devManifestResponse{
		Operation:    "manifest_get",
		Status:       "ok",
		PluginsRoot:  pluginsRoot,
		PluginID:     pluginID,
		PluginDir:    pluginDir,
		ManifestPath: manifestPath,
		Manifest:     string(raw),
		Validation:   validation,
		Error:        validationErr,
	})
}

func (h *PluginHandler) devManifestPut(w http.ResponseWriter, r *http.Request) {
	if !h.isSuperAdmin(r) {
		apiv1.WriteMessage(w, http.StatusForbidden, "forbidden", "super_admin role required")
		return
	}

	var req devManifestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	pluginsRoot, err := h.resolveDevPluginsRoot(req.PluginsRoot)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	pluginID := strings.TrimSpace(strings.ToLower(req.PluginID))
	if pluginID == "" {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("pluginId is required"))
		return
	}
	manifest := strings.TrimSpace(req.Manifest)
	if manifest == "" {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("manifest is required"))
		return
	}
	if !strings.HasSuffix(manifest, "\n") {
		manifest += "\n"
	}

	pluginDir, manifestPath, err := h.resolveManifestPath(pluginsRoot, pluginID)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	beforeRaw, err := os.ReadFile(manifestPath)
	if err != nil {
		apiv1.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		apiv1.WriteError(w, http.StatusInternalServerError, err)
		return
	}
	if _, loadErr := h.loader.Load(pluginDir); loadErr != nil {
		_ = os.WriteFile(manifestPath, beforeRaw, 0o644)
		apiv1.WriteError(w, http.StatusBadRequest, loadErr)
		return
	}
	if refresher, ok := h.manager.(PluginMetadataRefresher); ok {
		if refreshErr := refresher.ReloadPluginMetadata(pluginID); refreshErr != nil {
			_ = os.WriteFile(manifestPath, beforeRaw, 0o644)
			apiv1.WriteError(w, http.StatusBadRequest, refreshErr)
			return
		}
	}

	h.appendAudit(r, "dev_manifest_put", "plugin", pluginID, map[string]any{"pluginsRoot": pluginsRoot, "bytes": len(manifest)})
	apiv1.WriteJSON(w, http.StatusOK, devManifestResponse{
		Operation:    "manifest_put",
		Status:       "ok",
		PluginsRoot:  pluginsRoot,
		PluginID:     pluginID,
		PluginDir:    pluginDir,
		ManifestPath: manifestPath,
		Manifest:     manifest,
		Validation:   "ok",
	})
}

func (h *PluginHandler) devManifestValidate(w http.ResponseWriter, r *http.Request) {
	if !h.isSuperAdmin(r) {
		apiv1.WriteMessage(w, http.StatusForbidden, "forbidden", "super_admin role required")
		return
	}

	var req devManifestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	pluginsRoot, err := h.resolveDevPluginsRoot(req.PluginsRoot)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	pluginID := strings.TrimSpace(strings.ToLower(req.PluginID))
	if pluginID == "" {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("pluginId is required"))
		return
	}
	manifest := strings.TrimSpace(req.Manifest)
	if manifest == "" {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("manifest is required"))
		return
	}
	if !strings.HasSuffix(manifest, "\n") {
		manifest += "\n"
	}

	tmpDir, err := os.MkdirTemp("", "skoll-manifest-validate-*")
	if err != nil {
		apiv1.WriteError(w, http.StatusInternalServerError, err)
		return
	}
	defer os.RemoveAll(tmpDir)

	pluginDir := filepath.Join(tmpDir, pluginID)
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		apiv1.WriteError(w, http.StatusInternalServerError, err)
		return
	}
	manifestPath := filepath.Join(pluginDir, "plugin.yaml")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		apiv1.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	if _, loadErr := h.loader.Load(pluginDir); loadErr != nil {
		apiv1.WriteError(w, http.StatusBadRequest, loadErr)
		return
	}

	apiv1.WriteJSON(w, http.StatusOK, devManifestResponse{
		Operation:    "manifest_validate",
		Status:       "ok",
		PluginsRoot:  pluginsRoot,
		PluginID:     pluginID,
		PluginDir:    filepath.Join(pluginsRoot, pluginID),
		ManifestPath: filepath.Join(pluginsRoot, pluginID, "plugin.yaml"),
		Manifest:     manifest,
		Validation:   "ok",
	})
}

func (h *PluginHandler) devPipeline(w http.ResponseWriter, r *http.Request) {
	if !h.isSuperAdmin(r) {
		apiv1.WriteMessage(w, http.StatusForbidden, "forbidden", "super_admin role required")
		return
	}

	var req devPipelineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	pluginsRoot, err := h.resolveDevPluginsRoot(req.PluginsRoot)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	pluginID := strings.TrimSpace(strings.ToLower(req.PluginID))
	if pluginID == "" {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("pluginId is required"))
		return
	}

	pluginDir, _, err := h.resolveManifestPath(pluginsRoot, pluginID)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	started := time.Now().UTC()
	steps := make([]devPipelineStep, 0, 3)
	pipeStatus := "ok"

	stepStart := time.Now()
	info, loadErr := h.loader.Load(pluginDir)
	if loadErr != nil {
		steps = append(steps, devPipelineStep{Name: "validate_manifest", Status: "failed", Message: loadErr.Error(), DurationMs: time.Since(stepStart).Milliseconds()})
		pipeStatus = "failed"
	} else {
		steps = append(steps, devPipelineStep{Name: "validate_manifest", Status: "ok", Message: fmt.Sprintf("%s@%s", info.ID, info.Version), DurationMs: time.Since(stepStart).Milliseconds()})
	}

	if pipeStatus == "ok" {
		stepStart = time.Now()
		artifactDir := strings.TrimSpace(req.OutputDir)
		if artifactDir == "" {
			artifactDir = filepath.Join(pluginsRoot, "_dist")
		}
		if err := os.MkdirAll(artifactDir, 0o755); err != nil {
			steps = append(steps, devPipelineStep{Name: "prepare_artifact_dir", Status: "failed", Message: err.Error(), DurationMs: time.Since(stepStart).Milliseconds()})
			pipeStatus = "failed"
		} else {
			steps = append(steps, devPipelineStep{Name: "prepare_artifact_dir", Status: "ok", Message: artifactDir, DurationMs: time.Since(stepStart).Milliseconds()})
		}

		if pipeStatus == "ok" {
			stepStart = time.Now()
			artifactPath := filepath.Join(artifactDir, fmt.Sprintf("%s-%s.zip", info.ID, strings.TrimSpace(info.Version)))
			if err := zipDirectory(pluginDir, artifactPath); err != nil {
				steps = append(steps, devPipelineStep{Name: "package", Status: "failed", Message: err.Error(), DurationMs: time.Since(stepStart).Milliseconds()})
				pipeStatus = "failed"
			} else {
				steps = append(steps, devPipelineStep{Name: "package", Status: "ok", Artifact: artifactPath, DurationMs: time.Since(stepStart).Milliseconds()})
			}
		}
	}

	h.appendAudit(r, "dev_pipeline", "plugin", pluginID, map[string]any{"pluginsRoot": pluginsRoot, "status": pipeStatus})
	apiv1.WriteJSON(w, http.StatusOK, devPipelineResponse{
		Operation:   "pipeline",
		Status:      pipeStatus,
		PluginsRoot: pluginsRoot,
		PluginID:    pluginID,
		PluginDir:   pluginDir,
		StartedAt:   started.Format(time.RFC3339),
		FinishedAt:  time.Now().UTC().Format(time.RFC3339),
		Steps:       steps,
	})
}

func (h *PluginHandler) devProjects(w http.ResponseWriter, r *http.Request) {
	if !h.isSuperAdmin(r) {
		apiv1.WriteMessage(w, http.StatusForbidden, "forbidden", "super_admin role required")
		return
	}

	var req devProjectActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	pluginsRoot, err := h.resolveDevPluginsRoot(req.PluginsRoot)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	pluginDirs, err := discoverPluginManifestDirs(pluginsRoot)
	if err != nil {
		apiv1.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	installedMap := map[string]plugin.Info{}
	for _, item := range h.manager.List() {
		if strings.TrimSpace(item.ID) != "" {
			installedMap[item.ID] = item
		}
	}

	projects := make([]devProjectRecord, 0, len(pluginDirs))
	for _, dir := range pluginDirs {
		rec := devProjectRecord{
			Path:        dir,
			PluginsRoot: pluginsRoot,
			Status:      "ok",
		}
		info, loadErr := h.loader.Load(dir)
		if loadErr != nil {
			rec.Status = "invalid"
			rec.Error = loadErr.Error()
			rec.PluginID = filepath.Base(dir)
			projects = append(projects, rec)
			continue
		}

		rec.PluginID = info.ID
		rec.Name = info.Name
		rec.NameZhCN = strings.TrimSpace(info.NameZhCN)
		rec.NameEnUS = strings.TrimSpace(info.NameEnUS)
		rec.Version = info.Version
		rec.Mode = detectProjectMode(dir)
		rec.LastUpdatedAt = detectManifestUpdatedAt(filepath.Join(dir, "plugin.yaml"))
		rec.BuildHint, rec.PackageHint, rec.PublishHint = buildProjectHints(info.ID, dir, rec.Mode)
		rec.PreviewURL = fmt.Sprintf("/skoll/plugins/%s", info.ID)
		if installed, ok := installedMap[info.ID]; ok {
			rec.Installed = true
			rec.Enabled = installed.State == plugin.StateEnabled
		}
		projects = append(projects, rec)
	}

	h.appendAudit(r, "dev_projects", "plugin", "", map[string]any{"pluginsRoot": pluginsRoot, "count": len(projects)})
	apiv1.WriteJSON(w, http.StatusOK, devListProjectsResponse{
		Operation:   "projects",
		Status:      "ok",
		PluginsRoot: pluginsRoot,
		Projects:    projects,
	})
}

func (h *PluginHandler) devRemove(w http.ResponseWriter, r *http.Request) {
	if !h.isSuperAdmin(r) {
		apiv1.WriteMessage(w, http.StatusForbidden, "forbidden", "super_admin role required")
		return
	}

	var req devProjectActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	pluginsRoot, err := h.resolveDevPluginsRoot(req.PluginsRoot)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	pluginID := strings.TrimSpace(strings.ToLower(req.PluginID))
	if pluginID == "" {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("pluginId is required"))
		return
	}

	pluginDir := filepath.Join(pluginsRoot, pluginID)
	manifestPath := filepath.Join(pluginDir, "plugin.yaml")
	if _, statErr := os.Stat(manifestPath); statErr != nil {
		if errors.Is(statErr, os.ErrNotExist) {
			apiv1.WriteError(w, http.StatusBadRequest, errors.New("plugin project not found in selected root"))
			return
		}
		apiv1.WriteError(w, http.StatusInternalServerError, statErr)
		return
	}

	uninstalled := false
	if err := h.manager.Uninstall(pluginID); err == nil {
		uninstalled = true
	} else if !errors.Is(err, plugin.ErrPluginNotFound) {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	filesRemoved := false
	if req.RemoveFiles {
		if err := os.RemoveAll(pluginDir); err != nil {
			apiv1.WriteError(w, http.StatusInternalServerError, err)
			return
		}
		filesRemoved = true
	}

	h.appendAudit(r, "dev_remove", "plugin", pluginID, map[string]any{"pluginsRoot": pluginsRoot, "removeFiles": req.RemoveFiles})
	apiv1.WriteJSON(w, http.StatusOK, devRemoveProjectResponse{
		Operation:    "remove",
		Status:       "ok",
		PluginsRoot:  pluginsRoot,
		PluginID:     pluginID,
		PluginDir:    pluginDir,
		Uninstalled:  uninstalled,
		FilesRemoved: filesRemoved,
	})
}

func (h *PluginHandler) devPackage(w http.ResponseWriter, r *http.Request) {
	if !h.isSuperAdmin(r) {
		apiv1.WriteMessage(w, http.StatusForbidden, "forbidden", "super_admin role required")
		return
	}

	var req devProjectActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	pluginsRoot, err := h.resolveDevPluginsRoot(req.PluginsRoot)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	pluginID := strings.TrimSpace(strings.ToLower(req.PluginID))
	if pluginID == "" {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("pluginId is required"))
		return
	}

	pluginDir := filepath.Join(pluginsRoot, pluginID)
	info, err := h.loader.Load(pluginDir)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	outputDir := strings.TrimSpace(req.OutputDir)
	if outputDir == "" {
		outputDir = filepath.Join(pluginsRoot, "_dist")
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		apiv1.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	artifactPath := filepath.Join(outputDir, fmt.Sprintf("%s-%s.zip", info.ID, strings.TrimSpace(info.Version)))
	if err := zipDirectory(pluginDir, artifactPath); err != nil {
		apiv1.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	h.appendAudit(r, "dev_package", "plugin", info.ID, map[string]any{"pluginsRoot": pluginsRoot, "artifactPath": artifactPath})
	apiv1.WriteJSON(w, http.StatusOK, devPackageProjectResponse{
		Operation:    "package",
		Status:       "ok",
		PluginsRoot:  pluginsRoot,
		PluginID:     info.ID,
		PluginDir:    pluginDir,
		ArtifactPath: artifactPath,
	})
}

func normalizeDevScaffoldMode(raw string) (string, error) {
	v := strings.ToLower(strings.TrimSpace(raw))
	switch v {
	case "", devScaffoldModeWorkspace:
		return devScaffoldModeWorkspace, nil
	case devScaffoldModeRepository:
		return devScaffoldModeRepository, nil
	default:
		return "", fmt.Errorf("unsupported scaffold mode: %q", raw)
	}
}

func createScaffoldLayout(pluginDir, pluginID, mode string) error {
	dirs := []string{
		pluginDir,
		filepath.Join(pluginDir, "backend"),
		filepath.Join(pluginDir, "frontend"),
		filepath.Join(pluginDir, "migrations"),
		filepath.Join(pluginDir, "docs"),
	}
	if mode == devScaffoldModeRepository {
		dirs = append(dirs, filepath.Join(pluginDir, "backend", "cmd", pluginID), filepath.Join(pluginDir, "frontend", "src"))
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	if mode != devScaffoldModeRepository {
		return nil
	}

	moduleName := "github.com/example/" + strings.ReplaceAll(pluginID, "_", "-")
	goMod := strings.Join([]string{
		"module " + moduleName,
		"",
		"go 1.22",
		"",
	}, "\n")
	mainGo := strings.Join([]string{
		"package main",
		"",
		"import \"fmt\"",
		"",
		"func main() {",
		"\tfmt.Println(\"plugin backend bootstrap ready\")",
		"}",
		"",
	}, "\n")
	frontendPkg := strings.Join([]string{
		"{",
		"  \"name\": \"" + pluginID + "-frontend\",",
		"  \"version\": \"0.1.0\",",
		"  \"private\": true,",
		"  \"scripts\": {",
		"    \"dev\": \"echo TODO: frontend dev server\",",
		"    \"build\": \"echo TODO: frontend build\"",
		"  }",
		"}",
		"",
	}, "\n")

	if err := os.WriteFile(filepath.Join(pluginDir, "backend", "go.mod"), []byte(goMod), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "backend", "cmd", pluginID, "main.go"), []byte(mainGo), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "frontend", "package.json"), []byte(frontendPkg), 0o644); err != nil {
		return err
	}
	return nil
}

func (h *PluginHandler) devValidateAll(w http.ResponseWriter, r *http.Request) {
	if !h.isSuperAdmin(r) {
		apiv1.WriteMessage(w, http.StatusForbidden, "forbidden", "super_admin role required")
		return
	}

	var req devValidateAllRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	pluginsRoot, err := h.resolveDevPluginsRoot(req.PluginsRoot)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	pluginDirs, err := discoverPluginManifestDirs(pluginsRoot)
	if err != nil {
		apiv1.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	results := make([]devValidationResult, 0, len(pluginDirs))
	failures := 0
	for _, dir := range pluginDirs {
		info, loadErr := h.loader.Load(dir)
		if loadErr != nil {
			failures++
			results = append(results, devValidationResult{Path: dir, Status: "invalid", Error: loadErr.Error()})
			continue
		}
		results = append(results, devValidationResult{
			Path:    dir,
			Plugin:  info.ID,
			Name:    info.Name,
			Version: info.Version,
			Status:  "ok",
		})
	}

	h.appendAudit(r, "dev_validate_all", "plugin", "", map[string]any{"pluginsRoot": pluginsRoot, "count": len(pluginDirs), "failures": failures})
	apiv1.WriteJSON(w, http.StatusOK, devValidateAllResponse{
		Operation:   "validate_all",
		Status:      "ok",
		PluginsRoot: pluginsRoot,
		Summary: devValidateAllSummary{
			Total:   len(pluginDirs),
			Valid:   len(pluginDirs) - failures,
			Invalid: failures,
		},
		Results: results,
	})
}

func (h *PluginHandler) isSuperAdmin(r *http.Request) bool {
	if h == nil || r == nil {
		return false
	}
	claims, ok := security.JWTClaimsFromContext(r.Context())
	if !ok {
		return false
	}
	return claims.HasRole("super_admin")
}

func (h *PluginHandler) resolveDevPluginsRoot(candidate string) (string, error) {
	allowlist, primary := h.devRoots()

	requested := strings.TrimSpace(candidate)
	if requested == "" {
		return primary, nil
	}
	reqClean := filepath.Clean(requested)
	for _, allowed := range allowlist {
		if pathEquals(reqClean, allowed) {
			return allowed, nil
		}
	}
	return "", fmt.Errorf("pluginsRoot must match one of the allowlisted roots: %s", strings.Join(allowlist, ", "))
}

func (h *PluginHandler) resolveManifestPath(pluginsRoot, pluginID string) (pluginDir, manifestPath string, err error) {
	if strings.Contains(pluginID, "/") || strings.Contains(pluginID, string(filepath.Separator)) {
		return "", "", errors.New("pluginId cannot contain path separators")
	}
	pluginDir = filepath.Join(pluginsRoot, pluginID)
	manifestPath = filepath.Join(pluginDir, "plugin.yaml")
	if _, statErr := os.Stat(manifestPath); statErr != nil {
		if errors.Is(statErr, os.ErrNotExist) {
			return "", "", errors.New("plugin manifest not found in selected root")
		}
		return "", "", statErr
	}
	return pluginDir, manifestPath, nil
}

func (h *PluginHandler) devRoots() ([]string, string) {
	allowlist := make([]string, 0, len(h.devPortalRoots)+1)
	for _, root := range h.devPortalRoots {
		v := strings.TrimSpace(root)
		if v != "" {
			allowlist = append(allowlist, filepath.Clean(v))
		}
	}
	if len(allowlist) == 0 {
		primary := strings.TrimSpace(h.devPortalRoot)
		if primary == "" {
			primary = "plugins"
		}
		allowlist = append(allowlist, filepath.Clean(primary))
	}
	return allowlist, allowlist[0]
}

func pathEquals(a, b string) bool {
	a = filepath.Clean(strings.TrimSpace(a))
	b = filepath.Clean(strings.TrimSpace(b))
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func discoverPluginManifestDirs(root string) ([]string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, errors.New("root is required")
	}
	if _, err := os.Stat(root); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []string{}, nil
		}
		return nil, err
	}

	paths := make([]string, 0, 32)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() {
			return nil
		}

		name := d.Name()
		if strings.HasPrefix(name, ".") || name == "node_modules" {
			if path == root {
				return nil
			}
			return filepath.SkipDir
		}

		manifestPath := filepath.Join(path, "plugin.yaml")
		if info, statErr := os.Stat(manifestPath); statErr == nil && !info.IsDir() {
			paths = append(paths, filepath.Clean(path))
			return filepath.SkipDir
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(paths)
	return paths, nil
}

func detectProjectMode(projectDir string) string {
	backendGoMod := filepath.Join(projectDir, "backend", "go.mod")
	frontendPkg := filepath.Join(projectDir, "frontend", "package.json")
	if fileExists(backendGoMod) || fileExists(frontendPkg) {
		return devScaffoldModeRepository
	}
	return devScaffoldModeWorkspace
}

func detectManifestUpdatedAt(path string) string {
	if stat, err := os.Stat(path); err == nil {
		return stat.ModTime().UTC().Format(time.RFC3339)
	}
	return ""
}

func buildProjectHints(pluginID, projectDir, mode string) (buildHint, packageHint, publishHint string) {
	if mode == devScaffoldModeRepository {
		buildHint = fmt.Sprintf("cd %s/frontend && npm install && npm run build", filepath.ToSlash(projectDir))
		packageHint = fmt.Sprintf("POST /skoll/v1/plugins/dev/package { pluginsRoot, pluginId: \"%s\" }", pluginID)
		publishHint = "将 zip 包上传到内部制品库并走 CI 发布流程（签名/灰度/版本门禁）"
		return
	}
	buildHint = "workspace 模式通常不在当前仓库构建，请在独立仓库完成 frontend/backend 构建"
	packageHint = fmt.Sprintf("POST /skoll/v1/plugins/dev/package { pluginsRoot, pluginId: \"%s\" }", pluginID)
	publishHint = "打包后推送制品库，再由平台安装/升级"
	return
}

func fileExists(path string) bool {
	if stat, err := os.Stat(path); err == nil {
		return !stat.IsDir()
	}
	return false
}

func zipDirectory(sourceDir, zipPath string) error {
	file, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := zip.NewWriter(file)
	defer writer.Close()

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		name := info.Name()
		if info.IsDir() {
			if name == ".git" || name == "node_modules" || name == "dist" || name == "build" {
				return filepath.SkipDir
			}
			return nil
		}

		rel = filepath.ToSlash(rel)
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = rel
		header.Method = zip.Deflate
		entryWriter, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}

		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()
		_, err = io.Copy(entryWriter, src)
		return err
	})
}

func renderScaffoldManifest(info plugin.Info) string {
	lines := []string{
		"id: " + info.ID,
		"name: " + quoteYAML(info.Name),
		"version: " + info.Version,
		"api_version: " + info.APIVersion,
		"migration_version: " + info.MigrationVersion,
		"ui_mode: " + string(info.UIMode),
		"level: " + string(info.Level),
		"mount_policy: " + string(info.MountPolicy),
		"ui_nav_position: " + string(plugin.UINavPositionNone),
		"ui_open_mode: " + string(plugin.UIOpenModeIntegrated),
		"ui_tab_mode: " + string(plugin.UITabModeOptional),
		"i18n_locales:",
		"  - zh-CN",
		"  - en-US",
	}
	if strings.TrimSpace(info.AppID) != "" {
		lines = append(lines, "app_id: "+info.AppID)
	}
	lines = append(lines,
		"permissions:",
		"  - "+quoteYAML(info.Permissions[0]),
	)
	return strings.Join(lines, "\n") + "\n"
}

func quoteYAML(value string) string {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.ReplaceAll(trimmed, `"`, `\\"`)
	return `"` + trimmed + `"`
}

func decodePluginPathRequest(r *http.Request) (pluginPathRequest, error) {
	var req pluginPathRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return pluginPathRequest{}, err
	}
	req.Path = strings.TrimSpace(req.Path)
	if req.Path == "" {
		return pluginPathRequest{}, errors.New("path is required")
	}
	return req, nil
}

func defaultPluginRecords() []pluginRecord {
	return []pluginRecord{
		{
			ID:            "builtin-auth",
			Name:          "Builtin Auth",
			Version:       "1.0.0",
			Enabled:       true,
			UIMode:        string(plugin.UIModeSeparated),
			Level:         string(plugin.LevelSystem),
			MountPolicy:   string(plugin.MountPolicyAdmin),
			FrontendEntry: "/skoll/plugins/auth",
			SystemBuiltin: true,
		},
	}
}

func pluginRecordFromInfo(item plugin.Info) pluginRecord {
	if source := strings.TrimSpace(item.Source); source != "" {
		if loaded, err := plugin.NewFileLoader().Load(source); err == nil {
			// Manifest is the source of truth for UI placement; keep runtime states from current item.
			item.NameZhCN = loaded.NameZhCN
			item.NameEnUS = loaded.NameEnUS
			item.UINavPosition = loaded.UINavPosition
			item.UIOpenMode = loaded.UIOpenMode
			item.UITabMode = loaded.UITabMode
			item.I18nLocales = append([]string(nil), loaded.I18nLocales...)
			item.UIMenu = clonePluginUIMenu(loaded.UIMenu)
			item.ConfigSchema = clonePluginConfigSchema(loaded.ConfigSchema)
			item.FrontendEntry = loaded.FrontendEntry
		}
	}

	uiMode := string(item.UIMode)
	if uiMode == "" {
		uiMode = string(plugin.UIModeBackendOnly)
	}
	level := item.Level
	if level == "" {
		level = plugin.LevelSystem
	}
	mp := item.MountPolicy
	if mp == "" {
		mp = plugin.MountPolicyAdmin
	}
	navPosition := item.UINavPosition
	if navPosition == "" {
		navPosition = plugin.UINavPositionNone
	}
	openMode := item.UIOpenMode
	if openMode == "" {
		openMode = plugin.UIOpenModeIntegrated
	}
	tabMode := item.UITabMode
	if tabMode == "" {
		tabMode = plugin.UITabModeOptional
	}
	entry := plugin.ResolveFrontendEntry(item)
	locales := append([]string(nil), item.I18nLocales...)
	return pluginRecord{
		ID:            item.ID,
		Name:          item.Name,
		NameZhCN:      strings.TrimSpace(item.NameZhCN),
		NameEnUS:      strings.TrimSpace(item.NameEnUS),
		Version:       item.Version,
		Enabled:       item.State == plugin.StateEnabled,
		UIMode:        uiMode,
		Level:         string(level),
		AppID:         strings.TrimSpace(item.AppID),
		MountPolicy:   string(mp),
		UINavPosition: string(navPosition),
		UIOpenMode:    string(openMode),
		UITabMode:     string(tabMode),
		I18nLocales:   locales,
		UIMenu:        pluginMenuRecordFromInfo(item.UIMenu),
		ConfigSchema:  pluginConfigSchemaRecordFromInfo(item.ConfigSchema),
		FrontendEntry: entry,
		SystemBuiltin: item.SystemBuiltin || strings.EqualFold(strings.TrimSpace(item.Source), "builtin"),
	}
}

func pluginConfigSchemaRecordFromInfo(schema *plugin.ConfigSchema) *pluginConfigSchemaRecord {
	if schema == nil {
		return nil
	}
	out := &pluginConfigSchemaRecord{
		Title:       strings.TrimSpace(schema.Title),
		TitleZhCN:   strings.TrimSpace(schema.TitleZhCN),
		TitleEnUS:   strings.TrimSpace(schema.TitleEnUS),
		Description: strings.TrimSpace(schema.Description),
		Fields:      make([]pluginConfigFieldRecord, 0, len(schema.Fields)),
	}
	for _, field := range schema.Fields {
		record := pluginConfigFieldRecord{
			Key:         strings.TrimSpace(field.Key),
			Label:       strings.TrimSpace(field.Label),
			LabelZhCN:   strings.TrimSpace(field.LabelZhCN),
			LabelEnUS:   strings.TrimSpace(field.LabelEnUS),
			Type:        strings.TrimSpace(field.Type),
			Required:    field.Required,
			Default:     strings.TrimSpace(field.Default),
			Placeholder: strings.TrimSpace(field.Placeholder),
			Help:        strings.TrimSpace(field.Help),
			Min:         field.Min,
			Max:         field.Max,
			MinLength:   field.MinLength,
			MaxLength:   field.MaxLength,
			Pattern:     strings.TrimSpace(field.Pattern),
			Options:     make([]pluginConfigOptionRecord, 0, len(field.Options)),
		}
		for _, option := range field.Options {
			record.Options = append(record.Options, pluginConfigOptionRecord{
				Label:     strings.TrimSpace(option.Label),
				LabelZhCN: strings.TrimSpace(option.LabelZhCN),
				LabelEnUS: strings.TrimSpace(option.LabelEnUS),
				Value:     strings.TrimSpace(option.Value),
			})
		}
		out.Fields = append(out.Fields, record)
	}
	return out
}

func pluginConfigSchemaForInfo(info plugin.Info) *plugin.ConfigSchema {
	if source := strings.TrimSpace(info.Source); source != "" {
		if loaded, err := plugin.NewFileLoader().Load(source); err == nil && loaded.ConfigSchema != nil {
			return loaded.ConfigSchema
		}
	}
	return info.ConfigSchema
}

func pluginMenuRecordFromInfo(menu *plugin.UIMenu) *pluginMenuRecord {
	if menu == nil {
		return nil
	}
	return &pluginMenuRecord{
		Label:               strings.TrimSpace(menu.Label),
		LabelZhCN:           strings.TrimSpace(menu.LabelZhCN),
		LabelEnUS:           strings.TrimSpace(menu.LabelEnUS),
		Path:                strings.TrimSpace(menu.Path),
		Icon:                strings.TrimSpace(menu.Icon),
		Order:               menu.Order,
		RequiredRoles:       append([]string(nil), menu.RequiredRoles...),
		RequiredPermissions: append([]string(nil), menu.RequiredPermissions...),
	}
}

func clonePluginConfigSchema(schema *plugin.ConfigSchema) *plugin.ConfigSchema {
	if schema == nil {
		return nil
	}
	out := &plugin.ConfigSchema{
		Title:       schema.Title,
		TitleZhCN:   schema.TitleZhCN,
		TitleEnUS:   schema.TitleEnUS,
		Description: schema.Description,
		Fields:      make([]plugin.ConfigField, 0, len(schema.Fields)),
	}
	for _, field := range schema.Fields {
		cloned := field
		cloned.Options = append([]plugin.ConfigOption(nil), field.Options...)
		out.Fields = append(out.Fields, cloned)
	}
	return out
}

func clonePluginUIMenu(menu *plugin.UIMenu) *plugin.UIMenu {
	if menu == nil {
		return nil
	}
	return &plugin.UIMenu{
		Label:               menu.Label,
		LabelZhCN:           menu.LabelZhCN,
		LabelEnUS:           menu.LabelEnUS,
		Path:                menu.Path,
		Icon:                menu.Icon,
		Order:               menu.Order,
		RequiredRoles:       append([]string(nil), menu.RequiredRoles...),
		RequiredPermissions: append([]string(nil), menu.RequiredPermissions...),
	}
}

func (h *PluginHandler) appendPluginLog(pluginID, action, result, message string) {
	pluginID = strings.TrimSpace(pluginID)
	if pluginID == "" {
		return
	}

	if !h.shouldWritePluginFile() {
		if h.logger == nil {
			h.logger = logging.New("info")
		}
		h.logger.Info(
			"plugin_operation",
			"plugin_id", pluginID,
			"action", strings.TrimSpace(action),
			"result", strings.TrimSpace(result),
			"message", strings.TrimSpace(message),
		)
		return
	}

	logPath := h.logPathForPlugin(pluginID)
	if logPath == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return
	}
	line := fmt.Sprintf("%s action=%s result=%s plugin_id=%s message=%s\n", time.Now().UTC().Format(time.RFC3339), strings.TrimSpace(action), strings.TrimSpace(result), pluginID, strings.TrimSpace(message))
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(line)
}

func (h *PluginHandler) logPathForPlugin(pluginID string) string {
	if output := logging.ResolveLogFilePath(h.logDir, h.logFile); output != "" {
		return output
	}
	if !h.pluginPerFile {
		return ""
	}
	logDir := strings.TrimSpace(h.logDir)
	if logDir == "" {
		logDir = "log"
	}
	return filepath.Join(logDir, pluginID+".log")
}

func (h *PluginHandler) shouldWritePluginFile() bool {
	return strings.TrimSpace(h.logFile) != "" || h.pluginPerFile
}

func (h *PluginHandler) readPluginLogContent(pluginID string) ([]byte, error) {
	logPath := h.logPathForPlugin(pluginID)
	if logPath == "" {
		return nil, os.ErrNotExist
	}

	if strings.TrimSpace(h.logFile) == "" {
		if !h.pluginPerFile {
			return nil, os.ErrNotExist
		}
		return os.ReadFile(logPath)
	}

	f, err := os.Open(logPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	raw, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	needle := "plugin_id=" + pluginID
	lines := strings.Split(string(raw), "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.Contains(trimmed, needle) {
			filtered = append(filtered, trimmed)
		}
	}
	if len(filtered) == 0 {
		return nil, os.ErrNotExist
	}
	return []byte(strings.Join(filtered, "\n")), nil
}
