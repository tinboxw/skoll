package sdkconformance

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

type Report struct {
	Transaction   bool
	Scope         bool
	DataStore     bool
	Document      bool
	Collaboration bool
	File          bool
	Audit         bool
	Config        bool
	Secret        bool
	Workflow      bool
	Job           bool
}

func Run(ctx context.Context, host pluginsdk.HostServices) (Report, error) {
	if err := host.Validate(); err != nil {
		return Report{}, err
	}
	report := Report{}
	if err := host.Transactions.Within(ctx, func(tx pluginsdk.Transaction) error {
		if tx.Context() == nil {
			return fmt.Errorf("transaction context is missing")
		}
		report.Transaction = true
		return nil
	}); err != nil {
		return report, fmt.Errorf("transaction contract: %w", err)
	}
	scope, err := host.DataScopes.Resolve(ctx, pluginsdk.Permission{Resource: "conformance.record", Action: "read"})
	if err != nil || scope.Denied() {
		return report, fmt.Errorf("scope contract: %w", err)
	}
	report.Scope = true
	mutation := pluginsdk.DataMutation{
		Table: "records", Operation: pluginsdk.DataMutationInsert,
		Scope:     pluginsdk.DataScopeIntent{Permission: pluginsdk.Permission{Resource: "sdk_conformance.record", Action: "write"}},
		Key:       map[string]pluginsdk.DataValue{"id": {Type: pluginsdk.DataValueString, Value: "record-1"}},
		Values:    map[string]pluginsdk.DataValue{"name": {Type: pluginsdk.DataValueString, Value: "SDK record"}},
		Returning: []string{"id", "name"}, IdempotencyKey: "create-record-1",
	}
	mutated, err := host.DataStore.Mutate(ctx, mutation)
	if err != nil {
		return report, fmt.Errorf("datastore mutation contract: %w", err)
	}
	if mutated.RowsAffected != 1 || mutated.Record == nil || mutated.Record.Version != 1 {
		return report, fmt.Errorf("datastore mutation contract returned an invalid result")
	}
	page, err := host.DataStore.Query(ctx, pluginsdk.DataQuery{
		Table: "records", Fields: []string{"id", "name"},
		Scope: pluginsdk.DataScopeIntent{Permission: pluginsdk.Permission{Resource: "sdk_conformance.record", Action: "read"}},
		Sort:  []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 10},
	})
	if err != nil {
		return report, fmt.Errorf("datastore query contract: %w", err)
	}
	if len(page.Records) != 1 || page.Records[0].Values["name"].Value != "SDK record" {
		return report, fmt.Errorf("datastore query contract returned an invalid page")
	}
	report.DataStore = true
	file, err := host.Files.Store(ctx, pluginsdk.FileWrite{Key: "evidence/contract.txt", Name: "contract.txt", Content: []byte("sdk-conformance")})
	if err != nil {
		return report, fmt.Errorf("file store contract: %w", err)
	}
	if _, err = host.Files.Get(ctx, file.ID); err != nil {
		return report, fmt.Errorf("file get contract: %w", err)
	}
	report.File = true
	if _, err = host.Audit.Record(ctx, pluginsdk.AuditEntry{Action: "conformance.run", Resource: "contract", ResourceID: "sdk", Detail: map[string]any{"token": "must-redact"}}); err != nil {
		return report, fmt.Errorf("audit contract: %w", err)
	}
	report.Audit = true
	if _, err = host.Config.Replace(ctx, map[string]any{"mode": "strict"}); err != nil {
		return report, fmt.Errorf("config contract: %w", err)
	}
	config, err := host.Config.Get(ctx)
	if err != nil || config["mode"] != "strict" {
		return report, fmt.Errorf("config read contract: %w", err)
	}
	report.Config = true
	if err = host.Secrets.Set(ctx, "remote.api_key", "conformance-secret"); err != nil {
		return report, fmt.Errorf("secret set contract: %w", err)
	}
	secret, err := host.Secrets.Get(ctx, "remote.api_key")
	if err != nil || secret != "conformance-secret" {
		return report, fmt.Errorf("secret get contract: %w", err)
	}
	report.Secret = true
	definition, err := host.Workflows.CreateDefinition(ctx, pluginsdk.WorkflowDefinitionInput{
		ID: "approval", Key: "approval", Name: "SDK approval", Version: 1,
		Nodes: []pluginsdk.WorkflowNode{
			{ID: "start", Key: "start", Name: "Start", Type: pluginsdk.WorkflowNodeStart},
			{ID: "review", Key: "review", Name: "Review", Type: pluginsdk.WorkflowNodeApproval, AssigneeIDs: []string{"conformance-user"}},
			{ID: "end", Key: "end", Name: "End", Type: pluginsdk.WorkflowNodeEnd},
		},
		Transitions: []pluginsdk.WorkflowTransition{{From: "start", To: "review"}, {From: "review", To: "end"}},
	})
	if err != nil {
		return report, fmt.Errorf("workflow definition contract: %w", err)
	}
	if _, err = host.Workflows.PublishDefinition(ctx, definition.ID); err != nil {
		return report, fmt.Errorf("workflow publish contract: %w", err)
	}
	instance, err := host.Workflows.Start(ctx, pluginsdk.WorkflowStartInput{
		ID: "request", DefinitionID: definition.ID, BusinessType: "request", BusinessID: "request-1", Title: "SDK request",
	})
	if err != nil || len(instance.Tasks) != 1 {
		return report, fmt.Errorf("workflow start contract: %w", err)
	}
	instance, err = host.Workflows.Approve(ctx, pluginsdk.WorkflowTaskActionInput{InstanceID: instance.ID, TaskID: instance.Tasks[0].ID})
	if err != nil || instance.Status != pluginsdk.WorkflowInstanceApproved {
		return report, fmt.Errorf("workflow approve contract: %w", err)
	}
	report.Workflow = true
	documentInput := pluginsdk.DocumentWorkflowSubmitInput{
		TenantID: "tenant-conformance", Permission: pluginsdk.Permission{Resource: "sdk_conformance.approval_request", Action: "manage"},
		Schema: pluginsdk.DocumentSchema{
			Key: "approval_request", Name: "Approval Request", Version: 1, InitialState: "draft",
			Header: []pluginsdk.DocumentFieldSchema{{Key: "subject", Label: "Subject", Type: pluginsdk.DocumentFieldString, Required: true}},
			States: []pluginsdk.DocumentStateSchema{
				{Key: "draft", Name: "Draft"}, {Key: "pending", Name: "Pending"}, {Key: "approved", Name: "Approved", Terminal: true},
			},
			Actions: []pluginsdk.DocumentActionSchema{
				{Key: "submit", Name: "Submit", From: []string{"draft"}, To: "pending"},
				{Key: "approve", Name: "Approve", From: []string{"pending"}, To: "approved"},
			},
		},
		Draft: pluginsdk.DocumentDraft{
			ID: "approval-document", Type: "approval_request", SchemaVersion: 1, Number: "OA-000001", Title: "SDK document approval",
			Header: map[string]pluginsdk.DocumentValue{"subject": {Type: pluginsdk.DocumentFieldString, Value: "SDK conformance"}},
			Lines:  map[string][]pluginsdk.DocumentLine{},
		},
		DefinitionID: definition.ID, InstanceID: "document-request", IdempotencyKey: "submit-approval-document",
	}
	var document pluginsdk.DocumentWorkflowResult
	if err = host.Transactions.Within(ctx, func(tx pluginsdk.Transaction) error {
		document, err = host.Documents.Submit(tx.Context(), documentInput)
		return err
	}); err != nil || len(document.Workflow.Tasks) != 1 {
		return report, fmt.Errorf("document submit contract: %w", err)
	}
	if err = host.Transactions.Within(ctx, func(tx pluginsdk.Transaction) error {
		if _, err = host.Documents.AddAttachment(tx.Context(), pluginsdk.DocumentAttachmentAddInput{
			TenantID: "tenant-conformance", Permission: documentInput.Permission, DocumentID: document.Document.ID,
			AttachmentID: "evidence-1", FileID: file.ID,
		}); err != nil {
			return err
		}
		_, err = host.Documents.AddComment(tx.Context(), pluginsdk.DocumentCommentAddInput{
			TenantID: "tenant-conformance", Permission: documentInput.Permission, DocumentID: document.Document.ID,
			CommentID: "review-note-1", Body: "Evidence received.",
		})
		return err
	}); err != nil {
		return report, fmt.Errorf("document collaboration write contract: %w", err)
	}
	if err = host.Transactions.Within(ctx, func(tx pluginsdk.Transaction) error {
		document, err = host.Documents.Act(tx.Context(), pluginsdk.DocumentWorkflowActionInput{
			TenantID: "tenant-conformance", Permission: documentInput.Permission, DocumentID: document.Document.ID,
			Action: pluginsdk.DocumentWorkflowApprove, ExpectedVersion: document.Document.Version,
			TaskID: document.Workflow.Tasks[0].ID, IdempotencyKey: "approve-approval-document",
		})
		return err
	}); err != nil || document.Document.State != "approved" || document.Workflow.Status != pluginsdk.WorkflowInstanceApproved {
		return report, fmt.Errorf("document approval contract: %w", err)
	}
	attachments, err := host.Documents.ListAttachments(ctx, pluginsdk.DocumentCollaborationQueryInput{
		TenantID: "tenant-conformance", Permission: documentInput.Permission, DocumentID: document.Document.ID,
	})
	if err != nil || len(attachments) != 1 || attachments[0].File.ID != file.ID {
		return report, fmt.Errorf("document attachment read contract: %w", err)
	}
	comments, err := host.Documents.ListComments(ctx, pluginsdk.DocumentCollaborationQueryInput{
		TenantID: "tenant-conformance", Permission: documentInput.Permission, DocumentID: document.Document.ID,
	})
	if err != nil || len(comments) != 1 || comments[0].Body != "Evidence received." {
		return report, fmt.Errorf("document comment read contract: %w", err)
	}
	timeline, err := host.Documents.Timeline(ctx, pluginsdk.DocumentTimelineQueryInput{
		TenantID: "tenant-conformance", Permission: documentInput.Permission, DocumentID: document.Document.ID,
	})
	if err != nil || len(timeline.Events) != 4 || timeline.NextSequence != 4 {
		return report, fmt.Errorf("document timeline contract: %w", err)
	}
	report.Collaboration = true
	report.Document = true
	job, err := host.Jobs.Schedule(ctx, pluginsdk.JobScheduleInput{
		ID: "sync", Kind: "sync", IdempotencyKey: "sync-1", Payload: json.RawMessage(`{"mode":"full"}`), MaxAttempts: 1,
	})
	if err != nil {
		return report, fmt.Errorf("job schedule contract: %w", err)
	}
	leased, err := host.Jobs.LeaseDue(ctx, pluginsdk.JobLeaseInput{WorkerID: "worker", Limit: 1, LeaseDuration: time.Minute})
	if err != nil || len(leased) != 1 || leased[0].ID != job.ID {
		return report, fmt.Errorf("job lease contract: %w", err)
	}
	completed, err := host.Jobs.Complete(ctx, pluginsdk.JobCompleteInput{JobID: job.ID, LeaseToken: leased[0].LeaseToken, Result: json.RawMessage(`{"ok":true}`)})
	if err != nil || completed.Status != pluginsdk.JobStatusSucceeded {
		return report, fmt.Errorf("job complete contract: %w", err)
	}
	report.Job = true
	return report, nil
}
