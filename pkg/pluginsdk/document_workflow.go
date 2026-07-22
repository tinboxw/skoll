package pluginsdk

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const MaxDocumentWorkflowIdempotencyKey = 128

type DocumentDraft struct {
	ID            string                    `json:"id"`
	Type          string                    `json:"type"`
	SchemaVersion int                       `json:"schemaVersion"`
	Number        string                    `json:"number"`
	Title         string                    `json:"title"`
	Header        map[string]DocumentValue  `json:"header"`
	Lines         map[string][]DocumentLine `json:"lines"`
	Tags          []string                  `json:"tags,omitempty"`
}

type DocumentWorkflowSubmitInput struct {
	TenantID       string         `json:"tenantId"`
	Permission     Permission     `json:"permission"`
	Schema         DocumentSchema `json:"schema"`
	Draft          DocumentDraft  `json:"draft"`
	DefinitionID   string         `json:"definitionId"`
	InstanceID     string         `json:"instanceId"`
	Comment        string         `json:"comment,omitempty"`
	IdempotencyKey string         `json:"idempotencyKey"`
}

type DocumentWorkflowAction string

const (
	DocumentWorkflowWithdraw DocumentWorkflowAction = "withdraw"
	DocumentWorkflowApprove  DocumentWorkflowAction = "approve"
	DocumentWorkflowReject   DocumentWorkflowAction = "reject"
	DocumentWorkflowDelegate DocumentWorkflowAction = "delegate"
	DocumentWorkflowCancel   DocumentWorkflowAction = "cancel"
)

type DocumentWorkflowActionInput struct {
	TenantID        string                 `json:"tenantId"`
	Permission      Permission             `json:"permission"`
	DocumentID      string                 `json:"documentId"`
	Action          DocumentWorkflowAction `json:"action"`
	ExpectedVersion int64                  `json:"expectedVersion"`
	TaskID          string                 `json:"taskId,omitempty"`
	Target          WorkflowActor          `json:"target,omitempty"`
	Comment         string                 `json:"comment,omitempty"`
	IdempotencyKey  string                 `json:"idempotencyKey"`
}

type DocumentWorkflowGetInput struct {
	TenantID   string     `json:"tenantId"`
	Permission Permission `json:"permission"`
	DocumentID string     `json:"documentId"`
}

type DocumentWorkflowResult struct {
	Document  DocumentRecord   `json:"document"`
	Workflow  WorkflowInstance `json:"workflow"`
	Duplicate bool             `json:"duplicate"`
}

type DocumentWorkflowService interface {
	Submit(ctx context.Context, input DocumentWorkflowSubmitInput) (DocumentWorkflowResult, error)
	Act(ctx context.Context, input DocumentWorkflowActionInput) (DocumentWorkflowResult, error)
	Get(ctx context.Context, input DocumentWorkflowGetInput) (DocumentWorkflowResult, error)
}

func (in DocumentWorkflowSubmitInput) Validate() error {
	return documentWorkflowValidationError(in.validate())
}

func (in DocumentWorkflowSubmitInput) validate() error {
	if err := validateDocumentWorkflowScope(in.TenantID, in.Permission); err != nil {
		return err
	}
	if err := in.Schema.Validate(); err != nil {
		return err
	}
	if err := validateDocumentID("draft.id", in.Draft.ID); err != nil {
		return err
	}
	if err := validateDocumentID("definitionId", in.DefinitionID); err != nil {
		return err
	}
	if err := validateDocumentID("instanceId", in.InstanceID); err != nil {
		return err
	}
	if err := validateDocumentWorkflowKey(in.IdempotencyKey); err != nil {
		return err
	}
	if err := validateDocumentBoundedText("comment", in.Comment, 4096, false); err != nil {
		return err
	}
	now := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	record := DocumentRecord{
		ID: in.Draft.ID, Type: in.Draft.Type, SchemaVersion: in.Draft.SchemaVersion,
		Number: in.Draft.Number, Title: in.Draft.Title, State: in.Schema.InitialState, Version: 1,
		Header: in.Draft.Header, Lines: in.Draft.Lines,
		Metadata: DocumentMetadata{CreatedAt: now, UpdatedAt: now, CreatedBy: "validation", UpdatedBy: "validation", Tags: in.Draft.Tags},
	}
	return in.Schema.ValidateRecord(record)
}

func (in DocumentWorkflowActionInput) Validate() error {
	return documentWorkflowValidationError(in.validate())
}

