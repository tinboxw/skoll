package hostservice

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	documentworkflowsvc "github.com/tinboxw/skoll/internal/service/documentworkflow"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

type documentWorkflowBackend interface {
	Submit(context.Context, string, pluginsdk.WorkflowActor, pluginsdk.DocumentWorkflowSubmitInput) (pluginsdk.DocumentWorkflowResult, error)
	Act(context.Context, string, pluginsdk.WorkflowActor, pluginsdk.DocumentWorkflowActionInput) (pluginsdk.DocumentWorkflowResult, error)
	Get(context.Context, string, pluginsdk.DocumentWorkflowGetInput) (pluginsdk.DocumentWorkflowResult, error)
	AddAttachment(context.Context, string, pluginsdk.WorkflowActor, pluginsdk.DocumentAttachmentAddInput, pluginsdk.FileObject) (pluginsdk.DocumentAttachmentResult, error)
	RemoveAttachment(context.Context, string, pluginsdk.WorkflowActor, pluginsdk.DocumentAttachmentRemoveInput) (pluginsdk.DocumentAttachmentResult, error)
	ListAttachments(context.Context, string, pluginsdk.DocumentCollaborationQueryInput) ([]pluginsdk.DocumentAttachment, error)
	AddComment(context.Context, string, pluginsdk.WorkflowActor, pluginsdk.DocumentCommentAddInput) (pluginsdk.DocumentCommentResult, error)
	ListComments(context.Context, string, pluginsdk.DocumentCollaborationQueryInput) ([]pluginsdk.DocumentComment, error)
	Timeline(context.Context, string, pluginsdk.DocumentTimelineQueryInput) (pluginsdk.DocumentTimelinePage, error)
	Search(context.Context, string, pluginsdk.DocumentSearchInput) (pluginsdk.DocumentSearchPage, error)
	Print(context.Context, string, pluginsdk.DocumentPrintInput) (pluginsdk.DocumentPrintPayload, error)
}

type documentWorkflowService struct {
	pluginID string
	backend  documentWorkflowBackend
	scopes   pluginsdk.DataScopeService
	audit    pluginsdk.AuditService
	files    pluginsdk.FileService
	jobs     pluginsdk.JobService
}

func NewDocumentWorkflowService(pluginID string, backend documentWorkflowBackend, scopes pluginsdk.DataScopeService, files pluginsdk.FileService, jobs pluginsdk.JobService, audit pluginsdk.AuditService) (pluginsdk.DocumentService, error) {
	pluginID = strings.ToLower(strings.TrimSpace(pluginID))
	if pluginID == "" || backend == nil || scopes == nil || files == nil || jobs == nil || audit == nil {
		return nil, fmt.Errorf("document workflow host dependencies are required")
	}
	return &documentWorkflowService{pluginID: pluginID, backend: backend, scopes: scopes, files: files, jobs: jobs, audit: audit}, nil
}

func (s *documentWorkflowService) Search(ctx context.Context, input pluginsdk.DocumentSearchInput) (pluginsdk.DocumentSearchPage, error) {
	if err := input.Validate(); err != nil {
		return pluginsdk.DocumentSearchPage{}, documentWorkflowHostError(err)
	}
	if err := s.authorizeTenant(ctx, input.TenantID, input.Permission); err != nil {
		return pluginsdk.DocumentSearchPage{}, err
	}
	page, err := s.backend.Search(ctx, s.pluginID, input)
	return page, documentWorkflowHostError(err)
}

func (s *documentWorkflowService) Print(ctx context.Context, input pluginsdk.DocumentPrintInput) (pluginsdk.DocumentPrintPayload, error) {
	if err := input.Validate(); err != nil {
		return pluginsdk.DocumentPrintPayload{}, documentWorkflowHostError(err)
	}
	if err := s.authorizeTenant(ctx, input.TenantID, input.Permission); err != nil {
		return pluginsdk.DocumentPrintPayload{}, err
	}
	if input.IncludeSensitive {
		if err := s.authorizeTenant(ctx, input.TenantID, input.SensitivePermission); err != nil {
			return pluginsdk.DocumentPrintPayload{}, pluginsdk.NewDocumentWorkflowError(pluginsdk.DocumentWorkflowErrorForbidden, "sensitivePermission", "sensitive document fields are outside the trusted scope", false)
		}
	}
	payload, err := s.backend.Print(ctx, s.pluginID, input)
	if err != nil {
		return pluginsdk.DocumentPrintPayload{}, documentWorkflowHostError(err)
	}
	if _, err = s.audit.Record(ctx, pluginsdk.AuditEntry{
		Action: "document.print", Resource: payload.Document.Type, ResourceID: payload.Document.ID, Risk: pluginsdk.AuditRiskMedium,
		Detail: map[string]any{"includeSensitive": input.IncludeSensitive, "redactedFieldCount": len(payload.RedactedFields)},
	}); err != nil {
		return pluginsdk.DocumentPrintPayload{}, documentWorkflowHostError(err)
	}
	return payload, nil
}

