package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/plugin/quota"
	"github.com/tinboxw/skoll/pkg/pluginclient"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"github.com/tinboxw/skoll/pkg/security"
)

type gatewayTxMarker struct{}

type gatewayTransactions struct {
	mu        sync.Mutex
	commits   int
	rollbacks int
}

func (s *gatewayTransactions) Within(ctx context.Context, fn func(pluginsdk.Transaction) error) error {
	err := fn(gatewayTransaction{ctx: context.WithValue(ctx, gatewayTxMarker{}, true)})
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.rollbacks++
	} else {
		s.commits++
	}
	return err
}

type gatewayTransaction struct{ ctx context.Context }

func (t gatewayTransaction) Context() context.Context { return t.ctx }

type gatewayScopes struct{}

func (gatewayScopes) Resolve(ctx context.Context, _ pluginsdk.Permission) (pluginsdk.ScopePredicate, error) {
	claims, ok := security.JWTClaimsFromContext(ctx)
	if !ok {
		return pluginsdk.ScopePredicate{}, errors.New("user claims required")
	}
	return pluginsdk.NewScopePredicate(pluginsdk.TrustedScope{SubjectID: claims.Subject, AllTenants: true, AllOwners: true, AllOrganizations: true})
}

type gatewayDataStore struct{}

func (gatewayDataStore) Query(ctx context.Context, query pluginsdk.DataQuery) (pluginsdk.DataPage, error) {
	if query.Table != "assets" {
		return pluginsdk.DataPage{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorNotFound, "table", "table is not declared by the plugin", false)
	}
	id := "asset-1"
	if ctx.Value(gatewayTxMarker{}) == true {
		id = "asset-transaction"
	}
	return pluginsdk.DataPage{Records: []pluginsdk.DataRecord{{Values: map[string]pluginsdk.DataValue{
		"id": {Type: pluginsdk.DataValueString, Value: id},
	}, Version: 1}}}, nil
}

func (gatewayDataStore) Mutate(_ context.Context, mutation pluginsdk.DataMutation) (pluginsdk.DataMutationResult, error) {
	if mutation.Table != "assets" {
		return pluginsdk.DataMutationResult{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorForbidden, "table", "plugin does not own the requested table", false)
	}
	return pluginsdk.DataMutationResult{RowsAffected: 1}, nil
}

func (gatewayDataStore) Aggregate(_ context.Context, query pluginsdk.DataAggregateQuery) (pluginsdk.DataAggregatePage, error) {
	if query.Table != "assets" {
		return pluginsdk.DataAggregatePage{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorNotFound, "table", "table is not declared by the plugin", false)
	}
	page := pluginsdk.DataAggregatePage{
		Metrics: append([]pluginsdk.DataAggregateMetric(nil), query.Metrics...),
		GroupBy: append([]string(nil), query.GroupBy...),
		Rows:    []pluginsdk.DataAggregateRow{},
	}
	if len(query.GroupBy) == 0 {
		values := make([]pluginsdk.DataValue, len(query.Metrics))
		for index, metric := range query.Metrics {
			values[index] = pluginsdk.DataValue{Type: pluginsdk.DataValueNull}
			if metric.Operation == pluginsdk.DataAggregateCount {
				values[index] = pluginsdk.DataValue{Type: pluginsdk.DataValueInteger, Value: "1"}
			}
		}
		page.Rows = append(page.Rows, pluginsdk.DataAggregateRow{Group: map[string]pluginsdk.DataValue{}, Values: values})
	}
	return page, nil
}

type gatewayDocumentNumbers struct{}

func (gatewayDocumentNumbers) Preview(_ context.Context, input pluginsdk.DocumentNumberInput) (pluginsdk.DocumentNumberResult, error) {
	return pluginsdk.DocumentNumberResult{Number: input.Rule.Prefix + "-000001", Sequence: 1}, nil
}

func (gatewayDocumentNumbers) Issue(_ context.Context, input pluginsdk.DocumentNumberInput) (pluginsdk.DocumentNumberResult, error) {
	return pluginsdk.DocumentNumberResult{Number: input.Rule.Prefix + "-000001", Sequence: 1}, nil
}

type gatewayDocumentNumberFailure struct{}

func (gatewayDocumentNumberFailure) Preview(context.Context, pluginsdk.DocumentNumberInput) (pluginsdk.DocumentNumberResult, error) {
	return pluginsdk.DocumentNumberResult{}, pluginsdk.NewDocumentNumberError(pluginsdk.DocumentNumberErrorConflict, "rule", "numbering rule is frozen", false)
}

type gatewayDocuments struct{}

func gatewayDocumentActor() pluginsdk.WorkflowActor {
	return pluginsdk.WorkflowActor{ID: "user-7", Name: "User Seven"}
}

func gatewayDocumentTime() time.Time { return time.Date(2026, time.July, 22, 8, 0, 0, 0, time.UTC) }

func (gatewayDocuments) Submit(context.Context, pluginsdk.DocumentWorkflowSubmitInput) (pluginsdk.DocumentWorkflowResult, error) {
	return pluginsdk.DocumentWorkflowResult{}, nil
}

type gatewayDocumentWorkflowFailure struct{}

