package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	apiv1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	pluginruntime "github.com/tinboxw/skoll/internal/plugin"
)

const maxPluginMigrationRollbackSteps = 100

type pluginMigrationRollbackRequest struct {
	Limit           int    `json:"limit"`
	ConfirmPluginID string `json:"confirmPluginId"`
}

type pluginMigrationRollbackResult struct {
	OperationID     string                            `json:"operationId"`
	CompletedAt     time.Time                         `json:"completedAt"`
	RolledBackSteps int                               `json:"rolledBackSteps"`
	Snapshot        pluginruntime.DataControlSnapshot `json:"snapshot"`
}

func (h *PluginHandler) dataControl(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.dataControlProvider == nil {
		apiv1.WriteMessage(w, http.StatusServiceUnavailable, "plugin_data_control_unavailable", "plugin data control is not configured")
		return
	}
	pluginID := strings.TrimSpace(r.PathValue("id"))
	if pluginID == "" {
		apiv1.WriteMessage(w, http.StatusBadRequest, "invalid_plugin_id", "plugin ID is required")
		return
	}
	snapshot, err := h.dataControlProvider.PluginDataControl(r.Context(), pluginID)
	if err != nil {
		if errors.Is(err, pluginruntime.ErrPluginNotFound) {
			apiv1.WriteMessage(w, http.StatusNotFound, "plugin_not_found", "plugin not found")
			return
		}
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, snapshot)
}

func (h *PluginHandler) rollbackMigration(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.dataControlProvider == nil {
		apiv1.WriteMessage(w, http.StatusServiceUnavailable, "plugin_data_control_unavailable", "plugin data control is not configured")
		return
	}
	if !h.isSuperAdmin(r) {
		apiv1.WriteMessage(w, http.StatusForbidden, "super_admin_required", "super_admin role is required")
		return
	}
	pluginID := strings.TrimSpace(r.PathValue("id"))
	if pluginID == "" {
		apiv1.WriteMessage(w, http.StatusBadRequest, "invalid_plugin_id", "plugin ID is required")
		return
	}
	var request pluginMigrationRollbackRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		apiv1.WriteMessage(w, http.StatusBadRequest, "invalid_rollback_request", "rollback request must be valid JSON")
		return
	}
	if strings.TrimSpace(request.ConfirmPluginID) != pluginID {
		apiv1.WriteMessage(w, http.StatusBadRequest, "plugin_confirmation_mismatch", "confirmPluginId must exactly match the target plugin ID")
		return
	}
	if request.Limit <= 0 || request.Limit > maxPluginMigrationRollbackSteps {
		apiv1.WriteMessage(w, http.StatusBadRequest, "invalid_rollback_limit", fmt.Sprintf("limit must be between 1 and %d", maxPluginMigrationRollbackSteps))
		return
	}

	before, err := h.dataControlProvider.PluginDataControl(r.Context(), pluginID)
	if err != nil {
		h.writeDataControlError(w, err)
		return
	}
	if !before.Actions.CanRollback {
		apiv1.WriteResponse(w, http.StatusConflict, "migration_rollback_blocked", "plugin migration rollback is blocked", map[string]any{
			"blockedReason": before.Actions.BlockedReason,
			"snapshot":      before,
		})
		return
	}
	if request.Limit > before.Actions.RollbackMaxSteps {
		apiv1.WriteMessage(w, http.StatusBadRequest, "rollback_limit_exceeded", "limit exceeds applied migration count")
		return
	}

	operationID := fmt.Sprintf("plugin-migration-rollback-%d", time.Now().UTC().UnixNano())
	if err := h.dataControlProvider.RollbackPluginData(pluginID, request.Limit); err != nil {
		h.appendAudit(r, "migration_rollback_failed", "plugin", pluginID, map[string]any{
			"operationId": operationID, "limit": request.Limit, "error": err.Error(),
		})
		apiv1.WriteError(w, http.StatusConflict, err)
		return
	}
	after, err := h.dataControlProvider.PluginDataControl(r.Context(), pluginID)
	if err != nil {
		h.writeDataControlError(w, err)
		return
	}
	completedAt := time.Now().UTC()
	h.appendAudit(r, "migration_rollback", "plugin", pluginID, map[string]any{
		"operationId": operationID, "limit": request.Limit, "rollbackPolicy": before.Policy.Rollback,
		"beforeVersion": before.Migration.CurrentVersion, "afterVersion": after.Migration.CurrentVersion,
	})
	apiv1.WriteJSON(w, http.StatusOK, pluginMigrationRollbackResult{
		OperationID: operationID, CompletedAt: completedAt, RolledBackSteps: request.Limit, Snapshot: after,
	})
}

func (h *PluginHandler) writeDataControlError(w http.ResponseWriter, err error) {
	if errors.Is(err, pluginruntime.ErrPluginNotFound) {
		apiv1.WriteMessage(w, http.StatusNotFound, "plugin_not_found", "plugin not found")
		return
	}
	apiv1.WriteError(w, http.StatusBadRequest, err)
}
