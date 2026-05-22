package plugin

import (
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

	apiv1 "github.com/tinboxw/skoll/internal/handler/http/v1"
)

const (
	devReleaseTaskStatusPending   = "pending"
	devReleaseTaskStatusRunning   = "running"
	devReleaseTaskStatusSuccess   = "success"
	devReleaseTaskStatusFailed    = "failed"
	devReleaseTaskStatusCancelled = "cancelled"

	devReleaseTaskStepPrepare        = "prepare"
	devReleaseTaskStepVerifyArtifact = "verify_artifact"
	devReleaseTaskStepPublish        = "publish"
	devReleaseTaskStepPostCheck      = "post_check"
)

type devExecuteReleaseOrderRequest struct {
	PluginsRoot  string `json:"pluginsRoot"`
	PluginID     string `json:"pluginId"`
	TargetEnv    string `json:"targetEnv"`
	ArtifactPath string `json:"artifactPath"`
}

type devReleaseTaskStep struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
	StartedAt  string `json:"startedAt,omitempty"`
	FinishedAt string `json:"finishedAt,omitempty"`
	DurationMs int64  `json:"durationMs"`
}

type devReleaseTaskLogRecord struct {
	Timestamp string `json:"timestamp"`
	Step      string `json:"step,omitempty"`
	Level     string `json:"level"`
	Message   string `json:"message"`
}

type devReleaseTaskRecord struct {
	TaskID          string                    `json:"taskId"`
	OrderID         string                    `json:"orderId"`
	PluginID        string                    `json:"pluginId"`
	ReleaseVersion  string                    `json:"releaseVersion"`
	TargetEnv       string                    `json:"targetEnv"`
	ArtifactPath    string                    `json:"artifactPath"`
	TaskStatus      string                    `json:"taskStatus"`
	CreatedBy       string                    `json:"createdBy"`
	CreatedAt       string                    `json:"createdAt"`
	StartedAt       string                    `json:"startedAt,omitempty"`
	FinishedAt      string                    `json:"finishedAt,omitempty"`
	FailureStep     string                    `json:"failureStep,omitempty"`
	FailureReason   string                    `json:"failureReason,omitempty"`
	ExecutionSteps  []devReleaseTaskStep      `json:"steps,omitempty"`
	ExecutionLogs   []devReleaseTaskLogRecord `json:"logs,omitempty"`
	ExecutionResult string                    `json:"executionResult,omitempty"`
}

type devReleaseTaskEnvelope struct {
	Tasks []devReleaseTaskRecord `json:"tasks"`
}

type devExecuteReleaseOrderResponse struct {
	Operation string               `json:"operation"`
	Status    string               `json:"status"`
	Task      devReleaseTaskRecord `json:"task"`
}

type devListReleaseTasksResponse struct {
	Operation   string                 `json:"operation"`
	Status      string                 `json:"status"`
	PluginsRoot string                 `json:"pluginsRoot"`
	PluginID    string                 `json:"pluginId,omitempty"`
	OrderID     string                 `json:"orderId,omitempty"`
	TaskStatus  string                 `json:"taskStatus,omitempty"`
	Tasks       []devReleaseTaskRecord `json:"tasks"`
}

type devGetReleaseTaskResponse struct {
	Operation string               `json:"operation"`
	Status    string               `json:"status"`
	Task      devReleaseTaskRecord `json:"task"`
}

type devReleaseTaskLogsResponse struct {
	Operation string                    `json:"operation"`
	Status    string                    `json:"status"`
	TaskID    string                    `json:"taskId"`
	Logs      []devReleaseTaskLogRecord `json:"logs"`
}