func (gatewayDocumentWorkflowFailure) Submit(context.Context, pluginsdk.DocumentWorkflowSubmitInput) (pluginsdk.DocumentWorkflowResult, error) {
	return pluginsdk.DocumentWorkflowResult{}, pluginsdk.NewDocumentWorkflowError(pluginsdk.DocumentWorkflowErrorTransactionRequired, "transaction", "transaction required", false)
}
func (gatewayDocumentWorkflowFailure) Act(context.Context, pluginsdk.DocumentWorkflowActionInput) (pluginsdk.DocumentWorkflowResult, error) {
	return pluginsdk.DocumentWorkflowResult{}, pluginsdk.NewDocumentWorkflowError(pluginsdk.DocumentWorkflowErrorConflict, "expectedVersion", "document state changed", false)
}
func (gatewayDocumentWorkflowFailure) Get(context.Context, pluginsdk.DocumentWorkflowGetInput) (pluginsdk.DocumentWorkflowResult, error) {
	return pluginsdk.DocumentWorkflowResult{}, pluginsdk.NewDocumentWorkflowError(pluginsdk.DocumentWorkflowErrorNotFound, "documentId", "document not found", false)
}
func (gatewayDocuments) Act(context.Context, pluginsdk.DocumentWorkflowActionInput) (pluginsdk.DocumentWorkflowResult, error) {
	return pluginsdk.DocumentWorkflowResult{}, nil
}
func (gatewayDocuments) Get(context.Context, pluginsdk.DocumentWorkflowGetInput) (pluginsdk.DocumentWorkflowResult, error) {
	return pluginsdk.DocumentWorkflowResult{}, nil
}
func (gatewayDocuments) AddAttachment(_ context.Context, input pluginsdk.DocumentAttachmentAddInput) (pluginsdk.DocumentAttachmentResult, error) {
	return pluginsdk.DocumentAttachmentResult{
		Attachment: pluginsdk.DocumentAttachment{ID: input.AttachmentID, DocumentID: input.DocumentID, File: pluginsdk.FileObject{ID: input.FileID}, AddedBy: gatewayDocumentActor(), AddedAt: gatewayDocumentTime(), AddedSequence: 2},
		Event:      pluginsdk.DocumentTimelineEvent{ID: "attachment-added:" + input.AttachmentID, Sequence: 2, DocumentID: input.DocumentID, Kind: pluginsdk.DocumentTimelineAttachmentAdded, AttachmentID: input.AttachmentID, FileID: input.FileID, Actor: gatewayDocumentActor(), OccurredAt: gatewayDocumentTime()},
	}, nil
}
func (gatewayDocuments) RemoveAttachment(_ context.Context, input pluginsdk.DocumentAttachmentRemoveInput) (pluginsdk.DocumentAttachmentResult, error) {
	removedAt, removedSequence := gatewayDocumentTime(), int64(4)
	return pluginsdk.DocumentAttachmentResult{
		Attachment: pluginsdk.DocumentAttachment{ID: input.AttachmentID, DocumentID: input.DocumentID, File: pluginsdk.FileObject{ID: "file-1"}, AddedBy: gatewayDocumentActor(), AddedAt: gatewayDocumentTime(), AddedSequence: 2, RemovedBy: gatewayDocumentActor(), RemovedAt: &removedAt, RemovedSequence: &removedSequence},
		Event:      pluginsdk.DocumentTimelineEvent{ID: "attachment-removed:" + input.AttachmentID, Sequence: 4, DocumentID: input.DocumentID, Kind: pluginsdk.DocumentTimelineAttachmentRemoved, AttachmentID: input.AttachmentID, FileID: "file-1", Actor: gatewayDocumentActor(), OccurredAt: removedAt},
	}, nil
}
func (gatewayDocuments) ListAttachments(_ context.Context, input pluginsdk.DocumentCollaborationQueryInput) ([]pluginsdk.DocumentAttachment, error) {
	return []pluginsdk.DocumentAttachment{{ID: "attachment-1", DocumentID: input.DocumentID, File: pluginsdk.FileObject{ID: "file-1"}, AddedBy: gatewayDocumentActor(), AddedAt: gatewayDocumentTime(), AddedSequence: 2}}, nil
}
func (gatewayDocuments) AddComment(_ context.Context, input pluginsdk.DocumentCommentAddInput) (pluginsdk.DocumentCommentResult, error) {
	return pluginsdk.DocumentCommentResult{
		Comment: pluginsdk.DocumentComment{ID: input.CommentID, DocumentID: input.DocumentID, Body: input.Body, Author: gatewayDocumentActor(), CreatedAt: gatewayDocumentTime(), Sequence: 3},
		Event:   pluginsdk.DocumentTimelineEvent{ID: "comment-added:" + input.CommentID, Sequence: 3, DocumentID: input.DocumentID, Kind: pluginsdk.DocumentTimelineCommentAdded, CommentID: input.CommentID, Actor: gatewayDocumentActor(), OccurredAt: gatewayDocumentTime()},
	}, nil
}
func (gatewayDocuments) ListComments(_ context.Context, input pluginsdk.DocumentCollaborationQueryInput) ([]pluginsdk.DocumentComment, error) {
	return []pluginsdk.DocumentComment{{ID: "comment-1", DocumentID: input.DocumentID, Body: "Gateway comment", Author: gatewayDocumentActor(), CreatedAt: gatewayDocumentTime(), Sequence: 3}}, nil
}
func (gatewayDocuments) Timeline(_ context.Context, input pluginsdk.DocumentTimelineQueryInput) (pluginsdk.DocumentTimelinePage, error) {
	return pluginsdk.DocumentTimelinePage{Events: []pluginsdk.DocumentTimelineEvent{{ID: "event-1", Sequence: 1, DocumentID: input.DocumentID, Kind: pluginsdk.DocumentTimelineAction, Action: "submit", Actor: gatewayDocumentActor(), OccurredAt: gatewayDocumentTime()}}, NextSequence: 1}, nil
}
func (gatewayDocuments) Search(_ context.Context, _ pluginsdk.DocumentSearchInput) (pluginsdk.DocumentSearchPage, error) {
	return pluginsdk.DocumentSearchPage{Items: []pluginsdk.DocumentSummary{{
		ID: "work-order-1", Type: "work_order", Number: "WO-000001", Title: "Gateway work order", State: "pending", Version: 1,
		CreatedAt: gatewayDocumentTime(), UpdatedAt: gatewayDocumentTime(), CreatedBy: "user-7", UpdatedBy: "user-7",
	}}}, nil
}
func (gatewayDocuments) Print(_ context.Context, input pluginsdk.DocumentPrintInput) (pluginsdk.DocumentPrintPayload, error) {
	return pluginsdk.DocumentPrintPayload{
		Schema:   gatewayDocumentSchema(),
		Document: pluginsdk.DocumentRecord{ID: input.DocumentID, Type: "work_order", SchemaVersion: 1},
		Workflow: pluginsdk.WorkflowInstance{ID: "instance-1", BusinessID: input.DocumentID}, GeneratedAt: gatewayDocumentTime(),
	}, nil
}
func (gatewayDocuments) Export(_ context.Context, input pluginsdk.DocumentExportInput) (pluginsdk.Job, error) {
	return pluginsdk.Job{ID: input.JobID, Kind: pluginsdk.DocumentExportJobKind, Payload: []byte(`{}`), MaxAttempts: 3}, nil
}

func gatewayDocumentSchema() pluginsdk.DocumentSchema {
	return pluginsdk.DocumentSchema{
		Key: "work_order", Name: "Work Order", Version: 1, InitialState: "draft",
		Header:  []pluginsdk.DocumentFieldSchema{{Key: "subject", Label: "Subject", Type: pluginsdk.DocumentFieldString, Required: true}},
		States:  []pluginsdk.DocumentStateSchema{{Key: "draft", Name: "Draft"}, {Key: "pending", Name: "Pending"}, {Key: "approved", Name: "Approved", Terminal: true}},
		Actions: []pluginsdk.DocumentActionSchema{{Key: "submit", Name: "Submit", From: []string{"draft"}, To: "pending"}, {Key: "approve", Name: "Approve", From: []string{"pending"}, To: "approved"}},
	}
}
func (gatewayDocumentWorkflowFailure) AddAttachment(context.Context, pluginsdk.DocumentAttachmentAddInput) (pluginsdk.DocumentAttachmentResult, error) {
	return pluginsdk.DocumentAttachmentResult{}, pluginsdk.NewDocumentWorkflowError(pluginsdk.DocumentWorkflowErrorForbidden, "fileId", "file access denied", false)
}
func (gatewayDocumentWorkflowFailure) RemoveAttachment(context.Context, pluginsdk.DocumentAttachmentRemoveInput) (pluginsdk.DocumentAttachmentResult, error) {
	return pluginsdk.DocumentAttachmentResult{}, pluginsdk.NewDocumentWorkflowError(pluginsdk.DocumentWorkflowErrorNotFound, "attachmentId", "attachment not found", false)
}
func (gatewayDocumentWorkflowFailure) ListAttachments(context.Context, pluginsdk.DocumentCollaborationQueryInput) ([]pluginsdk.DocumentAttachment, error) {
	return nil, pluginsdk.NewDocumentWorkflowError(pluginsdk.DocumentWorkflowErrorForbidden, "documentId", "document access denied", false)
}
func (gatewayDocumentWorkflowFailure) AddComment(context.Context, pluginsdk.DocumentCommentAddInput) (pluginsdk.DocumentCommentResult, error) {
	return pluginsdk.DocumentCommentResult{}, pluginsdk.NewDocumentWorkflowError(pluginsdk.DocumentWorkflowErrorConflict, "commentId", "comment already exists", false)
}
func (gatewayDocumentWorkflowFailure) ListComments(context.Context, pluginsdk.DocumentCollaborationQueryInput) ([]pluginsdk.DocumentComment, error) {
	return nil, pluginsdk.NewDocumentWorkflowError(pluginsdk.DocumentWorkflowErrorForbidden, "documentId", "document access denied", false)
}
func (gatewayDocumentWorkflowFailure) Timeline(context.Context, pluginsdk.DocumentTimelineQueryInput) (pluginsdk.DocumentTimelinePage, error) {
	return pluginsdk.DocumentTimelinePage{}, pluginsdk.NewDocumentWorkflowError(pluginsdk.DocumentWorkflowErrorNotFound, "documentId", "document not found", false)
}
func (gatewayDocumentWorkflowFailure) Search(context.Context, pluginsdk.DocumentSearchInput) (pluginsdk.DocumentSearchPage, error) {
	return pluginsdk.DocumentSearchPage{}, pluginsdk.NewDocumentWorkflowError(pluginsdk.DocumentWorkflowErrorForbidden, "tenantId", "tenant access denied", false)
}
func (gatewayDocumentWorkflowFailure) Print(context.Context, pluginsdk.DocumentPrintInput) (pluginsdk.DocumentPrintPayload, error) {
	return pluginsdk.DocumentPrintPayload{}, pluginsdk.NewDocumentWorkflowError(pluginsdk.DocumentWorkflowErrorForbidden, "sensitivePermission", "sensitive fields denied", false)
}
func (gatewayDocumentWorkflowFailure) Export(context.Context, pluginsdk.DocumentExportInput) (pluginsdk.Job, error) {
	return pluginsdk.Job{}, pluginsdk.NewDocumentWorkflowError(pluginsdk.DocumentWorkflowErrorConflict, "jobId", "export job conflict", false)
}

