package workflow

import (
	"context"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainworkflow "github.com/tinboxw/skoll/internal/domain/workflow"
)

func TestWorkflowServiceStartApproveAndPersist(t *testing.T) {
	ctx := context.Background()
	now := fixedServiceWorkflowTime()
	repo := NewMemoryRepository()
	svc := NewService(repo)

	definition, err := svc.CreateDefinition(ctx, sampleCreateDefinitionInput(now))
	if err != nil {
		t.Fatalf("CreateDefinition error: %v", err)
	}
	if definition.Status != domainworkflow.DefinitionDraft {
		t.Fatalf("expected draft definition, got %s", definition.Status)
	}
	if _, err := svc.Start(ctx, StartInput{ID: "wf-before-publish", DefinitionID: definition.ID, BusinessType: "leave", BusinessID: "leave-1", Title: "Leave", Starter: domainworkflow.Actor{ID: "user-1"}, Now: now}); err == nil {
		t.Fatal("expected draft start to fail")
	}
	definition, err = svc.PublishDefinition(ctx, definition.ID, now.Add(time.Second))
	if err != nil {
		t.Fatalf("PublishDefinition error: %v", err)
	}
	if definition.Status != domainworkflow.DefinitionPublished {
		t.Fatalf("expected published definition, got %s", definition.Status)
	}

	instance, err := svc.Start(ctx, StartInput{
		ID:           "wf-1",
		DefinitionID: definition.ID,
		BusinessType: "leave",
		BusinessID:   "leave-1",
		Title:        "Leave Request",
		Starter:      domainworkflow.Actor{ID: "user-1"},
		Now:          now.Add(2 * time.Second),
	})
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
	if instance.Status != domainworkflow.InstanceRunning || len(instance.Tasks) != 1 {
		t.Fatalf("unexpected instance: %+v", instance)
	}
	taskID := instance.Tasks[0].ID
	instance, err = svc.Approve(ctx, TaskActionInput{InstanceID: instance.ID, TaskID: taskID, Actor: domainworkflow.Actor{ID: "manager-1"}, Comment: "ok", Now: now.Add(time.Minute)})
	if err != nil {
		t.Fatalf("Approve error: %v", err)
	}
	if instance.Status != domainworkflow.InstanceApproved || instance.Tasks[0].Status != domainworkflow.TaskApproved {
		t.Fatalf("unexpected approved instance: %+v", instance)
	}
	persisted, err := repo.GetInstance(ctx, "wf-1")
	if err != nil {
		t.Fatalf("GetInstance error: %v", err)
	}
	if persisted.Status != domainworkflow.InstanceApproved {
		t.Fatalf("expected persisted approved instance, got %+v", persisted)
	}
}

func TestWorkflowServiceTransferCopyRejectAndWithdraw(t *testing.T) {
	ctx := context.Background()
	now := fixedServiceWorkflowTime()
	svc := NewService(NewMemoryRepository())
	definition := mustPublishedDefinition(t, ctx, svc, now)
	instance := mustStartServiceInstance(t, ctx, svc, definition.ID, "wf-transfer", now)
	taskID := instance.Tasks[0].ID

	if _, err := svc.Copy(ctx, TaskTargetActionInput{InstanceID: instance.ID, TaskID: taskID, Actor: domainworkflow.Actor{ID: "manager-1"}, Target: domainworkflow.Actor{ID: "observer-1"}, Comment: "FYI", Now: now.Add(time.Minute)}); err != nil {
		t.Fatalf("Copy error: %v", err)
	}
	instance, err := svc.Transfer(ctx, TaskTargetActionInput{InstanceID: instance.ID, TaskID: taskID, Actor: domainworkflow.Actor{ID: "manager-1"}, Target: domainworkflow.Actor{ID: "manager-2"}, Comment: "delegate", Now: now.Add(2 * time.Minute)})
	if err != nil {
		t.Fatalf("Transfer error: %v", err)
	}
	if len(instance.Tasks) != 3 || instance.Tasks[2].Assignee.ID != "manager-2" {
		t.Fatalf("unexpected transferred instance: %+v", instance.Tasks)
	}
	if _, err := svc.Reject(ctx, TaskActionInput{InstanceID: instance.ID, TaskID: instance.Tasks[2].ID, Actor: domainworkflow.Actor{ID: "manager-2"}, Comment: "no", Now: now.Add(3 * time.Minute)}); err != nil {
		t.Fatalf("Reject error: %v", err)
	}

	withdraw := mustStartServiceInstance(t, ctx, svc, definition.ID, "wf-withdraw", now)
	if _, err := svc.Withdraw(ctx, InstanceActionInput{InstanceID: withdraw.ID, Actor: domainworkflow.Actor{ID: "user-1"}, Comment: "cancel", Now: now.Add(time.Minute)}); err != nil {
		t.Fatalf("Withdraw error: %v", err)
	}
}

