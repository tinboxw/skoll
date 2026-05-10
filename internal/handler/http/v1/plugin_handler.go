package v1

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/plugin"
	"github.com/tinboxw/skoll/pkg/logging"
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

type PluginHandler struct {
	manager           PluginManager
	extensionProvider PluginExtensionSnapshotProvider
	loader            plugin.MetadataLoader
	logger            logging.Logger
	logLevel          string
	logDir            string
	logFile           string
	pluginPerFile     bool
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

type pluginRecord struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Version       string `json:"version"`
	Enabled       bool   `json:"enabled"`
	UIMode        string `json:"uiMode"`
	FrontendEntry string `json:"frontendEntry,omitempty"`
	SystemBuiltin bool   `json:"systemBuiltin"`
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
	h := &PluginHandler{manager: manager, loader: plugin.NewFileLoader()}
	for _, opt := range opts {
		if opt != nil {
			opt(h)
		}
	}
	if provider, ok := manager.(PluginExtensionSnapshotProvider); ok {
		h.extensionProvider = provider
	}
	if h.logLevel == "" {
		h.logLevel = "info"
	}
	h.logger = logging.New(h.logLevel)
	mux.HandleFunc("GET /v1/plugins", h.list)
	mux.HandleFunc("POST /v1/plugins/install", h.install)
	mux.HandleFunc("POST /v1/plugins/link", h.createLink)
	mux.HandleFunc("POST /v1/plugins/embed", h.createEmbed)
	mux.HandleFunc("POST /v1/plugins/validate", h.validate)
	mux.HandleFunc("POST /v1/plugins/{id}/enable", h.enable)
	mux.HandleFunc("POST /v1/plugins/{id}/disable", h.disable)
	mux.HandleFunc("DELETE /v1/plugins/{id}", h.uninstall)
	mux.HandleFunc("GET /v1/plugins/{id}/debug", h.debug)
	mux.HandleFunc("GET /v1/plugins/{id}/logs", h.logs)
	mux.HandleFunc("GET /v1/plugins/{id}/page", h.page)
	mux.HandleFunc("GET /v1/plugins/{id}/assets/{asset...}", h.asset)
}

func (h *PluginHandler) list(w http.ResponseWriter, r *http.Request) {
	onlyEnabled := false
	if raw := r.URL.Query().Get("enabled"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
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

	writeJSON(w, http.StatusOK, records)
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
		uiMode := string(item.UIMode)
		if uiMode == "" {
			uiMode = string(plugin.UIModeBackendOnly)
		}
		records = append(records, pluginRecord{
			ID:            item.ID,
			Name:          item.Name,
			Version:       item.Version,
			Enabled:       item.State == plugin.StateEnabled,
			UIMode:        uiMode,
			FrontendEntry: strings.TrimSpace(item.FrontendEntry),
			SystemBuiltin: item.SystemBuiltin || strings.EqualFold(strings.TrimSpace(item.Source), "builtin"),
		})
	}

	if len(records) == 0 {
		return defaultPluginRecords()
	}

	return records
}