func (s *documentWorkflowService) Export(ctx context.Context, input pluginsdk.DocumentExportInput) (pluginsdk.Job, error) {
	if err := input.Validate(); err != nil {
		return pluginsdk.Job{}, documentWorkflowHostError(err)
	}
	if err := s.authorizeTenant(ctx, input.Search.TenantID, input.Search.Permission); err != nil {
		return pluginsdk.Job{}, err
	}
	if input.IncludeSensitive {
		if err := s.authorizeTenant(ctx, input.Search.TenantID, input.SensitivePermission); err != nil {
			return pluginsdk.Job{}, pluginsdk.NewDocumentWorkflowError(pluginsdk.DocumentWorkflowErrorForbidden, "sensitivePermission", "sensitive document fields are outside the trusted scope", false)
		}
	}
	search := input.Search
	search.Cursor, search.Limit = "", pluginsdk.MaxDocumentSearchPage
	plan := pluginsdk.DocumentExportPlan{
		Version: 1, Search: search, Format: input.Format, MaxRows: input.MaxRows,
		SensitiveAuthorized: input.IncludeSensitive, Actor: s.actor(ctx),
	}
	payload, err := plan.JSON()
	if err != nil {
		return pluginsdk.Job{}, documentWorkflowHostError(err)
	}
	job, err := s.jobs.Schedule(ctx, pluginsdk.JobScheduleInput{
		ID: input.JobID, Kind: pluginsdk.DocumentExportJobKind, IdempotencyKey: input.IdempotencyKey,
		Payload: payload, RunAt: time.Unix(0, 0).UTC(), MaxAttempts: 3,
	})
	if err != nil {
		return pluginsdk.Job{}, documentWorkflowHostError(err)
	}
	if _, err = s.audit.Record(ctx, pluginsdk.AuditEntry{
		Action: "document.export.schedule", Resource: "document_export", ResourceID: input.JobID, Risk: pluginsdk.AuditRiskMedium,
		Detail: map[string]any{"tenantId": input.Search.TenantID, "maxRows": input.MaxRows, "includeSensitive": input.IncludeSensitive},
	}); err != nil {
		return pluginsdk.Job{}, documentWorkflowHostError(err)
	}
	return job, nil
}

func (s *documentWorkflowService) AddAttachment(ctx context.Context, input pluginsdk.DocumentAttachmentAddInput) (pluginsdk.DocumentAttachmentResult, error) {
	if err := input.Validate(); err != nil {
		return pluginsdk.DocumentAttachmentResult{}, documentWorkflowHostError(err)
	}
	if err := s.authorizeTenant(ctx, input.TenantID, input.Permission); err != nil {
		return pluginsdk.DocumentAttachmentResult{}, err
	}
	file, err := s.files.Get(ctx, input.FileID)
	if err != nil || file.ID != input.FileID || file.Status != "available" {
		return pluginsdk.DocumentAttachmentResult{}, pluginsdk.NewDocumentWorkflowError(pluginsdk.DocumentWorkflowErrorForbidden, "fileId", "attachment file is outside the caller or plugin scope", false)
	}
	result, err := s.backend.AddAttachment(ctx, s.pluginID, s.actor(ctx), input, file)
	if err != nil {
		return pluginsdk.DocumentAttachmentResult{}, documentCollaborationHostError(err, "attachmentId", "attachment already exists or the document attachment limit was reached")
	}
	if err = s.recordActivity(ctx, "document.attachment.add", input.DocumentID, result.Event); err != nil {
		return pluginsdk.DocumentAttachmentResult{}, documentWorkflowHostError(err)
	}
	return result, nil
}

func (s *documentWorkflowService) RemoveAttachment(ctx context.Context, input pluginsdk.DocumentAttachmentRemoveInput) (pluginsdk.DocumentAttachmentResult, error) {
	if err := input.Validate(); err != nil {
		return pluginsdk.DocumentAttachmentResult{}, documentWorkflowHostError(err)
	}
	if err := s.authorizeTenant(ctx, input.TenantID, input.Permission); err != nil {
		return pluginsdk.DocumentAttachmentResult{}, err
	}
	result, err := s.backend.RemoveAttachment(ctx, s.pluginID, s.actor(ctx), input)
	if err != nil {
		return pluginsdk.DocumentAttachmentResult{}, documentCollaborationHostError(err, "attachmentId", "attachment was not found or was already removed")
	}
	if err = s.recordActivity(ctx, "document.attachment.remove", input.DocumentID, result.Event); err != nil {
		return pluginsdk.DocumentAttachmentResult{}, documentWorkflowHostError(err)
	}
	return result, nil
}

