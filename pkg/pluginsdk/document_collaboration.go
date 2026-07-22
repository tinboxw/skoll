package pluginsdk

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	MaxDocumentAttachments  = 200
	MaxDocumentComments     = 1000
	MaxDocumentCommentSize  = 8000
	MaxDocumentActivityPage = 200
)

type DocumentService interface {
	DocumentWorkflowService
	DocumentQueryService
	AddAttachment(context.Context, DocumentAttachmentAddInput) (DocumentAttachmentResult, error)
	RemoveAttachment(context.Context, DocumentAttachmentRemoveInput) (DocumentAttachmentResult, error)
	ListAttachments(context.Context, DocumentCollaborationQueryInput) ([]DocumentAttachment, error)
	AddComment(context.Context, DocumentCommentAddInput) (DocumentCommentResult, error)
	ListComments(context.Context, DocumentCollaborationQueryInput) ([]DocumentComment, error)
	Timeline(context.Context, DocumentTimelineQueryInput) (DocumentTimelinePage, error)
}

type DocumentAttachmentAddInput struct {
	TenantID     string     `json:"tenantId"`
	Permission   Permission `json:"permission"`
	DocumentID   string     `json:"documentId"`
	AttachmentID string     `json:"attachmentId"`
	FileID       string     `json:"fileId"`
}

type DocumentAttachmentRemoveInput struct {
	TenantID     string     `json:"tenantId"`
	Permission   Permission `json:"permission"`
	DocumentID   string     `json:"documentId"`
	AttachmentID string     `json:"attachmentId"`
}

type DocumentCommentAddInput struct {
	TenantID   string     `json:"tenantId"`
	Permission Permission `json:"permission"`
	DocumentID string     `json:"documentId"`
	CommentID  string     `json:"commentId"`
	Body       string     `json:"body"`
}

type DocumentCollaborationQueryInput struct {
	TenantID       string     `json:"tenantId"`
	Permission     Permission `json:"permission"`
	DocumentID     string     `json:"documentId"`
	Offset         int        `json:"offset,omitempty"`
	Limit          int        `json:"limit,omitempty"`
	IncludeRemoved bool       `json:"includeRemoved,omitempty"`
}

type DocumentTimelineQueryInput struct {
	TenantID      string     `json:"tenantId"`
	Permission    Permission `json:"permission"`
	DocumentID    string     `json:"documentId"`
	AfterSequence int64      `json:"afterSequence,omitempty"`
	Limit         int        `json:"limit,omitempty"`
}

type DocumentAttachment struct {
	ID              string        `json:"id"`
	DocumentID      string        `json:"documentId"`
	File            FileObject    `json:"file"`
	AddedBy         WorkflowActor `json:"addedBy"`
	AddedAt         time.Time     `json:"addedAt"`
	AddedSequence   int64         `json:"addedSequence"`
	RemovedBy       WorkflowActor `json:"removedBy,omitempty"`
	RemovedAt       *time.Time    `json:"removedAt,omitempty"`
	RemovedSequence *int64        `json:"removedSequence,omitempty"`
}

type DocumentComment struct {
	ID         string        `json:"id"`
	DocumentID string        `json:"documentId"`
	Body       string        `json:"body"`
	Author     WorkflowActor `json:"author"`
	CreatedAt  time.Time     `json:"createdAt"`
	Sequence   int64         `json:"sequence"`
}

type DocumentTimelineKind string

const (
	DocumentTimelineAction            DocumentTimelineKind = "action"
	DocumentTimelineAttachmentAdded   DocumentTimelineKind = "attachment_added"
	DocumentTimelineAttachmentRemoved DocumentTimelineKind = "attachment_removed"
	DocumentTimelineCommentAdded      DocumentTimelineKind = "comment_added"
)

type DocumentTimelineEvent struct {
	ID           string               `json:"id"`
	Sequence     int64                `json:"sequence"`
	DocumentID   string               `json:"documentId"`
	Kind         DocumentTimelineKind `json:"kind"`
	Action       string               `json:"action,omitempty"`
	AttachmentID string               `json:"attachmentId,omitempty"`
	FileID       string               `json:"fileId,omitempty"`
	CommentID    string               `json:"commentId,omitempty"`
	Actor        WorkflowActor        `json:"actor"`
	OccurredAt   time.Time            `json:"occurredAt"`
}

type DocumentAttachmentResult struct {
	Attachment DocumentAttachment    `json:"attachment"`
	Event      DocumentTimelineEvent `json:"event"`
}

