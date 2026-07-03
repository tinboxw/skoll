package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

var ErrPluginRollbackInvalid = errors.New("plugin rollback plan is invalid")

type PluginRollbackPlanInput struct {
	PluginID       string
	From           Info
	Target         Info
	TargetPercent  *int
	TargetVersion  string
	TargetAssetDir string
}

type PluginRollbackPlan struct {
	PluginID    string                     `json:"pluginId"`
	Status      string                     `json:"status"`
	FromVersion string                     `json:"fromVersion"`
	ToVersion   string                     `json:"toVersion"`
	Checkpoints []PluginRollbackCheckpoint `json:"checkpoints"`
}

type PluginRollbackCheckpoint struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Before  string `json:"before,omitempty"`
	After   string `json:"after,omitempty"`
	Message string `json:"message,omitempty"`
}

type PluginRollbackService struct{}

func NewPluginRollbackService() *PluginRollbackService {
	return &PluginRollbackService{}
}

func (s *PluginRollbackService) BuildPlan(in PluginRollbackPlanInput) (PluginRollbackPlan, error) {
	pluginID := strings.TrimSpace(in.PluginID)
	if pluginID == "" {
		pluginID = strings.TrimSpace(in.Target.ID)
	}
	if pluginID == "" {
		return PluginRollbackPlan{}, fmt.Errorf("%w: plugin id is required", ErrPluginRollbackInvalid)
	}
	if strings.TrimSpace(in.Target.ID) != "" && strings.TrimSpace(in.Target.ID) != pluginID {
		return PluginRollbackPlan{}, fmt.Errorf("%w: target plugin id mismatch", ErrPluginRollbackInvalid)
	}
	if strings.TrimSpace(in.From.ID) != "" && strings.TrimSpace(in.From.ID) != pluginID {
		return PluginRollbackPlan{}, fmt.Errorf("%w: source plugin id mismatch", ErrPluginRollbackInvalid)
	}

	toVersion := strings.TrimSpace(in.TargetVersion)
	if toVersion == "" {
		toVersion = strings.TrimSpace(in.Target.Version)
	}
	if toVersion == "" {
		return PluginRollbackPlan{}, fmt.Errorf("%w: target version is required", ErrPluginRollbackInvalid)
	}

	checkpoints := []PluginRollbackCheckpoint{
		{
			Name:    "version",
			Status:  "passed",
			Before:  strings.TrimSpace(in.From.Version),
			After:   toVersion,
			Message: "target version selected",
		},
		buildRollbackMenuCheckpoint(in.From, in.Target),
		buildRollbackPermissionCheckpoint(in.From, in.Target),
		buildRollbackConfigCheckpoint(in.From, in.Target, in.TargetPercent),
		buildRollbackAssetCheckpoint(in.Target, in.TargetAssetDir),
		buildRollbackMigrationCheckpoint(in.From, in.Target),
	}

	status := "passed"
	for _, checkpoint := range checkpoints {
		if checkpoint.Status != "passed" {
			status = "blocked"
			break
		}
	}
	return PluginRollbackPlan{
		PluginID:    pluginID,
		Status:      status,
		FromVersion: strings.TrimSpace(in.From.Version),
		ToVersion:   toVersion,
		Checkpoints: checkpoints,
	}, nil
}

func buildRollbackMenuCheckpoint(from Info, target Info) PluginRollbackCheckpoint {
	before := rollbackMenuFingerprint(from)
	after := rollbackMenuFingerprint(target)
	return PluginRollbackCheckpoint{
		Name:    "menu",
		Status:  "passed",
		Before:  before,
		After:   after,
		Message: "menu key, path, visibility, roles, and permissions are taken from rollback target",
	}
}

func buildRollbackPermissionCheckpoint(from Info, target Info) PluginRollbackCheckpoint {
	before, beforeErr := rollbackPermissionFingerprint(from)
	after, afterErr := rollbackPermissionFingerprint(target)
	status := "passed"
	msg := "permission catalog entries are taken from rollback target"
	if beforeErr != nil || afterErr != nil {
		status = "blocked"
		msg = strings.TrimSpace(strings.Join([]string{errorString(beforeErr), errorString(afterErr)}, "; "))
	}
	return PluginRollbackCheckpoint{Name: "permission", Status: status, Before: before, After: after, Message: msg}
}

