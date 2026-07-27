package workflow

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainworkflow "github.com/tinboxw/skoll/internal/domain/workflow"
)

func TestQuorumDecisionsAreAtomicIdempotentAndRejectStaleTasks(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	service := newTestService(NewMemoryRepository())
	definition, err := service.CreateDefinition(ctx, CreateDefinitionInput{
		ID: "quorum-definition", Key: "medical.quality.release", Name: "Quality Release", Version: 1,
		Nodes: []domainworkflow.Node{
			{ID: "start", Key: "start", Name: "Start", Type: domainworkflow.NodeStart},
			{
				ID: "quality", Key: "quality", Name: "Quality", Type: domainworkflow.NodeApproval,
				Assignees: []shared.ID{"quality-1", "quality-2", "quality-3"},
				Decision:  domainworkflow.DecisionRule{Strategy: domainworkflow.DecisionQuorum, Quorum: 2},
			},
			{ID: "end", Key: "end", Name: "End", Type: domainworkflow.NodeEnd},
		},
		Transitions: []domainworkflow.Transition{{From: "start", To: "quality"}, {From: "quality", To: "end"}},
		Now:         now,
	})
	if err != nil {
		t.Fatalf("CreateDefinition error: %v", err)
	}
	if _, err = service.PublishDefinition(ctx, definition.ID, now); err != nil {
		t.Fatalf("PublishDefinition error: %v", err)
	}
	instance, err := service.Start(ctx, StartInput{
		ID: "quorum-instance", DefinitionID: definition.ID, BusinessType: "quality-release",
		BusinessID: "batch-1", Title: "Release batch", Starter: domainworkflow.Actor{ID: "qa-owner"}, Variables: map[string]domainworkflow.Value{}, Now: now,
	})
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}

	tasks := make(map[shared.ID]domainworkflow.Task)
	for _, task := range instance.Tasks {
		tasks[task.Assignee.ID] = task
	}
	const workers = 64
	start := make(chan struct{})
	errors := make(chan error, workers)
	var group sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		actorID := shared.ID("quality-1")
		if worker%2 == 1 {
			actorID = "quality-2"
		}
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			_, actionErr := service.Approve(ctx, TaskActionInput{
				InstanceID: instance.ID, TaskID: tasks[actorID].ID,
				Actor: domainworkflow.Actor{ID: actorID}, Comment: "approved", Now: now.Add(time.Minute),
			})
			errors <- actionErr
		}()
	}
	close(start)
	group.Wait()
	close(errors)
	for actionErr := range errors {
		if actionErr != nil {
			t.Fatalf("duplicate quorum decision should be idempotent: %v", actionErr)
		}
	}

	result, err := service.GetInstance(ctx, instance.ID)
	if err != nil {
		t.Fatalf("GetInstance error: %v", err)
	}
	if result.Status != domainworkflow.InstanceApproved {
		t.Fatalf("quorum did not resolve instance: %+v", result)
	}
	approvals := 0
	for _, action := range result.Timeline {
		if action.Type == domainworkflow.ActionApprove {
			approvals++
		}
	}
	if approvals != 2 {
		t.Fatalf("expected exactly two approval effects, got %d", approvals)
	}
	if _, err := service.Approve(ctx, TaskActionInput{
		InstanceID: instance.ID, TaskID: tasks["quality-3"].ID,
		Actor: domainworkflow.Actor{ID: "quality-3"}, Comment: "stale", Now: now.Add(2 * time.Minute),
	}); err == nil {
		t.Fatal("expected stale third decision to fail after quorum resolution")
	}
}

func TestGovernedWorkflowQuorumProperty(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 27, 16, 0, 0, 0, time.UTC)
	for assigneeCount := 2; assigneeCount <= 7; assigneeCount++ {
		for quorum := 1; quorum <= assigneeCount; quorum++ {
			name := fmt.Sprintf("assignees_%d_quorum_%d", assigneeCount, quorum)
			t.Run(name, func(t *testing.T) {
				service := newTestService(NewMemoryRepository())
				assignees := make([]shared.ID, assigneeCount)
				for index := range assignees {
					assignees[index] = shared.ID(fmt.Sprintf("quality-%d", index+1))
				}
				definition, err := service.CreateDefinition(ctx, CreateDefinitionInput{
					ID: shared.ID("definition-" + name), Key: "regulated." + name, Name: name, Version: 1, Now: now,
					Nodes: []domainworkflow.Node{
						{ID: "start", Key: "start", Name: "Start", Type: domainworkflow.NodeStart},
						{
							ID: "quality", Key: "quality", Name: "Quality", Type: domainworkflow.NodeApproval,
							Assignees: assignees,
							Decision:  domainworkflow.DecisionRule{Strategy: domainworkflow.DecisionQuorum, Quorum: quorum},
						},
						{ID: "end", Key: "end", Name: "End", Type: domainworkflow.NodeEnd},
					},
					Transitions: []domainworkflow.Transition{{From: "start", To: "quality"}, {From: "quality", To: "end"}},
				})
				if err != nil {
					t.Fatal(err)
				}
				if _, err = service.PublishDefinition(ctx, definition.ID, now); err != nil {
					t.Fatal(err)
				}
				instance, err := service.Start(ctx, StartInput{
					ID: shared.ID("instance-" + name), DefinitionID: definition.ID, BusinessType: "batch-release",
					BusinessID: name, Title: name, Starter: domainworkflow.Actor{ID: "starter"}, Now: now,
				})
				if err != nil {
					t.Fatal(err)
				}
				tasks := make(map[shared.ID]domainworkflow.Task, len(instance.Tasks))
				for _, task := range instance.Tasks {
					tasks[task.Assignee.ID] = task
				}
				for index := 0; index < quorum; index++ {
					actorID := assignees[index]
					for duplicate := 0; duplicate < 3; duplicate++ {
						if _, err = service.Approve(ctx, TaskActionInput{
							InstanceID: instance.ID, TaskID: tasks[actorID].ID,
							Actor: domainworkflow.Actor{ID: actorID}, Comment: "property approval", Now: now.Add(time.Minute),
						}); err != nil {
							t.Fatalf("duplicate %d for %s failed: %v", duplicate, actorID, err)
						}
					}
				}
				resolved, err := service.GetInstance(ctx, instance.ID)
				if err != nil {
					t.Fatal(err)
				}
				approvalCount := 0
				for _, action := range resolved.Timeline {
					if action.Type == domainworkflow.ActionApprove {
						approvalCount++
					}
				}
				if resolved.Status != domainworkflow.InstanceApproved || approvalCount != quorum {
					t.Fatalf("quorum property failed: status=%s approvals=%d want=%d", resolved.Status, approvalCount, quorum)
				}
				for index := quorum; index < assigneeCount; index++ {
					actorID := assignees[index]
					if _, err = service.Approve(ctx, TaskActionInput{
						InstanceID: instance.ID, TaskID: tasks[actorID].ID,
						Actor: domainworkflow.Actor{ID: actorID}, Comment: "stale property decision", Now: now.Add(2 * time.Minute),
					}); err == nil {
						t.Fatalf("stale task for %s remained actionable", actorID)
					}
				}
			})
		}
	}
}