func (s *documentWorkflowService) ListAttachments(ctx context.Context, input pluginsdk.DocumentCollaborationQueryInput) ([]pluginsdk.DocumentAttachment, error) {
	if err := input.Validate(); err != nil {
		return nil, documentWorkflowHostError(err)
	}
	if err := s.authorizeTenant(ctx, input.TenantID, input.Permission); err != nil {
		return nil, err
	}
	items, err := s.backend.ListAttachments(ctx, s.pluginID, input)
	return items, documentWorkflowHostError(err)
}

func (s *documentWorkflowService) AddComment(ctx context.Context, input pluginsdk.DocumentCommentAddInput) (pluginsdk.DocumentCommentResult, error) {
	if err := input.Validate(); err != nil {
		return pluginsdk.DocumentCommentResult{}, documentWorkflowHostError(err)
	}
	if err := s.authorizeTenant(ctx, input.TenantID, input.Permission); err != nil {
		return pluginsdk.DocumentCommentResult{}, err
	}
	result, err := s.backend.AddComment(ctx, s.pluginID, s.actor(ctx), input)
	if err != nil {
		return pluginsdk.DocumentCommentResult{}, documentCollaborationHostError(err, "commentId", "comment already exists or the document comment limit was reached")
	}
	if err = s.recordActivity(ctx, "document.comment.add", input.DocumentID, result.Event); err != nil {
		return pluginsdk.DocumentCommentResult{}, documentWorkflowHostError(err)
	}
	return result, nil
}

func (s *documentWorkflowService) ListComments(ctx context.Context, input pluginsdk.DocumentCollaborationQueryInput) ([]pluginsdk.DocumentComment, error) {
	if err := input.Validate(); err != nil {
		return nil, documentWorkflowHostError(err)
	}
	if err := s.authorizeTenant(ctx, input.TenantID, input.Permission); err != nil {
		return nil, err
	}
	items, err := s.backend.ListComments(ctx, s.pluginID, input)
	return items, documentWorkflowHostError(err)
}

func (s *documentWorkflowService) Timeline(ctx context.Context, input pluginsdk.DocumentTimelineQueryInput) (pluginsdk.DocumentTimelinePage, error) {
	if err := input.Validate(); err != nil {
		return pluginsdk.DocumentTimelinePage{}, documentWorkflowHostError(err)
	}
	if err := s.authorizeTenant(ctx, input.TenantID, input.Permission); err != nil {
		return pluginsdk.DocumentTimelinePage{}, err
	}
	page, err := s.backend.Timeline(ctx, s.pluginID, input)
	return page, documentWorkflowHostError(err)
}

func (s *documentWorkflowService) Submit(ctx context.Context, input pluginsdk.DocumentWorkflowSubmitInput) (pluginsdk.DocumentWorkflowResult, error) {
	if err := input.Validate(); err != nil {
		return pluginsdk.DocumentWorkflowResult{}, documentWorkflowHostError(err)
	}
	if err := s.authorizeTenant(ctx, input.TenantID, input.Permission); err != nil {
		return pluginsdk.DocumentWorkflowResult{}, err
	}
	result, err := s.backend.Submit(ctx, s.pluginID, s.actor(ctx), input)
	if err != nil {
		return pluginsdk.DocumentWorkflowResult{}, documentWorkflowHostError(err)
	}
	if !result.Duplicate {
		if err = s.record(ctx, "document.submit", input.Draft.ID, result); err != nil {
			return pluginsdk.DocumentWorkflowResult{}, documentWorkflowHostError(err)
		}
	}
	return result, nil
}

func (s *documentWorkflowService) Act(ctx context.Context, input pluginsdk.DocumentWorkflowActionInput) (pluginsdk.DocumentWorkflowResult, error) {
	if err := input.Validate(); err != nil {
		return pluginsdk.DocumentWorkflowResult{}, documentWorkflowHostError(err)
	}
	if err := s.authorizeTenant(ctx, input.TenantID, input.Permission); err != nil {
		return pluginsdk.DocumentWorkflowResult{}, err
	}
	result, err := s.backend.Act(ctx, s.pluginID, s.actor(ctx), input)
	if err != nil {
		return pluginsdk.DocumentWorkflowResult{}, documentWorkflowHostError(err)
	}
	if !result.Duplicate {
		if err = s.record(ctx, "document."+string(input.Action), input.DocumentID, result); err != nil {
			return pluginsdk.DocumentWorkflowResult{}, documentWorkflowHostError(err)
		}
	}
	return result, nil
}

