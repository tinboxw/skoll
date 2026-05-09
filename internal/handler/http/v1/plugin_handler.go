package v1

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tinboxw/skoll/internal/plugin"
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

type PluginHandler struct {
	manager           PluginManager
	extensionProvider PluginExtensionSnapshotProvider
	loader            plugin.MetadataLoader
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

func RegisterPluginRoutes(mux *http.ServeMux, manager PluginManager) {
	h := &PluginHandler{manager: manager, loader: plugin.NewFileLoader()}
	if provider, ok := manager.(PluginExtensionSnapshotProvider); ok {
		h.extensionProvider = provider
	}
	mux.HandleFunc("GET /v1/plugins", h.list)
	mux.HandleFunc("POST /v1/plugins/install", h.install)
	mux.HandleFunc("POST /v1/plugins/validate", h.validate)
	mux.HandleFunc("POST /v1/plugins/{id}/enable", h.enable)
	mux.HandleFunc("POST /v1/plugins/{id}/disable", h.disable)
	mux.HandleFunc("DELETE /v1/plugins/{id}", h.uninstall)
	mux.HandleFunc("GET /v1/plugins/{id}/debug", h.debug)
	mux.HandleFunc("GET /v1/plugins/{id}/logs", h.logs)
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

	content, err := os.ReadFile(filepath.Join("plugins", "logs", id+".log"))
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

	writeJSON(w, http.StatusCreated, pluginRecord{
		ID:      info.ID,
		Name:    info.Name,
		Version: info.Version,
		Enabled: info.State == plugin.StateEnabled,
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
