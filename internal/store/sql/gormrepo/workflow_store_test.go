package gormrepo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainworkflow "github.com/tinboxw/skoll/internal/domain/workflow"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestWorkflowStorePersistsAggregateAcrossDatabaseRestart(t *testing.T) {
	ctx := context.Background()
	dsn := filepath.Join(t.TempDir(), "workflow.db")
	db := openWorkflowTestDB(t, dsn)
	repo := NewWorkflowStore(db)
	service := workflowsvc.NewService(repo)
	now := time.Date(2026, 7, 22, 9, 30, 0, 0, time.UTC)

	definition, err := service.CreateDefinition(ctx, workflowsvc.CreateDefinitionInput{
		ID: "definition-purchase", Key: "purchase.approval", Name: "Purchase Approval", Version: 3,
		Nodes: []domainworkflow.Node{
			{ID: "node-start", Key: "start", Name: "Start", Type: domainworkflow.NodeStart},
			{ID: "node-approve", Key: "approve", Name: "Manager Approval", Type: domainworkflow.NodeApproval, Assignees: []shared.ID{"manager-1"}},
			{ID: "node-end", Key: "end", Name: "End", Type: domainworkflow.NodeEnd},
		},
		Transitions: []domainworkflow.Transition{{From: "node-start", To: "node-approve"}, {From: "node-approve", To: "node-end"}},
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
		ID: "instance-purchase-1", DefinitionID: definition.ID, BusinessType: "purchase_order", BusinessID: "PO-20260722-001",
		Title: "Cold-chain Purchase", Starter: domainworkflow.Actor{ID: "employee-1", Name: "Alice Zhang"}, Now: now.Add(2 * time.Second),
	})
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
	initialTaskID := instance.Tasks[0].ID
	instance, err = service.Copy(ctx, workflowsvc.TaskTargetActionInput{
		InstanceID: instance.ID, TaskID: initialTaskID,
		Actor:   domainworkflow.Actor{ID: "manager-1", Name: "Manager One"},
		Target:  domainworkflow.Actor{ID: "quality-1", Name: "Quality Observer"},
		Comment: "quality review", Now: now.Add(3 * time.Second),
	})
	if err != nil {
		t.Fatalf("Copy error: %v", err)
	}
	instance, err = service.Transfer(ctx, workflowsvc.TaskTargetActionInput{
		InstanceID: instance.ID, TaskID: initialTaskID,
		Actor:   domainworkflow.Actor{ID: "manager-1", Name: "Manager One"},
		Target:  domainworkflow.Actor{ID: "manager-2", Name: "Manager Two"},
		Comment: "delegate approval", Now: now.Add(4 * time.Second),
	})
	if err != nil {
		t.Fatalf("Transfer error: %v", err)
	}
	transferredTaskID := instance.Tasks[len(instance.Tasks)-1].ID
	if _, err := service.Approve(ctx, workflowsvc.TaskActionInput{
		InstanceID: instance.ID, TaskID: transferredTaskID,
		Actor:   domainworkflow.Actor{ID: "manager-2", Name: "Manager Two"},
		Comment: "approved", Now: now.Add(5 * time.Second),
	}); err != nil {
		t.Fatalf("Approve error: %v", err)
	}

	closeWorkflowTestDB(t, db)
	restartedDB := openWorkflowTestDB(t, dsn)
	t.Cleanup(func() { closeWorkflowTestDB(t, restartedDB) })
	restartedRepo := NewWorkflowStore(restartedDB)

	persistedDefinition, err := restartedRepo.GetDefinition(ctx, definition.ID)
	if err != nil {
		t.Fatalf("GetDefinition after restart error: %v", err)
	}
	if persistedDefinition.Status != domainworkflow.DefinitionPublished || persistedDefinition.Version != 3 {
		t.Fatalf("definition state was not restored: %+v", persistedDefinition)
	}
	if len(persistedDefinition.Nodes) != 3 || len(persistedDefinition.Nodes[1].Assignees) != 1 || persistedDefinition.Nodes[1].Assignees[0] != "manager-1" {
		t.Fatalf("definition nodes and actors were not restored: %+v", persistedDefinition.Nodes)
	}
	if len(persistedDefinition.Transitions) != 2 || persistedDefinition.Transitions[1].To != "node-end" {
		t.Fatalf("definition transitions were not restored: %+v", persistedDefinition.Transitions)
	}

	persistedInstance, err := restartedRepo.GetInstance(ctx, instance.ID)
	if err != nil {
		t.Fatalf("GetInstance after restart error: %v", err)
	}
	if persistedInstance.Status != domainworkflow.InstanceApproved || persistedInstance.Starter.Name != "Alice Zhang" {
		t.Fatalf("instance state and starter were not restored: %+v", persistedInstance)
	}
	if len(persistedInstance.Tasks) != 3 || persistedInstance.Tasks[2].Assignee.Name != "Manager Two" || persistedInstance.Tasks[2].CompletedAt == nil {
		t.Fatalf("tasks were not restored: %+v", persistedInstance.Tasks)
	}
	if len(persistedInstance.Timeline) != 4 {
		t.Fatalf("expected complete action history, got %+v", persistedInstance.Timeline)
	}
	if persistedInstance.Timeline[1].Target.Name != "Quality Observer" || persistedInstance.Timeline[2].Target.Name != "Manager Two" || persistedInstance.Timeline[3].Actor.Name != "Manager Two" {
		t.Fatalf("workflow decisions and actors were not restored: %+v", persistedInstance.Timeline)
	}
}

