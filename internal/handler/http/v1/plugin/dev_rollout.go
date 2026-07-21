package plugin

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
	"time"

	"github.com/tinboxw/skoll/internal/adapter"
	apiv1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	pluginpkg "github.com/tinboxw/skoll/internal/plugin"
)

type devRolloutRequest struct {
	PluginID      string   `json:"pluginId"`
	Percent       *int     `json:"percent"`
	StrategyType  string   `json:"strategyType"`
	TargetEnv     string   `json:"targetEnv"`
	Tags          []string `json:"tags"`
	CanaryVersion string   `json:"canaryVersion"`
}

type devRollbackRequest struct {
	PluginID  string `json:"pluginId"`
	ToVersion string `json:"toVersion,omitempty"`
	ToPercent *int   `json:"toPercent,omitempty"`
}

type devRolloutResponse struct {
	Operation      string                        `json:"operation"`
	Status         string                        `json:"status"`
	PluginID       string                        `json:"pluginId"`
	RolloutPercent int                           `json:"rolloutPercent"`
	Persisted      bool                          `json:"persisted"`
	Message        string                        `json:"message,omitempty"`
	Task           devRolloutTaskRecord          `json:"task"`
	Rollback       *pluginpkg.PluginRollbackPlan `json:"rollback,omitempty"`
}

const (
	devRolloutTaskActionRollout  = "rollout"
	devRolloutTaskActionRollback = "rollback"

	devRolloutTaskStepPrepare = "prepare"
	devRolloutTaskStepPersist = "persist"
)

type devRolloutTaskStep struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
	StartedAt  string `json:"startedAt,omitempty"`
	FinishedAt string `json:"finishedAt,omitempty"`
	DurationMs int64  `json:"durationMs"`
}

type devRolloutTaskRecord struct {
	TaskID                  string                    `json:"taskId"`
	PluginID                string                    `json:"pluginId"`
	Action                  string                    `json:"action"`
	RequestedRolloutPercent int                       `json:"requestedRolloutPercent"`
	PreviousRolloutPercent  int                       `json:"previousRolloutPercent"`
	RolloutPercent          int                       `json:"rolloutPercent"`
	Version                 string                    `json:"version,omitempty"`
	RollbackToVersion       string                    `json:"rollbackToVersion,omitempty"`
	RollbackToPercent       *int                      `json:"rollbackToPercent,omitempty"`
	StrategyType            string                    `json:"strategyType,omitempty"`
	TargetEnv               string                    `json:"targetEnv,omitempty"`
	Tags                    []string                  `json:"tags,omitempty"`
	CanaryVersion           string                    `json:"canaryVersion,omitempty"`
	TaskStatus              string                    `json:"taskStatus"`
	CreatedBy               string                    `json:"createdBy"`
	CreatedAt               string                    `json:"createdAt"`
	StartedAt               string                    `json:"startedAt,omitempty"`
	FinishedAt              string                    `json:"finishedAt,omitempty"`
	FailureReason           string                    `json:"failureReason,omitempty"`
	ExecutionSteps          []devRolloutTaskStep      `json:"steps,omitempty"`
	ExecutionLogs           []devReleaseTaskLogRecord `json:"logs,omitempty"`
	ExecutionResult         string                    `json:"executionResult,omitempty"`
}

type devRolloutTaskEnvelope struct {
	Tasks []devRolloutTaskRecord `json:"tasks"`
}

