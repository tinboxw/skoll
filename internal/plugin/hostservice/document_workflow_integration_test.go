package hostservice

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	documentworkflowsvc "github.com/tinboxw/skoll/internal/service/documentworkflow"
	jobsvc "github.com/tinboxw/skoll/internal/service/job"
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
	documents pluginsdk.DocumentService
	workflows pluginsdk.WorkflowService
	tx        pluginsdk.TransactionService
	audit     *documentWorkflowAudit
	jobs      pluginsdk.JobService
	db        *gorm.DB
}

type documentWorkflowScopes struct {
	tenantID     string
	denyResource string
}

func (s documentWorkflowScopes) Resolve(_ context.Context, permission pluginsdk.Permission) (pluginsdk.ScopePredicate, error) {
	if permission.Resource == s.denyResource {
		return pluginsdk.ScopePredicate{}, errors.New("permission denied")
	}
	return pluginsdk.NewScopePredicate(pluginsdk.TrustedScope{
		SubjectID: "user-1", TenantIDs: []string{s.tenantID}, AllOwners: true, AllOrganizations: true,
	})
}

type documentWorkflowFiles struct{}

func (documentWorkflowFiles) Store(context.Context, pluginsdk.FileWrite) (pluginsdk.FileObject, error) {
	return pluginsdk.FileObject{}, errors.New("file store is not used by this fixture")
}

func (documentWorkflowFiles) List(context.Context, pluginsdk.FileQuery) ([]pluginsdk.FileObject, error) {
	return nil, errors.New("file list is not used by this fixture")
}

