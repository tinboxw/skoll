package plugin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	apiv1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	pluginruntime "github.com/tinboxw/skoll/internal/plugin"
	jobsvc "github.com/tinboxw/skoll/internal/service/job"
	"github.com/tinboxw/skoll/pkg/security"
)

type retryDeadLetterJobRequest struct {
	ConfirmPluginID string `json:"confirmPluginId"`
	ConfirmJobID    string `json:"confirmJobId"`
}

func (h *PluginHandler) diagnostics(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.diagnosticsProvider == nil {
		apiv1.WriteMessage(w, http.StatusServiceUnavailable, "plugin_diagnostics_unavailable", "plugin diagnostics are not configured")
		return
	}
	pluginID := strings.TrimSpace(r.PathValue("id"))
	if pluginID == "" {
		apiv1.WriteMessage(w, http.StatusBadRequest, "invalid_plugin_id", "plugin ID is required")
		return
	}
	limit := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			apiv1.WriteMessage(w, http.StatusBadRequest, "invalid_diagnostic_limit", "diagnostic limit must be an integer")
			return
		}
		limit = parsed
	}
	snapshot, err := h.diagnosticsProvider.Inspect(r.Context(), pluginID, pluginruntime.DiagnosticQuery{
		JobStatus:   jobsvc.Status(strings.TrimSpace(r.URL.Query().Get("jobStatus"))),
		AuditResult: strings.TrimSpace(r.URL.Query().Get("auditResult")),
		Correlation: strings.TrimSpace(r.URL.Query().Get("correlation")),
		Limit:       limit,
	})
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

func (h *PluginHandler) retryDeadLetterJob(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.diagnosticsProvider == nil {
		apiv1.WriteMessage(w, http.StatusServiceUnavailable, "plugin_diagnostics_unavailable", "plugin diagnostics are not configured")
		return
	}
	if !h.isSuperAdmin(r) {
		apiv1.WriteMessage(w, http.StatusForbidden, "plugin_job_retry_forbidden", "only super administrators can retry dead-letter jobs")
		return
	}
	pluginID := strings.TrimSpace(r.PathValue("id"))
	jobID := strings.TrimSpace(r.PathValue("jobId"))
	if pluginID == "" || jobID == "" {
		apiv1.WriteMessage(w, http.StatusBadRequest, "invalid_plugin_job", "plugin ID and job ID are required")
		return
	}
	var request retryDeadLetterJobRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		apiv1.WriteMessage(w, http.StatusBadRequest, "invalid_plugin_job_retry", "invalid retry confirmation")
		return
	}
	if request.ConfirmPluginID != pluginID || request.ConfirmJobID != jobID {
		apiv1.WriteMessage(w, http.StatusBadRequest, "plugin_job_retry_confirmation_mismatch", "plugin and job confirmation must match exactly")
		return
	}

	result, err := h.diagnosticsProvider.RetryDeadLetter(r.Context(), pluginID, jobID)
	if err != nil {
		switch {
		case errors.Is(err, pluginruntime.ErrPluginNotFound), errors.Is(err, jobsvc.ErrNotFound):
			apiv1.WriteMessage(w, http.StatusNotFound, "plugin_job_not_found", "plugin job not found")
		case strings.Contains(err.Error(), "only dead-letter"):
			apiv1.WriteMessage(w, http.StatusConflict, "plugin_job_not_dead_letter", "only dead-letter jobs can be retried")
		default:
			apiv1.WriteError(w, http.StatusBadRequest, err)
		}
		return
	}
	actorID := "system"
	if claims, ok := security.JWTClaimsFromContext(r.Context()); ok && strings.TrimSpace(claims.Subject) != "" {
		actorID = strings.TrimSpace(claims.Subject)
	}
	h.appendAudit(r, "retry_dead_letter", "plugin_job", jobID, map[string]any{
		"pluginId": pluginID, "sourceJobId": jobID, "retryJobId": result.RetryJob.ID,
		"operationId": result.OperationID, "actorId": actorID,
	})
	apiv1.WriteJSON(w, http.StatusOK, result)
}