func (h *PluginHandler) devExecuteReleaseOrder(w http.ResponseWriter, r *http.Request) {
	if !h.isSuperAdmin(r) {
		apiv1.WriteMessage(w, http.StatusForbidden, "forbidden", "super_admin role required")
		return
	}

	orderID := strings.TrimSpace(r.PathValue("orderId"))
	if orderID == "" {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("orderId is required"))
		return
	}

	var req devExecuteReleaseOrderRequest
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
	targetEnv := strings.TrimSpace(strings.ToLower(req.TargetEnv))
	if targetEnv == "" {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("targetEnv is required"))
		return
	}
	artifactPath := strings.TrimSpace(req.ArtifactPath)
	if artifactPath == "" {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("artifactPath is required"))
		return
	}

	orders, err := h.loadDevReleaseOrders(pluginsRoot, pluginID)
	if err != nil {
		apiv1.WriteError(w, http.StatusInternalServerError, err)
		return
	}
	orderIndex := findDevReleaseOrderIndex(orders, orderID)
	if orderIndex < 0 {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("release order not found"))
		return
	}
	if orders[orderIndex].OrderStatus != devReleaseOrderStatusApproved {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("only approved release order can be executed"))
		return
	}

	tasks, err := h.loadDevReleaseTasks(pluginsRoot, pluginID)
	if err != nil {
		apiv1.WriteError(w, http.StatusInternalServerError, err)
		return
	}
	for _, item := range tasks {
		if item.TaskStatus == devReleaseTaskStatusRunning && strings.EqualFold(item.TargetEnv, targetEnv) {
			apiv1.WriteMessage(w, http.StatusConflict, "task_conflict", "release task already running for plugin and targetEnv")
			return
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	createdBy := actorIDFromJWT(r)
	task := devReleaseTaskRecord{
		TaskID:         newDevReleaseTaskID(),
		OrderID:        orderID,
		PluginID:       pluginID,
		ReleaseVersion: orders[orderIndex].ReleaseVersion,
		TargetEnv:      targetEnv,
		ArtifactPath:   artifactPath,
		TaskStatus:     devReleaseTaskStatusPending,
		CreatedBy:      createdBy,
		CreatedAt:      now,
		ExecutionLogs: []devReleaseTaskLogRecord{
			{Timestamp: now, Level: "info", Message: "task accepted"},
		},
	}
	tasks = append(tasks, task)
	if err := h.saveDevReleaseTasks(pluginsRoot, pluginID, tasks); err != nil {
		apiv1.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	h.appendAudit(r, "dev_release_order_execute", "plugin_release_order", orderID, map[string]any{
		"pluginsRoot":  pluginsRoot,
		"pluginId":     pluginID,
		"taskId":       task.TaskID,
		"targetEnv":    targetEnv,
		"artifactPath": artifactPath,
	})

	go h.runDevReleaseTask(pluginsRoot, pluginID, task.TaskID)

	apiv1.WriteJSON(w, http.StatusAccepted, devExecuteReleaseOrderResponse{
		Operation: "release_order_execute",
		Status:    "accepted",
		Task:      task,
	})
}

func (h *PluginHandler) devListReleaseTasks(w http.ResponseWriter, r *http.Request) {
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
	orderID := strings.TrimSpace(r.URL.Query().Get("orderId"))
	taskStatus := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("status")))

	tasks := make([]devReleaseTaskRecord, 0, 16)
	if pluginID != "" {
		items, loadErr := h.loadDevReleaseTasks(pluginsRoot, pluginID)
		if loadErr != nil {
			apiv1.WriteError(w, http.StatusInternalServerError, loadErr)
			return
		}
		tasks = append(tasks, items...)
	} else {
		pluginDirs, dirErr := discoverPluginManifestDirs(pluginsRoot)
		if dirErr != nil {
			apiv1.WriteError(w, http.StatusInternalServerError, dirErr)
			return
		}
		for _, dir := range pluginDirs {
			id := strings.TrimSpace(filepath.Base(dir))
			if id == "" {
				continue
			}
			items, loadErr := h.loadDevReleaseTasks(pluginsRoot, id)
			if loadErr != nil {
				apiv1.WriteError(w, http.StatusInternalServerError, loadErr)
				return
			}
			tasks = append(tasks, items...)
		}
	}

	filtered := make([]devReleaseTaskRecord, 0, len(tasks))
	for _, item := range tasks {
		if orderID != "" && item.OrderID != orderID {
			continue
		}
		if taskStatus != "" && !strings.EqualFold(item.TaskStatus, taskStatus) {
			continue
		}
		filtered = append(filtered, item)
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt > filtered[j].CreatedAt
	})

	apiv1.WriteJSON(w, http.StatusOK, devListReleaseTasksResponse{
		Operation:   "release_task_list",
		Status:      "ok",
		PluginsRoot: pluginsRoot,
		PluginID:    pluginID,
		OrderID:     orderID,
		TaskStatus:  taskStatus,
		Tasks:       filtered,
	})
}

