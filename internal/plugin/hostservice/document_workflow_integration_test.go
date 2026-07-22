package hostservice

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	documentworkflowsvc "github.com/tinboxw/skoll/internal/service/documentworkflow"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"github.com/tinboxw/skoll/internal/store/sql/gormrepo"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"github.com/tinboxw/skoll/pkg/security"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type documentWorkflowFixture struct {
	documents pluginsdk.DocumentWorkflowService
	workflows pluginsdk.WorkflowService
	tx        pluginsdk.TransactionService
	audit     *documentWorkflowAudit
}

type documentWorkflowAudit struct {
	mu      sync.Mutex
	entries []pluginsdk.AuditEntry
}

func (a *documentWorkflowAudit) Record(_ context.Context, entry pluginsdk.AuditEntry) (pluginsdk.AuditReceipt, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.entries = append(a.entries, entry)
	return pluginsdk.AuditReceipt{ID: "audit"}, nil
}

func TestDocumentWorkflowActionsAndIdempotentDecision(t *testing.T) {
	fixture := newDocumentWorkflowFixture(t)
	user1 := documentWorkflowUserContext("user-1")
	user2 := documentWorkflowUserContext("user-2")

	tests := []struct {
		name   string
		action pluginsdk.DocumentWorkflowAction
		status pluginsdk.WorkflowInstanceStatus
		state  string
	}{
		{name: "approve", action: pluginsdk.DocumentWorkflowApprove, status: pluginsdk.WorkflowInstanceApproved, state: "approved"},
		{name: "reject", action: pluginsdk.DocumentWorkflowReject, status: pluginsdk.WorkflowInstanceRejected, state: "rejected"},
		{name: "withdraw", action: pluginsdk.DocumentWorkflowWithdraw, status: pluginsdk.WorkflowInstanceWithdrawn, state: "withdrawn"},
		{name: "cancel", action: pluginsdk.DocumentWorkflowCancel, status: pluginsdk.WorkflowInstanceCanceled, state: "canceled"},
	}
	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			id := test.name + "-document"
			submitted := submitDocument(t, fixture, user1, id, "instance-"+test.name)
			input := documentWorkflowActionInput(id, test.action, submitted)
			var decided pluginsdk.DocumentWorkflowResult
			if err := fixture.tx.Within(user1, func(tx pluginsdk.Transaction) error {
				var err error
				decided, err = fixture.documents.Act(tx.Context(), input)
				return err
			}); err != nil {
				t.Fatalf("action %d error: %v", index, err)
			}
			if decided.Document.State != test.state || decided.Document.Version != 2 || decided.Workflow.Status != test.status {
				t.Fatalf("unexpected decision result: %+v", decided)
			}
			if test.action == pluginsdk.DocumentWorkflowApprove {
				var duplicate pluginsdk.DocumentWorkflowResult
				if err := fixture.tx.Within(user1, func(tx pluginsdk.Transaction) error {
					var err error
					duplicate, err = fixture.documents.Act(tx.Context(), input)
					return err
				}); err != nil {
					t.Fatalf("duplicate approval error: %v", err)
				}
				if !duplicate.Duplicate || duplicate.Document.Version != decided.Document.Version || duplicate.Workflow.Status != decided.Workflow.Status {
					t.Fatalf("duplicate decision was not replayed: %+v", duplicate)
				}
			}
		})
	}

	submitted := submitDocument(t, fixture, user1, "delegate-document", "instance-delegate")
	taskID := pendingTaskID(t, submitted.Workflow)
	delegate := documentWorkflowActionInput("delegate-document", pluginsdk.DocumentWorkflowDelegate, submitted)
	delegate.TaskID = taskID
	delegate.Target = pluginsdk.WorkflowActor{ID: "user-2", Name: "User Two"}
	var delegated pluginsdk.DocumentWorkflowResult
	if err := fixture.tx.Within(user1, func(tx pluginsdk.Transaction) error {
		var err error
		delegated, err = fixture.documents.Act(tx.Context(), delegate)
		return err
	}); err != nil {
		t.Fatalf("delegate error: %v", err)
	}
	if delegated.Document.State != "pending" || delegated.Document.Version != 1 {
		t.Fatalf("delegation must not transition the document: %+v", delegated.Document)
	}
	approve := documentWorkflowActionInput("delegate-document", pluginsdk.DocumentWorkflowApprove, delegated)
	if err := fixture.tx.Within(user2, func(tx pluginsdk.Transaction) error {
		_, err := fixture.documents.Act(tx.Context(), approve)
		return err
	}); err != nil {
		t.Fatalf("delegated approval error: %v", err)
	}
}

