package gormrepo

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainworkflow "github.com/tinboxw/skoll/internal/domain/workflow"
	jobsvc "github.com/tinboxw/skoll/internal/service/job"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"gorm.io/gorm"
)

func TestWorkflowAssignmentAndTimerSurviveRestartAndExecuteOnce(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 27, 14, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	path := filepath.ToSlash(filepath.Join(t.TempDir(), "workflow-assignment-timer.db"))
	dsn := fmt.Sprintf("file:%s?_busy_timeout=5000&_journal_mode=WAL", path)

	db := openWorkflowTestDB(t, dsn)
	service := persistedWorkflowService(db, clock)
	window, err := service.CreateSubstitution(ctx, workflowsvc.CreateSubstitutionInput{
		ID: "absence-restart", Principal: domainworkflow.Actor{ID: "manager-1"}, Substitute: domainworkflow.Actor{ID: "backup-1"},
		StartsAt: now.Add(-time.Hour), EndsAt: now.Add(8 * time.Hour),
		CreatedBy: domainworkflow.Actor{ID: "manager-1"}, Reason: "planned absence", Now: now,
	})
	if err != nil {
		t.Fatalf("CreateSubstitution error: %v", err)
	}
	definition, err := service.CreateDefinition(ctx, workflowsvc.CreateDefinitionInput{
		ID: "definition-restart-timer", Key: "restart.timer", Name: "Restart timer", Version: 1, Now: now,
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
	instance, err := service.Start(ctx, workflowsvc.StartInput{
		ID: "instance-restart-timer", DefinitionID: definition.ID, BusinessType: "purchase", BusinessID: "PO-RESTART",
		Title: "Purchase", Starter: domainworkflow.Actor{ID: "buyer-1"}, Variables: map[string]domainworkflow.Value{}, Now: now,
	})
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
	if instance.Tasks[0].Assignee.ID != "backup-1" || instance.Tasks[0].AuthorizationID != window.ID {
		t.Fatalf("substitution was not persisted on task: %+v", instance.Tasks[0])
	}
	closeWorkflowTestDB(t, db)

	now = now.Add(2 * time.Minute)
	restartedDB := openWorkflowTestDB(t, dsn)
	restarted := persistedWorkflowService(restartedDB, clock)
	if completed, err := restarted.ProcessDueTimers(ctx, "restart-worker", 10, time.Minute); err != nil || completed != 1 {
		t.Fatalf("ProcessDueTimers after restart completed=%d err=%v", completed, err)
	}
	closeWorkflowTestDB(t, restartedDB)

	finalDB := openWorkflowTestDB(t, dsn)
	defer closeWorkflowTestDB(t, finalDB)
	finalService := persistedWorkflowService(finalDB, clock)
	if completed, err := finalService.ProcessDueTimers(ctx, "final-worker", 10, time.Minute); err != nil || completed != 0 {
		t.Fatalf("completed timer replayed after second restart: completed=%d err=%v", completed, err)
	}
	persisted, err := finalService.GetInstance(ctx, instance.ID)
	if err != nil {
		t.Fatalf("GetInstance after restart error: %v", err)
	}
	escalations := 0
	for _, action := range persisted.Timeline {
		if action.Type == domainworkflow.ActionEscalate {
			escalations++
		}
	}
	if escalations != 1 || len(persisted.Tasks) != 2 || persisted.Tasks[1].Assignment != domainworkflow.AssignmentEscalated {
		t.Fatalf("restart timer evidence is invalid: escalations=%d tasks=%+v", escalations, persisted.Tasks)
	}
	if _, err := finalService.RevokeSubstitution(ctx, workflowsvc.RevokeSubstitutionInput{
		ID: window.ID, Principal: window.Principal, Now: now,
	}); err != nil {
		t.Fatalf("RevokeSubstitution after restart error: %v", err)
	}
	if active, err := NewWorkflowStore(finalDB).ListActiveSubstitutions(ctx, []shared.ID{"manager-1"}, now); err != nil || len(active) != 0 {
		t.Fatalf("revoked substitution remained active: active=%+v err=%v", active, err)
	}
}

func persistedWorkflowService(db *gorm.DB, now func() time.Time) workflowsvc.Service {
	jobs := jobsvc.NewService(NewJobStore(db), now)
	return workflowsvc.NewService(NewWorkflowStore(db), workflowsvc.Options{
		Jobs: jobs, UnitOfWork: storesql.NewUnitOfWorkWithDB(db), Now: now,
	})
}
