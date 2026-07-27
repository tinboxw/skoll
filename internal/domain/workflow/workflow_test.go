package workflow

import (
	"strings"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

func TestWorkflowDefinitionValidationAndPublish(t *testing.T) {
	now := fixedWorkflowTime()
	def, err := sampleDefinition(now)
	if err != nil {
		t.Fatalf("sampleDefinition error: %v", err)
	}
	if def.Status != DefinitionPublished {
		t.Fatalf("expected published definition, got %s", def.Status)
	}
	if def.Meta.CreatedAt.IsZero() || def.Meta.UpdatedAt.IsZero() {
		t.Fatalf("expected definition meta timestamps")
	}

	_, err = NewDefinition("bad", "leave", "Leave", 1, []Node{
		{ID: "start", Key: "start", Name: "Start", Type: NodeStart},
		{ID: "approval", Key: "approval", Name: "Approval", Type: NodeApproval, Assignees: []shared.ID{"manager-1"}, Decision: DecisionRule{Strategy: DecisionAny, Quorum: 1}},
	}, nil, now)
	if err == nil || !strings.Contains(err.Error(), "one start and one end") {
		t.Fatalf("expected missing end validation error, got %v", err)
	}
}

func TestWorkflowStartApproveRejectAndWithdraw(t *testing.T) {
	now := fixedWorkflowTime()
	def, err := sampleDefinition(now)
	if err != nil {
		t.Fatalf("sampleDefinition error: %v", err)
	}
	instance, err := Start(StartInput{
		ID:           "wf-1",
		Definition:   *def,
		BusinessType: "leave",
		BusinessID:   "leave-1",
		Title:        "Leave Request",
		Starter:      Actor{ID: "user-1", Name: "Starter"},
		Now:          now,
	})
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
	if instance.Status != InstanceRunning || instance.CurrentNode != "approval" || len(instance.Tasks) != 1 {
		t.Fatalf("unexpected started instance: %+v", instance)
	}
	taskID := instance.Tasks[0].ID
	if err := instance.Approve(*def, taskID, Actor{ID: "manager-1"}, "ok", now.Add(time.Minute)); err != nil {
		t.Fatalf("Approve error: %v", err)
	}
	if instance.Status != InstanceApproved || instance.Tasks[0].Status != TaskApproved || len(instance.Timeline) != 2 {
		t.Fatalf("unexpected approved instance: %+v", instance)
	}
	if err := instance.Reject(taskID, Actor{ID: "manager-1"}, "late", now.Add(2*time.Minute)); err == nil {
		t.Fatal("expected terminal instance to reject repeated action")
	}

	rejected := mustStartInstance(t, *def, "wf-2", now)
	if err := rejected.Reject(rejected.Tasks[0].ID, Actor{ID: "manager-1"}, "no", now.Add(time.Minute)); err != nil {
		t.Fatalf("Reject error: %v", err)
	}
	if rejected.Status != InstanceRejected || rejected.Tasks[0].Status != TaskRejected {
		t.Fatalf("unexpected rejected instance: %+v", rejected)
	}

	withdrawn := mustStartInstance(t, *def, "wf-3", now)
	if err := withdrawn.Withdraw(Actor{ID: "user-1"}, "cancel", now.Add(time.Minute)); err != nil {
		t.Fatalf("Withdraw error: %v", err)
	}
	if withdrawn.Status != InstanceWithdrawn || withdrawn.Tasks[0].Status != TaskCanceled {
		t.Fatalf("unexpected withdrawn instance: %+v", withdrawn)
	}
}

func TestWorkflowTransferAndCopy(t *testing.T) {
	now := fixedWorkflowTime()
	def, err := sampleDefinition(now)
	if err != nil {
		t.Fatalf("sampleDefinition error: %v", err)
	}
	instance := mustStartInstance(t, *def, "wf-transfer", now)
	originalTask := instance.Tasks[0].ID
	if err := instance.Copy(originalTask, Actor{ID: "manager-1"}, Actor{ID: "observer-1"}, "FYI", now.Add(time.Minute)); err != nil {
		t.Fatalf("Copy error: %v", err)
	}
	if len(instance.Tasks) != 2 || instance.Tasks[1].Status != TaskCopied || instance.Tasks[0].Status != TaskPending {
		t.Fatalf("unexpected copy result: %+v", instance.Tasks)
	}
	if err := instance.Transfer(originalTask, Actor{ID: "manager-1"}, Actor{ID: "manager-2"}, "delegate", now.Add(2*time.Minute)); err != nil {
		t.Fatalf("Transfer error: %v", err)
	}
	if len(instance.Tasks) != 3 || instance.Tasks[0].Status != TaskTransferred || instance.Tasks[2].Status != TaskPending || instance.Tasks[2].Assignee.ID != "manager-2" {
		t.Fatalf("unexpected transfer result: %+v", instance.Tasks)
	}
	if err := instance.Approve(*def, instance.Tasks[2].ID, Actor{ID: "manager-2"}, "ok", now.Add(3*time.Minute)); err != nil {
		t.Fatalf("Approve transferred task error: %v", err)
	}
	if instance.Status != InstanceApproved {
		t.Fatalf("expected transferred task approval to finish instance, got %+v", instance)
	}
}

func TestWorkflowRejectsInvalidActorAndDefinitionStates(t *testing.T) {
	now := fixedWorkflowTime()
	def, err := sampleDefinition(now)
	if err != nil {
		t.Fatalf("sampleDefinition error: %v", err)
	}
	draft := *def
	draft.Status = DefinitionDraft
	if _, err := Start(StartInput{ID: "wf-draft", Definition: draft, BusinessType: "leave", BusinessID: "leave-1", Title: "Leave", Starter: Actor{ID: "user-1"}, Now: now}); err == nil {
		t.Fatal("expected draft definition start to fail")
	}
	instance := mustStartInstance(t, *def, "wf-invalid", now)
	if err := instance.Approve(*def, instance.Tasks[0].ID, Actor{ID: "other-manager"}, "ok", now.Add(time.Minute)); err == nil {
		t.Fatal("expected assignee mismatch to fail")
	}
	if err := instance.Withdraw(Actor{ID: "other-user"}, "cancel", now.Add(time.Minute)); err == nil {
		t.Fatal("expected non-starter withdraw to fail")
	}
}

func sampleDefinition(now time.Time) (*Definition, error) {
	def, err := NewDefinition("def-leave", "leave", "Leave Approval", 1, []Node{
		{ID: "start", Key: "start", Name: "Start", Type: NodeStart},
		{ID: "approval", Key: "approval", Name: "Manager Approval", Type: NodeApproval, Assignees: []shared.ID{"manager-1"}, Decision: DecisionRule{Strategy: DecisionAny, Quorum: 1}},
		{ID: "end", Key: "end", Name: "End", Type: NodeEnd},
	}, []Transition{
		{From: "start", To: "approval"},
		{From: "approval", To: "end"},
	}, now)
	if err != nil {
		return nil, err
	}
	return def, def.Publish(now.Add(time.Second))
}

func mustStartInstance(t *testing.T, def Definition, id shared.ID, now time.Time) *Instance {
	t.Helper()
	instance, err := Start(StartInput{
		ID:           id,
		Definition:   def,
		BusinessType: "leave",
		BusinessID:   id.String() + "-business",
		Title:        "Leave Request",
		Starter:      Actor{ID: "user-1"},
		Now:          now,
	})
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
	return instance
}

func fixedWorkflowTime() time.Time {
	return time.Date(2026, 7, 4, 8, 0, 0, 0, time.UTC)
}