func (h *PluginHandler) devGetReleaseTask(w http.ResponseWriter, r *http.Request) {
	if !h.isSuperAdmin(r) {
		apiv1.WriteMessage(w, http.StatusForbidden, "forbidden", "super_admin role required")
		return
	}

	taskID := strings.TrimSpace(r.PathValue("taskId"))
	if taskID == "" {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("taskId is required"))
		return
	}

	pluginsRoot, err := h.resolveDevPluginsRoot(r.URL.Query().Get("pluginsRoot"))
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	task, found, err := h.findDevReleaseTask(pluginsRoot, taskID)
	if err != nil {
		apiv1.WriteError(w, http.StatusInternalServerError, err)
		return
	}
	if !found {
		apiv1.WriteError(w, http.StatusNotFound, errors.New("release task not found"))
		return
	}

	apiv1.WriteJSON(w, http.StatusOK, devGetReleaseTaskResponse{
		Operation: "release_task_get",
		Status:    "ok",
		Task:      task,
	})
}

func (h *PluginHandler) devGetReleaseTaskLogs(w http.ResponseWriter, r *http.Request) {
	if !h.isSuperAdmin(r) {
		apiv1.WriteMessage(w, http.StatusForbidden, "forbidden", "super_admin role required")
		return
	}

	taskID := strings.TrimSpace(r.PathValue("taskId"))
	if taskID == "" {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("taskId is required"))
		return
	}

	pluginsRoot, err := h.resolveDevPluginsRoot(r.URL.Query().Get("pluginsRoot"))
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	task, found, err := h.findDevReleaseTask(pluginsRoot, taskID)
	if err != nil {
		apiv1.WriteError(w, http.StatusInternalServerError, err)
		return
	}
	if !found {
		apiv1.WriteError(w, http.StatusNotFound, errors.New("release task not found"))
		return
	}

	apiv1.WriteJSON(w, http.StatusOK, devReleaseTaskLogsResponse{
		Operation: "release_task_logs",
		Status:    "ok",
		TaskID:    taskID,
		Logs:      task.ExecutionLogs,
	})
}