func TestWorkflowMemoryRepositoryReturnsClones(t *testing.T) {
	ctx := context.Background()
	now := fixedServiceWorkflowTime()
	repo := NewMemoryRepository()
	def, err := domainworkflow.NewDefinition("def-1", "leave", "Leave", 1, []domainworkflow.Node{
		{ID: "start", Key: "start", Name: "Start", Type: domainworkflow.NodeStart},
		{ID: "approval", Key: "approval", Name: "Approval", Type: domainworkflow.NodeApproval, Assignees: []shared.ID{"manager-1"}},
		{ID: "end", Key: "end", Name: "End", Type: domainworkflow.NodeEnd},
	}, []domainworkflow.Transition{{From: "start", To: "approval"}, {From: "approval", To: "end"}}, now)
	if err != nil {
		t.Fatalf("NewDefinition error: %v", err)
	}
	if err := repo.SaveDefinition(ctx, *def); err != nil {
		t.Fatalf("SaveDefinition error: %v", err)
	}
	got, err := repo.GetDefinition(ctx, "def-1")
	if err != nil {
		t.Fatalf("GetDefinition error: %v", err)
	}
	got.Nodes[1].Assignees[0] = "mutated"
	again, err := repo.GetDefinition(ctx, "def-1")
	if err != nil {
		t.Fatalf("GetDefinition second error: %v", err)
	}
	if again.Nodes[1].Assignees[0] == "mutated" {
		t.Fatal("repository returned mutable definition internals")
	}
}

func sampleCreateDefinitionInput(now time.Time) CreateDefinitionInput {
	return CreateDefinitionInput{
		ID:      "def-leave",
		Key:     "leave",
		Name:    "Leave Approval",
		Version: 1,
		Nodes: []domainworkflow.Node{
			{ID: "start", Key: "start", Name: "Start", Type: domainworkflow.NodeStart},
			{ID: "approval", Key: "approval", Name: "Manager Approval", Type: domainworkflow.NodeApproval, Assignees: []shared.ID{"manager-1"}},
			{ID: "end", Key: "end", Name: "End", Type: domainworkflow.NodeEnd},
		},
		Transitions: []domainworkflow.Transition{{From: "start", To: "approval"}, {From: "approval", To: "end"}},
		Now:         now,
	}
}

func mustPublishedDefinition(t *testing.T, ctx context.Context, svc Service, now time.Time) *domainworkflow.Definition {
	t.Helper()
	definition, err := svc.CreateDefinition(ctx, sampleCreateDefinitionInput(now))
	if err != nil {
		t.Fatalf("CreateDefinition error: %v", err)
	}
	definition, err = svc.PublishDefinition(ctx, definition.ID, now.Add(time.Second))
	if err != nil {
		t.Fatalf("PublishDefinition error: %v", err)
	}
	return definition
}

func mustStartServiceInstance(t *testing.T, ctx context.Context, svc Service, definitionID shared.ID, id shared.ID, now time.Time) *domainworkflow.Instance {
	t.Helper()
	instance, err := svc.Start(ctx, StartInput{
		ID:           id,
		DefinitionID: definitionID,
		BusinessType: "leave",
		BusinessID:   id.String() + "-business",
		Title:        "Leave Request",
		Starter:      domainworkflow.Actor{ID: "user-1"},
		Now:          now,
	})
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
	return instance
}

func fixedServiceWorkflowTime() time.Time {
	return time.Date(2026, 7, 4, 9, 0, 0, 0, time.UTC)
}