func TestWorkflowStoreRollsBackAggregateReplacement(t *testing.T) {
	ctx := context.Background()
	db := TestDB(t)
	repo := NewWorkflowStore(db)
	now := time.Date(2026, 7, 22, 10, 0, 0, 0, time.UTC)
	instance := domainworkflow.Instance{
		ID: "instance-rollback", DefinitionID: "definition-rollback", DefinitionKey: "rollback.approval",
		BusinessType: "purchase", BusinessID: "PO-ROLLBACK", Title: "Original title", Status: domainworkflow.InstanceRunning,
		Starter: domainworkflow.Actor{ID: "employee-1", Name: "Employee One"}, CurrentNode: "node-approve",
		Tasks: []domainworkflow.Task{{
			ID: "task-rollback", InstanceID: "instance-rollback", NodeID: "node-approve",
			Assignee: domainworkflow.Actor{ID: "manager-1", Name: "Manager One"}, Status: domainworkflow.TaskPending, CreatedAt: now,
		}},
		Timeline: []domainworkflow.Action{{
			ID: "action-rollback", Type: domainworkflow.ActionStart, InstanceID: "instance-rollback", NodeID: "node-approve",
			Actor: domainworkflow.Actor{ID: "employee-1", Name: "Employee One"}, CreatedAt: now,
		}},
		Meta: shared.AuditMeta{CreatedAt: now, UpdatedAt: now},
	}
	if err := repo.SaveInstance(ctx, instance); err != nil {
		t.Fatalf("initial SaveInstance error: %v", err)
	}

	invalid := instance
	invalid.Title = "Must be rolled back"
	invalid.Meta.UpdatedAt = now.Add(time.Minute)
	invalid.Tasks = append(append([]domainworkflow.Task(nil), instance.Tasks...), instance.Tasks[0])
	if err := repo.SaveInstance(ctx, invalid); err == nil {
		t.Fatal("expected duplicate task persistence to fail")
	}

	persisted, err := repo.GetInstance(ctx, instance.ID)
	if err != nil {
		t.Fatalf("GetInstance after rollback error: %v", err)
	}
	if persisted.Title != "Original title" || len(persisted.Tasks) != 1 || persisted.Tasks[0].ID != "task-rollback" || len(persisted.Timeline) != 1 {
		t.Fatalf("failed aggregate replacement was not rolled back: %+v", persisted)
	}
}

func TestWorkflowStorePreservesRepositoryNotFoundErrors(t *testing.T) {
	repo := NewWorkflowStore(TestDB(t))
	if _, err := repo.GetDefinition(context.Background(), "missing"); err == nil || !strings.Contains(err.Error(), "workflow definition not found") {
		t.Fatalf("unexpected definition not-found error: %v", err)
	}
	if _, err := repo.GetInstance(context.Background(), "missing"); err == nil || !strings.Contains(err.Error(), "workflow instance not found") {
		t.Fatalf("unexpected instance not-found error: %v", err)
	}
}

func TestWorkflowMigrationScriptsCoverRelationalSchema(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "migrations")
	required := []string{
		"sk_workflow_definitions", "sk_workflow_nodes", "sk_workflow_node_assignees", "sk_workflow_transitions",
		"sk_workflow_instances", "sk_workflow_tasks", "sk_workflow_actions",
		"idx_workflow_definition_key_version", "idx_workflow_instance_business", "idx_workflow_task_assignee_status",
		"idx_workflow_action_instance_created", "foreign key",
	}
	for _, dialect := range []string{"mysql", "postgres"} {
		path := filepath.Join(root, dialect, "20260722_000024_create_workflow_persistence.sql")
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s workflow migration: %v", dialect, err)
		}
		text := strings.ToLower(string(body))
		for _, token := range required {
			if !strings.Contains(text, token) {
				t.Fatalf("%s workflow migration missing %q", dialect, token)
			}
		}
		if strings.Contains(text, "payload_json") {
			t.Fatalf("%s workflow migration must persist queryable relations, not aggregate JSON", dialect)
		}
	}
}

func openWorkflowTestDB(t *testing.T, dsn string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		if isSQLiteCGODisabledError(err) {
			t.Skipf("sqlite test requires cgo: %v", err)
		}
		t.Fatalf("open workflow test database: %v", err)
	}
	if err := db.AutoMigrate(
		&WorkflowDefinitionModel{}, &WorkflowNodeModel{}, &WorkflowNodeAssigneeModel{}, &WorkflowTransitionModel{},
		&WorkflowInstanceModel{}, &WorkflowTaskModel{}, &WorkflowActionModel{},
	); err != nil {
		t.Fatalf("migrate workflow test database: %v", err)
	}
	return db
}

func closeWorkflowTestDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("resolve workflow sql database: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close workflow test database: %v", err)
	}
}