func (gatewayDocumentNumberFailure) Issue(context.Context, pluginsdk.DocumentNumberInput) (pluginsdk.DocumentNumberResult, error) {
	return pluginsdk.DocumentNumberResult{}, pluginsdk.NewDocumentNumberError(pluginsdk.DocumentNumberErrorTransactionRequired, "transaction", "transaction required", false)
}

type gatewayFiles struct{}

func (gatewayFiles) Store(context.Context, pluginsdk.FileWrite) (pluginsdk.FileObject, error) {
	return pluginsdk.FileObject{ID: "file-1"}, nil
}
func (gatewayFiles) List(context.Context, pluginsdk.FileQuery) ([]pluginsdk.FileObject, error) {
	return []pluginsdk.FileObject{{ID: "file-1"}}, nil
}
func (gatewayFiles) Get(context.Context, string) (pluginsdk.FileObject, error) {
	return pluginsdk.FileObject{ID: "file-1"}, nil
}
func (gatewayFiles) Download(context.Context, string) (pluginsdk.FileDownload, error) {
	return pluginsdk.FileDownload{ID: "file-1"}, nil
}
func (gatewayFiles) Delete(context.Context, string) error { return nil }

type gatewayAudit struct{}

func (gatewayAudit) Record(context.Context, pluginsdk.AuditEntry) (pluginsdk.AuditReceipt, error) {
	return pluginsdk.AuditReceipt{ID: "audit-1"}, nil
}

type gatewayAuditRecorder struct {
	mu         sync.Mutex
	operations []pluginsdk.OperationContext
	entries    []pluginsdk.AuditEntry
}

func (a *gatewayAuditRecorder) Record(ctx context.Context, entry pluginsdk.AuditEntry) (pluginsdk.AuditReceipt, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	operation, _ := pluginsdk.OperationContextFromContext(ctx)
	a.operations = append(a.operations, operation)
	a.entries = append(a.entries, entry)
	return pluginsdk.AuditReceipt{ID: "evidence-1"}, nil
}

type gatewayConfig struct{}

func (gatewayConfig) Get(ctx context.Context) (map[string]any, error) {
	return map[string]any{"transaction": ctx.Value(gatewayTxMarker{}) == true}, nil
}
func (gatewayConfig) Replace(context.Context, map[string]any) (map[string]any, error) {
	return map[string]any{"saved": true}, nil
}

type gatewaySecrets struct {
	mu    sync.Mutex
	value string
}

func (s *gatewaySecrets) Get(context.Context, string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.value, nil
}
func (s *gatewaySecrets) Set(_ context.Context, _ string, value string) error {
	s.mu.Lock()
	s.value = value
	s.mu.Unlock()
	return nil
}

type gatewayWorkflows struct{}

func (gatewayWorkflows) CreateDefinition(context.Context, pluginsdk.WorkflowDefinitionInput) (pluginsdk.WorkflowDefinition, error) {
	return pluginsdk.WorkflowDefinition{ID: "definition-1"}, nil
}
func (gatewayWorkflows) GetDefinition(context.Context, string) (pluginsdk.WorkflowDefinition, error) {
	return pluginsdk.WorkflowDefinition{ID: "definition-1"}, nil
}
func (gatewayWorkflows) PublishDefinition(context.Context, string) (pluginsdk.WorkflowDefinition, error) {
	return pluginsdk.WorkflowDefinition{ID: "definition-1", Status: pluginsdk.WorkflowDefinitionPublished}, nil
}
func (gatewayWorkflows) Start(context.Context, pluginsdk.WorkflowStartInput) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{ID: "instance-1"}, nil
}
func (gatewayWorkflows) GetInstance(context.Context, string) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{ID: "instance-1"}, nil
}
func (gatewayWorkflows) Approve(context.Context, pluginsdk.WorkflowTaskActionInput) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{ID: "instance-1", Status: pluginsdk.WorkflowInstanceApproved}, nil
}
func (gatewayWorkflows) Reject(context.Context, pluginsdk.WorkflowTaskActionInput) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{ID: "instance-1", Status: pluginsdk.WorkflowInstanceRejected}, nil
}
func (gatewayWorkflows) Withdraw(context.Context, pluginsdk.WorkflowInstanceActionInput) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{ID: "instance-1", Status: pluginsdk.WorkflowInstanceWithdrawn}, nil
}
func (gatewayWorkflows) Cancel(context.Context, pluginsdk.WorkflowInstanceActionInput) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{ID: "instance-1", Status: pluginsdk.WorkflowInstanceCanceled}, nil
}
func (gatewayWorkflows) Delegate(context.Context, pluginsdk.WorkflowTargetActionInput) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{ID: "instance-1"}, nil
}
func (gatewayWorkflows) Copy(context.Context, pluginsdk.WorkflowTargetActionInput) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{ID: "instance-1"}, nil
}
func (gatewayWorkflows) CreateSubstitution(context.Context, pluginsdk.WorkflowSubstitutionInput) (pluginsdk.WorkflowSubstitution, error) {
	return pluginsdk.WorkflowSubstitution{ID: "substitution-1"}, nil
}
func (gatewayWorkflows) RevokeSubstitution(context.Context, string) (pluginsdk.WorkflowSubstitution, error) {
	return pluginsdk.WorkflowSubstitution{ID: "substitution-1"}, nil
}

type gatewayJobs struct{}

type gatewayEvents struct{}

func (gatewayEvents) Publish(_ context.Context, in pluginsdk.EventPublication) (pluginsdk.EventEnvelope, error) {
	return pluginsdk.EventEnvelope{
		ID: "event-1", Publisher: "equipment", Name: in.Name, SchemaVersion: in.SchemaVersion,
		PayloadType: "equipment.event", Scope: in.Scope, CorrelationID: in.CorrelationID,
		CausationID: in.CausationID, Subject: in.Subject, Payload: in.Payload, OccurredAt: time.Now().UTC(),
	}, nil
}

type gatewayEventFailure struct{}

func (gatewayEventFailure) Publish(context.Context, pluginsdk.EventPublication) (pluginsdk.EventEnvelope, error) {
	return pluginsdk.EventEnvelope{}, pluginsdk.NewEventError(
		pluginsdk.EventErrorConflict, "idempotencyKey", "event identity conflict", false,
	)
}

func (gatewayJobs) Schedule(context.Context, pluginsdk.JobScheduleInput) (pluginsdk.Job, error) {
	return pluginsdk.Job{ID: "job-1"}, nil
}
func (gatewayJobs) LeaseDue(context.Context, pluginsdk.JobLeaseInput) ([]pluginsdk.Job, error) {
	return []pluginsdk.Job{{ID: "job-1"}}, nil
}
func (gatewayJobs) Complete(context.Context, pluginsdk.JobCompleteInput) (pluginsdk.Job, error) {
	return pluginsdk.Job{ID: "job-1", Status: pluginsdk.JobStatusSucceeded}, nil
}
func (gatewayJobs) Fail(context.Context, pluginsdk.JobFailInput) (pluginsdk.Job, error) {
	return pluginsdk.Job{ID: "job-1", Status: pluginsdk.JobStatusRetryWait}, nil
}
func (gatewayJobs) Get(context.Context, string) (pluginsdk.Job, error) {
	return pluginsdk.Job{ID: "job-1"}, nil
}
func (gatewayJobs) List(context.Context, pluginsdk.JobQuery) ([]pluginsdk.Job, error) {
	return []pluginsdk.Job{{ID: "job-1"}}, nil
}

