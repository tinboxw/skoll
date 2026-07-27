package workflow

import (
	"fmt"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

const (
	minEscalationDelay = time.Minute
	maxEscalationDelay = 90 * 24 * time.Hour
	maxAbsenceWindow   = 366 * 24 * time.Hour
)

type AssignmentKind string

const (
	AssignmentDirect      AssignmentKind = "direct"
	AssignmentDelegated   AssignmentKind = "delegated"
	AssignmentSubstituted AssignmentKind = "substituted"
	AssignmentEscalated   AssignmentKind = "escalated"
)

type EscalationRule struct {
	After  time.Duration
	Target Actor
}

func (r EscalationRule) Validate() error {
	if r.After < minEscalationDelay || r.After > maxEscalationDelay {
		return fmt.Errorf("workflow escalation delay must be between one minute and ninety days")
	}
	if r.Target.ID.IsZero() {
		return fmt.Errorf("workflow escalation target is required")
	}
	return nil
}

type SubstitutionWindow struct {
	ID         shared.ID
	Principal  Actor
	Substitute Actor
	StartsAt   time.Time
	EndsAt     time.Time
	CreatedBy  Actor
	Reason     string
	CreatedAt  time.Time
	RevokedAt  *time.Time
}

func NewSubstitutionWindow(id shared.ID, principal, substitute, createdBy Actor, startsAt, endsAt, now time.Time, reason string) (*SubstitutionWindow, error) {
	window := &SubstitutionWindow{
		ID:         shared.ID(strings.TrimSpace(id.String())),
		Principal:  normalizeActor(principal),
		Substitute: normalizeActor(substitute),
		StartsAt:   startsAt.UTC(),
		EndsAt:     endsAt.UTC(),
		CreatedBy:  normalizeActor(createdBy),
		Reason:     strings.TrimSpace(reason),
		CreatedAt:  normalizeNow(now),
	}
	if err := window.Validate(); err != nil {
		return nil, err
	}
	return window, nil
}

func (w SubstitutionWindow) Validate() error {
	if w.ID.IsZero() || w.Principal.ID.IsZero() || w.Substitute.ID.IsZero() || w.CreatedBy.ID.IsZero() {
		return fmt.Errorf("workflow substitution identity is incomplete")
	}
	if w.Principal.ID == w.Substitute.ID {
		return fmt.Errorf("workflow substitute must differ from principal")
	}
	if w.CreatedBy.ID != w.Principal.ID {
		return fmt.Errorf("workflow substitution can only be created by the principal")
	}
	if w.StartsAt.IsZero() || w.EndsAt.IsZero() || !w.EndsAt.After(w.StartsAt) || w.EndsAt.Sub(w.StartsAt) > maxAbsenceWindow {
		return fmt.Errorf("workflow substitution window is invalid")
	}
	if w.CreatedAt.IsZero() {
		return fmt.Errorf("workflow substitution creation time is required")
	}
	if len(w.Reason) > 1024 {
		return fmt.Errorf("workflow substitution reason is too long")
	}
	if w.RevokedAt != nil && w.RevokedAt.Before(w.CreatedAt) {
		return fmt.Errorf("workflow substitution revocation time is invalid")
	}
	return nil
}

func (w SubstitutionWindow) Active(at time.Time) bool {
	at = normalizeNow(at)
	return w.RevokedAt == nil && !at.Before(w.StartsAt) && at.Before(w.EndsAt)
}

func (w *SubstitutionWindow) Revoke(actor Actor, now time.Time) error {
	if w == nil {
		return fmt.Errorf("workflow substitution is required")
	}
	if actor.ID.IsZero() || actor.ID != w.Principal.ID {
		return fmt.Errorf("workflow substitution can only be revoked by the principal")
	}
	if w.RevokedAt != nil {
		return nil
	}
	value := normalizeNow(now)
	w.RevokedAt = &value
	return w.Validate()
}

func (i *Instance) ApplySubstitutions(windows []SubstitutionWindow, now time.Time) error {
	if err := i.ensureRunning(); err != nil {
		return err
	}
	active := make(map[shared.ID]SubstitutionWindow)
	for _, window := range windows {
		if err := window.Validate(); err != nil {
			return err
		}
		if !window.Active(now) {
			continue
		}
		if _, exists := active[window.Principal.ID]; exists {
			return fmt.Errorf("workflow principal has overlapping substitution windows")
		}
		active[window.Principal.ID] = window
	}
	now = normalizeNow(now)
	for index := range i.Tasks {
		task := &i.Tasks[index]
		if task.Status != TaskPending || task.Assignment != AssignmentDirect {
			continue
		}
		window, ok := active[task.Assignee.ID]
		if !ok {
			continue
		}
		task.Assignee = window.Substitute
		task.Assignment = AssignmentSubstituted
		task.AuthorizedBy = window.CreatedBy
		task.AuthorizationID = window.ID
		i.appendAction(Action{
			ID: actionID(i.ID, ActionSubstitute, task.ID.String(), window.ID.String()), Type: ActionSubstitute,
			InstanceID: i.ID, TaskID: task.ID, NodeID: task.NodeID, Actor: window.Principal,
			Target: window.Substitute, Comment: window.Reason, CreatedAt: now,
		})
	}
	i.Meta.Touch(now)
	return nil
}

func (i *Instance) Escalate(taskID shared.ID, target Actor, now time.Time) error {
	if i == nil {
		return fmt.Errorf("workflow instance is required")
	}
	for _, action := range i.Timeline {
		if action.Type == ActionEscalate && action.TaskID == taskID && action.Target.ID == target.ID {
			return nil
		}
	}
	if i.Status != InstanceRunning {
		return nil
	}
	if target.ID.IsZero() {
		return fmt.Errorf("workflow escalation target is required")
	}
	var task *Task
	for index := range i.Tasks {
		if i.Tasks[index].ID == taskID {
			task = &i.Tasks[index]
			break
		}
	}
	if task == nil || task.Status != TaskPending {
		return nil
	}
	now = normalizeNow(now)
	task.Status = TaskDelegated
	task.CompletedAt = &now
	system := Actor{ID: shared.ID("system:workflow-timer"), Name: "Workflow timer"}
	authorizationID := actionID(i.ID, ActionEscalate, taskID.String(), target.ID.String())
	i.Tasks = append(i.Tasks, Task{
		ID: shared.ID(fmt.Sprintf("%s-escalate-%s", task.ID, target.ID)), InstanceID: i.ID, NodeID: task.NodeID,
		Assignee: normalizeActor(target), OriginalAssignee: task.OriginalAssignee, Assignment: AssignmentEscalated,
		AuthorizedBy: system, AuthorizationID: authorizationID, Status: TaskPending, CreatedAt: now,
	})
	i.appendAction(Action{
		ID: authorizationID, Type: ActionEscalate, InstanceID: i.ID, TaskID: taskID, NodeID: task.NodeID,
		Actor: system, Target: normalizeActor(target), Comment: "approval timer elapsed", CreatedAt: now,
	})
	i.Meta.Touch(now)
	return nil
}
