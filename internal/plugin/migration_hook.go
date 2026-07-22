package plugin

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type PluginMigrationAction string

const (
	PluginMigrationInstall   PluginMigrationAction = "install"
	PluginMigrationUpgrade   PluginMigrationAction = "upgrade"
	PluginMigrationDowngrade PluginMigrationAction = "downgrade"
	PluginMigrationUninstall PluginMigrationAction = "uninstall"
)

type PluginMigrationStatus string

const (
	PluginMigrationStarted   PluginMigrationStatus = "started"
	PluginMigrationSucceeded PluginMigrationStatus = "succeeded"
	PluginMigrationFailed    PluginMigrationStatus = "failed"
)

type PluginMigrationEvent struct {
	PluginID    string
	Action      PluginMigrationAction
	Status      PluginMigrationStatus
	FromVersion string
	ToVersion   string
	Steps       []MigrationStep
	Error       string
	OccurredAt  time.Time
}

type PluginMigrationHookInput struct {
	PluginID           string
	PluginDir          string
	MigrationDirectory string
	Action             PluginMigrationAction
	FromVersion        string
	ToVersion          string
	UninstallPolicy    DataUninstallPolicy
	RollbackPolicy     DataRollbackPolicy
	Limit              int
	TransformSQL       func(string) (string, error)
}

type PluginMigrationRecorder interface {
	RecordPluginMigration(event PluginMigrationEvent) error
}

type PluginMigrationHook struct {
	store    MigrationStore
	recorder PluginMigrationRecorder
	now      func() time.Time
}

type MemoryPluginMigrationRecorder struct {
	mu     sync.RWMutex
	events []PluginMigrationEvent
}

func NewPluginMigrationHook(store MigrationStore, recorder PluginMigrationRecorder) *PluginMigrationHook {
	if recorder == nil {
		recorder = NewMemoryPluginMigrationRecorder()
	}
	return &PluginMigrationHook{
		store:    store,
		recorder: recorder,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func NewMemoryPluginMigrationRecorder() *MemoryPluginMigrationRecorder {
	return &MemoryPluginMigrationRecorder{events: []PluginMigrationEvent{}}
}

func (h *PluginMigrationHook) Run(ctx context.Context, in PluginMigrationHookInput) ([]MigrationStep, error) {
	if h == nil {
		h = NewPluginMigrationHook(nil, nil)
	}
	in.PluginID = strings.TrimSpace(in.PluginID)
	in.PluginDir = strings.TrimSpace(in.PluginDir)
	if in.PluginID == "" || in.PluginDir == "" {
		return nil, fmt.Errorf("%w: plugin id and dir are required", ErrPluginManifestBroken)
	}
	if !isPluginMigrationAction(in.Action) {
		return nil, fmt.Errorf("%w: migration action is invalid", ErrPluginManifestBroken)
	}

	h.record(in, PluginMigrationStarted, nil, "")
	steps, err := h.run(ctx, in)
	if err != nil {
		h.record(in, PluginMigrationFailed, steps, err.Error())
		return steps, err
	}
	h.record(in, PluginMigrationSucceeded, steps, "")
	return steps, nil
}

func (r *MemoryPluginMigrationRecorder) RecordPluginMigration(event PluginMigrationEvent) error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	event.Steps = append([]MigrationStep(nil), event.Steps...)
	r.events = append(r.events, event)
	return nil
}

func (r *MemoryPluginMigrationRecorder) Events() []PluginMigrationEvent {
	if r == nil {
		return []PluginMigrationEvent{}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]PluginMigrationEvent, 0, len(r.events))
	for _, event := range r.events {
		event.Steps = append([]MigrationStep(nil), event.Steps...)
		out = append(out, event)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].OccurredAt.Before(out[j].OccurredAt)
	})
	return out
}

func (h *PluginMigrationHook) record(in PluginMigrationHookInput, status PluginMigrationStatus, steps []MigrationStep, errText string) {
	if h == nil || h.recorder == nil {
		return
	}
	now := time.Now().UTC()
	if h.now != nil {
		now = h.now().UTC()
	}
	_ = h.recorder.RecordPluginMigration(PluginMigrationEvent{
		PluginID:    in.PluginID,
		Action:      in.Action,
		Status:      status,
		FromVersion: strings.TrimSpace(in.FromVersion),
		ToVersion:   strings.TrimSpace(in.ToVersion),
		Steps:       append([]MigrationStep(nil), steps...),
		Error:       errText,
		OccurredAt:  now,
	})
}

func (h *PluginMigrationHook) run(ctx context.Context, in PluginMigrationHookInput) ([]MigrationStep, error) {
	migrator := NewMigratorWithTransformer(in.PluginID, in.PluginDir, in.MigrationDirectory, h.store, in.TransformSQL)
	switch in.Action {
	case PluginMigrationInstall, PluginMigrationUpgrade:
		return migrator.Apply(ctx, in.Limit)
	case PluginMigrationDowngrade:
		if in.RollbackPolicy != DataRollbackAutomatic {
			return nil, fmt.Errorf("plugin downgrade requires automatic rollback policy")
		}
		return migrator.Rollback(ctx, in.Limit)
	case PluginMigrationUninstall:
		switch in.UninstallPolicy {
		case DataUninstallRetain, DataUninstallArchive:
			return []MigrationStep{}, nil
		case DataUninstallDrop:
			return migrator.Rollback(ctx, in.Limit)
		default:
			return nil, fmt.Errorf("plugin uninstall requires an explicit data policy")
		}
	default:
		return nil, fmt.Errorf("%w: migration action is invalid", ErrPluginManifestBroken)
	}
}

func isPluginMigrationAction(action PluginMigrationAction) bool {
	switch action {
	case PluginMigrationInstall, PluginMigrationUpgrade, PluginMigrationDowngrade, PluginMigrationUninstall:
		return true
	default:
		return false
	}
}
