package gormrepo

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainworkflow "github.com/tinboxw/skoll/internal/domain/workflow"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
	"gorm.io/gorm"
)

func TestWorkflowStoreSerializesCompetingDecisions(t *testing.T) {
	ctx := context.Background()
	db := openConcurrentWorkflowTestDB(t, "competing-decisions")
	service := workflowsvc.NewService(NewWorkflowStore(db))
	instance := createConcurrentWorkflowInstance(t, ctx, service, "competing")
	taskID := instance.Tasks[0].ID
	start := make(chan struct{})
	results := make(chan error, 2)

	go func() {
		<-start
		_, err := service.Approve(ctx, workflowsvc.TaskActionInput{
			InstanceID: instance.ID, TaskID: taskID, Actor: domainworkflow.Actor{ID: "manager-1", Name: "Manager One"},
			Comment: "approve", Now: workflowConcurrencyTime().Add(time.Minute),
		})
		results <- err
	}()
	go func() {
		<-start
		_, err := service.Reject(ctx, workflowsvc.TaskActionInput{
			InstanceID: instance.ID, TaskID: taskID, Actor: domainworkflow.Actor{ID: "manager-1", Name: "Manager One"},
			Comment: "reject", Now: workflowConcurrencyTime().Add(2 * time.Minute),
		})
		results <- err
	}()
	close(start)

	succeeded := 0
	failed := 0
	for range 2 {
		err := <-results
		if err == nil {
			succeeded++
			continue
		}
		failed++
		message := strings.ToLower(err.Error())
		if strings.Contains(message, "locked") || strings.Contains(message, "deadlock") || strings.Contains(message, "serialize") {
			t.Fatalf("database contention leaked instead of a workflow conflict: %v", err)
		}
	}
	if succeeded != 1 || failed != 1 {
		t.Fatalf("expected one committed decision and one rejected competitor, got success=%d failed=%d", succeeded, failed)
	}

	persisted, err := service.GetInstance(ctx, instance.ID)
	if err != nil {
		t.Fatalf("GetInstance error: %v", err)
	}
	terminalActions := 0
	for _, action := range persisted.Timeline {
		if action.Type == domainworkflow.ActionApprove || action.Type == domainworkflow.ActionReject {
			terminalActions++
		}
	}
	if terminalActions != 1 || (persisted.Status != domainworkflow.InstanceApproved && persisted.Status != domainworkflow.InstanceRejected) {
		t.Fatalf("competing transitions produced an invalid final state: %+v", persisted)
	}
}