type DocumentCommentResult struct {
	Comment DocumentComment       `json:"comment"`
	Event   DocumentTimelineEvent `json:"event"`
}

type DocumentTimelinePage struct {
	Events       []DocumentTimelineEvent `json:"events"`
	NextSequence int64                   `json:"nextSequence"`
	HasMore      bool                    `json:"hasMore"`
}

func (in DocumentAttachmentAddInput) Validate() error {
	if err := validateDocumentResource(in.TenantID, in.Permission, in.DocumentID); err != nil {
		return err
	}
	if err := validateDocumentID("attachmentId", in.AttachmentID); err != nil {
		return documentWorkflowValidationError(err)
	}
	return documentWorkflowValidationError(validateDocumentID("fileId", in.FileID))
}

func (in DocumentAttachmentRemoveInput) Validate() error {
	if err := validateDocumentResource(in.TenantID, in.Permission, in.DocumentID); err != nil {
		return err
	}
	return documentWorkflowValidationError(validateDocumentID("attachmentId", in.AttachmentID))
}

func (in DocumentCommentAddInput) Validate() error {
	if err := validateDocumentResource(in.TenantID, in.Permission, in.DocumentID); err != nil {
		return err
	}
	if err := validateDocumentID("commentId", in.CommentID); err != nil {
		return documentWorkflowValidationError(err)
	}
	if strings.TrimSpace(in.Body) != in.Body || in.Body == "" || len([]byte(in.Body)) > MaxDocumentCommentSize {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "body", "comment body is invalid", false)
	}
	return nil
}

func (in DocumentCollaborationQueryInput) Validate() error {
	if err := validateDocumentResource(in.TenantID, in.Permission, in.DocumentID); err != nil {
		return err
	}
	return validateDocumentActivityPage(in.Offset, in.Limit)
}

func (in DocumentTimelineQueryInput) Validate() error {
	if err := validateDocumentResource(in.TenantID, in.Permission, in.DocumentID); err != nil {
		return err
	}
	if in.AfterSequence < 0 {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "afterSequence", "timeline sequence cannot be negative", false)
	}
	return validateDocumentActivityPage(0, in.Limit)
}

func (a DocumentAttachment) Validate() error {
	if err := validateDocumentID("attachment.id", a.ID); err != nil {
		return documentWorkflowValidationError(err)
	}
	if err := validateDocumentID("attachment.documentId", a.DocumentID); err != nil {
		return documentWorkflowValidationError(err)
	}
	if err := validateDocumentID("attachment.file.id", a.File.ID); err != nil {
		return documentWorkflowValidationError(err)
	}
	if err := validateDocumentActivityActor("attachment.addedBy", a.AddedBy); err != nil {
		return err
	}
	if a.AddedSequence <= 0 || !validDocumentActivityTime(a.AddedAt) {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "attachment.addedSequence", "attachment add attribution is invalid", false)
	}
	removed := a.RemovedAt != nil || a.RemovedSequence != nil || strings.TrimSpace(a.RemovedBy.ID) != "" || strings.TrimSpace(a.RemovedBy.Name) != ""
	if !removed {
		return nil
	}
	if a.RemovedAt == nil || a.RemovedSequence == nil || *a.RemovedSequence <= a.AddedSequence || !validDocumentActivityTime(*a.RemovedAt) || a.RemovedAt.Before(a.AddedAt) {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "attachment.removedSequence", "attachment removal attribution is invalid", false)
	}
	return validateDocumentActivityActor("attachment.removedBy", a.RemovedBy)
}

func (c DocumentComment) Validate() error {
	input := DocumentCommentAddInput{
		TenantID: "response", Permission: Permission{Resource: "response.document", Action: "read"},
		DocumentID: c.DocumentID, CommentID: c.ID, Body: c.Body,
	}
	if err := input.Validate(); err != nil {
		return err
	}
	if err := validateDocumentActivityActor("comment.author", c.Author); err != nil {
		return err
	}
	if c.Sequence <= 0 || !validDocumentActivityTime(c.CreatedAt) {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "comment.sequence", "comment attribution is invalid", false)
	}
	return nil
}