func (h *PluginHandler) enable(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.manager == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("plugin manager not configured"))
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, errors.New("plugin id is required"))
		return
	}

	if err := h.manager.Enable(id); err != nil {
		if errors.Is(err, plugin.ErrPluginNotFound) {
			writeMessage(w, http.StatusNotFound, "not_found", "plugin not found")
			return
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	h.appendPluginLog(id, "enable", "ok", "enabled")

	writeMessage(w, http.StatusOK, "ok", "enabled")
}

func (h *PluginHandler) disable(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.manager == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("plugin manager not configured"))
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, errors.New("plugin id is required"))
		return
	}

	if err := h.manager.Disable(id); err != nil {
		if errors.Is(err, plugin.ErrPluginNotFound) {
			writeMessage(w, http.StatusNotFound, "not_found", "plugin not found")
			return
		}
		if errors.Is(err, plugin.ErrPluginSystemProtected) {
			writeMessage(w, http.StatusForbidden, "forbidden", "system builtin plugin cannot be disabled")
			return
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	h.appendPluginLog(id, "disable", "ok", "disabled")

	writeMessage(w, http.StatusOK, "ok", "disabled")
}

func (h *PluginHandler) uninstall(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.manager == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("plugin manager not configured"))
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, errors.New("plugin id is required"))
		return
	}

	if err := h.manager.Uninstall(id); err != nil {
		if errors.Is(err, plugin.ErrPluginNotFound) {
			writeMessage(w, http.StatusNotFound, "not_found", "plugin not found")
			return
		}
		if errors.Is(err, plugin.ErrPluginSystemProtected) {
			writeMessage(w, http.StatusForbidden, "forbidden", "system builtin plugin cannot be uninstalled")
			return
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	h.appendPluginLog(id, "uninstall", "ok", "uninstalled")

	writeMessage(w, http.StatusOK, "ok", "uninstalled")
}

func (h *PluginHandler) debug(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.manager == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("plugin manager not configured"))
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, errors.New("plugin id is required"))
		return
	}

	item, err := h.manager.Get(id)
	if err != nil {
		if errors.Is(err, plugin.ErrPluginNotFound) {
			writeMessage(w, http.StatusNotFound, "not_found", "plugin not found")
			return
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusOK, pluginDebugRecord{
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
		writeError(w, http.StatusBadRequest, errors.New("plugin id is required"))
		return
	}

	content, err := h.readPluginLogContent(id)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeMessage(w, http.StatusNotFound, "not_found", "plugin log not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, pluginLogsRecord{PluginID: id, Content: strings.TrimSpace(string(content))})
}

func (h *PluginHandler) page(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, errors.New("plugin id is required"))
		return
	}

	info, err := h.getPluginInfo(id)
	if err != nil {
		if errors.Is(err, plugin.ErrPluginNotFound) {
			writeMessage(w, http.StatusNotFound, "not_found", "plugin not found")
			return
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}

	frontendDir, err := resolvePluginFrontendDir(info)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}

	indexPath := filepath.Join(frontendDir, "index.html")
	content, err := os.ReadFile(indexPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeError(w, http.StatusNotFound, errors.New("plugin frontend index not found"))
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	baseTag := []byte("<base href=\"/v1/plugins/" + id + "/assets/\">")
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
		writeError(w, http.StatusBadRequest, errors.New("plugin id is required"))
		return
	}

	assetPath := strings.TrimSpace(r.PathValue("asset"))
	if assetPath == "" {
		writeError(w, http.StatusBadRequest, errors.New("asset path is required"))
		return
	}
	cleanAsset := filepath.Clean(assetPath)
	if cleanAsset == "." || strings.HasPrefix(cleanAsset, "..") || strings.Contains(cleanAsset, "..") {
		writeError(w, http.StatusBadRequest, errors.New("invalid asset path"))
		return
	}

	info, err := h.getPluginInfo(id)
	if err != nil {
		if errors.Is(err, plugin.ErrPluginNotFound) {
			writeMessage(w, http.StatusNotFound, "not_found", "plugin not found")
			return
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}

	frontendDir, err := resolvePluginFrontendDir(info)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}

	assetFile := filepath.Join(frontendDir, cleanAsset)
	rel, err := filepath.Rel(frontendDir, assetFile)
	if err != nil || strings.HasPrefix(rel, "..") {
		writeError(w, http.StatusBadRequest, errors.New("invalid asset target"))
		return
	}

	fi, err := os.Stat(assetFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeMessage(w, http.StatusNotFound, "not_found", "plugin asset not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if fi.IsDir() {
		writeError(w, http.StatusBadRequest, errors.New("asset path points to directory"))
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
		writeError(w, http.StatusServiceUnavailable, errors.New("plugin manager not configured"))
		return
	}

	req, err := decodePluginPathRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	info, err := h.manager.Install(req.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	h.appendPluginLog(info.ID, "install", "ok", "installed from path")

	writeJSON(w, http.StatusCreated, pluginRecord{
		ID:      info.ID,
		Name:    info.Name,
		Version: info.Version,
		Enabled: info.State == plugin.StateEnabled,
	})
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
		writeError(w, http.StatusServiceUnavailable, errors.New("plugin external registration is not configured"))
		return
	}

	var req externalPluginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	req.PluginID = strings.TrimSpace(req.PluginID)
	req.Name = strings.TrimSpace(req.Name)
	req.Version = strings.TrimSpace(req.Version)
	req.URL = strings.TrimSpace(req.URL)
	req.OpenMode = strings.TrimSpace(req.OpenMode)
	req.RoutePath = strings.TrimSpace(req.RoutePath)
	if req.PluginID == "" || req.Name == "" || req.URL == "" {
		writeMessage(w, http.StatusBadRequest, "invalid_request", "pluginId, name and url are required")
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
		FrontendEntry: req.URL,
		SystemBuiltin: false,
	}

	if err := registrar.RegisterExternalPlugin(info); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	h.appendPluginLog(req.PluginID, "create_"+pluginType, "ok", "registered external plugin")
	writeJSON(w, http.StatusCreated, map[string]any{
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
		writeError(w, http.StatusServiceUnavailable, errors.New("plugin loader not configured"))
		return
	}

	req, err := decodePluginPathRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	info, err := h.loader.Load(req.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":           info.ID,
		"name":         info.Name,
		"version":      info.Version,
		"dependencies": len(info.Dependencies),
		"permissions":  len(info.Permissions),
	})
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
			FrontendEntry: "/plugins/auth",
			SystemBuiltin: true,
		},
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
