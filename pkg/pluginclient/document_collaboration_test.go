package pluginclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

func TestDocumentCollaborationClientRejectsMismatchedResponses(t *testing.T) {
	now := time.Date(2026, time.July, 22, 8, 0, 0, 0, time.UTC)
	actor := pluginsdk.WorkflowActor{ID: "user-1", Name: "User One"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/documents/add-attachment":
			writeClientJSON(t, w, http.StatusOK, pluginsdk.DocumentAttachmentResult{
				Attachment: pluginsdk.DocumentAttachment{
					ID: "attachment-1", DocumentID: "another-document", File: pluginsdk.FileObject{ID: "file-1"},
					AddedBy: actor, AddedAt: now, AddedSequence: 2,
				},
				Event: pluginsdk.DocumentTimelineEvent{
					ID: "attachment-added:attachment-1", Sequence: 2, DocumentID: "another-document", Kind: pluginsdk.DocumentTimelineAttachmentAdded,
					AttachmentID: "attachment-1", FileID: "file-1", Actor: actor, OccurredAt: now,
				},
			})
		case "/v1/documents/timeline":
			writeClientJSON(t, w, http.StatusOK, pluginsdk.DocumentTimelinePage{
				Events: []pluginsdk.DocumentTimelineEvent{{
					ID: "action:submit-1", Sequence: 1, DocumentID: "another-document", Kind: pluginsdk.DocumentTimelineAction,
					Action: "submit", Actor: actor, OccurredAt: now,
				}}, NextSequence: 1,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	host := newClientHost(t, server)
	permission := pluginsdk.Permission{Resource: "medical_oa.approval_request", Action: "manage"}

	_, err := host.Documents.AddAttachment(context.Background(), pluginsdk.DocumentAttachmentAddInput{
		TenantID: "tenant-a", Permission: permission, DocumentID: "document-1", AttachmentID: "attachment-1", FileID: "file-1",
	})
	assertDocumentCollaborationUnavailable(t, err)
	_, err = host.Documents.Timeline(context.Background(), pluginsdk.DocumentTimelineQueryInput{
		TenantID: "tenant-a", Permission: permission, DocumentID: "document-1",
	})
	assertDocumentCollaborationUnavailable(t, err)
}

func assertDocumentCollaborationUnavailable(t *testing.T, err error) {
	t.Helper()
	var documentErr *pluginsdk.DocumentWorkflowError
	if !errors.As(err, &documentErr) || documentErr.Code != pluginsdk.DocumentWorkflowErrorUnavailable || documentErr.Field != "response" {
		t.Fatalf("collaboration response error=%v", err)
	}
}
