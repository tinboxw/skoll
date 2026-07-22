package pluginsdk

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestDocumentCollaborationContractsAcceptValidInputs(t *testing.T) {
	permission := Permission{Resource: "medical_oa.approval_request", Action: "manage"}
	valid := []interface{ Validate() error }{
		DocumentAttachmentAddInput{TenantID: "tenant-a", Permission: permission, DocumentID: "document-1", AttachmentID: "attachment-1", FileID: "file-1"},
		DocumentAttachmentRemoveInput{TenantID: "tenant-a", Permission: permission, DocumentID: "document-1", AttachmentID: "attachment-1"},
		DocumentCommentAddInput{TenantID: "tenant-a", Permission: permission, DocumentID: "document-1", CommentID: "comment-1", Body: "Reviewed and approved."},
		DocumentCollaborationQueryInput{TenantID: "tenant-a", Permission: permission, DocumentID: "document-1", Limit: MaxDocumentActivityPage},
		DocumentTimelineQueryInput{TenantID: "tenant-a", Permission: permission, DocumentID: "document-1", AfterSequence: 1, Limit: MaxDocumentActivityPage},
	}
	for index, input := range valid {
		if err := input.Validate(); err != nil {
			t.Fatalf("valid collaboration input %d rejected: %v", index, err)
		}
	}
}

func TestDocumentCollaborationResponsesRequireAttributionAndStrictOrdering(t *testing.T) {
	now := time.Date(2026, time.July, 22, 8, 0, 0, 0, time.UTC)
	actor := WorkflowActor{ID: "user-1", Name: "User One"}
	attachment := DocumentAttachmentResult{
		Attachment: DocumentAttachment{
			ID: "attachment-1", DocumentID: "document-1", File: FileObject{ID: "file-1"},
			AddedBy: actor, AddedAt: now, AddedSequence: 2,
		},
		Event: DocumentTimelineEvent{
			ID: "attachment-added:attachment-1", Sequence: 2, DocumentID: "document-1", Kind: DocumentTimelineAttachmentAdded,
			AttachmentID: "attachment-1", FileID: "file-1", Actor: actor, OccurredAt: now,
		},
	}
	if err := attachment.Validate(); err != nil {
		t.Fatalf("valid attachment result rejected: %v", err)
	}
	attachment.Event.Sequence = 3
	if err := attachment.Validate(); err == nil {
		t.Fatal("attachment and event sequence mismatch was accepted")
	}

	comment := DocumentCommentResult{
		Comment: DocumentComment{ID: "comment-1", DocumentID: "document-1", Body: "Reviewed.", Author: actor, CreatedAt: now, Sequence: 3},
		Event: DocumentTimelineEvent{
			ID: "comment-added:comment-1", Sequence: 3, DocumentID: "document-1", Kind: DocumentTimelineCommentAdded,
			CommentID: "comment-1", Actor: actor, OccurredAt: now,
		},
	}
	if err := comment.Validate(); err != nil {
		t.Fatalf("valid comment result rejected: %v", err)
	}

	page := DocumentTimelinePage{Events: []DocumentTimelineEvent{comment.Event, {
		ID: "action:approve-1", Sequence: 2, DocumentID: "document-1", Kind: DocumentTimelineAction,
		Action: "approve", Actor: actor, OccurredAt: now,
	}}, NextSequence: 2}
	if err := page.Validate(); err == nil {
		t.Fatal("out-of-order timeline was accepted")
	}
}

func TestDocumentCollaborationContractsReturnStableValidationErrors(t *testing.T) {
	permission := Permission{Resource: "medical_oa.approval_request", Action: "manage"}
	tests := []struct {
		name  string
		input interface{ Validate() error }
		field string
	}{
		{name: "foreign whitespace in comment", input: DocumentCommentAddInput{TenantID: "tenant-a", Permission: permission, DocumentID: "document-1", CommentID: "comment-1", Body: " padded "}, field: "body"},
		{name: "comment byte limit", input: DocumentCommentAddInput{TenantID: "tenant-a", Permission: permission, DocumentID: "document-1", CommentID: "comment-1", Body: strings.Repeat("a", MaxDocumentCommentSize+1)}, field: "body"},
		{name: "negative offset", input: DocumentCollaborationQueryInput{TenantID: "tenant-a", Permission: permission, DocumentID: "document-1", Offset: -1}, field: "offset"},
		{name: "page limit", input: DocumentCollaborationQueryInput{TenantID: "tenant-a", Permission: permission, DocumentID: "document-1", Limit: MaxDocumentActivityPage + 1}, field: "limit"},
		{name: "negative timeline sequence", input: DocumentTimelineQueryInput{TenantID: "tenant-a", Permission: permission, DocumentID: "document-1", AfterSequence: -1}, field: "afterSequence"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var contractErr *DocumentWorkflowError
			err := test.input.Validate()
			if !errors.As(err, &contractErr) || contractErr.Code != DocumentWorkflowErrorInvalidRequest || contractErr.Field != test.field {
				t.Fatalf("validation error=%v", err)
			}
		})
	}
}