func (e DocumentTimelineEvent) Validate() error {
	if strings.TrimSpace(e.ID) != e.ID || e.ID == "" || len(e.ID) > 192 {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "timeline.event.id", "timeline event id is invalid", false)
	}
	if err := validateDocumentID("timeline.event.documentId", e.DocumentID); err != nil {
		return documentWorkflowValidationError(err)
	}
	if e.Sequence <= 0 || !validDocumentActivityTime(e.OccurredAt) {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "timeline.event.sequence", "timeline event position is invalid", false)
	}
	if err := validateDocumentActivityActor("timeline.event.actor", e.Actor); err != nil {
		return err
	}
	switch e.Kind {
	case DocumentTimelineAction:
		if strings.TrimSpace(e.Action) != e.Action || e.Action == "" || len(e.Action) > 32 || e.AttachmentID != "" || e.FileID != "" || e.CommentID != "" {
			return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "timeline.event.action", "timeline action event is invalid", false)
		}
	case DocumentTimelineAttachmentAdded, DocumentTimelineAttachmentRemoved:
		if validateDocumentID("timeline.event.attachmentId", e.AttachmentID) != nil || validateDocumentID("timeline.event.fileId", e.FileID) != nil || e.Action != "" || e.CommentID != "" {
			return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "timeline.event.attachmentId", "timeline attachment event is invalid", false)
		}
	case DocumentTimelineCommentAdded:
		if validateDocumentID("timeline.event.commentId", e.CommentID) != nil || e.Action != "" || e.AttachmentID != "" || e.FileID != "" {
			return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "timeline.event.commentId", "timeline comment event is invalid", false)
		}
	default:
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "timeline.event.kind", "timeline event kind is invalid", false)
	}
	return nil
}

func (r DocumentAttachmentResult) Validate() error {
	if err := r.Attachment.Validate(); err != nil {
		return err
	}
	if err := r.Event.Validate(); err != nil {
		return err
	}
	if r.Event.DocumentID != r.Attachment.DocumentID || r.Event.AttachmentID != r.Attachment.ID || r.Event.Sequence != attachmentResultSequence(r.Attachment, r.Event.Kind) {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "response", "attachment result does not match its timeline event", false)
	}
	return nil
}

func (r DocumentCommentResult) Validate() error {
	if err := r.Comment.Validate(); err != nil {
		return err
	}
	if err := r.Event.Validate(); err != nil {
		return err
	}
	if r.Event.Kind != DocumentTimelineCommentAdded || r.Event.DocumentID != r.Comment.DocumentID || r.Event.CommentID != r.Comment.ID || r.Event.Sequence != r.Comment.Sequence {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "response", "comment result does not match its timeline event", false)
	}
	return nil
}

func (p DocumentTimelinePage) Validate() error {
	if len(p.Events) > MaxDocumentActivityPage || p.NextSequence < 0 {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "timeline", "timeline page is invalid", false)
	}
	var prior int64
	for index, event := range p.Events {
		if err := event.Validate(); err != nil {
			return err
		}
		if index > 0 && event.Sequence <= prior {
			return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "timeline.events", "timeline events are not strictly ordered", false)
		}
		prior = event.Sequence
	}
	if len(p.Events) > 0 && p.NextSequence != p.Events[len(p.Events)-1].Sequence {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "timeline.nextSequence", "timeline cursor does not match the last event", false)
	}
	return nil
}

func validateDocumentResource(tenantID string, permission Permission, documentID string) error {
	if err := validateDocumentWorkflowScope(tenantID, permission); err != nil {
		return err
	}
	return documentWorkflowValidationError(validateDocumentID("documentId", documentID))
}

func validateDocumentActivityPage(offset, limit int) error {
	if offset < 0 {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "offset", "offset cannot be negative", false)
	}
	if limit < 0 || limit > MaxDocumentActivityPage {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "limit", fmt.Sprintf("limit must be between 0 and %d", MaxDocumentActivityPage), false)
	}
	return nil
}

func validateDocumentActivityActor(path string, actor WorkflowActor) error {
	id := strings.TrimSpace(actor.ID)
	if id == "" || id != actor.ID || len(id) > 512 || strings.TrimSpace(actor.Name) != actor.Name || len(actor.Name) > 256 {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, path, "document activity actor is invalid", false)
	}
	return nil
}

func validDocumentActivityTime(value time.Time) bool {
	if value.IsZero() {
		return false
	}
	_, offset := value.Zone()
	return offset == 0
}

func attachmentResultSequence(attachment DocumentAttachment, kind DocumentTimelineKind) int64 {
	if kind == DocumentTimelineAttachmentRemoved && attachment.RemovedSequence != nil {
		return *attachment.RemovedSequence
	}
	return attachment.AddedSequence
}