func (documentWorkflowFiles) Get(ctx context.Context, id string) (pluginsdk.FileObject, error) {
	claims, ok := security.JWTClaimsFromContext(ctx)
	if !ok || id != "file-"+claims.Subject {
		return pluginsdk.FileObject{}, errors.New("file access denied")
	}
	now := time.Date(2026, time.July, 22, 8, 0, 0, 0, time.UTC)
	return pluginsdk.FileObject{
		ID: id, Key: "documents/" + id, Name: id + ".pdf", Size: 1024,
		MIME: "application/pdf", Hash: "sha256:test", Visibility: pluginsdk.FileVisibilityPrivate,
		Status: "available", Metadata: map[string]string{"source": "acceptance"}, CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (documentWorkflowFiles) Download(context.Context, string) (pluginsdk.FileDownload, error) {
	return pluginsdk.FileDownload{}, errors.New("file download is not used by this fixture")
}

func (documentWorkflowFiles) Delete(context.Context, string) error {
	return errors.New("file delete is not used by this fixture")
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

func (a *documentWorkflowAudit) snapshot() []pluginsdk.AuditEntry {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]pluginsdk.AuditEntry(nil), a.entries...)
}

func TestDocumentCollaborationTimelineAttributionAndImmutability(t *testing.T) {
	fixture := newDocumentWorkflowFixture(t)
	ctx := documentWorkflowUserContext("user-1")
	submitDocument(t, fixture, ctx, "collaboration-document", "instance-collaboration")

	var added pluginsdk.DocumentAttachmentResult
	var commented pluginsdk.DocumentCommentResult
	var removed pluginsdk.DocumentAttachmentResult
	err := fixture.tx.Within(ctx, func(tx pluginsdk.Transaction) error {
		var err error
		added, err = fixture.documents.AddAttachment(tx.Context(), pluginsdk.DocumentAttachmentAddInput{
			TenantID: "tenant-a", Permission: documentWorkflowPermission(), DocumentID: "collaboration-document",
			AttachmentID: "attachment-1", FileID: "file-user-1",
		})
		if err != nil {
			return err
		}
		commented, err = fixture.documents.AddComment(tx.Context(), pluginsdk.DocumentCommentAddInput{
			TenantID: "tenant-a", Permission: documentWorkflowPermission(), DocumentID: "collaboration-document",
			CommentID: "comment-1", Body: "Reviewed by quality assurance.",
		})
		if err != nil {
			return err
		}
		removed, err = fixture.documents.RemoveAttachment(tx.Context(), pluginsdk.DocumentAttachmentRemoveInput{
			TenantID: "tenant-a", Permission: documentWorkflowPermission(), DocumentID: "collaboration-document", AttachmentID: "attachment-1",
		})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if added.Attachment.AddedBy.ID != "user-1" || added.Event.Sequence != 2 || commented.Comment.Author.ID != "user-1" || commented.Event.Sequence != 3 {
		t.Fatalf("collaboration attribution or ordering is invalid: added=%+v commented=%+v", added, commented)
	}
	if removed.Attachment.RemovedAt == nil || removed.Attachment.RemovedBy.ID != "user-1" || removed.Event.Sequence != 4 || removed.Event.FileID != "file-user-1" {
		t.Fatalf("attachment removal lost attribution: %+v", removed)
	}

	query := pluginsdk.DocumentCollaborationQueryInput{TenantID: "tenant-a", Permission: documentWorkflowPermission(), DocumentID: "collaboration-document"}
	active, err := fixture.documents.ListAttachments(ctx, query)
	if err != nil || len(active) != 0 {
		t.Fatalf("removed attachment remained active: items=%+v err=%v", active, err)
	}
	query.IncludeRemoved = true
	attachments, err := fixture.documents.ListAttachments(ctx, query)
	if err != nil || len(attachments) != 1 || attachments[0].RemovedSequence == nil || *attachments[0].RemovedSequence != 4 {
		t.Fatalf("attachment history is incomplete: items=%+v err=%v", attachments, err)
	}
	comments, err := fixture.documents.ListComments(ctx, query)
	if err != nil || len(comments) != 1 || comments[0].Body != "Reviewed by quality assurance." || comments[0].Sequence != 3 {
		t.Fatalf("immutable comment is incomplete: items=%+v err=%v", comments, err)
	}

	first, err := fixture.documents.Timeline(ctx, pluginsdk.DocumentTimelineQueryInput{
		TenantID: "tenant-a", Permission: documentWorkflowPermission(), DocumentID: "collaboration-document", Limit: 2,
	})
	if err != nil || !first.HasMore || first.NextSequence != 2 || len(first.Events) != 2 {
		t.Fatalf("first timeline page=%+v err=%v", first, err)
	}
	second, err := fixture.documents.Timeline(ctx, pluginsdk.DocumentTimelineQueryInput{
		TenantID: "tenant-a", Permission: documentWorkflowPermission(), DocumentID: "collaboration-document", AfterSequence: first.NextSequence, Limit: 2,
	})
	if err != nil || second.HasMore || second.NextSequence != 4 || len(second.Events) != 2 {
		t.Fatalf("second timeline page=%+v err=%v", second, err)
	}
	kinds := []pluginsdk.DocumentTimelineKind{first.Events[0].Kind, first.Events[1].Kind, second.Events[0].Kind, second.Events[1].Kind}
	want := []pluginsdk.DocumentTimelineKind{pluginsdk.DocumentTimelineAction, pluginsdk.DocumentTimelineAttachmentAdded, pluginsdk.DocumentTimelineCommentAdded, pluginsdk.DocumentTimelineAttachmentRemoved}
	for index := range want {
		if kinds[index] != want[index] {
			t.Fatalf("timeline kind %d=%q want=%q", index, kinds[index], want[index])
		}
	}

	err = fixture.tx.Within(ctx, func(tx pluginsdk.Transaction) error {
		_, err := fixture.documents.AddComment(tx.Context(), pluginsdk.DocumentCommentAddInput{
			TenantID: "tenant-a", Permission: documentWorkflowPermission(), DocumentID: "collaboration-document",
			CommentID: "comment-1", Body: "Duplicate comment.",
		})
		return err
	})
	assertDocumentCollaborationError(t, err, pluginsdk.DocumentWorkflowErrorConflict, "commentId")
	err = fixture.tx.Within(ctx, func(tx pluginsdk.Transaction) error {
		_, err := fixture.documents.RemoveAttachment(tx.Context(), pluginsdk.DocumentAttachmentRemoveInput{
			TenantID: "tenant-a", Permission: documentWorkflowPermission(), DocumentID: "collaboration-document", AttachmentID: "attachment-1",
		})
		return err
	})
	assertDocumentCollaborationError(t, err, pluginsdk.DocumentWorkflowErrorConflict, "attachmentId")

	actions := map[string]bool{}
	for _, entry := range fixture.audit.snapshot() {
		actions[entry.Action] = true
		if entry.Action == "document.comment.add" {
			if _, leaked := entry.Detail["body"]; leaked {
				t.Fatal("audit detail leaked comment body")
			}
		}
	}
	for _, action := range []string{"document.attachment.add", "document.comment.add", "document.attachment.remove"} {
		if !actions[action] {
			t.Fatalf("missing audit action %q", action)
		}
	}
}

func assertDocumentCollaborationError(t *testing.T, err error, code pluginsdk.DocumentWorkflowErrorCode, field string) {
	t.Helper()
	var publicErr *pluginsdk.DocumentWorkflowError
	if !errors.As(err, &publicErr) || publicErr.Code != code || publicErr.Field != field {
		t.Fatalf("document collaboration error=%v want code=%s field=%s", err, code, field)
	}
}

func TestDocumentCollaborationRejectsForeignFilesAndRollsBackAtomically(t *testing.T) {
	fixture := newDocumentWorkflowFixture(t)
	ctx := documentWorkflowUserContext("user-1")
	submitDocument(t, fixture, ctx, "scope-document", "instance-scope")

	err := fixture.tx.Within(ctx, func(tx pluginsdk.Transaction) error {
		_, err := fixture.documents.AddAttachment(tx.Context(), pluginsdk.DocumentAttachmentAddInput{
			TenantID: "tenant-a", Permission: documentWorkflowPermission(), DocumentID: "scope-document",
			AttachmentID: "foreign-attachment", FileID: "file-user-2",
		})
		return err
	})
	var publicErr *pluginsdk.DocumentWorkflowError
	if !errors.As(err, &publicErr) || publicErr.Code != pluginsdk.DocumentWorkflowErrorForbidden || publicErr.Field != "fileId" {
		t.Fatalf("foreign file error=%v", err)
	}

	rollback := errors.New("rollback collaboration")
	err = fixture.tx.Within(ctx, func(tx pluginsdk.Transaction) error {
		if _, err := fixture.documents.AddAttachment(tx.Context(), pluginsdk.DocumentAttachmentAddInput{
			TenantID: "tenant-a", Permission: documentWorkflowPermission(), DocumentID: "scope-document",
			AttachmentID: "rollback-attachment", FileID: "file-user-1",
		}); err != nil {
			return err
		}
		if _, err := fixture.documents.AddComment(tx.Context(), pluginsdk.DocumentCommentAddInput{
			TenantID: "tenant-a", Permission: documentWorkflowPermission(), DocumentID: "scope-document",
			CommentID: "rollback-comment", Body: "This must not persist.",
		}); err != nil {
			return err
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatalf("rollback error=%v", err)
	}
	query := pluginsdk.DocumentCollaborationQueryInput{TenantID: "tenant-a", Permission: documentWorkflowPermission(), DocumentID: "scope-document", IncludeRemoved: true}
	attachments, err := fixture.documents.ListAttachments(ctx, query)
	if err != nil || len(attachments) != 0 {
		t.Fatalf("rolled back attachment is visible: items=%+v err=%v", attachments, err)
	}
	comments, err := fixture.documents.ListComments(ctx, query)
	if err != nil || len(comments) != 0 {
		t.Fatalf("rolled back comment is visible: items=%+v err=%v", comments, err)
	}
	timeline, err := fixture.documents.Timeline(ctx, pluginsdk.DocumentTimelineQueryInput{TenantID: "tenant-a", Permission: documentWorkflowPermission(), DocumentID: "scope-document"})
	if err != nil || len(timeline.Events) != 1 || timeline.Events[0].Action != "submit" {
		t.Fatalf("rollback changed immutable timeline: page=%+v err=%v", timeline, err)
	}
}

func TestDocumentCollaborationWritesRequireTransaction(t *testing.T) {
	fixture := newDocumentWorkflowFixture(t)
	ctx := documentWorkflowUserContext("user-1")
	submitDocument(t, fixture, ctx, "transaction-document", "instance-transaction")
	writes := []func() error{
		func() error {
			_, err := fixture.documents.AddAttachment(ctx, pluginsdk.DocumentAttachmentAddInput{TenantID: "tenant-a", Permission: documentWorkflowPermission(), DocumentID: "transaction-document", AttachmentID: "attachment-1", FileID: "file-user-1"})
			return err
		},
		func() error {
			_, err := fixture.documents.AddComment(ctx, pluginsdk.DocumentCommentAddInput{TenantID: "tenant-a", Permission: documentWorkflowPermission(), DocumentID: "transaction-document", CommentID: "comment-1", Body: "Requires a transaction."})
			return err
		},
	}
	for index, write := range writes {
		var publicErr *pluginsdk.DocumentWorkflowError
		if err := write(); !errors.As(err, &publicErr) || publicErr.Code != pluginsdk.DocumentWorkflowErrorTransactionRequired {
			t.Fatalf("write %d transaction error=%v", index, err)
		}
	}
}

func TestDocumentSearchUsesScopedStableCursorPaging(t *testing.T) {
	fixture := newDocumentWorkflowFixture(t)
	ctx := documentWorkflowUserContext("user-1")
	for _, number := range []string{"001", "003", "005"} {
		input := documentWorkflowSubmitInput("search-"+number, "instance-search-"+number)
		input.Draft.Number = "OA-" + number
		input.Draft.Title = "Approval " + number
		submitDocumentInput(t, fixture, ctx, input)
	}
	search := pluginsdk.DocumentSearchInput{
		TenantID: "tenant-a", Permission: documentWorkflowPermission(), Types: []string{"approval_request"}, States: []string{"pending"},
		Text: "Approval", SortField: pluginsdk.DocumentSearchSortNumber, Direction: pluginsdk.DocumentSearchAscending, Limit: 2,
	}
	first, err := fixture.documents.Search(ctx, search)
	if err != nil || !first.HasMore || first.NextCursor == "" || len(first.Items) != 2 || first.Items[0].Number != "OA-001" || first.Items[1].Number != "OA-003" {
		t.Fatalf("first document search page=%+v err=%v", first, err)
	}
	inserted := documentWorkflowSubmitInput("search-002", "instance-search-002")
	inserted.Draft.Number, inserted.Draft.Title = "OA-002", "Approval 002"
	submitDocumentInput(t, fixture, ctx, inserted)
	search.Cursor = first.NextCursor
	second, err := fixture.documents.Search(ctx, search)
	if err != nil || second.HasMore || second.NextCursor != "" || len(second.Items) != 1 || second.Items[0].Number != "OA-005" {
		t.Fatalf("second document search page drifted: %+v err=%v", second, err)
	}

	mismatch := search
	mismatch.States = []string{"approved"}
	_, err = fixture.documents.Search(ctx, mismatch)
	assertDocumentCollaborationError(t, err, pluginsdk.DocumentWorkflowErrorInvalidRequest, "cursor")
	foreign := search
	foreign.Cursor, foreign.TenantID = "", "tenant-b"
	_, err = fixture.documents.Search(ctx, foreign)
	assertDocumentCollaborationError(t, err, pluginsdk.DocumentWorkflowErrorForbidden, "tenantId")

	literal := documentWorkflowSubmitInput("search-percent", "instance-search-percent")
	literal.Draft.Number, literal.Draft.Title = "OA-%-006", "Literal percent"
	submitDocumentInput(t, fixture, ctx, literal)
	percent, err := fixture.documents.Search(ctx, pluginsdk.DocumentSearchInput{
		TenantID: "tenant-a", Permission: documentWorkflowPermission(), Text: "%", SortField: pluginsdk.DocumentSearchSortNumber, Direction: pluginsdk.DocumentSearchAscending,
	})
	if err != nil || len(percent.Items) != 1 || percent.Items[0].ID != literal.Draft.ID {
		t.Fatalf("literal wildcard search=%+v err=%v", percent, err)
	}
}

func TestDocumentPrintRedactsSensitiveFieldsByPermission(t *testing.T) {
	fixture := newDocumentWorkflowFixture(t)
	ctx := documentWorkflowUserContext("user-1")
	input := sensitiveDocumentSubmitInput("print-document", "instance-print")
	submitDocumentInput(t, fixture, ctx, input)

	base := pluginsdk.DocumentPrintInput{TenantID: "tenant-a", Permission: documentWorkflowPermission(), DocumentID: input.Draft.ID}
	redacted, err := fixture.documents.Print(ctx, base)
	if err != nil || len(redacted.RedactedFields) != 1 || redacted.RedactedFields[0] != "header.subject" {
		t.Fatalf("redacted print payload=%+v err=%v", redacted, err)
	}
	if _, visible := redacted.Document.Header["subject"]; visible || redacted.Document.Header["department"].Value != "Quality" {
		t.Fatalf("print redaction removed or exposed the wrong fields: %+v", redacted.Document.Header)
	}
	base.IncludeSensitive = true
	base.SensitivePermission = sensitiveDocumentPermission()
	visible, err := fixture.documents.Print(ctx, base)
	if err != nil || len(visible.RedactedFields) != 0 || visible.Document.Header["subject"].Value != "Supplier qualification" {
		t.Fatalf("authorized print payload=%+v err=%v", visible, err)
	}

	deniedFixture := newDocumentWorkflowFixtureWithScopes(t, documentWorkflowScopes{tenantID: "tenant-a", denyResource: sensitiveDocumentPermission().Resource})
	submitDocumentInput(t, deniedFixture, ctx, sensitiveDocumentSubmitInput("denied-print", "instance-denied-print"))
	base.DocumentID = "denied-print"
	_, err = deniedFixture.documents.Print(ctx, base)
	assertDocumentCollaborationError(t, err, pluginsdk.DocumentWorkflowErrorForbidden, "sensitivePermission")
}

func TestDocumentExportSchedulesBoundedDurableJob(t *testing.T) {
	fixture := newDocumentWorkflowFixture(t)
	ctx := documentWorkflowUserContext("user-1")
	input := pluginsdk.DocumentExportInput{
		JobID: "document-export-1", IdempotencyKey: "document-export-1", Format: pluginsdk.DocumentExportCSV, MaxRows: 1000,
		Search: pluginsdk.DocumentSearchInput{
			TenantID: "tenant-a", Permission: documentWorkflowPermission(), Types: []string{"approval_request"},
			SortField: pluginsdk.DocumentSearchSortUpdatedAt, Direction: pluginsdk.DocumentSearchDescending,
		},
	}
	job, err := fixture.documents.Export(ctx, input)
	if err != nil || job.ID != input.JobID || job.Kind != pluginsdk.DocumentExportJobKind || job.Status != pluginsdk.JobStatusScheduled || job.MaxAttempts != 3 {
		t.Fatalf("document export job=%+v err=%v", job, err)
	}
	var plan pluginsdk.DocumentExportPlan
	if err = json.Unmarshal(job.Payload, &plan); err != nil || plan.Validate() != nil || plan.MaxRows != input.MaxRows || plan.Search.Limit != pluginsdk.MaxDocumentSearchPage || plan.Actor.ID != "user-1" || plan.SensitiveAuthorized {
		t.Fatalf("document export plan=%+v err=%v", plan, err)
	}
	duplicate, err := fixture.documents.Export(ctx, input)
	if err != nil || duplicate.ID != job.ID || !duplicate.CreatedAt.Equal(job.CreatedAt) {
		t.Fatalf("idempotent document export=%+v err=%v", duplicate, err)
	}
	freshJobs, err := NewJobService("medical_oa", jobsvc.NewService(gormrepo.NewJobStore(fixture.db), nil), fixture.audit)
	if err != nil {
		t.Fatal(err)
	}
	persisted, err := freshJobs.Get(ctx, input.JobID)
	if err != nil || persisted.ID != job.ID || string(persisted.Payload) != string(job.Payload) {
		t.Fatalf("persisted document export=%+v err=%v", persisted, err)
	}

	deniedFixture := newDocumentWorkflowFixtureWithScopes(t, documentWorkflowScopes{tenantID: "tenant-a", denyResource: sensitiveDocumentPermission().Resource})
	input.JobID, input.IdempotencyKey = "sensitive-export", "sensitive-export"
	input.IncludeSensitive, input.SensitivePermission = true, sensitiveDocumentPermission()
	_, err = deniedFixture.documents.Export(ctx, input)
	assertDocumentCollaborationError(t, err, pluginsdk.DocumentWorkflowErrorForbidden, "sensitivePermission")
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
	return newDocumentWorkflowFixtureWithScopes(t, documentWorkflowScopes{tenantID: "tenant-a"})
}

func newDocumentWorkflowFixtureWithScopes(t *testing.T, scopes pluginsdk.DataScopeService) documentWorkflowFixture {
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
		&gormrepo.DocumentAttachmentModel{}, &gormrepo.DocumentCommentModel{}, &gormrepo.DocumentTimelineEventModel{},
		&gormrepo.JobModel{},
	); err != nil {
		t.Fatal(err)
	}
	audit := &documentWorkflowAudit{}
	workflow, err := NewWorkflowService("medical_oa", workflowsvc.NewService(gormrepo.NewWorkflowStore(db)), audit)
	if err != nil {
		t.Fatal(err)
	}
	jobs, err := NewJobService("medical_oa", jobsvc.NewService(gormrepo.NewJobStore(db), nil), audit)
	if err != nil {
		t.Fatal(err)
	}
	documents, err := NewDocumentWorkflowService(
		"medical_oa", documentworkflowsvc.NewService(gormrepo.NewDocumentWorkflowStore(db), workflow),
		scopes, documentWorkflowFiles{}, jobs, audit,
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
			{ID: "approval", Key: "approval", Name: "Approval", Type: pluginsdk.WorkflowNodeApproval, AssigneeIDs: []string{"user-1"}, Decision: &pluginsdk.WorkflowDecisionRule{Strategy: pluginsdk.WorkflowDecisionAny, Quorum: 1}},
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
	return documentWorkflowFixture{documents: documents, workflows: workflow, tx: tx, audit: audit, jobs: jobs, db: db}
}

func submitDocument(t *testing.T, fixture documentWorkflowFixture, ctx context.Context, documentID, instanceID string) pluginsdk.DocumentWorkflowResult {
	return submitDocumentInput(t, fixture, ctx, documentWorkflowSubmitInput(documentID, instanceID))
}

func submitDocumentInput(t *testing.T, fixture documentWorkflowFixture, ctx context.Context, input pluginsdk.DocumentWorkflowSubmitInput) pluginsdk.DocumentWorkflowResult {
	t.Helper()
	var result pluginsdk.DocumentWorkflowResult
	if err := fixture.tx.Within(ctx, func(tx pluginsdk.Transaction) error {
		var err error
		result, err = fixture.documents.Submit(tx.Context(), input)
		return err
	}); err != nil {
		t.Fatalf("submit %s: %v", input.Draft.ID, err)
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

func sensitiveDocumentPermission() pluginsdk.Permission {
	return pluginsdk.Permission{Resource: "medical_oa.approval_request_sensitive", Action: "read"}
}

func sensitiveDocumentSubmitInput(documentID, instanceID string) pluginsdk.DocumentWorkflowSubmitInput {
	input := documentWorkflowSubmitInput(documentID, instanceID)
	input.Schema.Header = []pluginsdk.DocumentFieldSchema{
		{Key: "subject", Label: "Subject", Type: pluginsdk.DocumentFieldString, Required: true, Sensitive: true},
		{Key: "department", Label: "Department", Type: pluginsdk.DocumentFieldString, Required: true},
	}
	input.Draft.Header = map[string]pluginsdk.DocumentValue{
		"subject":    {Type: pluginsdk.DocumentFieldString, Value: "Supplier qualification"},
		"department": {Type: pluginsdk.DocumentFieldString, Value: "Quality"},
	}
	return input
}

func documentWorkflowUserContext(subject string) context.Context {
	return security.WithJWTClaimsContext(context.Background(), &security.JWTClaims{Subject: subject, Role: "employee", Roles: []string{"employee"}})
}