type devRolloutPoint struct {
	Percent       int      `json:"percent"`
	TaskID        string   `json:"taskId"`
	Version       string   `json:"version"`
	Type          string   `json:"type"`
	StrategyType  string   `json:"strategyType,omitempty"`
	TargetEnv     string   `json:"targetEnv,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	CanaryVersion string   `json:"canaryVersion,omitempty"`
	CreatedAt     string   `json:"createdAt"`
	CreatedBy     string   `json:"createdBy"`
}

type devRolloutHistoryEntry struct {
	Points      []devRolloutPoint `json:"points"`
	LastSuccess string            `json:"last_success"`
	UpdatedBy   string            `json:"updated_by"`
	UpdatedAt   string            `json:"updated_at"`
}

type devListRolloutTasksResponse struct {
	Operation  string                 `json:"operation"`
	Status     string                 `json:"status"`
	PluginID   string                 `json:"pluginId,omitempty"`
	TaskStatus string                 `json:"taskStatus,omitempty"`
	Tasks      []devRolloutTaskRecord `json:"tasks"`
}

type devGetRolloutTaskResponse struct {
	Operation string               `json:"operation"`
	Status    string               `json:"status"`
	Task      devRolloutTaskRecord `json:"task"`
}

type devRolloutTaskLogsResponse struct {
	Operation string                    `json:"operation"`
	Status    string                    `json:"status"`
	TaskID    string                    `json:"taskId"`
	Logs      []devReleaseTaskLogRecord `json:"logs"`
}

func (h *PluginHandler) devRollout(w http.ResponseWriter, r *http.Request) {
	if !h.isSuperAdmin(r) {
		apiv1.WriteMessage(w, http.StatusForbidden, "forbidden", "super_admin role required")
		return
	}

	var req devRolloutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	pluginID := strings.TrimSpace(strings.ToLower(req.PluginID))
	if pluginID == "" {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("pluginId is required"))
		return
	}

	strategyType := strings.TrimSpace(req.StrategyType)
	if strategyType == "" {
		strategyType = "percent"
	}
	targetEnv := strings.TrimSpace(req.TargetEnv)
	if targetEnv == "" {
		targetEnv = "production"
	}

	var rolloutPercent int
	switch strategyType {
	case "percent":
		if req.Percent == nil {
			apiv1.WriteError(w, http.StatusBadRequest, errors.New("percent is required when strategyType is percent"))
			return
		}
		if *req.Percent < 0 || *req.Percent > 100 {
			apiv1.WriteError(w, http.StatusBadRequest, errors.New("percent must be in range [0,100]"))
			return
		}
		rolloutPercent = *req.Percent
	case "tag":
		if len(req.Tags) == 0 {
			apiv1.WriteError(w, http.StatusBadRequest, errors.New("tags is required when strategyType is tag"))
			return
		}
	case "canary":
		if strings.TrimSpace(req.CanaryVersion) == "" {
			apiv1.WriteError(w, http.StatusBadRequest, errors.New("canaryVersion is required when strategyType is canary"))
			return
		}
	default:
		apiv1.WriteError(w, http.StatusBadRequest, fmt.Errorf("unsupported strategyType: %s", strategyType))
		return
	}

	task, err := h.runDevRolloutTask(pluginID, devRolloutTaskActionRollout, rolloutPercent, actorIDFromJWT(r), "", nil, strategyType, targetEnv, req.Tags, req.CanaryVersion)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	persisted := task.TaskStatus == devReleaseTaskStatusSuccess
	msg := task.ExecutionResult
	if strings.TrimSpace(msg) == "" {
		msg = "rollout updated"
	}

	h.appendAudit(r, "dev_rollout", "plugin", pluginID, map[string]any{
		"rolloutPercent": rolloutPercent,
		"strategyType":   strategyType,
		"targetEnv":      targetEnv,
		"persisted":      persisted,
		"taskId":         task.TaskID,
	})
	apiv1.WriteJSON(w, http.StatusOK, devRolloutResponse{
		Operation:      "rollout",
		Status:         "ok",
		PluginID:       pluginID,
		RolloutPercent: rolloutPercent,
		Persisted:      persisted,
		Message:        msg,
		Task:           task,
	})
}

func (h *PluginHandler) devRollback(w http.ResponseWriter, r *http.Request) {
	if !h.isSuperAdmin(r) {
		apiv1.WriteMessage(w, http.StatusForbidden, "forbidden", "super_admin role required")
		return
	}

	var req devRollbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	pluginID := strings.TrimSpace(strings.ToLower(req.PluginID))
	if pluginID == "" {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("pluginId is required"))
		return
	}
	if req.ToVersion != "" && req.ToPercent != nil {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("toVersion and toPercent are mutually exclusive"))
		return
	}
	if req.ToPercent != nil && (*req.ToPercent < 0 || *req.ToPercent > 100) {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("toPercent must be in range [0,100]"))
		return
	}

	beforeInfo, err := h.manager.Get(pluginID)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	task, err := h.runDevRolloutTask(pluginID, devRolloutTaskActionRollback, 0, actorIDFromJWT(r), req.ToVersion, req.ToPercent, "", "", nil, "")
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	persisted := task.TaskStatus == devReleaseTaskStatusSuccess
	msg := task.ExecutionResult
	if strings.TrimSpace(msg) == "" {
		msg = "rollback applied"
	}
	prev := task.RolloutPercent
	afterInfo, err := h.manager.Get(pluginID)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	rollbackPlan, err := pluginpkg.NewPluginRollbackService().BuildPlan(pluginpkg.PluginRollbackPlanInput{
		PluginID:      pluginID,
		From:          beforeInfo,
		Target:        afterInfo,
		TargetPercent: &prev,
		TargetVersion: req.ToVersion,
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	h.appendAudit(r, "dev_rollback", "plugin", pluginID, map[string]any{"rolloutPercent": prev, "persisted": persisted, "taskId": task.TaskID, "rollbackStatus": rollbackPlan.Status})
	apiv1.WriteJSON(w, http.StatusOK, devRolloutResponse{
		Operation:      "rollback",
		Status:         "ok",
		PluginID:       pluginID,
		RolloutPercent: prev,
		Persisted:      persisted,
		Message:        msg,
		Task:           task,
		Rollback:       &rollbackPlan,
	})
}

func (h *PluginHandler) devListRolloutTasks(w http.ResponseWriter, r *http.Request) {
	if !h.isSuperAdmin(r) {
		apiv1.WriteMessage(w, http.StatusForbidden, "forbidden", "super_admin role required")
		return
	}

	pluginID := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("pluginId")))
	taskStatus := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("status")))

	tasks, err := h.listDevRolloutTasks(pluginID)
	if err != nil {
		apiv1.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	filtered := make([]devRolloutTaskRecord, 0, len(tasks))
	for _, item := range tasks {
		if taskStatus != "" && !strings.EqualFold(item.TaskStatus, taskStatus) {
			continue
		}
		filtered = append(filtered, item)
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt > filtered[j].CreatedAt
	})

	apiv1.WriteJSON(w, http.StatusOK, devListRolloutTasksResponse{
		Operation:  "rollout_task_list",
		Status:     "ok",
		PluginID:   pluginID,
		TaskStatus: taskStatus,
		Tasks:      filtered,
	})
}

func (h *PluginHandler) devGetRolloutTask(w http.ResponseWriter, r *http.Request) {
	if !h.isSuperAdmin(r) {
		apiv1.WriteMessage(w, http.StatusForbidden, "forbidden", "super_admin role required")
		return
	}

	taskID := strings.TrimSpace(r.PathValue("taskId"))
	if taskID == "" {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("taskId is required"))
		return
	}

	task, found, err := h.findDevRolloutTask(taskID)
	if err != nil {
		apiv1.WriteError(w, http.StatusInternalServerError, err)
		return
	}
	if !found {
		apiv1.WriteError(w, http.StatusNotFound, errors.New("rollout task not found"))
		return
	}

	apiv1.WriteJSON(w, http.StatusOK, devGetRolloutTaskResponse{
		Operation: "rollout_task_get",
		Status:    "ok",
		Task:      task,
	})
}

func (h *PluginHandler) devGetRolloutTaskLogs(w http.ResponseWriter, r *http.Request) {
	if !h.isSuperAdmin(r) {
		apiv1.WriteMessage(w, http.StatusForbidden, "forbidden", "super_admin role required")
		return
	}

	taskID := strings.TrimSpace(r.PathValue("taskId"))
	if taskID == "" {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("taskId is required"))
		return
	}

	task, found, err := h.findDevRolloutTask(taskID)
	if err != nil {
		apiv1.WriteError(w, http.StatusInternalServerError, err)
		return
	}
	if !found {
		apiv1.WriteError(w, http.StatusNotFound, errors.New("rollout task not found"))
		return
	}

	apiv1.WriteJSON(w, http.StatusOK, devRolloutTaskLogsResponse{
		Operation: "rollout_task_logs",
		Status:    "ok",
		TaskID:    taskID,
		Logs:      task.ExecutionLogs,
	})
}

func (h *PluginHandler) applyDevRollout(pluginID string, percent int, actorID string, taskID string, strategyType, targetEnv string, tags []string, canaryVersion string) (persisted bool, message string, version string, err error) {
	item, getErr := h.manager.Get(pluginID)
	if getErr != nil {
		return false, "", "", getErr
	}

	current := 100
	if raw := strings.TrimSpace(item.ConfigJSON); raw != "" {
		cfg := map[string]any{}
		if jsonErr := json.Unmarshal([]byte(raw), &cfg); jsonErr == nil {
			if v, ok := cfg[rolloutStrategyConfigKey("percent", targetEnv)]; ok {
				switch n := v.(type) {
				case float64:
					current = int(n)
				case int:
					current = n
				}
			}
		}
	}

	entry, loadErr := h.loadDevRolloutHistory(pluginID)
	if loadErr != nil && !errors.Is(loadErr, os.ErrNotExist) {
		return false, "", "", loadErr
	}

	version = nextRolloutVersion(entry)
	point := devRolloutPoint{
		Percent:       current,
		TaskID:        taskID,
		Version:       version,
		Type:          "rollout",
		StrategyType:  strategyType,
		TargetEnv:     targetEnv,
		Tags:          tags,
		CanaryVersion: canaryVersion,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		CreatedBy:     actorID,
	}
	entry.Points = append(entry.Points, point)
	entry.LastSuccess = taskID
	entry.UpdatedBy = actorID
	entry.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if saveErr := h.saveDevRolloutHistory(pluginID, entry); saveErr != nil {
		return false, "", "", saveErr
	}

	// 调用灰度执行器（mock 版本打印日志，生产环境对接真实网关）
	strategy := adapter.DevRolloutStrategy{
		PluginID:      pluginID,
		StrategyType:  strategyType,
		TargetEnv:     targetEnv,
		Percent:       percent,
		Tags:          tags,
		CanaryVersion: canaryVersion,
		TaskID:        taskID,
		ActorID:       actorID,
	}
	execResult, execErr := h.devRolloutExecutor.ApplyRollout(context.Background(), strategy)
	if execErr != nil {
		return false, "", "", execErr
	}
	if execResult != nil && !execResult.Success {
		return false, "", "", fmt.Errorf("rollout executor failed: %s", execResult.Message)
	}

	persisted, message, err = h.persistDevRolloutStrategy(pluginID, strategyType, targetEnv, percent, tags, canaryVersion)
	if err != nil {
		return false, "", "", err
	}
	if message == "" {
		message = "rollout updated"
	}
	return persisted, message, version, nil
}

func (h *PluginHandler) applyDevRollback(pluginID string, actorID string, taskID string, toVersion string, toPercent *int) (persisted bool, message string, rollbackPercent int, err error) {
	entry, loadErr := h.loadDevRolloutHistory(pluginID)
	if loadErr != nil {
		return false, "", 0, loadErr
	}
	if len(entry.Points) == 0 {
		return false, "", 0, errors.New("no rollback history for plugin")
	}

	var targetPercent int
	var previousState adapter.DevRolloutStrategy

	if toVersion != "" {
		found := false
		for _, p := range entry.Points {
			if p.Version == toVersion {
				targetPercent = p.Percent
				previousState = adapter.DevRolloutStrategy{
					PluginID:      pluginID,
					StrategyType:  p.StrategyType,
					TargetEnv:     p.TargetEnv,
					Percent:       p.Percent,
					Tags:          p.Tags,
					CanaryVersion: p.CanaryVersion,
					TaskID:        p.TaskID,
				}
				found = true
				break
			}
		}
		if !found {
			return false, "", 0, fmt.Errorf("rollback target version %s not found in history", toVersion)
		}
	} else if toPercent != nil {
		targetPercent = *toPercent
		// Use the last point as previous state
		last := entry.Points[len(entry.Points)-1]
		previousState = adapter.DevRolloutStrategy{
			PluginID:      pluginID,
			StrategyType:  last.StrategyType,
			TargetEnv:     last.TargetEnv,
			Percent:       last.Percent,
			Tags:          last.Tags,
			CanaryVersion: last.CanaryVersion,
			TaskID:        last.TaskID,
		}
	} else {
		// Roll back to the state before the last rollout.
		// Remove the last point and use the new last point's percent
		last := entry.Points[len(entry.Points)-1]
		previousState = adapter.DevRolloutStrategy{
			PluginID:      pluginID,
			StrategyType:  last.StrategyType,
			TargetEnv:     last.TargetEnv,
			Percent:       last.Percent,
			Tags:          last.Tags,
			CanaryVersion: last.CanaryVersion,
			TaskID:        last.TaskID,
		}
		entry.Points = entry.Points[:len(entry.Points)-1]
		if len(entry.Points) == 0 {
			targetPercent = 100
		} else {
			targetPercent = entry.Points[len(entry.Points)-1].Percent
		}
	}

	// 调用灰度执行器执行回滚
	targetStrategy := adapter.DevRolloutStrategy{
		PluginID: pluginID,
		Percent:  targetPercent,
	}
	execResult, execErr := h.devRolloutExecutor.ApplyRollback(context.Background(), previousState, targetStrategy)
	if execErr != nil {
		return false, "", 0, execErr
	}
	if execResult != nil && !execResult.Success {
		return false, "", 0, fmt.Errorf("rollback executor failed: %s", execResult.Message)
	}

	// Create rollback point recording the restored state
	version := nextRolloutVersion(entry)
	point := devRolloutPoint{
		Percent:   targetPercent,
		TaskID:    taskID,
		Version:   version,
		Type:      "rollback",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		CreatedBy: actorID,
	}
	entry.Points = append(entry.Points, point)
	entry.LastSuccess = taskID
	entry.UpdatedBy = actorID
	entry.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	if saveErr := h.saveDevRolloutHistory(pluginID, entry); saveErr != nil {
		return false, "", 0, saveErr
	}

	persisted, message, err = h.persistDevRollout(pluginID, targetPercent)
	if err != nil {
		return false, "", 0, err
	}
	if message == "" {
		message = "rollback applied"
	}
	return persisted, message, targetPercent, nil
}

func (h *PluginHandler) persistDevRollout(pluginID string, percent int) (bool, string, error) {
	item, err := h.manager.Get(pluginID)
	if err != nil {
		return false, "", err
	}

	updater, ok := h.manager.(PluginConfigUpdater)
	if !ok {
		return false, "", errors.New("runtime config updater is required")
	}

	cfg := map[string]any{}
	if raw := strings.TrimSpace(item.ConfigJSON); raw != "" {
		if unmarshalErr := json.Unmarshal([]byte(raw), &cfg); unmarshalErr != nil {
			return false, "", unmarshalErr
		}
	}
	cfg[rolloutStrategyConfigKey("percent", "")] = percent
	cfg["dev_rollout_updated_at"] = time.Now().UTC().Format(time.RFC3339)

	if saveErr := updater.SavePluginConfig(pluginID, cfg); saveErr != nil {
		return false, "", saveErr
	}
	return true, "rollout persisted to plugin config", nil
}

func (h *PluginHandler) runDevRolloutTask(pluginID, action string, requestedPercent int, actorID string, rollbackToVersion string, rollbackToPercent *int, strategyType, targetEnv string, tags []string, canaryVersion string) (devRolloutTaskRecord, error) {
	task, err := h.createDevRolloutTask(pluginID, action, requestedPercent, actorID, rollbackToVersion, rollbackToPercent, strategyType, targetEnv, tags, canaryVersion)
	if err != nil {
		return devRolloutTaskRecord{}, err
	}

	if err := h.updateDevRolloutTask(pluginID, task.TaskID, func(target *devRolloutTaskRecord) error {
		now := time.Now().UTC().Format(time.RFC3339)
		target.TaskStatus = devReleaseTaskStatusRunning
		target.StartedAt = now
		target.ExecutionLogs = append(target.ExecutionLogs, devReleaseTaskLogRecord{Timestamp: now, Level: "info", Message: "task started"})
		return nil
	}); err != nil {
		return devRolloutTaskRecord{}, err
	}

	if err := h.runDevRolloutTaskStep(pluginID, task.TaskID, devRolloutTaskStepPrepare, func(target *devRolloutTaskRecord) error {
		current, currentErr := h.currentDevRolloutPercent(pluginID)
		if currentErr != nil {
			return currentErr
		}
		target.PreviousRolloutPercent = current
		return nil
	}); err != nil {
		return h.finishFailedDevRolloutTask(pluginID, task.TaskID, err)
	}

	if err := h.runDevRolloutTaskStep(pluginID, task.TaskID, devRolloutTaskStepPersist, func(target *devRolloutTaskRecord) error {
		if target.Action == devRolloutTaskActionRollout {
			persisted, message, version, applyErr := h.applyDevRollout(pluginID, target.RequestedRolloutPercent, actorID, task.TaskID, strategyType, targetEnv, tags, canaryVersion)
			if applyErr != nil {
				return applyErr
			}
			target.RolloutPercent = target.RequestedRolloutPercent
			target.Version = version
			target.ExecutionResult = message
			if !persisted {
				return errors.New("rollout not persisted")
			}
			return nil
		}
		persisted, message, rollbackPercent, applyErr := h.applyDevRollback(pluginID, actorID, task.TaskID, target.RollbackToVersion, target.RollbackToPercent)
		if applyErr != nil {
			return applyErr
		}
		target.RolloutPercent = rollbackPercent
		target.ExecutionResult = message
		if !persisted {
			return errors.New("rollback not persisted")
		}
		return nil
	}); err != nil {
		return h.finishFailedDevRolloutTask(pluginID, task.TaskID, err)
	}

	if err := h.updateDevRolloutTask(pluginID, task.TaskID, func(target *devRolloutTaskRecord) error {
		now := time.Now().UTC().Format(time.RFC3339)
		target.TaskStatus = devReleaseTaskStatusSuccess
		target.FinishedAt = now
		target.ExecutionLogs = append(target.ExecutionLogs, devReleaseTaskLogRecord{Timestamp: now, Level: "info", Message: "task completed"})
		return nil
	}); err != nil {
		return devRolloutTaskRecord{}, err
	}

	result, found, getErr := h.getDevRolloutTask(pluginID, task.TaskID)
	if getErr != nil {
		return devRolloutTaskRecord{}, getErr
	}
	if !found {
		return devRolloutTaskRecord{}, errors.New("rollout task not found")
	}
	return result, nil
}

func (h *PluginHandler) finishFailedDevRolloutTask(pluginID, taskID string, cause error) (devRolloutTaskRecord, error) {
	if updateErr := h.updateDevRolloutTask(pluginID, taskID, func(target *devRolloutTaskRecord) error {
		now := time.Now().UTC().Format(time.RFC3339)
		target.TaskStatus = devReleaseTaskStatusFailed
		target.FinishedAt = now
		target.FailureReason = cause.Error()
		target.ExecutionLogs = append(target.ExecutionLogs, devReleaseTaskLogRecord{Timestamp: now, Level: "error", Message: cause.Error()})
		return nil
	}); updateErr != nil {
		return devRolloutTaskRecord{}, updateErr
	}
	result, found, getErr := h.getDevRolloutTask(pluginID, taskID)
	if getErr != nil {
		return devRolloutTaskRecord{}, getErr
	}
	if !found {
		return devRolloutTaskRecord{}, errors.New("rollout task not found")
	}
	return result, cause
}

func (h *PluginHandler) runDevRolloutTaskStep(pluginID, taskID, stepName string, stepFn func(task *devRolloutTaskRecord) error) error {
	startedAt := time.Now().UTC()
	step := devRolloutTaskStep{Name: stepName, Status: devReleaseTaskStatusRunning, StartedAt: startedAt.Format(time.RFC3339)}

	if err := h.updateDevRolloutTask(pluginID, taskID, func(task *devRolloutTaskRecord) error {
		task.ExecutionSteps = append(task.ExecutionSteps, step)
		task.ExecutionLogs = append(task.ExecutionLogs, devReleaseTaskLogRecord{Timestamp: startedAt.Format(time.RFC3339), Step: stepName, Level: "info", Message: "step started"})
		return nil
	}); err != nil {
		return err
	}

	var stepErr error
	err := h.updateDevRolloutTask(pluginID, taskID, func(task *devRolloutTaskRecord) error {
		idx := findDevRolloutTaskStepIndex(task.ExecutionSteps, stepName)
		if idx < 0 {
			return errors.New("rollout task step not found")
		}
		stepErr = stepFn(task)
		finishedAt := time.Now().UTC()
		task.ExecutionSteps[idx].FinishedAt = finishedAt.Format(time.RFC3339)
		task.ExecutionSteps[idx].DurationMs = finishedAt.Sub(startedAt).Milliseconds()
		if stepErr != nil {
			task.ExecutionSteps[idx].Status = devReleaseTaskStatusFailed
			task.ExecutionSteps[idx].Message = stepErr.Error()
			task.ExecutionLogs = append(task.ExecutionLogs, devReleaseTaskLogRecord{Timestamp: finishedAt.Format(time.RFC3339), Step: stepName, Level: "error", Message: stepErr.Error()})
			return nil
		}
		task.ExecutionSteps[idx].Status = devReleaseTaskStatusSuccess
		task.ExecutionLogs = append(task.ExecutionLogs, devReleaseTaskLogRecord{Timestamp: finishedAt.Format(time.RFC3339), Step: stepName, Level: "info", Message: "step completed"})
		return nil
	})
	if err != nil {
		return err
	}
	return stepErr
}

func (h *PluginHandler) createDevRolloutTask(pluginID, action string, requestedPercent int, actorID string, rollbackToVersion string, rollbackToPercent *int, strategyType, targetEnv string, tags []string, canaryVersion string) (devRolloutTaskRecord, error) {
	if strings.TrimSpace(actorID) == "" {
		actorID = "system"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	task := devRolloutTaskRecord{
		TaskID:                  newDevRolloutTaskID(),
		PluginID:                pluginID,
		Action:                  action,
		RequestedRolloutPercent: requestedPercent,
		RollbackToVersion:       rollbackToVersion,
		RollbackToPercent:       rollbackToPercent,
		StrategyType:            strategyType,
		TargetEnv:               targetEnv,
		Tags:                    tags,
		CanaryVersion:           canaryVersion,
		TaskStatus:              devReleaseTaskStatusPending,
		CreatedBy:               actorID,
		CreatedAt:               now,
		ExecutionLogs:           []devReleaseTaskLogRecord{{Timestamp: now, Level: "info", Message: "task accepted"}},
	}
	tasks, err := h.loadDevRolloutTasks(pluginID)
	if err != nil {
		return devRolloutTaskRecord{}, err
	}
	tasks = append(tasks, task)
	if err := h.saveDevRolloutTasks(pluginID, tasks); err != nil {
		return devRolloutTaskRecord{}, err
	}
	return task, nil
}

func (h *PluginHandler) currentDevRolloutPercent(pluginID string) (int, error) {
	item, err := h.manager.Get(pluginID)
	if err != nil {
		return 0, err
	}
	current := 100
	if raw := strings.TrimSpace(item.ConfigJSON); raw != "" {
		cfg := map[string]any{}
		if jsonErr := json.Unmarshal([]byte(raw), &cfg); jsonErr == nil {
			if v, ok := cfg[rolloutStrategyConfigKey("percent", "")]; ok {
				switch n := v.(type) {
				case float64:
					current = int(n)
				case int:
					current = n
				}
			}
		}
	}
	return current, nil
}

func (h *PluginHandler) getDevRolloutTask(pluginID, taskID string) (devRolloutTaskRecord, bool, error) {
	tasks, err := h.loadDevRolloutTasks(pluginID)
	if err != nil {
		return devRolloutTaskRecord{}, false, err
	}
	for _, task := range tasks {
		if task.TaskID == taskID {
			return task, true, nil
		}
	}
	return devRolloutTaskRecord{}, false, nil
}

func (h *PluginHandler) findDevRolloutTask(taskID string) (devRolloutTaskRecord, bool, error) {
	items, err := h.listDevRolloutTasks("")
	if err != nil {
		return devRolloutTaskRecord{}, false, err
	}
	for _, item := range items {
		if item.TaskID == taskID {
			return item, true, nil
		}
	}
	return devRolloutTaskRecord{}, false, nil
}

func (h *PluginHandler) listDevRolloutTasks(pluginID string) ([]devRolloutTaskRecord, error) {
	if pluginID != "" {
		return h.loadDevRolloutTasks(pluginID)
	}

	root, err := h.defaultDevRoot()
	if err != nil {
		return nil, err
	}

	pluginDirs, err := discoverPluginManifestDirs(root)
	if err != nil {
		return nil, err
	}
	tasks := make([]devRolloutTaskRecord, 0, len(pluginDirs))
	for _, dir := range pluginDirs {
		id := strings.TrimSpace(filepath.Base(dir))
		if id == "" {
			continue
		}
		items, loadErr := h.loadDevRolloutTasks(id)
		if loadErr != nil {
			return nil, loadErr
		}
		tasks = append(tasks, items...)
	}
	return tasks, nil
}

func (h *PluginHandler) updateDevRolloutTask(pluginID, taskID string, updateFn func(task *devRolloutTaskRecord) error) error {
	tasks, err := h.loadDevRolloutTasks(pluginID)
	if err != nil {
		return err
	}
	idx := findDevRolloutTaskIndex(tasks, taskID)
	if idx < 0 {
		return errors.New("rollout task not found")
	}
	if err := updateFn(&tasks[idx]); err != nil {
		return err
	}
	return h.saveDevRolloutTasks(pluginID, tasks)
}

func (h *PluginHandler) loadDevRolloutTasks(pluginID string) ([]devRolloutTaskRecord, error) {
	path, err := h.devRolloutTasksPath(pluginID)
	if err != nil {
		return nil, err
	}
	h.devMu.Lock()
	defer h.devMu.Unlock()

	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []devRolloutTaskRecord{}, nil
		}
		return nil, err
	}
	env := devRolloutTaskEnvelope{}
	if len(bytesTrim(raw)) == 0 {
		return []devRolloutTaskRecord{}, nil
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	return env.Tasks, nil
}

func (h *PluginHandler) saveDevRolloutTasks(pluginID string, tasks []devRolloutTaskRecord) error {
	path, err := h.devRolloutTasksPath(pluginID)
	if err != nil {
		return err
	}
	h.devMu.Lock()
	defer h.devMu.Unlock()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(devRolloutTaskEnvelope{Tasks: tasks}, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

func (h *PluginHandler) defaultDevRoot() (string, error) {
	_, primary := h.devRoots()
	if strings.TrimSpace(primary) == "" {
		return "", errors.New("dev portal root is required")
	}
	return primary, nil
}

func (h *PluginHandler) devRolloutTasksPath(pluginID string) (string, error) {
	root, err := h.defaultDevRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, pluginID, ".devportal", fmt.Sprintf("%s.rollout.tasks.json", pluginID)), nil
}

func devRolloutHistoryPath(pluginsRoot, pluginID string) string {
	return filepath.Join(pluginsRoot, pluginID, ".devportal", fmt.Sprintf("%s.rollout.history.json", pluginID))
}

func (h *PluginHandler) loadDevRolloutHistory(pluginID string) (devRolloutHistoryEntry, error) {
	root, err := h.defaultDevRoot()
	if err != nil {
		return devRolloutHistoryEntry{}, err
	}
	path := devRolloutHistoryPath(root, pluginID)
	h.devMu.Lock()
	defer h.devMu.Unlock()

	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return devRolloutHistoryEntry{}, nil
		}
		return devRolloutHistoryEntry{}, err
	}
	if len(bytesTrim(raw)) == 0 {
		return devRolloutHistoryEntry{}, nil
	}
	var entry devRolloutHistoryEntry
	if err := json.Unmarshal(raw, &entry); err != nil {
		return devRolloutHistoryEntry{}, err
	}
	return entry, nil
}

func (h *PluginHandler) saveDevRolloutHistory(pluginID string, entry devRolloutHistoryEntry) error {
	root, err := h.defaultDevRoot()
	if err != nil {
		return err
	}
	path := devRolloutHistoryPath(root, pluginID)
	h.devMu.Lock()
	defer h.devMu.Unlock()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

func nextRolloutVersion(entry devRolloutHistoryEntry) string {
	return fmt.Sprintf("v%d", len(entry.Points)+1)
}

func findDevRolloutTaskIndex(tasks []devRolloutTaskRecord, taskID string) int {
	for idx := range tasks {
		if tasks[idx].TaskID == taskID {
			return idx
		}
	}
	return -1
}

func findDevRolloutTaskStepIndex(steps []devRolloutTaskStep, name string) int {
	for idx := len(steps) - 1; idx >= 0; idx-- {
		if steps[idx].Name == name {
			return idx
		}
	}
	return -1
}

func newDevRolloutTaskID() string {
	unixNano := time.Now().UTC().UnixNano()
	return "gt-" + strconv.FormatInt(unixNano, 36)
}

// rolloutStrategyConfigKey returns the config key for a strategy type and environment.
// percent → dev_rollout_percent, tag → dev_rollout_tags, canary → dev_canary_version.
// Non-production targetEnv appends a suffix, e.g. dev_rollout_percent_staging.
func rolloutStrategyConfigKey(strategyType, targetEnv string) string {
	var key string
	switch strategyType {
	case "percent":
		key = "dev_rollout_percent"
	case "tag":
		key = "dev_rollout_tags"
	case "canary":
		key = "dev_canary_version"
	default:
		key = "dev_rollout_percent"
	}
	if targetEnv != "" && targetEnv != "production" {
		key = key + "_" + targetEnv
	}
	return key
}

func (h *PluginHandler) persistDevRolloutStrategy(pluginID, strategyType, targetEnv string, percent int, tags []string, canaryVersion string) (bool, string, error) {
	item, err := h.manager.Get(pluginID)
	if err != nil {
		return false, "", err
	}

	updater, ok := h.manager.(PluginConfigUpdater)
	if !ok {
		return false, "", errors.New("runtime config updater is required")
	}

	cfg := map[string]any{}
	if raw := strings.TrimSpace(item.ConfigJSON); raw != "" {
		if unmarshalErr := json.Unmarshal([]byte(raw), &cfg); unmarshalErr != nil {
			return false, "", unmarshalErr
		}
	}

	switch strategyType {
	case "percent":
		cfg[rolloutStrategyConfigKey("percent", targetEnv)] = percent
	case "tag":
		cfg[rolloutStrategyConfigKey("tag", targetEnv)] = tags
	case "canary":
		cfg[rolloutStrategyConfigKey("canary", targetEnv)] = canaryVersion
	}
	cfg["dev_rollout_updated_at"] = time.Now().UTC().Format(time.RFC3339)

	if saveErr := updater.SavePluginConfig(pluginID, cfg); saveErr != nil {
		return false, "", saveErr
	}
	return true, "rollout persisted to plugin config", nil
}