func gatewayAllHostCapabilities() []pluginsdk.HostCapability {
	return []pluginsdk.HostCapability{
		pluginsdk.HostCapabilityTransactionsWithin,
		pluginsdk.HostCapabilityScopesResolve,
		pluginsdk.HostCapabilityDatastoreQuery,
		pluginsdk.HostCapabilityDatastoreMutate,
		pluginsdk.HostCapabilityDatastoreAggregate,
		pluginsdk.HostCapabilityEventsPublish,
		pluginsdk.HostCapabilityDocumentNumbersPreview,
		pluginsdk.HostCapabilityDocumentNumbersIssue,
		pluginsdk.HostCapabilityDocumentsSubmit,
		pluginsdk.HostCapabilityDocumentsAct,
		pluginsdk.HostCapabilityDocumentsGet,
		pluginsdk.HostCapabilityDocumentsAddAttachment,
		pluginsdk.HostCapabilityDocumentsRemoveAttachment,
		pluginsdk.HostCapabilityDocumentsListAttachments,
		pluginsdk.HostCapabilityDocumentsAddComment,
		pluginsdk.HostCapabilityDocumentsListComments,
		pluginsdk.HostCapabilityDocumentsTimeline,
		pluginsdk.HostCapabilityDocumentsSearch,
		pluginsdk.HostCapabilityDocumentsPrint,
		pluginsdk.HostCapabilityDocumentsExport,
		pluginsdk.HostCapabilityFilesStore,
		pluginsdk.HostCapabilityFilesList,
		pluginsdk.HostCapabilityFilesGet,
		pluginsdk.HostCapabilityFilesDownload,
		pluginsdk.HostCapabilityFilesDelete,
		pluginsdk.HostCapabilityAuditRecord,
		pluginsdk.HostCapabilityConfigGet,
		pluginsdk.HostCapabilityConfigReplace,
		pluginsdk.HostCapabilitySecretsGet,
		pluginsdk.HostCapabilitySecretsSet,
		pluginsdk.HostCapabilityWorkflowsCreateDefinition,
		pluginsdk.HostCapabilityWorkflowsGetDefinition,
		pluginsdk.HostCapabilityWorkflowsPublishDefinition,
		pluginsdk.HostCapabilityWorkflowsStart,
		pluginsdk.HostCapabilityWorkflowsGetInstance,
		pluginsdk.HostCapabilityWorkflowsApprove,
		pluginsdk.HostCapabilityWorkflowsReject,
		pluginsdk.HostCapabilityWorkflowsWithdraw,
		pluginsdk.HostCapabilityWorkflowsCancel,
		pluginsdk.HostCapabilityWorkflowsDelegate,
		pluginsdk.HostCapabilityWorkflowsCopy,
		pluginsdk.HostCapabilityWorkflowsCreateSubstitution,
		pluginsdk.HostCapabilityWorkflowsRevokeSubstitution,
		pluginsdk.HostCapabilityJobsSchedule,
		pluginsdk.HostCapabilityJobsLeaseDue,
		pluginsdk.HostCapabilityJobsComplete,
		pluginsdk.HostCapabilityJobsFail,
		pluginsdk.HostCapabilityJobsGet,
		pluginsdk.HostCapabilityJobsList,
	}
}

func gatewayHostWithCapabilities(capabilities ...pluginsdk.HostCapability) pluginsdk.HostServices {
	return pluginsdk.HostServices{
		PluginID:        "equipment",
		Capabilities:    append([]pluginsdk.HostCapability(nil), capabilities...),
		Transactions:    &gatewayTransactions{},
		DataScopes:      gatewayScopes{},
		DataStore:       gatewayDataStore{},
		Events:          gatewayEvents{},
		DocumentNumbers: gatewayDocumentNumbers{},
		Documents:       gatewayDocuments{},
		Files:           gatewayFiles{},
		Audit:           gatewayAudit{},
		Config:          gatewayConfig{},
		Secrets:         &gatewaySecrets{},
		Workflows:       gatewayWorkflows{},
		Jobs:            gatewayJobs{},
	}
}

func TestHostGatewayDeniesUndeclaredCapabilitiesBeforeRequestDecode(t *testing.T) {
	host := gatewayHostWithCapabilities(pluginsdk.HostCapabilityConfigGet)
	gateway, err := NewHostGateway(func(string) (pluginsdk.HostServices, error) { return host, nil }, "gateway-jwt-secret", newPluginTestQuotaController(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer gateway.Close()
	credential, err := gateway.Issue("equipment")
	if err != nil {
		t.Fatal(err)
	}
	client, err := pluginclient.New(pluginclient.Options{PluginID: "equipment", HostURL: credential.HostURL, HostToken: credential.Token})
	if err != nil {
		t.Fatal(err)
	}
	clientHost, err := client.HostServices()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := clientHost.Config.Get(context.Background()); err != nil {
		t.Fatalf("declared capability was denied: %v", err)
	}

	deniedPaths := []string{
		"/v1/transactions/start",
		"/v1/scopes/resolve",
		"/v1/datastore/query",
		"/v1/events/publish",
		"/v1/document-numbers/issue",
		"/v1/documents/submit",
		"/v1/files/delete",
		"/v1/audit/record",
		"/v1/config/replace",
		"/v1/secrets/get",
		"/v1/workflows/start",
		"/v1/jobs/schedule",
	}
	for _, path := range deniedPaths {
		t.Run(path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, path, strings.NewReader("{"))
			request.RemoteAddr = "127.0.0.1:1234"
			request.Header.Set(pluginclient.AuthorizationHeader, "Bearer "+credential.Token)
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			gateway.ServeHTTP(response, request)
			assertGatewayError(t, response, http.StatusForbidden, "host_capability_denied", "")
		})
	}
}

