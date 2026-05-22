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

type devRolloutRequest struct {
	PluginID       string `json:"pluginId"`
	RolloutPercent int    `json:"rolloutPercent"`
}

type devRollbackRequest struct {
	PluginID string `json:"pluginId"`
}

type devRolloutResponse struct {
	Operation      string               `json:"operation"`
	Status         string               `json:"status"`
	PluginID       string               `json:"pluginId"`
	RolloutPercent int                  `json:"rolloutPercent"`
	Persisted      bool                 `json:"persisted"`
	Message        string               `json:"message,omitempty"`
	Task           devRolloutTaskRecord `json:"task"`
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
	if req.RolloutPercent < 0 || req.RolloutPercent > 100 {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("rolloutPercent must be in range [0,100]"))
		return
	}

	task, err := h.runDevRolloutTask(pluginID, devRolloutTaskActionRollout, req.RolloutPercent, actorIDFromJWT(r))
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	persisted := task.TaskStatus == devReleaseTaskStatusSuccess
	msg := task.ExecutionResult
	if strings.TrimSpace(msg) == "" {
		msg = "rollout updated"
	}

	h.appendAudit(r, "dev_rollout", "plugin", pluginID, map[string]any{"rolloutPercent": req.RolloutPercent, "persisted": persisted, "taskId": task.TaskID})
	apiv1.WriteJSON(w, http.StatusOK, devRolloutResponse{
		Operation:      "rollout",
		Status:         "ok",
		PluginID:       pluginID,
		RolloutPercent: req.RolloutPercent,
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

	task, err := h.runDevRolloutTask(pluginID, devRolloutTaskActionRollback, 0, actorIDFromJWT(r))
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

	h.appendAudit(r, "dev_rollback", "plugin", pluginID, map[string]any{"rolloutPercent": prev, "persisted": persisted, "taskId": task.TaskID})
	apiv1.WriteJSON(w, http.StatusOK, devRolloutResponse{
		Operation:      "rollback",
		Status:         "ok",
		PluginID:       pluginID,
		RolloutPercent: prev,
		Persisted:      persisted,
		Message:        msg,
		Task:           task,
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

func (h *PluginHandler) applyDevRollout(pluginID string, percent int) (persisted bool, message string, err error) {
	item, getErr := h.manager.Get(pluginID)
	if getErr != nil {
		return false, "", getErr
	}

	current := 100
	if raw := strings.TrimSpace(item.ConfigJSON); raw != "" {
		cfg := map[string]any{}
		if jsonErr := json.Unmarshal([]byte(raw), &cfg); jsonErr == nil {
			if v, ok := cfg["dev_rollout_percent"]; ok {
				switch n := v.(type) {
				case float64:
					current = int(n)
				case int:
					current = n
				}
			}
		}
	}

	h.devMu.Lock()
	if h.devRolloutHistory == nil {
		h.devRolloutHistory = make(map[string][]int)
	}
	h.devRolloutHistory[pluginID] = append(h.devRolloutHistory[pluginID], current)
	h.devMu.Unlock()

	persisted, message, err = h.persistDevRollout(pluginID, percent)
	if err != nil {
		return false, "", err
	}
	if message == "" {
		message = "rollout updated"
	}
	return persisted, message, nil
}

func (h *PluginHandler) applyDevRollback(pluginID string) (persisted bool, message string, rollbackPercent int, err error) {
	h.devMu.Lock()
	history := h.devRolloutHistory[pluginID]
	if len(history) == 0 {
		h.devMu.Unlock()
		return false, "", 0, errors.New("no rollback history for plugin")
	}
	prev := history[len(history)-1]
	h.devRolloutHistory[pluginID] = history[:len(history)-1]
	h.devMu.Unlock()

	persisted, message, err = h.persistDevRollout(pluginID, prev)
	if err != nil {
		return false, "", 0, err
	}
	if message == "" {
		message = "rollback applied"
	}
	return persisted, message, prev, nil
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
	cfg["dev_rollout_percent"] = percent
	cfg["dev_rollout_updated_at"] = time.Now().UTC().Format(time.RFC3339)

	if saveErr := updater.SavePluginConfig(pluginID, cfg); saveErr != nil {
		return false, "", saveErr
	}
	return true, "rollout persisted to plugin config", nil
}

func (h *PluginHandler) runDevRolloutTask(pluginID, action string, requestedPercent int, actorID string) (devRolloutTaskRecord, error) {
	task, err := h.createDevRolloutTask(pluginID, action, requestedPercent, actorID)
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
			persisted, message, applyErr := h.applyDevRollout(pluginID, target.RequestedRolloutPercent)
			if applyErr != nil {
				return applyErr
			}
			target.RolloutPercent = target.RequestedRolloutPercent
			target.ExecutionResult = message
			if !persisted {
				return errors.New("rollout not persisted")
			}
			return nil
		}
		persisted, message, rollbackPercent, applyErr := h.applyDevRollback(pluginID)
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

func (h *PluginHandler) createDevRolloutTask(pluginID, action string, requestedPercent int, actorID string) (devRolloutTaskRecord, error) {
	if strings.TrimSpace(actorID) == "" {
		actorID = "system"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	task := devRolloutTaskRecord{
		TaskID:                  newDevRolloutTaskID(),
		PluginID:                pluginID,
		Action:                  action,
		RequestedRolloutPercent: requestedPercent,
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
			if v, ok := cfg["dev_rollout_percent"]; ok {
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