func TestWorkflowStoreDeduplicatesConcurrentNonTerminalActions(t *testing.T) {
	ctx := context.Background()
	db := openConcurrentWorkflowTestDB(t, "duplicate-copy")
	service := workflowsvc.NewService(NewWorkflowStore(db))
	instance := createConcurrentWorkflowInstance(t, ctx, service, "duplicate-copy")
	taskID := instance.Tasks[0].ID

	const workers = 12
	start := make(chan struct{})
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := service.Copy(ctx, workflowsvc.TaskTargetActionInput{
				InstanceID: instance.ID, TaskID: taskID,
				Actor:   domainworkflow.Actor{ID: "manager-1", Name: "Manager One"},
				Target:  domainworkflow.Actor{ID: "quality-1", Name: "Quality One"},
				Comment: "quality copy", Now: workflowConcurrencyTime().Add(time.Minute),
			})
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("duplicate copy should return the committed result: %v", err)
		}
	}

	persisted, err := service.GetInstance(ctx, instance.ID)
	if err != nil {
		t.Fatalf("GetInstance error: %v", err)
	}
	copyActions := 0
	for _, action := range persisted.Timeline {
		if action.Type == domainworkflow.ActionCopy {
			copyActions++
		}
	}
	if persisted.Status != domainworkflow.InstanceRunning || len(persisted.Tasks) != 2 || copyActions != 1 {
		t.Fatalf("duplicate copy produced duplicate effects: tasks=%+v timeline=%+v", persisted.Tasks, persisted.Timeline)
	}

	distinctTargets := []domainworkflow.Actor{{ID: "observer-1", Name: "Observer One"}, {ID: "observer-2", Name: "Observer Two"}}
	start = make(chan struct{})
	errs = make(chan error, len(distinctTargets))
	for _, target := range distinctTargets {
		target := target
		go func() {
			<-start
			_, err := service.Copy(ctx, workflowsvc.TaskTargetActionInput{
				InstanceID: instance.ID, TaskID: taskID,
				Actor: domainworkflow.Actor{ID: "manager-1", Name: "Manager One"}, Target: target,
				Comment: "distinct target", Now: workflowConcurrencyTime().Add(2 * time.Minute),
			})
			errs <- err
		}()
	}
	close(start)
	for range distinctTargets {
		if err := <-errs; err != nil {
			t.Fatalf("independent concurrent copy error: %v", err)
		}
	}
	persisted, err = service.GetInstance(ctx, instance.ID)
	if err != nil {
		t.Fatalf("GetInstance after independent copies error: %v", err)
	}
	copyActions = 0
	actionIDs := map[shared.ID]struct{}{}
	for _, action := range persisted.Timeline {
		if action.Type != domainworkflow.ActionCopy {
			continue
		}
		copyActions++
		actionIDs[action.ID] = struct{}{}
	}
	if len(persisted.Tasks) != 4 || copyActions != 3 || len(actionIDs) != 3 {
		t.Fatalf("independent copies were lost or collided: tasks=%+v timeline=%+v", persisted.Tasks, persisted.Timeline)
	}
}