func TestHostGatewayCredentialCapturesImmutableCapabilitySnapshot(t *testing.T) {
	capabilities := make([]pluginsdk.HostCapability, 1, 2)
	capabilities[0] = pluginsdk.HostCapabilityConfigGet
	host := gatewayHostWithCapabilities(capabilities...)
	gateway, err := NewHostGateway(func(string) (pluginsdk.HostServices, error) { return host, nil }, "gateway-jwt-secret", newPluginTestQuotaController(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer gateway.Close()
	credential, err := gateway.Issue("equipment")
	if err != nil {
		t.Fatal(err)
	}
	host.Capabilities[0] = pluginsdk.HostCapabilitySecretsGet

	allowed := callGatewayJSON(t, gateway, credential.Token, "/v1/config/get", struct{}{})
	if allowed.Code != http.StatusOK {
		t.Fatalf("credential lost issued capability status=%d body=%s", allowed.Code, allowed.Body.String())
	}
	denied := callGatewayJSON(t, gateway, credential.Token, "/v1/secrets/get", map[string]string{"key": "private"})
	assertGatewayError(t, denied, http.StatusForbidden, "host_capability_denied", "")
}

func TestHostGatewayRejectsQuotaExhaustionWithRetryableEvidence(t *testing.T) {
	policy := newPluginTestQuotaController().Policy()
	policy.HostCall = quota.Limit{RatePerSecond: 0.001, Burst: 1, MaxConcurrent: 1}
	quotas, err := quota.NewController(policy)
	if err != nil {
		t.Fatal(err)
	}
	audit := &gatewayAuditRecorder{}
	host := gatewayHostWithCapabilities(pluginsdk.HostCapabilityConfigGet)
	host.Audit = audit
	gateway, err := NewHostGateway(func(string) (pluginsdk.HostServices, error) { return host, nil }, "gateway-jwt-secret", quotas, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = gateway.Close() })
	credential, err := gateway.Issue("equipment")
	if err != nil {
		t.Fatal(err)
	}

	first := callGatewayJSON(t, gateway, credential.Token, "/v1/config/get", struct{}{})
	if first.Code != http.StatusOK {
		t.Fatalf("initial host call status=%d body=%s", first.Code, first.Body.String())
	}
	rejected := callGatewayJSON(t, gateway, credential.Token, "/v1/config/get", struct{}{})
	assertGatewayError(t, rejected, http.StatusTooManyRequests, "plugin_quota_exceeded", "")
	if rejected.Header().Get("Retry-After") == "" || rejected.Header().Get("X-Skoll-Quota-Resource") != string(quota.ResourceHostCall) {
		t.Fatalf("quota response headers=%v", rejected.Header())
	}

	audit.mu.Lock()
	defer audit.mu.Unlock()
	if len(audit.entries) != 2 {
		t.Fatalf("host operation evidence entries=%d want=2", len(audit.entries))
	}
	entry := audit.entries[1]
	if entry.Result != pluginsdk.AuditResultFailure ||
		entry.Detail["errorCode"] != "plugin_quota_exceeded" ||
		entry.Detail["retryable"] != true ||
		entry.Detail["owner"] != "equipment" {
		t.Fatalf("quota rejection evidence=%+v", entry)
	}
}

func TestHostGatewayClientConformanceIdentityAndTransactions(t *testing.T) {
	transactions := &gatewayTransactions{}
	secrets := &gatewaySecrets{}
	host := pluginsdk.HostServices{PluginID: "equipment", Capabilities: gatewayAllHostCapabilities(), Transactions: transactions, DataScopes: gatewayScopes{}, DataStore: gatewayDataStore{}, Events: gatewayEvents{}, DocumentNumbers: gatewayDocumentNumbers{}, Documents: gatewayDocuments{}, Files: gatewayFiles{}, Audit: gatewayAudit{}, Config: gatewayConfig{}, Secrets: secrets, Workflows: gatewayWorkflows{}, Jobs: gatewayJobs{}}
	gateway, err := NewHostGateway(func(id string) (pluginsdk.HostServices, error) {
		if id != "equipment" {
			return pluginsdk.HostServices{}, errors.New("wrong identity")
		}
		return host, nil
	}, "gateway-jwt-secret", newPluginTestQuotaController(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = gateway.Close() })
	credential, err := gateway.Issue("equipment")
	if err != nil {
		t.Fatal(err)
	}
	client, err := pluginclient.New(pluginclient.Options{PluginID: "equipment", HostURL: credential.HostURL, HostToken: credential.Token})
	if err != nil {
		t.Fatal(err)
	}
	services, err := client.HostServices()
	if err != nil {
		t.Fatal(err)
	}

	userToken, err := security.SignJWT("gateway-jwt-secret", security.JWTIdentity{Subject: "user-7", Role: "admin", Roles: []string{"admin"}}, time.Minute, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	ctx := pluginclient.WithUserToken(context.Background(), userToken)
	scope, err := services.DataScopes.Resolve(ctx, pluginsdk.Permission{Resource: "asset", Action: "read"})
	if err != nil || scope.SubjectID() != "user-7" {
		t.Fatalf("scope=%+v err=%v", scope, err)
	}
	page, err := services.DataStore.Query(ctx, basicGatewayQuery())
	if err != nil || len(page.Records) != 1 || page.Records[0].Values["id"].Value != "asset-1" {
		t.Fatalf("datastore page=%+v err=%v", page, err)
	}
	aggregatePage, err := services.DataStore.Aggregate(ctx, basicGatewayAggregateQuery())
	if err != nil || len(aggregatePage.Rows) != 1 || aggregatePage.Rows[0].Values[0].Value != "1" {
		t.Fatalf("datastore aggregate=%+v err=%v", aggregatePage, err)
	}
	eventEnvelope, err := services.Events.Publish(ctx, pluginsdk.EventPublication{
		IdempotencyKey: "equipment-event-1", Name: "equipment-changed", SchemaVersion: 1,
		Scope: pluginsdk.EventScope{TenantID: "tenant-a"}, CorrelationID: "request-1",
		Payload: pluginsdk.EventPayload{"asset_id": {Type: pluginsdk.DataValueString, Value: "asset-1"}},
	})
	if err != nil || eventEnvelope.Publisher != "equipment" || eventEnvelope.Name != "equipment-changed" {
		t.Fatalf("event envelope=%+v err=%v", eventEnvelope, err)
	}
	numberInput := pluginsdk.DocumentNumberInput{
		Rule: pluginsdk.DocumentNumberRule{
			DocumentType: "work_order", Prefix: "WO", Separator: "-", Period: pluginsdk.DocumentNumberPeriodNone,
			Width: 6, Start: 1, GapPolicy: pluginsdk.DocumentNumberGapTransactional,
		},
		TenantID: "tenant-a", Permission: pluginsdk.Permission{Resource: "equipment.work_order", Action: "issue"},
		OccurredAt: time.Date(2026, 7, 22, 0, 0, 0, 0, time.UTC), IdempotencyKey: "work-order-1",
	}
	if preview, previewErr := services.DocumentNumbers.Preview(ctx, numberInput); previewErr != nil || preview.Number != "WO-000001" {
		t.Fatalf("document number preview=%+v err=%v", preview, previewErr)
	}
	if issued, issueErr := services.DocumentNumbers.Issue(ctx, numberInput); issueErr != nil || issued.Number != "WO-000001" {
		t.Fatalf("document number issue=%+v err=%v", issued, issueErr)
	}
	documentPermission := pluginsdk.Permission{Resource: "equipment.work_order", Action: "manage"}
	attachmentInput := pluginsdk.DocumentAttachmentAddInput{
		TenantID: "tenant-a", Permission: documentPermission, DocumentID: "work-order-1", AttachmentID: "attachment-1", FileID: "file-1",
	}
	if added, addErr := services.Documents.AddAttachment(ctx, attachmentInput); addErr != nil || added.Attachment.ID != attachmentInput.AttachmentID || added.Event.Sequence != 2 {
		t.Fatalf("document attachment=%+v err=%v", added, addErr)
	}
	commentInput := pluginsdk.DocumentCommentAddInput{
		TenantID: "tenant-a", Permission: documentPermission, DocumentID: "work-order-1", CommentID: "comment-1", Body: "Gateway comment",
	}
	if commented, commentErr := services.Documents.AddComment(ctx, commentInput); commentErr != nil || commented.Comment.Body != commentInput.Body || commented.Event.Sequence != 3 {
		t.Fatalf("document comment=%+v err=%v", commented, commentErr)
	}
	documentQuery := pluginsdk.DocumentCollaborationQueryInput{TenantID: "tenant-a", Permission: documentPermission, DocumentID: "work-order-1"}
	if attachments, listErr := services.Documents.ListAttachments(ctx, documentQuery); listErr != nil || len(attachments) != 1 {
		t.Fatalf("document attachments=%+v err=%v", attachments, listErr)
	}
	if comments, listErr := services.Documents.ListComments(ctx, documentQuery); listErr != nil || len(comments) != 1 {
		t.Fatalf("document comments=%+v err=%v", comments, listErr)
	}
	if page, timelineErr := services.Documents.Timeline(ctx, pluginsdk.DocumentTimelineQueryInput{TenantID: "tenant-a", Permission: documentPermission, DocumentID: "work-order-1"}); timelineErr != nil || len(page.Events) != 1 || page.NextSequence != 1 {
		t.Fatalf("document timeline=%+v err=%v", page, timelineErr)
	}
	if removed, removeErr := services.Documents.RemoveAttachment(ctx, pluginsdk.DocumentAttachmentRemoveInput{TenantID: "tenant-a", Permission: documentPermission, DocumentID: "work-order-1", AttachmentID: "attachment-1"}); removeErr != nil || removed.Event.Sequence != 4 {
		t.Fatalf("document attachment removal=%+v err=%v", removed, removeErr)
	}
	searchInput := pluginsdk.DocumentSearchInput{
		TenantID: "tenant-a", Permission: documentPermission, Types: []string{"work_order"}, States: []string{"pending"}, Text: "Gateway",
		SortField: pluginsdk.DocumentSearchSortNumber, Direction: pluginsdk.DocumentSearchAscending,
	}
	if page, searchErr := services.Documents.Search(ctx, searchInput); searchErr != nil || len(page.Items) != 1 || page.Items[0].ID != "work-order-1" {
		t.Fatalf("document search=%+v err=%v", page, searchErr)
	}
	if payload, printErr := services.Documents.Print(ctx, pluginsdk.DocumentPrintInput{TenantID: "tenant-a", Permission: documentPermission, DocumentID: "work-order-1"}); printErr != nil || payload.Document.ID != "work-order-1" {
		t.Fatalf("document print=%+v err=%v", payload, printErr)
	}
	exportInput := pluginsdk.DocumentExportInput{JobID: "export-1", IdempotencyKey: "export-1", Search: searchInput, Format: pluginsdk.DocumentExportCSV, MaxRows: 1000}
	if job, exportErr := services.Documents.Export(ctx, exportInput); exportErr != nil || job.ID != exportInput.JobID || job.Kind != pluginsdk.DocumentExportJobKind {
		t.Fatalf("document export=%+v err=%v", job, exportErr)
	}
	if result, mutateErr := services.DataStore.Mutate(ctx, pluginsdk.DataMutation{
		Table: "assets", Operation: pluginsdk.DataMutationDelete,
		Scope: pluginsdk.DataScopeIntent{Permission: pluginsdk.Permission{Resource: "equipment.asset", Action: "delete"}},
		Key:   map[string]pluginsdk.DataValue{"id": {Type: pluginsdk.DataValueString, Value: "asset-1"}}, IdempotencyKey: "delete-asset-1",
	}); mutateErr != nil || result.RowsAffected != 1 {
		t.Fatalf("datastore mutation=%+v err=%v", result, mutateErr)
	}
	if _, err := services.Files.Store(ctx, pluginsdk.FileWrite{Name: "proof.txt"}); err != nil {
		t.Fatal(err)
	}
	if _, err := services.Files.List(ctx, pluginsdk.FileQuery{}); err != nil {
		t.Fatal(err)
	}
	if _, err := services.Files.Get(ctx, "file-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := services.Files.Download(ctx, "file-1"); err != nil {
		t.Fatal(err)
	}
	if err := services.Files.Delete(ctx, "file-1"); err != nil {
		t.Fatal(err)
	}
	if receipt, err := services.Audit.Record(ctx, pluginsdk.AuditEntry{Action: "asset.read", Resource: "asset"}); err != nil || receipt.ID != "audit-1" {
		t.Fatalf("audit=%+v err=%v", receipt, err)
	}
	if err := services.Secrets.Set(ctx, "api.key", "secret"); err != nil {
		t.Fatal(err)
	}
	if value, err := services.Secrets.Get(ctx, "api.key"); err != nil || value != "secret" {
		t.Fatalf("secret=%q err=%v", value, err)
	}
	if values, err := services.Config.Replace(ctx, map[string]any{"enabled": true}); err != nil || values["saved"] != true {
		t.Fatalf("config=%v err=%v", values, err)
	}
	if item, err := services.Workflows.Start(ctx, pluginsdk.WorkflowStartInput{}); err != nil || item.ID != "instance-1" {
		t.Fatalf("workflow=%+v err=%v", item, err)
	}
	definition, err := services.Workflows.CreateDefinition(ctx, pluginsdk.WorkflowDefinitionInput{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = services.Workflows.GetDefinition(ctx, definition.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = services.Workflows.PublishDefinition(ctx, definition.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = services.Workflows.GetInstance(ctx, "instance-1"); err != nil {
		t.Fatal(err)
	}
	if _, err = services.Workflows.Approve(ctx, pluginsdk.WorkflowTaskActionInput{}); err != nil {
		t.Fatal(err)
	}
	if _, err = services.Workflows.Reject(ctx, pluginsdk.WorkflowTaskActionInput{}); err != nil {
		t.Fatal(err)
	}
	if _, err = services.Workflows.Withdraw(ctx, pluginsdk.WorkflowInstanceActionInput{}); err != nil {
		t.Fatal(err)
	}
	if _, err = services.Workflows.Delegate(ctx, pluginsdk.WorkflowTargetActionInput{}); err != nil {
		t.Fatal(err)
	}
	if _, err = services.Workflows.CreateSubstitution(ctx, pluginsdk.WorkflowSubstitutionInput{}); err != nil {
		t.Fatal(err)
	}
	if _, err = services.Workflows.RevokeSubstitution(ctx, "substitution-1"); err != nil {
		t.Fatal(err)
	}
	if _, err = services.Workflows.Copy(ctx, pluginsdk.WorkflowTargetActionInput{}); err != nil {
		t.Fatal(err)
	}
	if item, err := services.Jobs.Schedule(ctx, pluginsdk.JobScheduleInput{}); err != nil || item.ID != "job-1" {
		t.Fatalf("job=%+v err=%v", item, err)
	}
	if _, err = services.Jobs.LeaseDue(ctx, pluginsdk.JobLeaseInput{}); err != nil {
		t.Fatal(err)
	}
	if _, err = services.Jobs.Complete(ctx, pluginsdk.JobCompleteInput{}); err != nil {
		t.Fatal(err)
	}
	if _, err = services.Jobs.Fail(ctx, pluginsdk.JobFailInput{}); err != nil {
		t.Fatal(err)
	}
	if _, err = services.Jobs.Get(ctx, "job-1"); err != nil {
		t.Fatal(err)
	}
	if _, err = services.Jobs.List(ctx, pluginsdk.JobQuery{}); err != nil {
		t.Fatal(err)
	}
	if err := services.Transactions.Within(ctx, func(tx pluginsdk.Transaction) error {
		values, err := services.Config.Get(tx.Context())
		if err != nil {
			return err
		}
		if values["transaction"] != true {
			return errors.New("transaction context missing")
		}
		page, err := services.DataStore.Query(tx.Context(), basicGatewayQuery())
		if err != nil {
			return err
		}
		if len(page.Records) != 1 || page.Records[0].Values["id"].Value != "asset-transaction" {
			return errors.New("datastore transaction context missing")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	rollback := errors.New("rollback proof")
	if err := services.Transactions.Within(ctx, func(pluginsdk.Transaction) error { return rollback }); !errors.Is(err, rollback) {
		t.Fatalf("rollback error=%v", err)
	}
	canceled, cancel := context.WithCancel(ctx)
	if err := services.Transactions.Within(canceled, func(pluginsdk.Transaction) error { cancel(); return context.Canceled }); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled transaction error=%v", err)
	}
	transactions.mu.Lock()
	commits, rollbacks := transactions.commits, transactions.rollbacks
	transactions.mu.Unlock()
	if commits != 1 || rollbacks != 2 {
		t.Fatalf("transactions commits=%d rollbacks=%d", commits, rollbacks)
	}

	gateway.Revoke(credential.Token)
	if _, err := services.Config.Get(ctx); err == nil {
		t.Fatal("revoked plugin credential retained host access")
	}
}

func TestHostGatewayRecordsCorrelatedPayloadFreeOperationEvidence(t *testing.T) {
	audit := &gatewayAuditRecorder{}
	host := gatewayHostWithCapabilities(pluginsdk.HostCapabilityConfigReplace)
	host.Audit = audit
	gateway, err := NewHostGateway(func(string) (pluginsdk.HostServices, error) { return host, nil }, "gateway-jwt-secret", newPluginTestQuotaController(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = gateway.Close() })
	credential, err := gateway.Issue("equipment")
	if err != nil {
		t.Fatal(err)
	}
	client, err := pluginclient.New(pluginclient.Options{PluginID: "equipment", HostURL: credential.HostURL, HostToken: credential.Token})
	if err != nil {
		t.Fatal(err)
	}
	services, err := client.HostServices()
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := pluginsdk.WithOperationContext(context.Background(), pluginsdk.OperationContext{
		CorrelationID: "operation-evidence-42",
		RequestID:     "request-evidence-42",
		TraceID:       "trace-evidence-42",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = services.Config.Replace(ctx, map[string]any{"apiToken": "top-secret", "enabled": true}); err != nil {
		t.Fatal(err)
	}
	audit.mu.Lock()
	defer audit.mu.Unlock()
	if len(audit.entries) != 1 || len(audit.operations) != 1 {
		t.Fatalf("evidence entries=%d operations=%d", len(audit.entries), len(audit.operations))
	}
	if audit.operations[0].CorrelationID != "operation-evidence-42" ||
		audit.operations[0].RequestID != "request-evidence-42" ||
		audit.operations[0].TraceID != "trace-evidence-42" {
		t.Fatalf("operation correlation=%+v", audit.operations[0])
	}
	entry := audit.entries[0]
	if entry.Action != "host.call" ||
		entry.ResourceID != string(pluginsdk.HostCapabilityConfigReplace) ||
		entry.Detail["owner"] != "equipment" ||
		entry.Detail["stage"] != string(pluginsdk.HostCapabilityConfigReplace) ||
		entry.Detail["retryable"] != false {
		t.Fatalf("operation evidence=%+v", entry)
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "top-secret") || strings.Contains(string(raw), "apiToken") {
		t.Fatalf("operation evidence leaked request payload: %s", raw)
	}
}

func TestHostGatewayRejectsMissingCredentialNonLoopbackAndInvalidUserToken(t *testing.T) {
	host := pluginsdk.HostServices{PluginID: "equipment", Capabilities: gatewayAllHostCapabilities(), Transactions: &gatewayTransactions{}, DataScopes: gatewayScopes{}, DataStore: gatewayDataStore{}, Events: gatewayEvents{}, DocumentNumbers: gatewayDocumentNumbers{}, Documents: gatewayDocuments{}, Files: gatewayFiles{}, Audit: gatewayAudit{}, Config: gatewayConfig{}, Secrets: &gatewaySecrets{}, Workflows: gatewayWorkflows{}, Jobs: gatewayJobs{}}
	gateway, err := NewHostGateway(func(string) (pluginsdk.HostServices, error) { return host, nil }, "gateway-jwt-secret", newPluginTestQuotaController(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer gateway.Close()
	credential, err := gateway.Issue("equipment")
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/v1/config/get", strings.NewReader(`{}`))
	request.RemoteAddr = "10.0.0.8:1234"
	request.Header.Set("Authorization", "Bearer "+credential.Token)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	gateway.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("non-loopback status=%d", recorder.Code)
	}
	request = httptest.NewRequest(http.MethodPost, "/v1/config/get", strings.NewReader(`{}`))
	request.RemoteAddr = "127.0.0.1:1234"
	request.Header.Set("Content-Type", "application/json")
	recorder = httptest.NewRecorder()
	gateway.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("missing credential status=%d", recorder.Code)
	}
	request = httptest.NewRequest(http.MethodPost, "/v1/scopes/resolve", strings.NewReader(`{"Resource":"asset","Action":"read"}`))
	request.RemoteAddr = "127.0.0.1:1234"
	request.Header.Set("Authorization", "Bearer "+credential.Token)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(pluginclient.UserTokenHeader, "not-a-jwt")
	recorder = httptest.NewRecorder()
	gateway.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("invalid user token status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var body pluginclient.ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil || body.Code != "host_context_invalid" {
		t.Fatalf("error=%+v decode=%v", body, err)
	}
	request = httptest.NewRequest(http.MethodPost, "/v1/config/get?extra=true", strings.NewReader(`{"unknown":true}`))
	request.RemoteAddr = "127.0.0.1:1234"
	request.Header.Set("Authorization", "Bearer "+credential.Token)
	request.Header.Set("Content-Type", "application/json")
	recorder = httptest.NewRecorder()
	gateway.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("query request status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	request = httptest.NewRequest(http.MethodPost, "/v1/config/get", strings.NewReader(`{"unknown":true}`))
	request.RemoteAddr = "127.0.0.1:1234"
	request.Header.Set("Authorization", "Bearer "+credential.Token)
	request.Header.Set("Content-Type", "application/json")
	recorder = httptest.NewRecorder()
	gateway.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("unknown field status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestHostGatewayPreservesDocumentNumberContractErrors(t *testing.T) {
	host := pluginsdk.HostServices{
		PluginID: "equipment", Capabilities: gatewayAllHostCapabilities(), Transactions: &gatewayTransactions{}, DataScopes: gatewayScopes{}, DataStore: gatewayDataStore{},
		Events:          gatewayEvents{},
		DocumentNumbers: gatewayDocumentNumberFailure{}, Documents: gatewayDocuments{}, Files: gatewayFiles{}, Audit: gatewayAudit{}, Config: gatewayConfig{}, Secrets: &gatewaySecrets{}, Workflows: gatewayWorkflows{}, Jobs: gatewayJobs{},
	}
	gateway, err := NewHostGateway(func(string) (pluginsdk.HostServices, error) { return host, nil }, "gateway-jwt-secret", newPluginTestQuotaController(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer gateway.Close()
	credential, err := gateway.Issue("equipment")
	if err != nil {
		t.Fatal(err)
	}
	client, err := pluginclient.New(pluginclient.Options{PluginID: "equipment", HostURL: credential.HostURL, HostToken: credential.Token})
	if err != nil {
		t.Fatal(err)
	}
	services, err := client.HostServices()
	if err != nil {
		t.Fatal(err)
	}
	userToken, err := security.SignJWT("gateway-jwt-secret", security.JWTIdentity{Subject: "user-7", Role: "admin", Roles: []string{"admin"}}, time.Minute, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	ctx := pluginclient.WithUserToken(context.Background(), userToken)
	input := pluginsdk.DocumentNumberInput{
		Rule: pluginsdk.DocumentNumberRule{
			DocumentType: "work_order", Prefix: "WO", Separator: "-", Period: pluginsdk.DocumentNumberPeriodNone,
			Width: 6, Start: 1, GapPolicy: pluginsdk.DocumentNumberGapTransactional,
		},
		TenantID: "tenant-a", Permission: pluginsdk.Permission{Resource: "equipment.work_order", Action: "issue"},
		OccurredAt: time.Date(2026, 7, 22, 0, 0, 0, 0, time.UTC), IdempotencyKey: "work-order-1",
	}
	_, err = services.DocumentNumbers.Preview(ctx, input)
	var numberErr *pluginsdk.DocumentNumberError
	if !errors.As(err, &numberErr) || numberErr.Code != pluginsdk.DocumentNumberErrorConflict || numberErr.Field != "rule" {
		t.Fatalf("preview error=%v", err)
	}
	_, err = services.DocumentNumbers.Issue(ctx, input)
	if !errors.As(err, &numberErr) || numberErr.Code != pluginsdk.DocumentNumberErrorTransactionRequired || numberErr.Field != "transaction" {
		t.Fatalf("issue error=%v", err)
	}
}

func TestHostGatewayPreservesEventContractErrors(t *testing.T) {
	host := pluginsdk.HostServices{
		PluginID: "equipment", Capabilities: gatewayAllHostCapabilities(), Transactions: &gatewayTransactions{}, DataScopes: gatewayScopes{},
		DataStore: gatewayDataStore{}, Events: gatewayEventFailure{},
		DocumentNumbers: gatewayDocumentNumbers{}, Documents: gatewayDocuments{}, Files: gatewayFiles{},
		Audit: gatewayAudit{}, Config: gatewayConfig{}, Secrets: &gatewaySecrets{},
		Workflows: gatewayWorkflows{}, Jobs: gatewayJobs{},
	}
	gateway, err := NewHostGateway(func(string) (pluginsdk.HostServices, error) { return host, nil }, "gateway-jwt-secret", newPluginTestQuotaController(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer gateway.Close()
	credential, err := gateway.Issue("equipment")
	if err != nil {
		t.Fatal(err)
	}
	client, err := pluginclient.New(pluginclient.Options{PluginID: "equipment", HostURL: credential.HostURL, HostToken: credential.Token})
	if err != nil {
		t.Fatal(err)
	}
	services, err := client.HostServices()
	if err != nil {
		t.Fatal(err)
	}
	_, err = services.Events.Publish(context.Background(), pluginsdk.EventPublication{
		IdempotencyKey: "event-1", Name: "equipment-changed", SchemaVersion: 1,
		Scope: pluginsdk.EventScope{TenantID: "tenant-a"}, CorrelationID: "request-1",
		Payload: pluginsdk.EventPayload{},
	})
	var eventErr *pluginsdk.EventError
	if !errors.As(err, &eventErr) || eventErr.Code != pluginsdk.EventErrorConflict || eventErr.Field != "idempotencyKey" {
		t.Fatalf("event error=%v", err)
	}
}

func TestHostGatewayPreservesDocumentWorkflowContractErrors(t *testing.T) {
	host := pluginsdk.HostServices{
		PluginID: "equipment", Capabilities: gatewayAllHostCapabilities(), Transactions: &gatewayTransactions{}, DataScopes: gatewayScopes{}, DataStore: gatewayDataStore{},
		Events:          gatewayEvents{},
		DocumentNumbers: gatewayDocumentNumbers{}, Documents: gatewayDocumentWorkflowFailure{}, Files: gatewayFiles{}, Audit: gatewayAudit{},
		Config: gatewayConfig{}, Secrets: &gatewaySecrets{}, Workflows: gatewayWorkflows{}, Jobs: gatewayJobs{},
	}
	gateway, err := NewHostGateway(func(string) (pluginsdk.HostServices, error) { return host, nil }, "gateway-jwt-secret", newPluginTestQuotaController(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer gateway.Close()
	credential, err := gateway.Issue("equipment")
	if err != nil {
		t.Fatal(err)
	}
	client, err := pluginclient.New(pluginclient.Options{PluginID: "equipment", HostURL: credential.HostURL, HostToken: credential.Token})
	if err != nil {
		t.Fatal(err)
	}
	services, err := client.HostServices()
	if err != nil {
		t.Fatal(err)
	}
	input := pluginsdk.DocumentWorkflowActionInput{
		TenantID: "tenant-a", Permission: pluginsdk.Permission{Resource: "equipment.work_order", Action: "approve"},
		DocumentID: "work-order-1", Action: pluginsdk.DocumentWorkflowApprove, ExpectedVersion: 1,
		TaskID: "task-1", IdempotencyKey: "approve-work-order-1",
	}
	_, err = services.Documents.Act(context.Background(), input)
	var documentErr *pluginsdk.DocumentWorkflowError
	if !errors.As(err, &documentErr) || documentErr.Code != pluginsdk.DocumentWorkflowErrorConflict || documentErr.Field != "expectedVersion" {
		t.Fatalf("document action error=%v", err)
	}
}

func TestHostGatewayDatastoreFailsClosedAndPreservesContractErrors(t *testing.T) {
	host := pluginsdk.HostServices{
		PluginID: "equipment", Capabilities: gatewayAllHostCapabilities(), Transactions: &gatewayTransactions{}, DataScopes: gatewayScopes{}, DataStore: gatewayDataStore{},
		Events:          gatewayEvents{},
		DocumentNumbers: gatewayDocumentNumbers{}, Documents: gatewayDocuments{}, Files: gatewayFiles{}, Audit: gatewayAudit{}, Config: gatewayConfig{}, Secrets: &gatewaySecrets{}, Workflows: gatewayWorkflows{}, Jobs: gatewayJobs{},
	}
	gateway, err := NewHostGateway(func(pluginID string) (pluginsdk.HostServices, error) {
		if pluginID != "equipment" {
			return pluginsdk.HostServices{}, errors.New("plugin is disabled")
		}
		return host, nil
	}, "gateway-jwt-secret", newPluginTestQuotaController(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer gateway.Close()
	if _, err = gateway.Issue("disabled_plugin"); err == nil {
		t.Fatal("disabled plugin received a host credential")
	}
	credential, err := gateway.Issue("equipment")
	if err != nil {
		t.Fatal(err)
	}

	query := pluginsdk.DataQuery{
		Table: "assets", Fields: []string{"id"},
		Scope: pluginsdk.DataScopeIntent{Permission: pluginsdk.Permission{Resource: "equipment.asset", Action: "read"}},
		Sort:  []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 20},
	}
	response := callGatewayJSON(t, gateway, credential.Token, "/v1/datastore/query", query)
	if response.Code != http.StatusOK {
		t.Fatalf("query status=%d body=%s", response.Code, response.Body.String())
	}
	var page pluginsdk.DataPage
	if err = json.Unmarshal(response.Body.Bytes(), &page); err != nil || len(page.Records) != 1 || page.Records[0].Values["id"].Value != "asset-1" {
		t.Fatalf("page=%+v err=%v", page, err)
	}
	response = callGatewayJSON(t, gateway, credential.Token, "/v1/datastore/aggregate", basicGatewayAggregateQuery())
	if response.Code != http.StatusOK {
		t.Fatalf("aggregate status=%d body=%s", response.Code, response.Body.String())
	}
	var aggregatePage pluginsdk.DataAggregatePage
	if err = json.Unmarshal(response.Body.Bytes(), &aggregatePage); err != nil || len(aggregatePage.Rows) != 1 || aggregatePage.Rows[0].Values[0].Value != "1" {
		t.Fatalf("aggregate page=%+v err=%v", aggregatePage, err)
	}

	mutation := pluginsdk.DataMutation{
		Table: "foreign_assets", Operation: pluginsdk.DataMutationDelete,
		Scope: pluginsdk.DataScopeIntent{Permission: pluginsdk.Permission{Resource: "equipment.asset", Action: "delete"}},
		Key:   map[string]pluginsdk.DataValue{"id": {Type: pluginsdk.DataValueString, Value: "asset-1"}}, IdempotencyKey: "delete-asset-1",
	}
	response = callGatewayJSON(t, gateway, credential.Token, "/v1/datastore/mutate", mutation)
	assertGatewayError(t, response, http.StatusForbidden, "forbidden", "table")

	query.Fields = nil
	response = callGatewayJSON(t, gateway, credential.Token, "/v1/datastore/query", query)
	assertGatewayError(t, response, http.StatusBadRequest, "invalid_request", "fields")
	invalidAggregate := basicGatewayAggregateQuery()
	invalidAggregate.Metrics = nil
	response = callGatewayJSON(t, gateway, credential.Token, "/v1/datastore/aggregate", invalidAggregate)
	assertGatewayError(t, response, http.StatusBadRequest, "invalid_request", "metrics")

	request := httptest.NewRequest(http.MethodPost, "/v1/datastore/query", strings.NewReader(`{}`))
	request.RemoteAddr = "127.0.0.1:1234"
	request.Header.Set("Authorization", "Bearer "+credential.Token)
	request.Header.Set("Content-Type", "application/json")
	request.ContentLength = maxHostRequestBytes + 1
	response = httptest.NewRecorder()
	gateway.ServeHTTP(response, request)
	assertGatewayError(t, response, http.StatusRequestEntityTooLarge, "host_request_too_large", "")

	gateway.Revoke(credential.Token)
	response = callGatewayJSON(t, gateway, credential.Token, "/v1/datastore/query", basicGatewayQuery())
	assertGatewayError(t, response, http.StatusUnauthorized, "plugin_identity_invalid", "")
	response = callGatewayJSON(t, gateway, "", "/v1/datastore/query", basicGatewayQuery())
	assertGatewayError(t, response, http.StatusUnauthorized, "plugin_identity_invalid", "")
}

func basicGatewayQuery() pluginsdk.DataQuery {
	return pluginsdk.DataQuery{
		Table: "assets", Fields: []string{"id"},
		Scope: pluginsdk.DataScopeIntent{Permission: pluginsdk.Permission{Resource: "equipment.asset", Action: "read"}},
		Sort:  []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 20},
	}
}

func basicGatewayAggregateQuery() pluginsdk.DataAggregateQuery {
	return pluginsdk.DataAggregateQuery{
		Table: "assets",
		Scope: pluginsdk.DataScopeIntent{Permission: pluginsdk.Permission{Resource: "equipment.asset", Action: "read"}},
		Metrics: []pluginsdk.DataAggregateMetric{
			{Operation: pluginsdk.DataAggregateCount},
		},
		Page: pluginsdk.DataPageRequest{Limit: 1},
	}
}

func callGatewayJSON(t *testing.T, gateway *HostGateway, token, path string, value any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(string(raw)))
	request.RemoteAddr = "127.0.0.1:1234"
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	gateway.ServeHTTP(response, request)
	return response
}

func assertGatewayError(t *testing.T, response *httptest.ResponseRecorder, status int, code, field string) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status=%d want=%d body=%s", response.Code, status, response.Body.String())
	}
	var failure pluginclient.ErrorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &failure); err != nil || failure.Code != code || failure.Field != field {
		t.Fatalf("failure=%+v err=%v", failure, err)
	}
}