func TestDocumentWorkflowRollsBackDocumentAndWorkflowTogether(t *testing.T) {
	fixture := newDocumentWorkflowFixture(t)
	ctx := documentWorkflowUserContext("user-1")
	rollback := errors.New("rollback acceptance transaction")
	input := documentWorkflowSubmitInput("rollback-document", "instance-rollback")
	if err := fixture.tx.Within(ctx, func(tx pluginsdk.Transaction) error {
		if _, err := fixture.documents.Submit(tx.Context(), input); err != nil {
			return err
		}
		return rollback
	}); !errors.Is(err, rollback) {
		t.Fatalf("rollback error=%v", err)
	}
	_, err := fixture.documents.Get(ctx, pluginsdk.DocumentWorkflowGetInput{TenantID: "tenant-a", Permission: documentWorkflowPermission(), DocumentID: input.Draft.ID})
	var publicErr *pluginsdk.DocumentWorkflowError
	if !errors.As(err, &publicErr) || publicErr.Code != pluginsdk.DocumentWorkflowErrorNotFound {
		t.Fatalf("rolled back document is visible: %v", err)
	}
	if _, err = fixture.workflows.GetInstance(ctx, input.InstanceID); err == nil {
		t.Fatal("rolled back workflow instance is visible")
	}

	submitted := submitDocument(t, fixture, ctx, "action-rollback-document", "instance-action-rollback")
	action := documentWorkflowActionInput(submitted.Document.ID, pluginsdk.DocumentWorkflowApprove, submitted)
	if err = fixture.tx.Within(ctx, func(tx pluginsdk.Transaction) error {
		if _, actionErr := fixture.documents.Act(tx.Context(), action); actionErr != nil {
			return actionErr
		}
		return rollback
	}); !errors.Is(err, rollback) {
		t.Fatalf("action rollback error=%v", err)
	}
	persisted, err := fixture.documents.Get(ctx, pluginsdk.DocumentWorkflowGetInput{TenantID: "tenant-a", Permission: documentWorkflowPermission(), DocumentID: submitted.Document.ID})
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Document.State != "pending" || persisted.Document.Version != 1 || persisted.Workflow.Status != pluginsdk.WorkflowInstanceRunning {
		t.Fatalf("action rollback split document and workflow state: %+v", persisted)
	}
}

func TestDocumentWorkflowRequiresTransactionAndTenantScope(t *testing.T) {
	fixture := newDocumentWorkflowFixture(t)
	ctx := documentWorkflowUserContext("user-1")
	_, err := fixture.documents.Submit(ctx, documentWorkflowSubmitInput("no-transaction", "instance-no-transaction"))
	var publicErr *pluginsdk.DocumentWorkflowError
	if !errors.As(err, &publicErr) || publicErr.Code != pluginsdk.DocumentWorkflowErrorTransactionRequired {
		t.Fatalf("missing transaction error=%v", err)
	}
	denied := documentWorkflowSubmitInput("denied", "instance-denied")
	denied.TenantID = "tenant-b"
	if err = fixture.tx.Within(ctx, func(tx pluginsdk.Transaction) error {
		_, submitErr := fixture.documents.Submit(tx.Context(), denied)
		return submitErr
	}); !errors.As(err, &publicErr) || publicErr.Code != pluginsdk.DocumentWorkflowErrorForbidden {
		t.Fatalf("tenant denial error=%v", err)
	}
}