func (h *PluginHandler) runDevReleaseTask(pluginsRoot, pluginID, taskID string) {
	if err := h.updateDevReleaseTask(pluginsRoot, pluginID, taskID, func(task *devReleaseTaskRecord) error {
		now := time.Now().UTC().Format(time.RFC3339)
		task.TaskStatus = devReleaseTaskStatusRunning
		task.StartedAt = now
		task.ExecutionLogs = append(task.ExecutionLogs, devReleaseTaskLogRecord{Timestamp: now, Level: "info", Message: "task started"})
		return nil
	}); err != nil {
		return
	}

	recordedFailure := false
	if err := h.runDevReleaseTaskStep(pluginsRoot, pluginID, taskID, devReleaseTaskStepPrepare, func(task *devReleaseTaskRecord) error {
		_, _, stepErr := h.resolveManifestPath(pluginsRoot, task.PluginID)
		if stepErr != nil {
			return stepErr
		}
		return nil
	}); err != nil {
		recordedFailure = true
	}

	if !recordedFailure {
		if err := h.runDevReleaseTaskStep(pluginsRoot, pluginID, taskID, devReleaseTaskStepVerifyArtifact, func(task *devReleaseTaskRecord) error {
			stat, stepErr := os.Stat(task.ArtifactPath)
			if stepErr != nil {
				if errors.Is(stepErr, os.ErrNotExist) {
					return errors.New("artifact not found")
				}
				return stepErr
			}
			if stat.IsDir() {
				return errors.New("artifactPath must be a file")
			}
			if !strings.HasSuffix(strings.ToLower(task.ArtifactPath), ".zip") {
				return errors.New("artifactPath must be a .zip file")
			}
			return nil
		}); err != nil {
			recordedFailure = true
		}
	}

	if !recordedFailure {
		if err := h.runDevReleaseTaskStep(pluginsRoot, pluginID, taskID, devReleaseTaskStepPublish, func(task *devReleaseTaskRecord) error {
			marker := filepath.Join(pluginsRoot, task.PluginID, ".devportal", fmt.Sprintf("%s.last.release.json", task.PluginID))
			if err := os.MkdirAll(filepath.Dir(marker), 0o755); err != nil {
				return err
			}
			payload := map[string]any{
				"taskId":         task.TaskID,
				"orderId":        task.OrderID,
				"pluginId":       task.PluginID,
				"releaseVersion": task.ReleaseVersion,
				"targetEnv":      task.TargetEnv,
				"artifactPath":   task.ArtifactPath,
				"publishedAt":    time.Now().UTC().Format(time.RFC3339),
			}
			raw, err := json.MarshalIndent(payload, "", "  ")
			if err != nil {
				return err
			}
			raw = append(raw, '\n')
			return os.WriteFile(marker, raw, 0o644)
		}); err != nil {
			recordedFailure = true
		}
	}

	if !recordedFailure {
		_ = h.runDevReleaseTaskStep(pluginsRoot, pluginID, taskID, devReleaseTaskStepPostCheck, func(task *devReleaseTaskRecord) error {
			marker := filepath.Join(pluginsRoot, task.PluginID, ".devportal", fmt.Sprintf("%s.last.release.json", task.PluginID))
			if _, err := os.Stat(marker); err != nil {
				return err
			}
			return nil
		})
	}

	if recordedFailure {
		return
	}

	_ = h.updateDevReleaseTask(pluginsRoot, pluginID, taskID, func(task *devReleaseTaskRecord) error {
		now := time.Now().UTC().Format(time.RFC3339)
		task.TaskStatus = devReleaseTaskStatusSuccess
		task.FinishedAt = now
		task.ExecutionResult = "released"
		task.ExecutionLogs = append(task.ExecutionLogs, devReleaseTaskLogRecord{Timestamp: now, Level: "info", Message: "task completed"})
		return nil
	})
}

func (h *PluginHandler) runDevReleaseTaskStep(pluginsRoot, pluginID, taskID, stepName string, stepFn func(task *devReleaseTaskRecord) error) error {
	startedAt := time.Now().UTC()
	step := devReleaseTaskStep{
		Name:      stepName,
		Status:    devReleaseTaskStatusRunning,
		StartedAt: startedAt.Format(time.RFC3339),
	}

	if err := h.updateDevReleaseTask(pluginsRoot, pluginID, taskID, func(task *devReleaseTaskRecord) error {
		task.ExecutionSteps = append(task.ExecutionSteps, step)
		task.ExecutionLogs = append(task.ExecutionLogs, devReleaseTaskLogRecord{
			Timestamp: startedAt.Format(time.RFC3339),
			Step:      stepName,
			Level:     "info",
			Message:   "step started",
		})
		return nil
	}); err != nil {
		return err
	}

	stepErr := h.updateDevReleaseTask(pluginsRoot, pluginID, taskID, func(task *devReleaseTaskRecord) error {
		idx := findDevReleaseTaskStepIndex(task.ExecutionSteps, stepName)
		if idx < 0 {
			return errors.New("release task step not found")
		}

		execErr := stepFn(task)
		finishedAt := time.Now().UTC()
		task.ExecutionSteps[idx].FinishedAt = finishedAt.Format(time.RFC3339)
		task.ExecutionSteps[idx].DurationMs = finishedAt.Sub(startedAt).Milliseconds()
		if execErr != nil {
			task.ExecutionSteps[idx].Status = devReleaseTaskStatusFailed
			task.ExecutionSteps[idx].Message = execErr.Error()
			task.TaskStatus = devReleaseTaskStatusFailed
			task.FinishedAt = finishedAt.Format(time.RFC3339)
			task.FailureStep = stepName
			task.FailureReason = execErr.Error()
			task.ExecutionLogs = append(task.ExecutionLogs, devReleaseTaskLogRecord{
				Timestamp: finishedAt.Format(time.RFC3339),
				Step:      stepName,
				Level:     "error",
				Message:   execErr.Error(),
			})
			return nil
		}

		task.ExecutionSteps[idx].Status = devReleaseTaskStatusSuccess
		task.ExecutionLogs = append(task.ExecutionLogs, devReleaseTaskLogRecord{
			Timestamp: finishedAt.Format(time.RFC3339),
			Step:      stepName,
			Level:     "info",
			Message:   "step completed",
		})
		return nil
	})
	if stepErr != nil {
		return stepErr
	}

	task, found, err := h.getDevReleaseTask(pluginsRoot, pluginID, taskID)
	if err != nil {
		return err
	}
	if !found {
		return errors.New("release task not found")
	}
	if task.TaskStatus == devReleaseTaskStatusFailed {
		return errors.New(task.FailureReason)
	}
	return nil
}

