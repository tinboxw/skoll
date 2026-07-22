package pluginsdk

import (
	"errors"
	"testing"
	"time"
)

func TestDocumentWorkflowContractsValidateActionsAndBinding(t *testing.T) {
	submit := documentWorkflowContractSubmit()
	if err := submit.Validate(); err != nil {
		t.Fatalf("valid submit rejected: %v", err)
	}
	for _, action := range []DocumentWorkflowAction{DocumentWorkflowApprove, DocumentWorkflowReject, DocumentWorkflowDelegate} {
		input := DocumentWorkflowActionInput{
			TenantID: "tenant-a", Permission: submit.Permission, DocumentID: submit.Draft.ID,
			Action: action, ExpectedVersion: 1, TaskID: "task-1", IdempotencyKey: string(action) + "-1",
		}
		if action == DocumentWorkflowDelegate {
			input.Target = WorkflowActor{ID: "user-2", Name: "User Two"}
		}
		if err := input.Validate(); err != nil {
			t.Fatalf("valid %s rejected: %v", action, err)
		}
	}
	for _, action := range []DocumentWorkflowAction{DocumentWorkflowWithdraw, DocumentWorkflowCancel} {
		input := DocumentWorkflowActionInput{
			TenantID: "tenant-a", Permission: submit.Permission, DocumentID: submit.Draft.ID,
			Action: action, ExpectedVersion: 1, IdempotencyKey: string(action) + "-1",
		}
		if err := input.Validate(); err != nil {
			t.Fatalf("valid %s rejected: %v", action, err)
		}
	}

	now := time.Date(2026, 7, 22, 0, 0, 0, 0, time.UTC)
	result := DocumentWorkflowResult{
		Document: DocumentRecord{ID: submit.Draft.ID},
		Workflow: WorkflowInstance{ID: submit.InstanceID, BusinessID: submit.Draft.ID, CreatedAt: now, UpdatedAt: now},
	}
	if err := result.Validate(); err != nil {
		t.Fatalf("valid binding rejected: %v", err)
	}
	result.Workflow.BusinessID = "another-document"
	if err := result.Validate(); err == nil {
		t.Fatal("mismatched workflow business id was accepted")
	}
}

func TestDocumentWorkflowValidationReturnsStableTypedErrors(t *testing.T) {
	submit := documentWorkflowContractSubmit()
	delete(submit.Draft.Header, "subject")
	err := submit.Validate()
	var contractErr *DocumentWorkflowError
	if !errors.As(err, &contractErr) || contractErr.Code != DocumentWorkflowErrorInvalidRequest || contractErr.Field == "" {
		t.Fatalf("submit validation error=%v", err)
	}

	action := DocumentWorkflowActionInput{
		TenantID: "tenant-a", Permission: documentWorkflowContractSubmit().Permission, DocumentID: "document-1",
		Action: DocumentWorkflowDelegate, ExpectedVersion: 1, TaskID: "task-1", IdempotencyKey: "delegate-1",
	}
	err = action.Validate()
	if !errors.As(err, &contractErr) || contractErr.Code != DocumentWorkflowErrorInvalidRequest || contractErr.Field != "target.id" {
		t.Fatalf("delegate validation error=%v", err)
	}
}

func documentWorkflowContractSubmit() DocumentWorkflowSubmitInput {
	schema := DocumentSchema{
		Key: "approval_request", Name: "Approval Request", Version: 1, InitialState: "draft",
		Header: []DocumentFieldSchema{{Key: "subject", Label: "Subject", Type: DocumentFieldString, Required: true}},
		States: []DocumentStateSchema{
			{Key: "draft", Name: "Draft"}, {Key: "pending", Name: "Pending"}, {Key: "approved", Name: "Approved", Terminal: true},
		},
		Actions: []DocumentActionSchema{
			{Key: "submit", Name: "Submit", From: []string{"draft"}, To: "pending"},
			{Key: "approve", Name: "Approve", From: []string{"pending"}, To: "approved"},
		},
	}
	return DocumentWorkflowSubmitInput{
		TenantID: "tenant-a", Permission: Permission{Resource: "medical_oa.approval_request", Action: "manage"}, Schema: schema,
		Draft: DocumentDraft{
			ID: "document-1", Type: schema.Key, SchemaVersion: schema.Version, Number: "OA-000001", Title: "Approval",
			Header: map[string]DocumentValue{"subject": {Type: DocumentFieldString, Value: "Purchase"}}, Lines: map[string][]DocumentLine{},
		},
		DefinitionID: "approval", InstanceID: "instance-1", IdempotencyKey: "submit-1",
	}
}