func TestDocumentWorkflowInvalidDocumentActionCannotAdvanceWorkflow(t *testing.T) {
	fixture := newDocumentWorkflowFixture(t)
	ctx := documentWorkflowUserContext("user-1")
	input := documentWorkflowSubmitInput("invalid-action-document", "instance-invalid-action")
	input.Schema.Actions = []pluginsdk.DocumentActionSchema{
		{Key: "submit", Name: "Submit", From: []string{"draft"}, To: "pending"},
		{Key: "reject", Name: "Reject", From: []string{"pending"}, To: "rejected"},
	}
	var submitted pluginsdk.DocumentWorkflowResult
	if err := fixture.tx.Within(ctx, func(tx pluginsdk.Transaction) error {
		var err error
		submitted, err = fixture.documents.Submit(tx.Context(), input)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	action := documentWorkflowActionInput(input.Draft.ID, pluginsdk.DocumentWorkflowApprove, submitted)
	if err := fixture.tx.Within(ctx, func(tx pluginsdk.Transaction) error {
		if _, actionErr := fixture.documents.Act(tx.Context(), action); actionErr == nil {
			t.Fatal("undeclared document action was accepted")
		}
		// Deliberately swallow the service error. The service must validate before
		// any workflow mutation instead of relying on the caller to roll back.
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	persisted, err := fixture.documents.Get(ctx, pluginsdk.DocumentWorkflowGetInput{
		TenantID: "tenant-a", Permission: documentWorkflowPermission(), DocumentID: input.Draft.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Document.State != "pending" || persisted.Document.Version != 1 || persisted.Workflow.Status != pluginsdk.WorkflowInstanceRunning {
		t.Fatalf("invalid action advanced persisted state: %+v", persisted)
	}
}

func newDocumentWorkflowFixture(t *testing.T) documentWorkflowFixture {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "document-workflow.db") + "?_busy_timeout=5000&_journal_mode=WAL"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent), TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(8)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err = db.AutoMigrate(
		&gormrepo.WorkflowDefinitionModel{}, &gormrepo.WorkflowNodeModel{}, &gormrepo.WorkflowNodeAssigneeModel{}, &gormrepo.WorkflowTransitionModel{},
		&gormrepo.WorkflowInstanceModel{}, &gormrepo.WorkflowTaskModel{}, &gormrepo.WorkflowActionModel{},
		&gormrepo.DocumentWorkflowBindingModel{}, &gormrepo.DocumentWorkflowActionModel{},
	); err != nil {
		t.Fatal(err)
	}
	audit := &documentWorkflowAudit{}
	workflow, err := NewWorkflowService("medical_oa", workflowsvc.NewService(gormrepo.NewWorkflowStore(db)), audit)
	if err != nil {
		t.Fatal(err)
	}
	documents, err := NewDocumentWorkflowService(
		"medical_oa", documentworkflowsvc.NewService(gormrepo.NewDocumentWorkflowStore(db), workflow),
		documentNumberTestScopes{tenantID: "tenant-a"}, audit,
	)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := NewTransactionService(storesql.NewUnitOfWorkWithDB(db))
	if err != nil {
		t.Fatal(err)
	}
	ctx := documentWorkflowUserContext("user-1")
	definition, err := workflow.CreateDefinition(ctx, pluginsdk.WorkflowDefinitionInput{
		ID: "document-approval", Key: "document_approval", Name: "Document Approval", Version: 1,
		Nodes: []pluginsdk.WorkflowNode{
			{ID: "start", Key: "start", Name: "Start", Type: pluginsdk.WorkflowNodeStart},
			{ID: "approval", Key: "approval", Name: "Approval", Type: pluginsdk.WorkflowNodeApproval, AssigneeIDs: []string{"user-1"}},
			{ID: "end", Key: "end", Name: "End", Type: pluginsdk.WorkflowNodeEnd},
		},
		Transitions: []pluginsdk.WorkflowTransition{{From: "start", To: "approval"}, {From: "approval", To: "end"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = workflow.PublishDefinition(ctx, definition.ID); err != nil {
		t.Fatal(err)
	}
	return documentWorkflowFixture{documents: documents, workflows: workflow, tx: tx, audit: audit}
}

func submitDocument(t *testing.T, fixture documentWorkflowFixture, ctx context.Context, documentID, instanceID string) pluginsdk.DocumentWorkflowResult {
	t.Helper()
	var result pluginsdk.DocumentWorkflowResult
	if err := fixture.tx.Within(ctx, func(tx pluginsdk.Transaction) error {
		var err error
		result, err = fixture.documents.Submit(tx.Context(), documentWorkflowSubmitInput(documentID, instanceID))
		return err
	}); err != nil {
		t.Fatalf("submit %s: %v", documentID, err)
	}
	return result
}

func documentWorkflowSubmitInput(documentID, instanceID string) pluginsdk.DocumentWorkflowSubmitInput {
	return pluginsdk.DocumentWorkflowSubmitInput{
		TenantID: "tenant-a", Permission: documentWorkflowPermission(), Schema: documentWorkflowSchema(),
		Draft: pluginsdk.DocumentDraft{
			ID: documentID, Type: "approval_request", SchemaVersion: 1, Number: "OA-" + documentID, Title: "Approval " + documentID,
			Header: map[string]pluginsdk.DocumentValue{"subject": {Type: pluginsdk.DocumentFieldString, Value: "Purchase approval"}},
			Lines:  map[string][]pluginsdk.DocumentLine{}, Tags: []string{"oa"},
		},
		DefinitionID: "document-approval", InstanceID: instanceID, IdempotencyKey: "submit-" + documentID,
	}
}

func documentWorkflowActionInput(documentID string, action pluginsdk.DocumentWorkflowAction, current pluginsdk.DocumentWorkflowResult) pluginsdk.DocumentWorkflowActionInput {
	input := pluginsdk.DocumentWorkflowActionInput{
		TenantID: "tenant-a", Permission: documentWorkflowPermission(), DocumentID: documentID,
		Action: action, ExpectedVersion: current.Document.Version, IdempotencyKey: string(action) + "-" + documentID,
	}
	if action == pluginsdk.DocumentWorkflowApprove || action == pluginsdk.DocumentWorkflowReject || action == pluginsdk.DocumentWorkflowDelegate {
		input.TaskID = pendingTaskIDValue(current.Workflow)
	}
	return input
}

func pendingTaskID(t *testing.T, workflow pluginsdk.WorkflowInstance) string {
	t.Helper()
	value := pendingTaskIDValue(workflow)
	if value == "" {
		t.Fatalf("workflow has no pending task: %+v", workflow)
	}
	return value
}

func pendingTaskIDValue(workflow pluginsdk.WorkflowInstance) string {
	for _, task := range workflow.Tasks {
		if task.Status == pluginsdk.WorkflowTaskPending {
			return task.ID
		}
	}
	return ""
}

func documentWorkflowSchema() pluginsdk.DocumentSchema {
	return pluginsdk.DocumentSchema{
		Key: "approval_request", Name: "Approval Request", Version: 1, InitialState: "draft",
		Header: []pluginsdk.DocumentFieldSchema{{Key: "subject", Label: "Subject", Type: pluginsdk.DocumentFieldString, Required: true}},
		States: []pluginsdk.DocumentStateSchema{
			{Key: "draft", Name: "Draft"}, {Key: "pending", Name: "Pending"},
			{Key: "approved", Name: "Approved", Terminal: true}, {Key: "rejected", Name: "Rejected", Terminal: true},
			{Key: "withdrawn", Name: "Withdrawn", Terminal: true}, {Key: "canceled", Name: "Canceled", Terminal: true},
		},
		Actions: []pluginsdk.DocumentActionSchema{
			{Key: "submit", Name: "Submit", From: []string{"draft"}, To: "pending"},
			{Key: "approve", Name: "Approve", From: []string{"pending"}, To: "approved"},
			{Key: "reject", Name: "Reject", From: []string{"pending"}, To: "rejected"},
			{Key: "withdraw", Name: "Withdraw", From: []string{"pending"}, To: "withdrawn"},
			{Key: "cancel", Name: "Cancel", From: []string{"pending"}, To: "canceled"},
		},
	}
}

func documentWorkflowPermission() pluginsdk.Permission {
	return pluginsdk.Permission{Resource: "medical_oa.approval_request", Action: "manage"}
}

func documentWorkflowUserContext(subject string) context.Context {
	return security.WithJWTClaimsContext(context.Background(), &security.JWTClaims{Subject: subject, Role: "employee", Roles: []string{"employee"}})
}