func (in DocumentWorkflowActionInput) validate() error {
	if err := validateDocumentWorkflowScope(in.TenantID, in.Permission); err != nil {
		return err
	}
	if err := validateDocumentID("documentId", in.DocumentID); err != nil {
		return err
	}
	if in.ExpectedVersion < 1 {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "expectedVersion", "expected version must be positive", false)
	}
	if err := validateDocumentWorkflowKey(in.IdempotencyKey); err != nil {
		return err
	}
	if err := validateDocumentBoundedText("comment", in.Comment, 4096, false); err != nil {
		return err
	}
	switch in.Action {
	case DocumentWorkflowApprove, DocumentWorkflowReject:
		if err := validateDocumentID("taskId", in.TaskID); err != nil {
			return err
		}
		if strings.TrimSpace(in.Target.ID) != "" || strings.TrimSpace(in.Target.Name) != "" {
			return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "target", "target is only valid for delegate", false)
		}
	case DocumentWorkflowDelegate:
		if err := validateDocumentID("taskId", in.TaskID); err != nil {
			return err
		}
		if err := validateDocumentID("target.id", in.Target.ID); err != nil {
			return err
		}
		if err := validateDocumentBoundedText("target.name", in.Target.Name, 256, false); err != nil {
			return err
		}
	case DocumentWorkflowWithdraw, DocumentWorkflowCancel:
		if strings.TrimSpace(in.TaskID) != "" || strings.TrimSpace(in.Target.ID) != "" || strings.TrimSpace(in.Target.Name) != "" {
			return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "taskId", "task and target are not valid for this action", false)
		}
	default:
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "action", "document workflow action is unsupported", false)
	}
	return nil
}

func (in DocumentWorkflowGetInput) Validate() error {
	return documentWorkflowValidationError(in.validate())
}

func (in DocumentWorkflowGetInput) validate() error {
	if err := validateDocumentWorkflowScope(in.TenantID, in.Permission); err != nil {
		return err
	}
	return validateDocumentID("documentId", in.DocumentID)
}

func documentWorkflowValidationError(err error) error {
	if err == nil {
		return nil
	}
	var workflowErr *DocumentWorkflowError
	if errors.As(err, &workflowErr) {
		return err
	}
	var contractErr *DocumentContractError
	if errors.As(err, &contractErr) {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, contractErr.Field, contractErr.Message, false)
	}
	return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "", err.Error(), false)
}

func (r DocumentWorkflowResult) Validate() error {
	if err := validateDocumentID("document.id", r.Document.ID); err != nil {
		return err
	}
	if strings.TrimSpace(r.Workflow.ID) == "" || strings.TrimSpace(r.Workflow.BusinessID) != r.Document.ID {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "workflow", "workflow is not bound to the document", false)
	}
	return nil
}

type DocumentWorkflowErrorCode string

const (
	DocumentWorkflowErrorInvalidRequest      DocumentWorkflowErrorCode = "invalid_request"
	DocumentWorkflowErrorForbidden           DocumentWorkflowErrorCode = "forbidden"
	DocumentWorkflowErrorNotFound            DocumentWorkflowErrorCode = "not_found"
	DocumentWorkflowErrorConflict            DocumentWorkflowErrorCode = "conflict"
	DocumentWorkflowErrorTransactionRequired DocumentWorkflowErrorCode = "transaction_required"
	DocumentWorkflowErrorUnavailable         DocumentWorkflowErrorCode = "unavailable"
)

type DocumentWorkflowError struct {
	Code      DocumentWorkflowErrorCode `json:"code"`
	Field     string                    `json:"field,omitempty"`
	Message   string                    `json:"message"`
	Retryable bool                      `json:"retryable"`
}

func (e *DocumentWorkflowError) Error() string {
	if e == nil {
		return "document workflow operation failed"
	}
	if e.Field == "" {
		return fmt.Sprintf("document workflow operation failed: %s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("document workflow operation failed: %s: %s: %s", e.Code, e.Field, e.Message)
}

func NewDocumentWorkflowError(code DocumentWorkflowErrorCode, field, message string, retryable bool) *DocumentWorkflowError {
	return &DocumentWorkflowError{Code: code, Field: strings.TrimSpace(field), Message: strings.TrimSpace(message), Retryable: retryable}
}

func validateDocumentWorkflowScope(tenantID string, permission Permission) error {
	if !documentNumberTenantPattern.MatchString(tenantID) {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "tenantId", "tenant id must be a bounded identifier", false)
	}
	if err := (DataScopeIntent{Permission: permission}).Validate(); err != nil {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "permission", "permission is invalid", false)
	}
	return nil
}

func validateDocumentWorkflowKey(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || trimmed != value || len(trimmed) > MaxDocumentWorkflowIdempotencyKey || strings.ContainsAny(trimmed, "\r\n\t") {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "idempotencyKey", "idempotency key is invalid", false)
	}
	return nil
}
