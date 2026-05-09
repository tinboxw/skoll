package v1

import (
	"net/http"
	"strconv"

	"github.com/tinboxw/skoll/internal/plugin"
)

type PluginListProvider interface {
	List() []plugin.Info
}

type PluginHandler struct {
	provider PluginListProvider
}

type pluginRecord struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Enabled bool   `json:"enabled"`
}

func RegisterPluginRoutes(mux *http.ServeMux, provider PluginListProvider) {
	h := &PluginHandler{provider: provider}
	mux.HandleFunc("GET /v1/plugins", h.list)
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
	if h == nil || h.provider == nil {
		return defaultPluginRecords()
	}

	items := h.provider.List()
	if len(items) == 0 {
		return defaultPluginRecords()
	}

	records := make([]pluginRecord, 0, len(items))
	for _, item := range items {
		if item.State == plugin.StateUninstalled {
			continue
		}
		records = append(records, pluginRecord{
			ID:      item.ID,
			Name:    item.Name,
			Version: item.Version,
			Enabled: item.State == plugin.StateEnabled,
		})
	}

	if len(records) == 0 {
		return defaultPluginRecords()
	}

	return records
}

func defaultPluginRecords() []pluginRecord {
	return []pluginRecord{
		{
			ID:      "builtin-auth",
			Name:    "Builtin Auth",
			Version: "1.0.0",
			Enabled: true,
		},
	}
}