func (h *PluginHandler) getDevReleaseTask(pluginsRoot, pluginID, taskID string) (devReleaseTaskRecord, bool, error) {
	tasks, err := h.loadDevReleaseTasks(pluginsRoot, pluginID)
	if err != nil {
		return devReleaseTaskRecord{}, false, err
	}
	for _, task := range tasks {
		if task.TaskID == taskID {
			return task, true, nil
		}
	}
	return devReleaseTaskRecord{}, false, nil
}

func (h *PluginHandler) findDevReleaseTask(pluginsRoot, taskID string) (devReleaseTaskRecord, bool, error) {
	pluginDirs, err := discoverPluginManifestDirs(pluginsRoot)
	if err != nil {
		return devReleaseTaskRecord{}, false, err
	}
	for _, dir := range pluginDirs {
		id := strings.TrimSpace(filepath.Base(dir))
		if id == "" {
			continue
		}
		task, found, readErr := h.getDevReleaseTask(pluginsRoot, id, taskID)
		if readErr != nil {
			return devReleaseTaskRecord{}, false, readErr
		}
		if found {
			return task, true, nil
		}
	}
	return devReleaseTaskRecord{}, false, nil
}

func (h *PluginHandler) updateDevReleaseTask(pluginsRoot, pluginID, taskID string, updateFn func(task *devReleaseTaskRecord) error) error {
	tasks, err := h.loadDevReleaseTasks(pluginsRoot, pluginID)
	if err != nil {
		return err
	}

	idx := findDevReleaseTaskIndex(tasks, taskID)
	if idx < 0 {
		return errors.New("release task not found")
	}
	if err := updateFn(&tasks[idx]); err != nil {
		return err
	}
	return h.saveDevReleaseTasks(pluginsRoot, pluginID, tasks)
}

func (h *PluginHandler) loadDevReleaseTasks(pluginsRoot, pluginID string) ([]devReleaseTaskRecord, error) {
	path := devReleaseTasksPath(pluginsRoot, pluginID)
	h.devMu.Lock()
	defer h.devMu.Unlock()

	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []devReleaseTaskRecord{}, nil
		}
		return nil, err
	}

	envelope := devReleaseTaskEnvelope{}
	if len(bytesTrim(raw)) == 0 {
		return []devReleaseTaskRecord{}, nil
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	return envelope.Tasks, nil
}

func (h *PluginHandler) saveDevReleaseTasks(pluginsRoot, pluginID string, tasks []devReleaseTaskRecord) error {
	path := devReleaseTasksPath(pluginsRoot, pluginID)
	h.devMu.Lock()
	defer h.devMu.Unlock()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(devReleaseTaskEnvelope{Tasks: tasks}, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

func devReleaseTasksPath(pluginsRoot, pluginID string) string {
	return filepath.Join(pluginsRoot, pluginID, ".devportal", fmt.Sprintf("%s.release.tasks.json", pluginID))
}

func findDevReleaseTaskIndex(tasks []devReleaseTaskRecord, taskID string) int {
	for idx := range tasks {
		if tasks[idx].TaskID == taskID {
			return idx
		}
	}
	return -1
}

func findDevReleaseTaskStepIndex(steps []devReleaseTaskStep, name string) int {
	for idx := len(steps) - 1; idx >= 0; idx-- {
		if steps[idx].Name == name {
			return idx
		}
	}
	return -1
}

func newDevReleaseTaskID() string {
	unixNano := time.Now().UTC().UnixNano()
	return "rt-" + strconv.FormatInt(unixNano, 36)
}