func TestWorkflowDecisionRecoversAfterCommittedResponseInterruption(t *testing.T) {
	ctx := context.Background()
	dsn := filepath.Join(t.TempDir(), "workflow-recovery.db")
	db := openWorkflowTestDB(t, dsn)
	repository := NewWorkflowStore(db)
	service := workflowsvc.NewService(repository)
	instance := createConcurrentWorkflowInstance(t, ctx, service, "response-interruption")
	taskID := instance.Tasks[0].ID
	commitTime := workflowConcurrencyTime().Add(time.Minute)
	interrupted := &interruptAfterCommitRepository{Repository: repository}
	interruptedService := workflowsvc.NewService(interrupted)

	_, err := interruptedService.Approve(ctx, workflowsvc.TaskActionInput{
		InstanceID: instance.ID, TaskID: taskID, Actor: domainworkflow.Actor{ID: "manager-1", Name: "Manager One"},
		Comment: "committed before disconnect", Now: commitTime,
	})
	if !errors.Is(err, errWorkflowResponseInterrupted) {
		t.Fatalf("expected simulated response interruption, got %v", err)
	}
	closeWorkflowTestDB(t, db)

	restartedDB := openWorkflowTestDB(t, dsn)
	t.Cleanup(func() { closeWorkflowTestDB(t, restartedDB) })
	restartedService := workflowsvc.NewService(NewWorkflowStore(restartedDB))
	recovered, err := restartedService.Approve(ctx, workflowsvc.TaskActionInput{
		InstanceID: instance.ID, TaskID: taskID, Actor: domainworkflow.Actor{ID: "manager-1", Name: "Manager One"},
		Comment: "retry after restart", Now: commitTime.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("retry after committed interruption error: %v", err)
	}
	approveActions := 0
	for _, action := range recovered.Timeline {
		if action.Type == domainworkflow.ActionApprove {
			approveActions++
			if action.Comment != "committed before disconnect" || !action.CreatedAt.Equal(commitTime) {
				t.Fatalf("retry replaced the committed decision: %+v", action)
			}
		}
	}
	if recovered.Status != domainworkflow.InstanceApproved || approveActions != 1 || recovered.Tasks[0].CompletedAt == nil || !recovered.Tasks[0].CompletedAt.Equal(commitTime) {
		t.Fatalf("recovery did not return the original committed state: %+v", recovered)
	}
}

func TestWorkflowStoreFailedAtomicUpdateLeavesCommittedState(t *testing.T) {
	ctx := context.Background()
	db := openConcurrentWorkflowTestDB(t, "failed-update")
	repository := NewWorkflowStore(db)
	service := workflowsvc.NewService(repository)
	instance := createConcurrentWorkflowInstance(t, ctx, service, "failed-update")

	_, err := repository.UpdateInstance(ctx, instance.ID, func(candidate *domainworkflow.Instance) (bool, error) {
		candidate.Title = "uncommitted title"
		candidate.Tasks = append(candidate.Tasks, candidate.Tasks[0])
		return true, nil
	})
	if err == nil {
		t.Fatal("expected duplicate task to fail the atomic update")
	}
	persisted, err := repository.GetInstance(ctx, instance.ID)
	if err != nil {
		t.Fatalf("GetInstance after failed update error: %v", err)
	}
	if persisted.Title == "uncommitted title" || len(persisted.Tasks) != 1 || len(persisted.Timeline) != 1 {
		t.Fatalf("failed atomic update leaked candidate state: %+v", persisted)
	}
}

var errWorkflowResponseInterrupted = errors.New("workflow response interrupted after commit")

type interruptAfterCommitRepository struct {
	workflowsvc.Repository
	once sync.Once
}

func (r *interruptAfterCommitRepository) UpdateInstance(ctx context.Context, id shared.ID, mutate workflowsvc.InstanceMutation) (*domainworkflow.Instance, error) {
	instance, err := r.Repository.UpdateInstance(ctx, id, mutate)
	if err != nil {
		return nil, err
	}
	interrupted := false
	r.once.Do(func() { interrupted = true })
	if interrupted {
		return nil, errWorkflowResponseInterrupted
	}
	return instance, nil
}

func createConcurrentWorkflowInstance(t *testing.T, ctx context.Context, service workflowsvc.Service, suffix string) *domainworkflow.Instance {
	t.Helper()
	now := workflowConcurrencyTime()
	definitionID := shared.ID("definition-" + suffix)
	definition, err := service.CreateDefinition(ctx, workflowsvc.CreateDefinitionInput{
		ID: definitionID, Key: "approval." + suffix, Name: "Approval " + suffix, Version: 1,
		Nodes: []domainworkflow.Node{
			{ID: "node-start", Key: "start", Name: "Start", Type: domainworkflow.NodeStart},
			{ID: "node-approval", Key: "approval", Name: "Approval", Type: domainworkflow.NodeApproval, Assignees: []shared.ID{"manager-1"}},
			{ID: "node-end", Key: "end", Name: "End", Type: domainworkflow.NodeEnd},
		},
		Transitions: []domainworkflow.Transition{{From: "node-start", To: "node-approval"}, {From: "node-approval", To: "node-end"}},
		Now:         now,
	})
	if err != nil {
		t.Fatalf("CreateDefinition error: %v", err)
	}
	definition, err = service.PublishDefinition(ctx, definition.ID, now.Add(time.Second))
	if err != nil {
		t.Fatalf("PublishDefinition error: %v", err)
	}
	instance, err := service.Start(ctx, workflowsvc.StartInput{
		ID: shared.ID("instance-" + suffix), DefinitionID: definition.ID, BusinessType: "approval_test", BusinessID: "business-" + suffix,
		Title: "Approval " + suffix, Starter: domainworkflow.Actor{ID: "employee-1", Name: "Employee One"}, Now: now.Add(2 * time.Second),
	})
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
	return instance
}

func openConcurrentWorkflowTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	path := filepath.ToSlash(filepath.Join(t.TempDir(), name+".db"))
	db := openWorkflowTestDB(t, fmt.Sprintf("file:%s?_busy_timeout=5000&_journal_mode=WAL", path))
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("resolve concurrent workflow sql database: %v", err)
	}
	sqlDB.SetMaxOpenConns(16)
	t.Cleanup(func() { closeWorkflowTestDB(t, db) })
	return db
}

func workflowConcurrencyTime() time.Time {
	return time.Date(2026, 7, 22, 13, 0, 0, 0, time.UTC)
}