func buildRollbackConfigCheckpoint(from Info, target Info, targetPercent *int) PluginRollbackCheckpoint {
	before := rollbackConfigFingerprint(from.ConfigJSON)
	after := rollbackConfigFingerprint(target.ConfigJSON)
	status := "passed"
	msg := "configuration snapshot is available for rollback target"
	if targetPercent != nil && !rollbackConfigContainsPercent(target.ConfigJSON, *targetPercent) {
		status = "blocked"
		msg = fmt.Sprintf("target config does not contain rollout percent %d", *targetPercent)
	}
	return PluginRollbackCheckpoint{Name: "config", Status: status, Before: before, After: after, Message: msg}
}

func buildRollbackAssetCheckpoint(target Info, assetDir string) PluginRollbackCheckpoint {
	assets, err := rollbackAssetList(target, assetDir)
	if err != nil {
		return PluginRollbackCheckpoint{Name: "asset", Status: "blocked", Message: err.Error()}
	}
	return PluginRollbackCheckpoint{
		Name:    "asset",
		Status:  "passed",
		After:   strings.Join(assets, ","),
		Message: fmt.Sprintf("%d frontend assets available for rollback target", len(assets)),
	}
}

func buildRollbackMigrationCheckpoint(from Info, target Info) PluginRollbackCheckpoint {
	return PluginRollbackCheckpoint{
		Name:    "migration",
		Status:  "passed",
		Before:  strings.TrimSpace(from.MigrationVersion),
		After:   strings.TrimSpace(target.MigrationVersion),
		Message: "migration version is recorded for rollback hook consistency",
	}
}

func rollbackMenuFingerprint(info Info) string {
	if info.UIMenu == nil {
		return "none"
	}
	visible := "default"
	if info.UIMenu.Visible != nil {
		visible = fmt.Sprintf("%t", *info.UIMenu.Visible)
	}
	return strings.Join([]string{
		strings.TrimSpace(info.UIMenu.Key),
		strings.TrimSpace(info.UIMenu.Path),
		visible,
		strings.Join(normalizeRolloutStrings(info.UIMenu.RequiredRoles), ","),
		strings.Join(normalizeRolloutStrings(info.UIMenu.RequiredPermissions), ","),
	}, "|")
}

func rollbackPermissionFingerprint(info Info) (string, error) {
	items, err := info.CatalogPermissions()
	if err != nil {
		return "", err
	}
	keys := make([]string, 0, len(items))
	for _, item := range items {
		keys = append(keys, item.Key())
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return "none", nil
	}
	return strings.Join(keys, ","), nil
}

func rollbackConfigFingerprint(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "empty"
	}
	cfg := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return "invalid"
	}
	keys := make([]string, 0, len(cfg))
	for key := range cfg {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return strings.Join(keys, ",")
}

func rollbackConfigContainsPercent(raw string, percent int) bool {
	cfg := map[string]any{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &cfg); err != nil {
		return false
	}
	for key, value := range cfg {
		if !strings.Contains(strings.ToLower(key), "rollout") && !strings.Contains(strings.ToLower(key), "percent") {
			continue
		}
		switch v := value.(type) {
		case float64:
			if int(v) == percent {
				return true
			}
		case int:
			if v == percent {
				return true
			}
		}
	}
	return false
}

func rollbackAssetList(target Info, assetDir string) ([]string, error) {
	mode := target.UIMode
	if mode == "" {
		mode = UIModeBackendOnly
	}
	if mode == UIModeBackendOnly {
		return []string{}, nil
	}
	root := strings.TrimSpace(assetDir)
	if root == "" {
		root = strings.TrimSpace(target.Source)
	}
	if root == "" {
		return nil, fmt.Errorf("%w: target asset source is required", ErrPluginRollbackInvalid)
	}
	assets, err := discoverFrontendAssets(root, mode)
	if err != nil {
		return nil, err
	}
	sort.Strings(assets)
	return assets, nil
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