func (s *documentWorkflowService) Get(ctx context.Context, input pluginsdk.DocumentWorkflowGetInput) (pluginsdk.DocumentWorkflowResult, error) {
	if err := input.Validate(); err != nil {
		return pluginsdk.DocumentWorkflowResult{}, documentWorkflowHostError(err)
	}
	if err := s.authorizeTenant(ctx, input.TenantID, input.Permission); err != nil {
		return pluginsdk.DocumentWorkflowResult{}, err
	}
	result, err := s.backend.Get(ctx, s.pluginID, input)
	return result, documentWorkflowHostError(err)
}

func (s *documentWorkflowService) authorizeTenant(ctx context.Context, tenantID string, permission pluginsdk.Permission) error {
	predicate, err := s.scopes.Resolve(ctx, permission)
	if err != nil {
		return pluginsdk.NewDocumentWorkflowError(pluginsdk.DocumentWorkflowErrorForbidden, "permission", "document permission was denied", false)
	}
	constrained := predicate.Constrain(pluginsdk.ScopeFilter{TenantIDs: []string{tenantID}})
	if constrained.Denied() || constrained.AllTenants() || len(constrained.TenantIDs()) != 1 || constrained.TenantIDs()[0] != tenantID {
		return pluginsdk.NewDocumentWorkflowError(pluginsdk.DocumentWorkflowErrorForbidden, "tenantId", "tenant is outside the trusted scope", false)
	}
	return nil
}

func (s *documentWorkflowService) actor(ctx context.Context) pluginsdk.WorkflowActor {
	actor := trustedHostActor(ctx, s.pluginID)
	return pluginsdk.WorkflowActor{ID: actor.id, Name: actor.name}
}

func (s *documentWorkflowService) record(ctx context.Context, action, documentID string, result pluginsdk.DocumentWorkflowResult) error {
	_, err := s.audit.Record(ctx, pluginsdk.AuditEntry{
		Action: action, Resource: result.Document.Type, ResourceID: documentID, Risk: pluginsdk.AuditRiskMedium,
		Detail: map[string]any{"state": result.Document.State, "version": result.Document.Version, "workflowInstanceId": result.Workflow.ID},
	})
	return err
}

func (s *documentWorkflowService) recordActivity(ctx context.Context, action, documentID string, event pluginsdk.DocumentTimelineEvent) error {
	_, err := s.audit.Record(ctx, pluginsdk.AuditEntry{
		Action: action, Resource: "document", ResourceID: documentID, Risk: pluginsdk.AuditRiskMedium,
		Detail: map[string]any{"eventId": event.ID, "sequence": event.Sequence, "attachmentId": event.AttachmentID, "fileId": event.FileID, "commentId": event.CommentID},
	})
	return err
}

func documentWorkflowHostError(err error) error {
	if err == nil {
		return nil
	}
	var publicErr *pluginsdk.DocumentWorkflowError
	if errors.As(err, &publicErr) {
		return err
	}
	var contractErr *pluginsdk.DocumentContractError
	if errors.As(err, &contractErr) {
		return pluginsdk.NewDocumentWorkflowError(pluginsdk.DocumentWorkflowErrorInvalidRequest, contractErr.Field, contractErr.Message, false)
	}
	switch {
	case errors.Is(err, documentworkflowsvc.ErrNotFound):
		return pluginsdk.NewDocumentWorkflowError(pluginsdk.DocumentWorkflowErrorNotFound, "documentId", "document workflow was not found", false)
	case errors.Is(err, documentworkflowsvc.ErrConflict):
		return pluginsdk.NewDocumentWorkflowError(pluginsdk.DocumentWorkflowErrorConflict, "expectedVersion", "document workflow action conflicts with current state or idempotency input", false)
	case errors.Is(err, documentworkflowsvc.ErrTransactionRequired):
		return pluginsdk.NewDocumentWorkflowError(pluginsdk.DocumentWorkflowErrorTransactionRequired, "transaction", "document workflow writes require an active host transaction", false)
	default:
		return pluginsdk.NewDocumentWorkflowError(pluginsdk.DocumentWorkflowErrorUnavailable, "", "document workflow service is unavailable", true)
	}
}

func documentCollaborationHostError(err error, field, conflictMessage string) error {
	switch {
	case errors.Is(err, documentworkflowsvc.ErrNotFound):
		return pluginsdk.NewDocumentWorkflowError(pluginsdk.DocumentWorkflowErrorNotFound, field, "document collaboration resource was not found", false)
	case errors.Is(err, documentworkflowsvc.ErrConflict):
		return pluginsdk.NewDocumentWorkflowError(pluginsdk.DocumentWorkflowErrorConflict, field, conflictMessage, false)
	default:
		return documentWorkflowHostError(err)
	}
}

var _ pluginsdk.DocumentService = (*documentWorkflowService)(nil)
