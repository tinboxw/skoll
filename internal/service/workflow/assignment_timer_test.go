package workflow

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainworkflow "github.com/tinboxw/skoll/internal/domain/workflow"
	jobsvc "github.com/tinboxw/skoll/internal/service/job"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
)

func TestSubstitutionAndEscalationTimersExecuteOnceUnderCompetition(t *testing.T) {
	ctx := context.Background()
	var clockMu sync.RWMutex
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time {
		clockMu.RLock()
		defer clockMu.RUnlock()
		return now
	}
	repository := NewMemoryRepository()
	jobs := jobsvc.NewService(jobsvc.NewMemoryRepository(), clock)
	service := NewService(repository, Options{Jobs: jobs, UnitOfWork: storesql.NewUnitOfWork(), Now: clock})

	if _, err := service.CreateSubstitution(ctx, CreateSubstitutionInput{
		ID: "absence-1", Principal: domainworkflow.Actor{ID: "manager-1"}, Substitute: domainworkflow.Actor{ID: "backup-1"},
		StartsAt: now.Add(-time.Hour), EndsAt: now.Add(8 * time.Hour),
		CreatedBy: domainworkflow.Actor{ID: "manager-1"}, Reason: "planned absence", Now: now,
	}); err != nil {
		t.Fatalf("CreateSubstitution error: %v", err)
	}
	if _, err := service.CreateSubstitution(ctx, CreateSubstitutionInput{
		ID: "absence-overlap", Principal: domainworkflow.Actor{ID: "manager-1"}, Substitute: domainworkflow.Actor{ID: "backup-2"},
		StartsAt: now, EndsAt: now.Add(time.Hour), CreatedBy: domainworkflow.Actor{ID: "manager-1"}, Now: now,
	}); err == nil {
		t.Fatal("overlapping substitution window was accepted")
	}
	if _, err := service.CreateSubstitution(ctx, CreateSubstitutionInput{
		ID: "absence-forged", Principal: domainworkflow.Actor{ID: "manager-2"}, Substitute: domainworkflow.Actor{ID: "backup-2"},
		StartsAt: now, EndsAt: now.Add(time.Hour), CreatedBy: domainworkflow.Actor{ID: "administrator"}, Now: now,
	}); err == nil {
		t.Fatal("substitution created by a different principal was accepted")
	}
	definition, err := service.CreateDefinition(ctx, CreateDefinitionInput{
		ID: "definition-timer", Key: "timer.approval", Name: "Timer approval", Version: 1, Now: now,
		Nodes: []domainworkflow.Node{
			{ID: "start", Key: "start", Name: "Start", Type: domainworkflow.NodeStart},
			{
				ID: "approval", Key: "approval", Name: "Approval", Type: domainworkflow.NodeApproval,
				Assignees: []shared.ID{"manager-1"}, Decision: domainworkflow.DecisionRule{Strategy: domainworkflow.DecisionAny, Quorum: 1},
				Escalation: &domainworkflow.EscalationRule{After: time.Minute, Target: domainworkflow.Actor{ID: "director-1"}},
			},
			{ID: "end", Key: "end", Name: "End", Type: domainworkflow.NodeEnd},
		},
		Transitions: []domainworkflow.Transition{{From: "start", To: "approval"}, {From: "approval", To: "end"}},
	})
	if err != nil {
		t.Fatalf("CreateDefinition error: %v", err)
	}
	if _, err = service.PublishDefinition(ctx, definition.ID, now); err != nil {
		t.Fatalf("PublishDefinition error: %v", err)
	}
	instance, err := service.Start(ctx, StartInput{
		ID: "instance-timer", DefinitionID: definition.ID, BusinessType: "purchase", BusinessID: "PO-TIMER",
		Title: "Purchase", Starter: domainworkflow.Actor{ID: "buyer-1"}, Variables: map[string]domainworkflow.Value{}, Now: now,
	})
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
	if task := instance.Tasks[0]; task.Assignee.ID != "backup-1" || task.Assignment != domainworkflow.AssignmentSubstituted {
		t.Fatalf("absence substitution was not applied: %+v", task)
	}

	clockMu.Lock()
	now = now.Add(2 * time.Minute)
	clockMu.Unlock()
	var completed atomic.Int64
	var wait sync.WaitGroup
	for index := range 24 {
		wait.Add(1)
		go func(worker int) {
			defer wait.Done()
			count, processErr := service.ProcessDueTimers(ctx, fmt.Sprintf("worker-%d", worker), 1, time.Minute)
			if processErr != nil {
				t.Errorf("ProcessDueTimers error: %v", processErr)
				return
			}
			completed.Add(int64(count))
		}(index)
	}
	wait.Wait()
	if completed.Load() != 1 {
		t.Fatalf("timer completion count=%d, want 1", completed.Load())
	}
	persisted, err := service.GetInstance(ctx, instance.ID)
	if err != nil {
		t.Fatalf("GetInstance error: %v", err)
	}
	escalations := 0
	for _, action := range persisted.Timeline {
		if action.Type == domainworkflow.ActionEscalate {
			escalations++
		}
	}
	if escalations != 1 || len(persisted.Tasks) != 2 {
		t.Fatalf("timer was not exactly once: escalations=%d tasks=%+v", escalations, persisted.Tasks)
	}
	if task := persisted.Tasks[1]; task.Assignee.ID != "director-1" || task.Assignment != domainworkflow.AssignmentEscalated ||
		task.OriginalAssignee.ID != "manager-1" {
		t.Fatalf("escalated task provenance is incomplete: %+v", task)
	}
}
