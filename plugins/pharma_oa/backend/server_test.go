package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

type testTransactionKey struct{}

type testTransaction struct{ ctx context.Context }

func (t testTransaction) Context() context.Context { return t.ctx }

type testTransactions struct {
	mu    sync.Mutex
	calls int
}

func (t *testTransactions) Within(ctx context.Context, fn func(pluginsdk.Transaction) error) error {
	t.mu.Lock()
	t.calls++
	t.mu.Unlock()
	return fn(testTransaction{ctx: context.WithValue(ctx, testTransactionKey{}, true)})
}

type testScopes struct {
	predicate pluginsdk.ScopePredicate
	err       error
}

func (s testScopes) Resolve(context.Context, pluginsdk.Permission) (pluginsdk.ScopePredicate, error) {
	return s.predicate, s.err
}

type testDataStore struct {
	mu          sync.Mutex
	records     map[string]pluginsdk.DataRecord
	tables      map[string]string
	idempotency map[string]string
	scope       employeeScope
	mutations   []pluginsdk.DataMutation
}

func (s *testDataStore) Query(_ context.Context, query pluginsdk.DataQuery) (pluginsdk.DataPage, error) {
	if err := query.Validate(); err != nil {
		return pluginsdk.DataPage{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := make([]string, 0, len(s.records))
	for id := range s.records {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	records := make([]pluginsdk.DataRecord, 0, len(ids))
	for _, id := range ids {
		if s.tables[id] != query.Table {
			continue
		}
		record := s.records[id]
		if dataString(record, "tenant_id") != s.scope.TenantID || dataString(record, "organization_id") != s.scope.OrganizationID || dataString(record, "owner_id") != s.scope.OwnerID {
			continue
		}
		if query.Filter != nil && !testFilterMatches(record, *query.Filter) {
			continue
		}
		records = append(records, testProjectRecord(record, query.Fields))
	}
	start := 0
	if query.Page.Cursor != "" {
		var err error
		start, err = strconv.Atoi(query.Page.Cursor)
		if err != nil || start < 0 || start > len(records) {
			return pluginsdk.DataPage{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorInvalidRequest, "cursor", "cursor is invalid", false)
		}
	}
	end := min(len(records), start+query.Page.Limit)
	nextCursor := ""
	if end < len(records) {
		nextCursor = strconv.Itoa(end)
	}
	return pluginsdk.DataPage{Records: records[start:end], NextCursor: nextCursor, HasMore: nextCursor != ""}, nil
}

func (s *testDataStore) Mutate(ctx context.Context, mutation pluginsdk.DataMutation) (pluginsdk.DataMutationResult, error) {
	if err := mutation.Validate(); err != nil {
		return pluginsdk.DataMutationResult{}, err
	}
	if ctx.Value(testTransactionKey{}) != true {
		return pluginsdk.DataMutationResult{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, "transaction", "mutation escaped transaction", false)
	}
	if len(mutation.Scope.Filter.TenantIDs) != 1 || mutation.Scope.Filter.TenantIDs[0] != s.scope.TenantID ||
		len(mutation.Scope.Filter.OrganizationIDs) != 1 || mutation.Scope.Filter.OrganizationIDs[0] != s.scope.OrganizationID ||
		len(mutation.Scope.Filter.OwnerIDs) != 1 || mutation.Scope.Filter.OwnerIDs[0] != s.scope.OwnerID {
		return pluginsdk.DataMutationResult{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorForbidden, "scope", "write scope is not exact", false)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if id, ok := s.idempotency[mutation.IdempotencyKey]; ok {
		record := testProjectRecord(s.records[id], mutation.Returning)
		return pluginsdk.DataMutationResult{RowsAffected: 1, Record: &record}, nil
	}
	id := mutation.Key["id"].Value
	now := "2026-07-23T08:00:00Z"
	switch mutation.Operation {
	case pluginsdk.DataMutationInsert:
		if _, exists := s.records[id]; exists {
			return pluginsdk.DataMutationResult{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorConflict, "id", "employee exists", false)
		}
		values := testCloneValues(mutation.Values)
		values["id"] = stringValue(id)
		values["tenant_id"] = stringValue(s.scope.TenantID)
		values["organization_id"] = stringValue(s.scope.OrganizationID)
		values["owner_id"] = stringValue(s.scope.OwnerID)
		values["created_at"] = timestampValue(now)
		values["updated_at"] = timestampValue(now)
		s.records[id] = pluginsdk.DataRecord{Values: values, Version: 1}
		s.tables[id] = mutation.Table
	case pluginsdk.DataMutationUpdate:
		record, exists := s.records[id]
		if !exists || s.tables[id] != mutation.Table {
			return pluginsdk.DataMutationResult{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorNotFound, "id", "employee not found", false)
		}
		if mutation.ExpectedVersion == nil || *mutation.ExpectedVersion != record.Version {
			return pluginsdk.DataMutationResult{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorConflict, "version", "employee version is stale", false)
		}
		for field, value := range mutation.Values {
			record.Values[field] = value
		}
		record.Values["updated_at"] = timestampValue(now)
		record.Version++
		s.records[id] = record
	default:
		return pluginsdk.DataMutationResult{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnsupported, "operation", "unsupported mutation", false)
	}
	s.idempotency[mutation.IdempotencyKey] = id
	s.mutations = append(s.mutations, mutation)
	record := testProjectRecord(s.records[id], mutation.Returning)
	return pluginsdk.DataMutationResult{RowsAffected: 1, Record: &record}, nil
}

type testFiles struct {
	mu      sync.Mutex
	items   map[string]pluginsdk.FileObject
	stores  int
	deletes int
}

type testJobs struct {
	mu            sync.Mutex
	items         map[string]pluginsdk.Job
	byIdempotency map[string]string
	scheduleCalls int
}

type testDocumentNumbers struct {
	mu            sync.Mutex
	sequences     map[string]int64
	byIdempotency map[string]pluginsdk.DocumentNumberResult
}

func (n *testDocumentNumbers) Preview(_ context.Context, input pluginsdk.DocumentNumberInput) (pluginsdk.DocumentNumberResult, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	sequence := n.sequences[input.Rule.DocumentType] + 1
	return pluginsdk.DocumentNumberResult{Number: fmt.Sprintf("%s-%06d", input.Rule.Prefix, sequence), Sequence: sequence}, nil
}

func (n *testDocumentNumbers) Issue(ctx context.Context, input pluginsdk.DocumentNumberInput) (pluginsdk.DocumentNumberResult, error) {
	if ctx.Value(testTransactionKey{}) != true {
		return pluginsdk.DocumentNumberResult{}, errors.New("document number escaped transaction")
	}
	if err := input.Validate(true); err != nil {
		return pluginsdk.DocumentNumberResult{}, err
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	key := input.Rule.DocumentType + ":" + input.IdempotencyKey
	if item, exists := n.byIdempotency[key]; exists {
		item.Duplicate = true
		return item, nil
	}
	sequence := n.sequences[input.Rule.DocumentType] + 1
	n.sequences[input.Rule.DocumentType] = sequence
	item := pluginsdk.DocumentNumberResult{Number: fmt.Sprintf("%s-%06d", input.Rule.Prefix, sequence), Sequence: sequence}
	n.byIdempotency[key] = item
	return item, nil
}

type testWorkflows struct {
	mu          sync.Mutex
	definitions map[string]pluginsdk.WorkflowDefinition
	instances   map[string]pluginsdk.WorkflowInstance
}

func (w *testWorkflows) CreateDefinition(_ context.Context, input pluginsdk.WorkflowDefinitionInput) (pluginsdk.WorkflowDefinition, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if item, exists := w.definitions[input.ID]; exists {
		return item, nil
	}
	now := time.Now().UTC()
	item := pluginsdk.WorkflowDefinition{ID: input.ID, Key: input.Key, Name: input.Name, Version: input.Version, Status: pluginsdk.WorkflowDefinitionDraft, Nodes: input.Nodes, Transitions: input.Transitions, CreatedAt: now, UpdatedAt: now}
	w.definitions[item.ID] = item
	return item, nil
}

func (w *testWorkflows) GetDefinition(_ context.Context, id string) (pluginsdk.WorkflowDefinition, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	item, exists := w.definitions[id]
	if !exists {
		return pluginsdk.WorkflowDefinition{}, errors.New("workflow definition not found")
	}
	return item, nil
}

func (w *testWorkflows) PublishDefinition(_ context.Context, id string) (pluginsdk.WorkflowDefinition, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	item, exists := w.definitions[id]
	if !exists {
		return pluginsdk.WorkflowDefinition{}, errors.New("workflow definition not found")
	}
	item.Status, item.UpdatedAt = pluginsdk.WorkflowDefinitionPublished, time.Now().UTC()
	w.definitions[id] = item
	return item, nil
}

func (w *testWorkflows) Start(_ context.Context, input pluginsdk.WorkflowStartInput) (pluginsdk.WorkflowInstance, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if item, exists := w.instances[input.ID]; exists {
		return item, nil
	}
	definition, exists := w.definitions[input.DefinitionID]
	if !exists || definition.Status != pluginsdk.WorkflowDefinitionPublished {
		return pluginsdk.WorkflowInstance{}, errors.New("published workflow definition not found")
	}
	var approval pluginsdk.WorkflowNode
	for _, node := range definition.Nodes {
		if node.Type == pluginsdk.WorkflowNodeApproval {
			approval = node
			break
		}
	}
	if len(approval.AssigneeIDs) != 1 {
		return pluginsdk.WorkflowInstance{}, errors.New("one workflow approver is required")
	}
	now := time.Now().UTC()
	item := pluginsdk.WorkflowInstance{
		ID: input.ID, DefinitionID: input.DefinitionID, DefinitionKey: definition.Key, BusinessType: input.BusinessType, BusinessID: input.BusinessID,
		Title: input.Title, Status: pluginsdk.WorkflowInstanceRunning, CurrentNode: approval.ID, CreatedAt: now, UpdatedAt: now,
		Tasks:    []pluginsdk.WorkflowTask{{ID: input.ID + "-task", InstanceID: input.ID, NodeID: approval.ID, Assignee: pluginsdk.WorkflowActor{ID: approval.AssigneeIDs[0]}, Status: pluginsdk.WorkflowTaskPending, CreatedAt: now}},
		Timeline: []pluginsdk.WorkflowAction{{ID: input.ID + "-start", Type: pluginsdk.WorkflowActionStart, InstanceID: input.ID, NodeID: "start", CreatedAt: now}},
	}
	w.instances[item.ID] = item
	return item, nil
}

func (w *testWorkflows) GetInstance(_ context.Context, id string) (pluginsdk.WorkflowInstance, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	item, exists := w.instances[id]
	if !exists {
		return pluginsdk.WorkflowInstance{}, errors.New("workflow instance not found")
	}
	return item, nil
}

func (w *testWorkflows) Approve(_ context.Context, input pluginsdk.WorkflowTaskActionInput) (pluginsdk.WorkflowInstance, error) {
	return w.taskAction(input, true)
}
func (w *testWorkflows) Reject(_ context.Context, input pluginsdk.WorkflowTaskActionInput) (pluginsdk.WorkflowInstance, error) {
	return w.taskAction(input, false)
}
func (w *testWorkflows) taskAction(input pluginsdk.WorkflowTaskActionInput, approve bool) (pluginsdk.WorkflowInstance, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	item, ok := w.instances[input.InstanceID]
	if !ok {
		return pluginsdk.WorkflowInstance{}, errors.New("workflow instance not found")
	}
	now := time.Now().UTC()
	found := false
	for index := range item.Tasks {
		if item.Tasks[index].ID == input.TaskID && item.Tasks[index].Status == pluginsdk.WorkflowTaskPending {
			found = true
			item.Tasks[index].CompletedAt = &now
			if approve {
				item.Tasks[index].Status = pluginsdk.WorkflowTaskApproved
			} else {
				item.Tasks[index].Status = pluginsdk.WorkflowTaskRejected
			}
			break
		}
	}
	if !found {
		return pluginsdk.WorkflowInstance{}, errors.New("pending workflow task not found")
	}
	action := pluginsdk.WorkflowActionReject
	if approve {
		item.Status, action = pluginsdk.WorkflowInstanceApproved, pluginsdk.WorkflowActionApprove
	} else {
		item.Status = pluginsdk.WorkflowInstanceRejected
	}
	item.Timeline = append(item.Timeline, pluginsdk.WorkflowAction{ID: input.InstanceID + "-" + string(action), Type: action, InstanceID: item.ID, TaskID: input.TaskID, Actor: pluginsdk.WorkflowActor{ID: "actor-1"}, Comment: input.Comment, CreatedAt: now})
	item.UpdatedAt = now
	w.instances[item.ID] = item
	return item, nil
}
func (w *testWorkflows) Withdraw(_ context.Context, input pluginsdk.WorkflowInstanceActionInput) (pluginsdk.WorkflowInstance, error) {
	return w.instanceAction(input, pluginsdk.WorkflowInstanceWithdrawn, pluginsdk.WorkflowActionWithdraw)
}
func (w *testWorkflows) Cancel(_ context.Context, input pluginsdk.WorkflowInstanceActionInput) (pluginsdk.WorkflowInstance, error) {
	return w.instanceAction(input, pluginsdk.WorkflowInstanceCanceled, pluginsdk.WorkflowActionCancel)
}
func (w *testWorkflows) instanceAction(input pluginsdk.WorkflowInstanceActionInput, status pluginsdk.WorkflowInstanceStatus, action pluginsdk.WorkflowActionType) (pluginsdk.WorkflowInstance, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	item, ok := w.instances[input.InstanceID]
	if !ok {
		return pluginsdk.WorkflowInstance{}, errors.New("workflow instance not found")
	}
	now := time.Now().UTC()
	item.Status, item.UpdatedAt = status, now
	for index := range item.Tasks {
		if item.Tasks[index].Status == pluginsdk.WorkflowTaskPending {
			item.Tasks[index].Status = pluginsdk.WorkflowTaskCanceled
			item.Tasks[index].CompletedAt = &now
		}
	}
	item.Timeline = append(item.Timeline, pluginsdk.WorkflowAction{ID: item.ID + "-" + string(action), Type: action, InstanceID: item.ID, Actor: pluginsdk.WorkflowActor{ID: "actor-1"}, Comment: input.Comment, CreatedAt: now})
	w.instances[item.ID] = item
	return item, nil
}
func (w *testWorkflows) Transfer(_ context.Context, input pluginsdk.WorkflowTargetActionInput) (pluginsdk.WorkflowInstance, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	item, ok := w.instances[input.InstanceID]
	if !ok {
		return pluginsdk.WorkflowInstance{}, errors.New("workflow instance not found")
	}
	now := time.Now().UTC()
	found := false
	for index := range item.Tasks {
		if item.Tasks[index].ID == input.TaskID && item.Tasks[index].Status == pluginsdk.WorkflowTaskPending {
			found = true
			item.Tasks[index].Status = pluginsdk.WorkflowTaskTransferred
			item.Tasks[index].CompletedAt = &now
			break
		}
	}
	if !found {
		return pluginsdk.WorkflowInstance{}, errors.New("pending workflow task not found")
	}
	item.Tasks = append(item.Tasks, pluginsdk.WorkflowTask{ID: input.TaskID + "-delegated", InstanceID: item.ID, NodeID: item.CurrentNode, Assignee: input.Target, Status: pluginsdk.WorkflowTaskPending, CreatedAt: now})
	item.Timeline = append(item.Timeline, pluginsdk.WorkflowAction{ID: item.ID + "-transfer", Type: pluginsdk.WorkflowActionTransfer, InstanceID: item.ID, TaskID: input.TaskID, Actor: pluginsdk.WorkflowActor{ID: "actor-1"}, Target: input.Target, Comment: input.Comment, CreatedAt: now})
	item.UpdatedAt = now
	w.instances[item.ID] = item
	return item, nil
}
func (*testWorkflows) Copy(context.Context, pluginsdk.WorkflowTargetActionInput) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{}, errors.New("not implemented")
}

func (j *testJobs) Schedule(_ context.Context, input pluginsdk.JobScheduleInput) (pluginsdk.Job, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if id, exists := j.byIdempotency[input.IdempotencyKey]; exists {
		return j.items[id], nil
	}
	j.scheduleCalls++
	now := time.Now().UTC()
	item := pluginsdk.Job{ID: input.ID, Kind: input.Kind, IdempotencyKey: input.IdempotencyKey, Payload: input.Payload, Status: pluginsdk.JobStatusScheduled, RunAt: input.RunAt, MaxAttempts: input.MaxAttempts, CreatedAt: now, UpdatedAt: now}
	j.items[item.ID] = item
	j.byIdempotency[item.IdempotencyKey] = item.ID
	return item, nil
}

func (j *testJobs) LeaseDue(context.Context, pluginsdk.JobLeaseInput) ([]pluginsdk.Job, error) {
	return nil, nil
}

func (j *testJobs) Complete(_ context.Context, input pluginsdk.JobCompleteInput) (pluginsdk.Job, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	item, exists := j.items[input.JobID]
	if !exists {
		return pluginsdk.Job{}, errors.New("job not found")
	}
	item.Status, item.Result = pluginsdk.JobStatusSucceeded, input.Result
	j.items[item.ID] = item
	return item, nil
}

func (j *testJobs) Fail(_ context.Context, input pluginsdk.JobFailInput) (pluginsdk.Job, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	item, exists := j.items[input.JobID]
	if !exists {
		return pluginsdk.Job{}, errors.New("job not found")
	}
	item.Status, item.LastError = pluginsdk.JobStatusRetryWait, input.Error
	j.items[item.ID] = item
	return item, nil
}

func (j *testJobs) Get(_ context.Context, id string) (pluginsdk.Job, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	item, exists := j.items[id]
	if !exists {
		return pluginsdk.Job{}, errors.New("job not found")
	}
	return item, nil
}

func (j *testJobs) List(_ context.Context, query pluginsdk.JobQuery) ([]pluginsdk.Job, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	items := make([]pluginsdk.Job, 0, len(j.items))
	for _, item := range j.items {
		if query.Kind != "" && item.Kind != query.Kind || query.Status != "" && item.Status != query.Status {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(left, right int) bool { return items[left].ID < items[right].ID })
	if query.Limit > 0 && len(items) > query.Limit {
		items = items[:query.Limit]
	}
	return items, nil
}

func (f *testFiles) Store(_ context.Context, input pluginsdk.FileWrite) (pluginsdk.FileObject, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stores++
	id := "file-" + string(rune('0'+f.stores))
	item := pluginsdk.FileObject{ID: id, Key: input.Key, Name: input.Name, Size: int64(len(input.Content)), MIME: "application/octet-stream", Visibility: input.Visibility, Metadata: input.Metadata}
	f.items[id] = item
	return item, nil
}
func (f *testFiles) List(context.Context, pluginsdk.FileQuery) ([]pluginsdk.FileObject, error) {
	return nil, nil
}
func (f *testFiles) Get(_ context.Context, id string) (pluginsdk.FileObject, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	item, ok := f.items[id]
	if !ok {
		return pluginsdk.FileObject{}, errors.New("file not found")
	}
	return item, nil
}
func (f *testFiles) Download(context.Context, string) (pluginsdk.FileDownload, error) {
	return pluginsdk.FileDownload{}, errors.New("not used")
}
func (f *testFiles) Delete(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.items, id)
	f.deletes++
	return nil
}

type testAudit struct {
	mu      sync.Mutex
	entries []pluginsdk.AuditEntry
}

func (a *testAudit) Record(ctx context.Context, entry pluginsdk.AuditEntry) (pluginsdk.AuditReceipt, error) {
	if ctx.Value(testTransactionKey{}) != true {
		return pluginsdk.AuditReceipt{}, errors.New("audit escaped transaction")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.entries = append(a.entries, entry)
	return pluginsdk.AuditReceipt{ID: "audit", OccurredAt: time.Now()}, nil
}

type testRuntime struct {
	handler      http.Handler
	transactions *testTransactions
	store        *testDataStore
	numbers      *testDocumentNumbers
	files        *testFiles
	audit        *testAudit
	jobs         *testJobs
	workflows    *testWorkflows
	scopes       testScopes
}

func newTestRuntime(t *testing.T) testRuntime {
	t.Helper()
	predicate, err := pluginsdk.NewScopePredicate(pluginsdk.TrustedScope{
		SubjectID: "actor-1", TenantIDs: []string{"tenant-a"}, OwnerIDs: []string{"actor-1"}, OrganizationIDs: []string{"org-a"},
	})
	if err != nil {
		t.Fatal(err)
	}
	transactions := &testTransactions{}
	store := &testDataStore{records: make(map[string]pluginsdk.DataRecord), tables: make(map[string]string), idempotency: make(map[string]string), scope: employeeScope{TenantID: "tenant-a", OrganizationID: "org-a", OwnerID: "actor-1"}}
	numbers := &testDocumentNumbers{sequences: make(map[string]int64), byIdempotency: make(map[string]pluginsdk.DocumentNumberResult)}
	files := &testFiles{items: make(map[string]pluginsdk.FileObject)}
	audit := &testAudit{}
	jobs := &testJobs{items: make(map[string]pluginsdk.Job), byIdempotency: make(map[string]string)}
	workflows := &testWorkflows{definitions: make(map[string]pluginsdk.WorkflowDefinition), instances: make(map[string]pluginsdk.WorkflowInstance)}
	scopes := testScopes{predicate: predicate}
	handler, err := newHandler(pluginsdk.HostServices{
		PluginID: pluginID, Transactions: transactions, DataScopes: scopes, DataStore: store, DocumentNumbers: numbers,
		Files: files, Audit: audit, Workflows: workflows, Jobs: jobs,
	})
	if err != nil {
		t.Fatal(err)
	}
	return testRuntime{handler: handler, transactions: transactions, store: store, numbers: numbers, files: files, audit: audit, jobs: jobs, workflows: workflows, scopes: scopes}
}

func TestFoundationEndpoints(t *testing.T) {
	runtime := newTestRuntime(t)
	health := testRequest(t, runtime.handler, http.MethodGet, "/health", nil, "", false, http.StatusOK)
	if testString(t, health, "status") != "ready" {
		t.Fatalf("unexpected health data: %v", health)
	}
	meta := testRequest(t, runtime.handler, http.MethodGet, apiBase+"/meta", nil, "", false, http.StatusOK)
	if testString(t, meta, "pluginId") != pluginID || testString(t, meta, "contractVersion") != "0.9.0" || len(meta["modules"].([]any)) != 12 || len(meta["documentTypes"].([]any)) != 12 || len(meta["events"].([]any)) != 5 {
		t.Fatalf("unexpected foundation contract: %v", meta)
	}
}

func TestFoundationEventContract(t *testing.T) {
	runtime := newTestRuntime(t)
	for handler, eventName := range foundationEvents {
		testRequest(t, runtime.handler, http.MethodPost, "/_skoll/events", map[string]any{"deliveryId": "delivery-1", "pluginId": pluginID, "handler": handler, "eventName": eventName}, "", false, http.StatusNoContent)
	}
	testRequest(t, runtime.handler, http.MethodPost, "/_skoll/events", map[string]any{"deliveryId": "delivery-2", "pluginId": pluginID, "handler": "privateHandler", "eventName": "approval-completed"}, "", false, http.StatusBadRequest)
}

func TestOARequestDraftUpdateAndSubmitUsesHostWorkflow(t *testing.T) {
	cases := []struct {
		kind string
		form map[string]any
	}{
		{kind: "leave", form: map[string]any{"leaveType": "annual", "startDate": "2026-08-01", "endDate": "2026-08-03"}},
		{kind: "expense", form: map[string]any{"category": "travel", "amount": 1280.50}},
		{kind: "procurement", form: map[string]any{"purpose": "质量检验耗材", "amount": 8600.0}},
		{kind: "contract", form: map[string]any{"counterparty": "华东医药商业有限公司", "amount": 200000.0, "effectiveDate": "2026-08-01"}},
		{kind: "custom", form: map[string]any{"formKey": "office-supplies", "quantity": 3.0}},
	}
	for _, item := range cases {
		t.Run(item.kind, func(t *testing.T) {
			runtime := newTestRuntime(t)
			body := map[string]any{
				"tenantId": "tenant-a", "organizationId": "org-a", "requestType": item.kind, "title": "通用申请-" + item.kind,
				"description": "用于验证公开宿主工作流桥接", "formData": item.form, "approverId": "manager-1", "approverName": "王经理",
			}
			created := testMap(t, testRequest(t, runtime.handler, http.MethodPost, apiBase+"/oa-requests", body, "oa-"+item.kind+"-create", true, http.StatusCreated), "item")
			id := testString(t, created, "id")
			if testString(t, created, "status") != "draft" || testString(t, created, "requestType") != item.kind || testInt64(t, created, "version") != 1 {
				t.Fatalf("unexpected request draft: %v", created)
			}

			body["title"], body["version"] = "已修改-"+item.kind, 1
			updated := testMap(t, testRequest(t, runtime.handler, http.MethodPut, apiBase+"/oa-requests/"+id, body, "oa-"+item.kind+"-update", true, http.StatusOK), "item")
			if testString(t, updated, "title") != "已修改-"+item.kind || testInt64(t, updated, "version") != 2 {
				t.Fatalf("unexpected request update: %v", updated)
			}

			submitted := testRequest(t, runtime.handler, http.MethodPost, apiBase+"/oa-requests/"+id+"/submit", map[string]any{"version": 2}, "oa-"+item.kind+"-submit", true, http.StatusOK)
			submittedItem := testMap(t, submitted, "item")
			workflow := testMap(t, submitted, "workflow")
			if testString(t, submittedItem, "status") != "pending" || testInt64(t, submittedItem, "version") != 3 || testString(t, workflow, "status") != string(pluginsdk.WorkflowInstanceRunning) {
				t.Fatalf("unexpected submitted request: %v", submitted)
			}
			if len(workflow["tasks"].([]any)) != 1 || len(runtime.workflows.definitions) != 1 || len(runtime.workflows.instances) != 1 {
				t.Fatalf("host workflow was not created exactly once: workflow=%v definitions=%v instances=%v", workflow, runtime.workflows.definitions, runtime.workflows.instances)
			}
			duplicate := testRequest(t, runtime.handler, http.MethodPost, apiBase+"/oa-requests/"+id+"/submit", map[string]any{"version": 2}, "oa-"+item.kind+"-submit", true, http.StatusOK)
			if duplicate["duplicate"] != true || len(runtime.workflows.definitions) != 1 || len(runtime.workflows.instances) != 1 {
				t.Fatalf("submit idempotency failed: %v", duplicate)
			}

			detail := testRequest(t, runtime.handler, http.MethodGet, apiBase+"/oa-requests/"+id, nil, "", true, http.StatusOK)
			if testString(t, testMap(t, detail, "item"), "workflowInstanceId") == "" || testString(t, testMap(t, detail, "workflow"), "businessId") != id {
				t.Fatalf("request detail lost workflow binding: %v", detail)
			}
			listed := testRequest(t, runtime.handler, http.MethodGet, apiBase+"/oa-requests?requestType="+item.kind+"&status=pending", nil, "", true, http.StatusOK)
			if testInt64(t, listed, "total") != 1 {
				t.Fatalf("request list did not return submitted item: %v", listed)
			}
			if runtime.transactions.calls != 3 || len(runtime.audit.entries) != 3 {
				t.Fatalf("request operations escaped transaction/audit: transactions=%d audit=%v", runtime.transactions.calls, runtime.audit.entries)
			}
			for index, action := range []string{"pharma_oa.oa_request.create", "pharma_oa.oa_request.update", "pharma_oa.oa_request.submit"} {
				if runtime.audit.entries[index].Action != action {
					t.Fatalf("audit[%d]=%q want=%q", index, runtime.audit.entries[index].Action, action)
				}
			}
		})
	}
}

func TestOARequestRejectsInvalidKindSpecificForms(t *testing.T) {
	cases := []struct {
		kind string
		form map[string]any
	}{
		{kind: "leave", form: map[string]any{"leaveType": "annual", "startDate": "2026-08-03", "endDate": "2026-08-01"}},
		{kind: "expense", form: map[string]any{"category": "travel", "amount": 0}},
		{kind: "procurement", form: map[string]any{"purpose": "", "amount": 100}},
		{kind: "contract", form: map[string]any{"counterparty": "", "amount": 100, "effectiveDate": "bad"}},
		{kind: "custom", form: map[string]any{"formKey": ""}},
	}
	for _, item := range cases {
		t.Run(item.kind, func(t *testing.T) {
			runtime := newTestRuntime(t)
			testRequest(t, runtime.handler, http.MethodPost, apiBase+"/oa-requests", map[string]any{
				"tenantId": "tenant-a", "organizationId": "org-a", "requestType": item.kind, "title": "无效申请", "formData": item.form, "approverId": "manager-1", "approverName": "王经理",
			}, "oa-invalid-"+item.kind, true, http.StatusBadRequest)
			if runtime.transactions.calls != 0 || len(runtime.workflows.instances) != 0 {
				t.Fatalf("invalid request reached persistence: transactions=%d workflows=%v", runtime.transactions.calls, runtime.workflows.instances)
			}
		})
	}
}

func TestOARequestCollaborationAndApprovalLifecycle(t *testing.T) {
	runtime, id, taskID := createSubmittedOARequest(t)
	attached := testRequest(t, runtime.handler, http.MethodPost, apiBase+"/oa-requests/"+id+"/attachments", map[string]any{"name": "凭证.pdf", "contentBase64": base64.StdEncoding.EncodeToString([]byte("attachment")), "version": 2}, "oa-attach-1", true, http.StatusOK)
	if len(testMap(t, attached, "item")["attachments"].([]any)) != 1 {
		t.Fatalf("attachment missing: %v", attached)
	}
	commented := testRequest(t, runtime.handler, http.MethodPost, apiBase+"/oa-requests/"+id+"/comments", map[string]any{"content": "请优先处理", "version": 3}, "oa-comment-1", true, http.StatusOK)
	if len(testMap(t, commented, "item")["comments"].([]any)) != 1 {
		t.Fatalf("comment missing: %v", commented)
	}
	reminded := testRequest(t, runtime.handler, http.MethodPost, apiBase+"/oa-requests/"+id+"/reminders", map[string]any{"runAt": time.Now().UTC().Add(time.Hour).Format(time.RFC3339), "version": 4}, "oa-remind-1", true, http.StatusOK)
	if testString(t, testMap(t, reminded, "item"), "reminderAt") == "" || len(runtime.jobs.items) != 1 {
		t.Fatalf("reminder missing: %v jobs=%v", reminded, runtime.jobs.items)
	}
	delegated := testRequest(t, runtime.handler, http.MethodPost, apiBase+"/oa-requests/"+id+"/delegate", map[string]any{"taskId": taskID, "targetId": "manager-2", "targetName": "赵经理", "comment": "转交复核", "version": 5}, "oa-delegate-1", true, http.StatusOK)
	if testString(t, testMap(t, delegated, "item"), "approverId") != "manager-2" {
		t.Fatalf("delegate failed: %v", delegated)
	}
	workflow := testMap(t, delegated, "workflow")
	tasks := workflow["tasks"].([]any)
	delegatedTask := tasks[len(tasks)-1].(map[string]any)
	approved := testRequest(t, runtime.handler, http.MethodPost, apiBase+"/oa-requests/"+id+"/approve", map[string]any{"taskId": delegatedTask["id"], "comment": "同意", "version": 6}, "oa-approve-1", true, http.StatusOK)
	if testString(t, testMap(t, approved, "item"), "status") != "approved" || testString(t, testMap(t, approved, "workflow"), "status") != "approved" {
		t.Fatalf("approve failed: %v", approved)
	}
	if runtime.files.stores != 1 || runtime.transactions.calls != 7 {
		t.Fatalf("collaboration host services mismatch: files=%d transactions=%d", runtime.files.stores, runtime.transactions.calls)
	}
}

func TestOARequestTerminalActions(t *testing.T) {
	for _, testCase := range []struct {
		action, want string
		task         bool
	}{{"reject", "rejected", true}, {"withdraw", "withdrawn", false}, {"cancel", "canceled", false}} {
		t.Run(testCase.action, func(t *testing.T) {
			runtime, id, taskID := createSubmittedOARequest(t)
			body := map[string]any{"comment": "终止流程", "version": 2}
			if testCase.task {
				body["taskId"] = taskID
			}
			result := testRequest(t, runtime.handler, http.MethodPost, apiBase+"/oa-requests/"+id+"/"+testCase.action, body, "oa-"+testCase.action+"-1", true, http.StatusOK)
			if testString(t, testMap(t, result, "item"), "status") != testCase.want || testString(t, testMap(t, result, "workflow"), "status") != testCase.want {
				t.Fatalf("%s failed: %v", testCase.action, result)
			}
		})
	}
}

func createSubmittedOARequest(t *testing.T) (testRuntime, string, string) {
	t.Helper()
	runtime := newTestRuntime(t)
	created := testMap(t, testRequest(t, runtime.handler, http.MethodPost, apiBase+"/oa-requests", map[string]any{"tenantId": "tenant-a", "organizationId": "org-a", "requestType": "expense", "title": "差旅报销", "formData": map[string]any{"category": "travel", "amount": 1280.0}, "approverId": "manager-1", "approverName": "王经理"}, "oa-lifecycle-create", true, http.StatusCreated), "item")
	id := testString(t, created, "id")
	submitted := testRequest(t, runtime.handler, http.MethodPost, apiBase+"/oa-requests/"+id+"/submit", map[string]any{"version": 1}, "oa-lifecycle-submit", true, http.StatusOK)
	tasks := testMap(t, submitted, "workflow")["tasks"].([]any)
	return runtime, id, tasks[0].(map[string]any)["id"].(string)
}

func TestEmployeeLifecycleUsesPublicScopedHostServices(t *testing.T) {
	runtime := newTestRuntime(t)
	testRequest(t, runtime.handler, http.MethodGet, apiBase+"/employees", nil, "", false, http.StatusUnauthorized)
	createBody := map[string]any{
		"tenantId": "tenant-a", "organizationId": "org-a", "code": "EMP-001", "name": "李华", "departmentId": "sales", "positionId": "manager", "phone": "13800000000", "email": "lihua@example.com", "hireDate": "2026-01-02",
		"certificates": []map[string]any{{"id": "cert-1", "name": "执业药师证", "number": "ZY-001", "expiresAt": "2000-01-01"}},
	}
	testRequest(t, runtime.handler, http.MethodPost, apiBase+"/employees", createBody, "", true, http.StatusBadRequest)
	created := testRequest(t, runtime.handler, http.MethodPost, apiBase+"/employees", createBody, "employee-create-1", true, http.StatusCreated)
	item := testMap(t, created, "item")
	id := testString(t, item, "id")
	if testString(t, item, "employmentStatus") != "active" || testInt64(t, item, "version") != 1 {
		t.Fatalf("unexpected created employee: %v", item)
	}

	listed := testRequest(t, runtime.handler, http.MethodGet, apiBase+"/employees?keyword=%E6%9D%8E&status=active", nil, "", true, http.StatusOK)
	if testInt64(t, listed, "total") != 1 {
		t.Fatalf("unexpected employee list: %v", listed)
	}

	updateBody := map[string]any{
		"code": "EMP-001", "name": "李华", "departmentId": "quality", "positionId": "director", "phone": "13800000000", "email": "lihua@example.com", "hireDate": testString(t, item, "hireDate"), "version": 1,
		"certificates": createBody["certificates"],
	}
	updated := testRequest(t, runtime.handler, http.MethodPut, apiBase+"/employees/"+id, updateBody, "employee-update-1", true, http.StatusOK)
	item = testMap(t, updated, "item")
	if testString(t, item, "departmentId") != "quality" || testInt64(t, item, "version") != 2 {
		t.Fatalf("unexpected updated employee: %v", item)
	}
	testRequest(t, runtime.handler, http.MethodPut, apiBase+"/employees/"+id, updateBody, "employee-update-stale", true, http.StatusConflict)

	attachment := testRequest(t, runtime.handler, http.MethodPost, apiBase+"/employees/"+id+"/attachments", map[string]any{
		"name": "certificate.pdf", "contentBase64": base64.StdEncoding.EncodeToString([]byte("owned employee document")), "version": 2,
	}, "employee-attach-1", true, http.StatusOK)
	item = testMap(t, attachment, "item")
	if testInt64(t, item, "version") != 3 || attachment["duplicate"] != false {
		t.Fatalf("unexpected attachment result: %v", attachment)
	}
	duplicate := testRequest(t, runtime.handler, http.MethodPost, apiBase+"/employees/"+id+"/attachments", map[string]any{
		"name": "certificate.pdf", "contentBase64": base64.StdEncoding.EncodeToString([]byte("owned employee document")), "version": 2,
	}, "employee-attach-1", true, http.StatusOK)
	if duplicate["duplicate"] != true || runtime.files.stores != 1 {
		t.Fatalf("attachment idempotency failed: response=%v stores=%d", duplicate, runtime.files.stores)
	}

	reminders := testRequest(t, runtime.handler, http.MethodGet, apiBase+"/employees/qualification-reminders?days=30", nil, "", true, http.StatusOK)
	if len(reminders["items"].([]any)) != 1 {
		t.Fatalf("unexpected qualification reminders: %v", reminders)
	}
	left := testRequest(t, runtime.handler, http.MethodPost, apiBase+"/employees/"+id+"/leave", map[string]any{"reason": "合同到期", "version": 3}, "employee-leave-1", true, http.StatusOK)
	if testString(t, testMap(t, left, "item"), "employmentStatus") != "left" {
		t.Fatalf("unexpected leave result: %v", left)
	}
	reminders = testRequest(t, runtime.handler, http.MethodGet, apiBase+"/employees/qualification-reminders?days=30", nil, "", true, http.StatusOK)
	if len(reminders["items"].([]any)) != 0 {
		t.Fatalf("departed employee retained reminder: %v", reminders)
	}

	wantActions := []string{"pharma_oa.employee.create", "pharma_oa.employee.update", "pharma_oa.employee.attach", "pharma_oa.employee.leave"}
	if len(runtime.audit.entries) != len(wantActions) || runtime.transactions.calls != len(wantActions) {
		t.Fatalf("transaction/audit coverage mismatch: transactions=%d audit=%v", runtime.transactions.calls, runtime.audit.entries)
	}
	for index, action := range wantActions {
		if runtime.audit.entries[index].Action != action {
			t.Fatalf("audit[%d]=%q want=%q", index, runtime.audit.entries[index].Action, action)
		}
	}
	for _, mutation := range runtime.store.mutations {
		if mutation.Scope.Filter.TenantIDs[0] != "tenant-a" || mutation.Scope.Filter.OrganizationIDs[0] != "org-a" || mutation.Scope.Filter.OwnerIDs[0] != "actor-1" {
			t.Fatalf("mutation escaped exact trusted scope: %+v", mutation.Scope.Filter)
		}
	}
}

func TestEmployeeCreateRejectsDeniedScope(t *testing.T) {
	runtime := newTestRuntime(t)
	handler, err := newHandler(pluginsdk.HostServices{
		PluginID: pluginID, Transactions: runtime.transactions, DataScopes: testScopes{predicate: pluginsdk.NewDeniedScopePredicate("actor-1")},
		DataStore: runtime.store, DocumentNumbers: runtime.numbers, Files: runtime.files, Audit: runtime.audit, Workflows: runtime.workflows, Jobs: runtime.jobs,
	})
	if err != nil {
		t.Fatal(err)
	}
	testRequest(t, handler, http.MethodPost, apiBase+"/employees", map[string]any{"code": "EMP-002", "name": "王芳", "departmentId": "sales", "positionId": "staff"}, "employee-denied-1", true, http.StatusForbidden)
}

func TestCustomerAndSupplierMasterDataLifecycle(t *testing.T) {
	runtime := newTestRuntime(t)
	base := map[string]any{
		"tenantId": "tenant-a", "organizationId": "org-a", "code": "CUS-001", "name": "华东医药商业有限公司",
		"unifiedSocialCreditCode": "91330000MA000001", "region": "浙江省杭州市", "rating": 5,
		"contacts":        []map[string]any{{"id": "contact-1", "name": "陈经理", "title": "采购经理", "phone": "13800000001", "email": "buyer@example.com", "primary": true}},
		"addresses":       []map[string]any{{"id": "address-1", "label": "总部", "province": "浙江省", "city": "杭州市", "district": "拱墅区", "detail": "康桥路 1 号", "default": true}},
		"settlementTerms": map[string]any{"currency": "CNY", "paymentDays": 30, "creditLimit": 500000},
	}
	created := testRequest(t, runtime.handler, http.MethodPost, apiBase+"/customers", base, "customer-create-1", true, http.StatusCreated)
	item := testMap(t, created, "item")
	id := testString(t, item, "id")
	if testString(t, item, "type") != "customer" || testString(t, item, "status") != "active" || testInt64(t, item, "version") != 1 {
		t.Fatalf("unexpected customer: %v", item)
	}
	testRequest(t, runtime.handler, http.MethodPost, apiBase+"/customers", base, "customer-duplicate-1", true, http.StatusConflict)
	listed := testRequest(t, runtime.handler, http.MethodGet, apiBase+"/customers?keyword=%E5%8D%8E%E4%B8%9C&status=active", nil, "", true, http.StatusOK)
	if testInt64(t, listed, "total") != 1 {
		t.Fatalf("unexpected customer list: %v", listed)
	}
	base["name"], base["rating"], base["version"] = "华东医药商业集团", 4, 1
	updated := testRequest(t, runtime.handler, http.MethodPut, apiBase+"/customers/"+id, base, "customer-update-1", true, http.StatusOK)
	if testString(t, testMap(t, updated, "item"), "name") != "华东医药商业集团" {
		t.Fatalf("customer update failed: %v", updated)
	}
	disabled := testRequest(t, runtime.handler, http.MethodPost, apiBase+"/customers/"+id+"/disable", map[string]any{"reason": "合作暂停", "version": 2}, "customer-disable-1", true, http.StatusOK)
	if testString(t, testMap(t, disabled, "item"), "status") != "disabled" {
		t.Fatalf("customer disable failed: %v", disabled)
	}
	enabled := testRequest(t, runtime.handler, http.MethodPost, apiBase+"/customers/"+id+"/enable", map[string]any{"version": 3}, "customer-enable-1", true, http.StatusOK)
	if testString(t, testMap(t, enabled, "item"), "status") != "active" {
		t.Fatalf("customer enable failed: %v", enabled)
	}

	supplier := map[string]any{}
	for key, value := range base {
		supplier[key] = value
	}
	supplier["code"], supplier["name"], supplier["unifiedSocialCreditCode"] = "SUP-001", "国药器械供应有限公司", "91330000MA000002"
	delete(supplier, "version")
	createdSupplier := testRequest(t, runtime.handler, http.MethodPost, apiBase+"/suppliers", supplier, "supplier-create-1", true, http.StatusCreated)
	if testString(t, testMap(t, createdSupplier, "item"), "type") != "supplier" {
		t.Fatalf("unexpected supplier: %v", createdSupplier)
	}
	wantActions := []string{"pharma_oa.customer.create", "pharma_oa.customer.update", "pharma_oa.customer.disable", "pharma_oa.customer.enable", "pharma_oa.supplier.create"}
	if len(runtime.audit.entries) != len(wantActions) {
		t.Fatalf("party audit count=%d want=%d", len(runtime.audit.entries), len(wantActions))
	}
	for index, action := range wantActions {
		if runtime.audit.entries[index].Action != action {
			t.Fatalf("party audit[%d]=%q want=%q", index, runtime.audit.entries[index].Action, action)
		}
	}
}

func TestProductCatalogLifecycleAndReferenceGuards(t *testing.T) {
	runtime := newTestRuntime(t)
	category := testRequest(t, runtime.handler, http.MethodPost, apiBase+"/categories", map[string]any{
		"tenantId": "tenant-a", "organizationId": "org-a", "code": "RX", "name": "处方药", "description": "处方药分类",
	}, "category-create-1", true, http.StatusCreated)
	categoryItem := testMap(t, category, "item")
	categoryID := testString(t, categoryItem, "id")

	unit := testRequest(t, runtime.handler, http.MethodPost, apiBase+"/units", map[string]any{
		"tenantId": "tenant-a", "organizationId": "org-a", "code": "BOX", "name": "盒", "description": "销售包装", "symbol": "盒", "decimalPlaces": 0,
	}, "unit-create-1", true, http.StatusCreated)
	unitItem := testMap(t, unit, "item")
	unitID := testString(t, unitItem, "id")

	manufacturer := testRequest(t, runtime.handler, http.MethodPost, apiBase+"/manufacturers", map[string]any{
		"tenantId": "tenant-a", "organizationId": "org-a", "code": "MFG-001", "name": "华东制药有限公司", "description": "药品生产企业", "unifiedSocialCreditCode": "91330000MA100001", "licenseNumber": "浙20260001",
	}, "manufacturer-create-1", true, http.StatusCreated)
	manufacturerItem := testMap(t, manufacturer, "item")
	manufacturerID := testString(t, manufacturerItem, "id")

	productBody := map[string]any{
		"tenantId": "tenant-a", "organizationId": "org-a", "code": "MED-001", "sku": "SKU-001", "name": "阿莫西林胶囊", "genericName": "阿莫西林", "categoryId": categoryID, "unitId": unitID, "manufacturerId": manufacturerID,
		"dosageForm": "胶囊剂", "specification": "0.25g*24粒", "approvalNumber": "国药准字H20260001", "barcode": "690000000001", "storageCondition": "密封，阴凉干燥处保存", "temperatureMin": 2, "temperatureMax": 25,
	}
	invalidReference := make(map[string]any, len(productBody))
	for key, value := range productBody {
		invalidReference[key] = value
	}
	invalidReference["categoryId"] = "category-missing"
	testRequest(t, runtime.handler, http.MethodPost, apiBase+"/products", invalidReference, "product-invalid-reference-1", true, http.StatusUnprocessableEntity)

	created := testRequest(t, runtime.handler, http.MethodPost, apiBase+"/products", productBody, "product-create-1", true, http.StatusCreated)
	productItem := testMap(t, created, "item")
	productID := testString(t, productItem, "id")
	if testString(t, productItem, "status") != "active" || testString(t, productItem, "sku") != "SKU-001" || testInt64(t, productItem, "version") != 1 {
		t.Fatalf("unexpected product: %v", productItem)
	}
	testRequest(t, runtime.handler, http.MethodPost, apiBase+"/products", productBody, "product-duplicate-1", true, http.StatusConflict)
	listed := testRequest(t, runtime.handler, http.MethodGet, apiBase+"/products?keyword=%E9%98%BF%E8%8E%AB&status=active&limit=50", nil, "", true, http.StatusOK)
	if len(listed["items"].([]any)) != 1 {
		t.Fatalf("unexpected product list: %v", listed)
	}

	testRequest(t, runtime.handler, http.MethodPost, apiBase+"/units/"+unitID+"/disable", map[string]any{"reason": "停用计量单位", "version": 1}, "unit-disable-blocked-1", true, http.StatusConflict)
	disabled := testRequest(t, runtime.handler, http.MethodPost, apiBase+"/products/"+productID+"/disable", map[string]any{"reason": "暂停销售", "version": 1}, "product-disable-1", true, http.StatusOK)
	if testString(t, testMap(t, disabled, "item"), "status") != "disabled" {
		t.Fatalf("product disable failed: %v", disabled)
	}
	testRequest(t, runtime.handler, http.MethodPost, apiBase+"/units/"+unitID+"/disable", map[string]any{"reason": "停用计量单位", "version": 1}, "unit-disable-1", true, http.StatusOK)
	testRequest(t, runtime.handler, http.MethodPost, apiBase+"/units/"+unitID+"/enable", map[string]any{"version": 2}, "unit-enable-1", true, http.StatusOK)
	testRequest(t, runtime.handler, http.MethodPost, apiBase+"/products/"+productID+"/enable", map[string]any{"version": 2}, "product-enable-1", true, http.StatusOK)

	wantActions := []string{"pharma_oa.category.create", "pharma_oa.unit.create", "pharma_oa.manufacturer.create", "pharma_oa.product.create", "pharma_oa.product.disable", "pharma_oa.unit.disable", "pharma_oa.unit.enable", "pharma_oa.product.enable"}
	if len(runtime.audit.entries) != len(wantActions) {
		t.Fatalf("catalog audit count=%d want=%d entries=%v", len(runtime.audit.entries), len(wantActions), runtime.audit.entries)
	}
	for index, action := range wantActions {
		entry := runtime.audit.entries[index]
		if entry.Action != action || entry.Resource != action[:strings.LastIndex(action, ".")] {
			t.Fatalf("catalog audit[%d]=%+v want action=%q", index, entry, action)
		}
	}
}

func TestCatalogListUsesBoundedCursorPages(t *testing.T) {
	runtime := newTestRuntime(t)
	for index := 0; index < 205; index++ {
		id := fmt.Sprintf("category-%03d", index)
		runtime.store.records[id] = pluginsdk.DataRecord{Values: map[string]pluginsdk.DataValue{
			"id": stringValue(id), "catalog_type": stringValue("category"), "code": stringValue(fmt.Sprintf("CAT-%03d", index)), "name": stringValue(fmt.Sprintf("分类 %03d", index)), "description": stringValue("性能验收数据"), "parent_id": nullableStringValue(""), "symbol": nullableStringValue(""), "decimal_places": integerValue(0), "unified_social_credit_code": nullableStringValue(""), "license_number": nullableStringValue(""), "status": stringValue("active"), "disable_reason": nullableStringValue(""), "tenant_id": stringValue("tenant-a"), "organization_id": stringValue("org-a"), "owner_id": stringValue("actor-1"), "created_at": timestampValue("2026-07-23T08:00:00Z"), "updated_at": timestampValue("2026-07-23T08:00:00Z"),
		}, Version: 1}
		runtime.store.tables[id] = catalogTable
	}
	first := testRequest(t, runtime.handler, http.MethodGet, apiBase+"/categories?limit=200", nil, "", true, http.StatusOK)
	if len(first["items"].([]any)) != 200 || !testMap(t, first, "pageInfo")["hasMore"].(bool) || testString(t, testMap(t, first, "pageInfo"), "nextCursor") != "200" {
		t.Fatalf("unexpected first catalog page: %v", first["pageInfo"])
	}
	second := testRequest(t, runtime.handler, http.MethodGet, apiBase+"/categories?limit=200&cursor=200", nil, "", true, http.StatusOK)
	if len(second["items"].([]any)) != 5 || testMap(t, second, "pageInfo")["hasMore"].(bool) {
		t.Fatalf("unexpected second catalog page: %v", second["pageInfo"])
	}
}

func TestCategoryHierarchyRejectsCycles(t *testing.T) {
	runtime := newTestRuntime(t)
	rootBody := map[string]any{"tenantId": "tenant-a", "organizationId": "org-a", "code": "ROOT", "name": "药品", "description": "根分类"}
	root := testMap(t, testRequest(t, runtime.handler, http.MethodPost, apiBase+"/categories", rootBody, "category-root-1", true, http.StatusCreated), "item")
	childBody := map[string]any{"tenantId": "tenant-a", "organizationId": "org-a", "code": "CHILD", "name": "处方药", "description": "子分类", "parentId": testString(t, root, "id")}
	child := testMap(t, testRequest(t, runtime.handler, http.MethodPost, apiBase+"/categories", childBody, "category-child-1", true, http.StatusCreated), "item")
	rootBody["parentId"] = testString(t, child, "id")
	rootBody["version"] = 1
	testRequest(t, runtime.handler, http.MethodPut, apiBase+"/categories/"+testString(t, root, "id"), rootBody, "category-cycle-1", true, http.StatusConflict)
}

func TestQualificationLifecycleEligibilityAndExpiryIdempotency(t *testing.T) {
	runtime := newTestRuntime(t)
	customer := testMap(t, testRequest(t, runtime.handler, http.MethodPost, apiBase+"/customers", map[string]any{
		"tenantId": "tenant-a", "organizationId": "org-a", "code": "CUS-QA-001", "name": "华东合规药房", "unifiedSocialCreditCode": "91330000MAQA0001", "region": "浙江省杭州市", "rating": 5,
		"contacts":        []map[string]any{{"name": "质量负责人", "phone": "13800000001", "email": "qa@example.com", "primary": true}},
		"addresses":       []map[string]any{{"label": "总部", "province": "浙江省", "city": "杭州市", "district": "拱墅区", "detail": "康桥路 8 号", "default": true}},
		"settlementTerms": map[string]any{"currency": "CNY", "paymentDays": 30, "creditLimit": 100000},
	}, "qualification-customer-1", true, http.StatusCreated), "item")
	customerID := testString(t, customer, "id")

	typeItem := testMap(t, testRequest(t, runtime.handler, http.MethodPost, apiBase+"/qualification-types", map[string]any{
		"tenantId": "tenant-a", "organizationId": "org-a", "code": "DRUG-BUSINESS", "name": "药品经营许可证", "subjectType": "customer", "businessGate": "sales", "description": "客户销售业务准入证照", "validityDays": 365, "alertDays": 30, "evidenceRequired": true, "businessRequired": true,
	}, "qualification-type-create-1", true, http.StatusCreated), "item")
	typeID := testString(t, typeItem, "id")

	now := time.Now().UTC()
	qualificationBody := map[string]any{
		"tenantId": "tenant-a", "organizationId": "org-a", "typeId": typeID, "subjectType": "customer", "subjectId": customerID, "certificateNumber": "浙药经许-2026-001", "issuer": "浙江省药品监督管理局",
		"validFrom": now.AddDate(0, 0, -30).Format("2006-01-02"), "validTo": now.AddDate(0, 0, 10).Format("2006-01-02"),
		"evidence": map[string]any{"name": "license.txt", "contentBase64": base64.StdEncoding.EncodeToString([]byte("not a license"))},
	}
	testRequest(t, runtime.handler, http.MethodPost, apiBase+"/qualifications", qualificationBody, "qualification-invalid-file-1", true, http.StatusBadRequest)
	qualificationBody["evidence"] = map[string]any{"name": "license.pdf", "contentBase64": base64.StdEncoding.EncodeToString([]byte("%PDF-1.4\n1 0 obj\n<<>>\nendobj\n%%EOF"))}
	created := testRequest(t, runtime.handler, http.MethodPost, apiBase+"/qualifications", qualificationBody, "qualification-create-1", true, http.StatusCreated)
	qualificationItem := testMap(t, created, "item")
	qualificationID := testString(t, qualificationItem, "id")
	if testString(t, qualificationItem, "status") != "draft" || testString(t, qualificationItem, "evidenceFileId") == "" || runtime.files.stores != 1 {
		t.Fatalf("unexpected qualification evidence result: item=%v stores=%d", qualificationItem, runtime.files.stores)
	}
	file := runtime.files.items[testString(t, qualificationItem, "evidenceFileId")]
	if file.Visibility != pluginsdk.FileVisibilityPrivate || file.Metadata["qualificationId"] != qualificationID || file.Metadata["subjectId"] != customerID {
		t.Fatalf("qualification evidence ownership is incomplete: %+v", file)
	}
	duplicate := testRequest(t, runtime.handler, http.MethodPost, apiBase+"/qualifications", qualificationBody, "qualification-create-1", true, http.StatusOK)
	if duplicate["duplicate"] != true || runtime.files.stores != 1 {
		t.Fatalf("qualification create is not idempotent: response=%v stores=%d", duplicate, runtime.files.stores)
	}
	testRequest(t, runtime.handler, http.MethodPut, apiBase+"/qualification-types/"+typeID, map[string]any{
		"code": "DRUG-BUSINESS", "name": "药品经营许可证", "subjectType": "customer", "businessGate": "sales", "description": "不得改写已引用规则", "validityDays": 364, "alertDays": 30, "evidenceRequired": true, "businessRequired": true, "version": 1,
	}, "qualification-type-policy-change-1", true, http.StatusConflict)

	eligibility := testRequest(t, runtime.handler, http.MethodGet, apiBase+"/customers/"+customerID+"/sales-eligibility", nil, "", true, http.StatusOK)
	if eligibility["eligible"] != false || len(eligibility["missing"].([]any)) != 1 {
		t.Fatalf("draft qualification unexpectedly passed business gate: %v", eligibility)
	}
	submitted := testMap(t, testRequest(t, runtime.handler, http.MethodPost, apiBase+"/qualifications/"+qualificationID+"/submit", map[string]any{"version": 1}, "qualification-submit-1", true, http.StatusOK), "item")
	if testString(t, submitted, "status") != "pending" || testInt64(t, submitted, "version") != 2 {
		t.Fatalf("unexpected qualification submission: %v", submitted)
	}
	testRequest(t, runtime.handler, http.MethodPost, apiBase+"/qualification-types/"+typeID+"/disable", map[string]any{"reason": "不应停用", "version": 1}, "qualification-type-disable-blocked-1", true, http.StatusConflict)
	approved := testMap(t, testRequest(t, runtime.handler, http.MethodPost, apiBase+"/qualifications/"+qualificationID+"/approve", map[string]any{"comment": "证照真实有效", "version": 2}, "qualification-approve-1", true, http.StatusOK), "item")
	if testString(t, approved, "status") != "approved" || testString(t, approved, "reviewedBy") != "actor-1" || testInt64(t, approved, "version") != 3 {
		t.Fatalf("unexpected qualification approval: %v", approved)
	}
	eligibility = testRequest(t, runtime.handler, http.MethodGet, apiBase+"/customers/"+customerID+"/sales-eligibility", nil, "", true, http.StatusOK)
	if eligibility["eligible"] != true || len(eligibility["missing"].([]any)) != 0 {
		t.Fatalf("approved qualification did not pass business gate: %v", eligibility)
	}

	firstScan := testRequest(t, runtime.handler, http.MethodPost, apiBase+"/qualifications/expiry-scan", map[string]any{}, "qualification-expiry-1", true, http.StatusOK)
	if testInt64(t, firstScan, "scheduled") != 1 || runtime.jobs.scheduleCalls != 1 {
		t.Fatalf("expiry scan did not schedule exactly once: response=%v calls=%d", firstScan, runtime.jobs.scheduleCalls)
	}
	secondScan := testRequest(t, runtime.handler, http.MethodPost, apiBase+"/qualifications/expiry-scan", map[string]any{}, "qualification-expiry-2", true, http.StatusOK)
	if testInt64(t, secondScan, "scheduled") != 0 || testInt64(t, secondScan, "skipped") != 1 || runtime.jobs.scheduleCalls != 1 {
		t.Fatalf("expiry scan duplicate was not skipped: response=%v calls=%d", secondScan, runtime.jobs.scheduleCalls)
	}
	restarted, err := newHandler(pluginsdk.HostServices{
		PluginID: pluginID, Transactions: runtime.transactions, DataScopes: runtime.scopes, DataStore: runtime.store,
		DocumentNumbers: runtime.numbers, Files: runtime.files, Audit: runtime.audit, Workflows: runtime.workflows, Jobs: runtime.jobs,
	})
	if err != nil {
		t.Fatal(err)
	}
	restartScan := testRequest(t, restarted, http.MethodPost, apiBase+"/qualifications/expiry-scan", map[string]any{}, "qualification-expiry-after-restart-1", true, http.StatusOK)
	if testInt64(t, restartScan, "scheduled") != 0 || testInt64(t, restartScan, "skipped") != 1 || runtime.jobs.scheduleCalls != 1 {
		t.Fatalf("expiry scan lost idempotency after restart: response=%v calls=%d", restartScan, runtime.jobs.scheduleCalls)
	}

	revoked := testMap(t, testRequest(t, runtime.handler, http.MethodPost, apiBase+"/qualifications/"+qualificationID+"/revoke", map[string]any{"comment": "监管机构撤销证照", "version": 4}, "qualification-revoke-1", true, http.StatusOK), "item")
	if testString(t, revoked, "status") != "revoked" {
		t.Fatalf("qualification was not revoked: %v", revoked)
	}
	eligibility = testRequest(t, runtime.handler, http.MethodGet, apiBase+"/customers/"+customerID+"/sales-eligibility", nil, "", true, http.StatusOK)
	if eligibility["eligible"] != false {
		t.Fatalf("revoked qualification still passed business gate: %v", eligibility)
	}
}

func TestPurchaseRequestApprovalCreatesOneGovernedOrder(t *testing.T) {
	runtime := newTestRuntime(t)
	supplierID, productID, manufacturerID := createPurchaseMasterData(t, runtime)
	purchaseBody := map[string]any{
		"tenantId": "tenant-a", "organizationId": "org-a", "supplierId": supplierID,
		"reason": "Replenish validated medicine stock", "currency": "CNY", "approverId": "manager-1", "approverName": "Purchase manager",
		"lines": []map[string]any{{"productId": productID, "quantity": "2.5", "unitPrice": "10.20"}},
	}
	testRequest(t, runtime.handler, http.MethodPost, apiBase+"/purchase-requests", purchaseBody, "purchase-create-without-qualification", true, http.StatusUnprocessableEntity)

	approvePurchaseQualification(t, runtime, "supplier", supplierID, "purchase", "supplier")
	approvePurchaseQualification(t, runtime, "product", productID, "purchase", "product")
	approvePurchaseQualification(t, runtime, "manufacturer", manufacturerID, "supply", "manufacturer")

	created := testRequest(t, runtime.handler, http.MethodPost, apiBase+"/purchase-requests", purchaseBody, "purchase-create-1", true, http.StatusCreated)
	requestItem := testMap(t, created, "item")
	requestID := testString(t, requestItem, "id")
	if testString(t, requestItem, "number") != "PR-000001" || testString(t, requestItem, "status") != "pending" ||
		testString(t, requestItem, "totalAmount") != "25.50" || testInt64(t, requestItem, "version") != 1 {
		t.Fatalf("unexpected purchase request: %v", requestItem)
	}
	lines := requestItem["lines"].([]any)
	if len(lines) != 1 || testString(t, lines[0].(map[string]any), "quantity") != "2.5" || testString(t, lines[0].(map[string]any), "amount") != "25.50" {
		t.Fatalf("purchase lines were not normalized: %v", lines)
	}
	taskID := testPendingWorkflowTaskID(t, testMap(t, created, "workflow"))
	duplicate := testRequest(t, runtime.handler, http.MethodPost, apiBase+"/purchase-requests", purchaseBody, "purchase-create-1", true, http.StatusOK)
	if duplicate["duplicate"] != true || testString(t, testMap(t, duplicate, "item"), "id") != requestID || runtime.numbers.sequences["purchase_request"] != 1 {
		t.Fatalf("purchase create idempotency failed: response=%v sequences=%v", duplicate, runtime.numbers.sequences)
	}
	testRequest(t, runtime.handler, http.MethodPost, apiBase+"/purchase-requests/"+requestID+"/approve", map[string]any{
		"taskId": taskID, "comment": "stale", "version": 2,
	}, "purchase-approve-stale", true, http.StatusConflict)

	testRequest(t, runtime.handler, http.MethodPost, apiBase+"/products/"+productID+"/disable", map[string]any{"reason": "Quality review", "version": 1}, "purchase-product-disable", true, http.StatusOK)
	testRequest(t, runtime.handler, http.MethodPost, apiBase+"/purchase-requests/"+requestID+"/approve", map[string]any{
		"taskId": taskID, "comment": "must fail while product is disabled", "version": 1,
	}, "purchase-approve-disabled-product", true, http.StatusUnprocessableEntity)
	testRequest(t, runtime.handler, http.MethodPost, apiBase+"/products/"+productID+"/enable", map[string]any{"version": 2}, "purchase-product-enable", true, http.StatusOK)

	approved := testRequest(t, runtime.handler, http.MethodPost, apiBase+"/purchase-requests/"+requestID+"/approve", map[string]any{
		"taskId": taskID, "comment": "Supplier, product, and manufacturer qualifications confirmed", "version": 1,
	}, "purchase-approve-1", true, http.StatusOK)
	approvedRequest, order := testMap(t, approved, "item"), testMap(t, approved, "order")
	if testString(t, approvedRequest, "status") != "approved" || testString(t, order, "number") != "PO-000001" ||
		testString(t, order, "status") != "open" || testString(t, order, "purchaseRequestId") != requestID ||
		testString(t, approvedRequest, "purchaseOrderId") != testString(t, order, "id") {
		t.Fatalf("purchase approval did not create the governed order: %v", approved)
	}
	duplicateApproval := testRequest(t, runtime.handler, http.MethodPost, apiBase+"/purchase-requests/"+requestID+"/approve", map[string]any{
		"taskId": taskID, "comment": "Supplier, product, and manufacturer qualifications confirmed", "version": 1,
	}, "purchase-approve-1", true, http.StatusOK)
	if duplicateApproval["duplicate"] != true || testString(t, testMap(t, duplicateApproval, "order"), "id") != testString(t, order, "id") ||
		runtime.numbers.sequences["purchase_order"] != 1 {
		t.Fatalf("purchase approval was not idempotent: response=%v sequences=%v", duplicateApproval, runtime.numbers.sequences)
	}

	second := testRequest(t, runtime.handler, http.MethodPost, apiBase+"/purchase-requests", purchaseBody, "purchase-create-2", true, http.StatusCreated)
	secondItem := testMap(t, second, "item")
	rejected := testRequest(t, runtime.handler, http.MethodPost, apiBase+"/purchase-requests/"+testString(t, secondItem, "id")+"/reject", map[string]any{
		"taskId": testPendingWorkflowTaskID(t, testMap(t, second, "workflow")), "comment": "Budget is not approved", "version": 1,
	}, "purchase-reject-1", true, http.StatusOK)
	if testString(t, testMap(t, rejected, "item"), "status") != "rejected" {
		t.Fatalf("purchase rejection failed: %v", rejected)
	}

	requests := testRequest(t, runtime.handler, http.MethodGet, apiBase+"/purchase-requests?status=approved", nil, "", true, http.StatusOK)
	orders := testRequest(t, runtime.handler, http.MethodGet, apiBase+"/purchase-orders?status=open", nil, "", true, http.StatusOK)
	if testInt64(t, requests, "total") != 1 || testInt64(t, orders, "total") != 1 {
		t.Fatalf("purchase list filters are inconsistent: requests=%v orders=%v", requests, orders)
	}
	crossScope := employeeScope{TenantID: "tenant-a", OrganizationID: "org-b", OwnerID: "actor-b"}
	crossPredicate, err := pluginsdk.NewScopePredicate(pluginsdk.TrustedScope{
		SubjectID: crossScope.OwnerID, TenantIDs: []string{crossScope.TenantID},
		OrganizationIDs: []string{crossScope.OrganizationID}, OwnerIDs: []string{crossScope.OwnerID},
	})
	if err != nil {
		t.Fatal(err)
	}
	runtime.store.scope = crossScope
	crossHandler, err := newHandler(pluginsdk.HostServices{
		PluginID: pluginID, Transactions: runtime.transactions, DataScopes: testScopes{predicate: crossPredicate}, DataStore: runtime.store,
		DocumentNumbers: runtime.numbers, Files: runtime.files, Audit: runtime.audit, Workflows: runtime.workflows, Jobs: runtime.jobs,
	})
	if err != nil {
		t.Fatal(err)
	}
	crossRequests := testRequest(t, crossHandler, http.MethodGet, apiBase+"/purchase-requests", nil, "", true, http.StatusOK)
	crossOrders := testRequest(t, crossHandler, http.MethodGet, apiBase+"/purchase-orders", nil, "", true, http.StatusOK)
	if testInt64(t, crossRequests, "total") != 0 || testInt64(t, crossOrders, "total") != 0 {
		t.Fatalf("cross-scope purchase lists leaked records: requests=%v orders=%v", crossRequests, crossOrders)
	}
	testRequest(t, crossHandler, http.MethodGet, apiBase+"/purchase-requests/"+requestID, nil, "", true, http.StatusNotFound)
	testRequest(t, crossHandler, http.MethodGet, apiBase+"/purchase-orders/"+testString(t, order, "id"), nil, "", true, http.StatusNotFound)
	runtime.store.scope = employeeScope{TenantID: "tenant-a", OrganizationID: "org-a", OwnerID: "actor-1"}
	for _, action := range []string{"pharma_oa.purchase.create", "pharma_oa.purchase.approve", "pharma_oa.purchase.order.create", "pharma_oa.purchase.reject"} {
		if !testAuditHasAction(runtime.audit.entries, action) {
			t.Fatalf("missing purchase audit action %q: %v", action, runtime.audit.entries)
		}
	}
	for _, mutation := range runtime.store.mutations {
		if mutation.Table == purchaseRequestTable || mutation.Table == purchaseOrderTable {
			if mutation.Scope.Filter.TenantIDs[0] != "tenant-a" || mutation.Scope.Filter.OrganizationIDs[0] != "org-a" || mutation.Scope.Filter.OwnerIDs[0] != "actor-1" {
				t.Fatalf("purchase mutation escaped exact trusted scope: %+v", mutation.Scope.Filter)
			}
		}
	}
}

func createPurchaseMasterData(t *testing.T, runtime testRuntime) (string, string, string) {
	t.Helper()
	supplier := testMap(t, testRequest(t, runtime.handler, http.MethodPost, apiBase+"/suppliers", map[string]any{
		"tenantId": "tenant-a", "organizationId": "org-a", "code": "SUP-PO-001", "name": "Validated Supplier",
		"unifiedSocialCreditCode": "91330000MAPO0001", "region": "Zhejiang", "rating": 5,
		"contacts":        []map[string]any{{"name": "Quality Owner", "phone": "13800000001", "email": "quality@supplier.example", "primary": true}},
		"addresses":       []map[string]any{{"label": "Headquarters", "province": "Zhejiang", "city": "Hangzhou", "district": "Gongshu", "detail": "88 Compliance Road", "default": true}},
		"settlementTerms": map[string]any{"currency": "CNY", "paymentDays": 30, "creditLimit": 500000},
	}, "purchase-supplier-create", true, http.StatusCreated), "item")
	category := testMap(t, testRequest(t, runtime.handler, http.MethodPost, apiBase+"/categories", map[string]any{
		"tenantId": "tenant-a", "organizationId": "org-a", "code": "PO-RX", "name": "Prescription", "description": "Purchase acceptance category",
	}, "purchase-category-create", true, http.StatusCreated), "item")
	unit := testMap(t, testRequest(t, runtime.handler, http.MethodPost, apiBase+"/units", map[string]any{
		"tenantId": "tenant-a", "organizationId": "org-a", "code": "PO-BOX", "name": "Box", "description": "Purchase package", "symbol": "box", "decimalPlaces": 3,
	}, "purchase-unit-create", true, http.StatusCreated), "item")
	manufacturer := testMap(t, testRequest(t, runtime.handler, http.MethodPost, apiBase+"/manufacturers", map[string]any{
		"tenantId": "tenant-a", "organizationId": "org-a", "code": "PO-MFG-001", "name": "Validated Manufacturer",
		"description": "Medicine manufacturer", "unifiedSocialCreditCode": "91330000MAPO1001", "licenseNumber": "MFG-PO-2026-001",
	}, "purchase-manufacturer-create", true, http.StatusCreated), "item")
	product := testMap(t, testRequest(t, runtime.handler, http.MethodPost, apiBase+"/products", map[string]any{
		"tenantId": "tenant-a", "organizationId": "org-a", "code": "PO-MED-001", "sku": "PO-SKU-001",
		"name": "Acceptance Capsule", "genericName": "Acceptance Medicine", "categoryId": testString(t, category, "id"),
		"unitId": testString(t, unit, "id"), "manufacturerId": testString(t, manufacturer, "id"), "dosageForm": "capsule",
		"specification": "0.25g x 24", "approvalNumber": "NMPA-PO-2026-001", "barcode": "690000009001",
		"storageCondition": "sealed and dry", "temperatureMin": 2, "temperatureMax": 25,
	}, "purchase-product-create", true, http.StatusCreated), "item")
	return testString(t, supplier, "id"), testString(t, product, "id"), testString(t, manufacturer, "id")
}

func approvePurchaseQualification(t *testing.T, runtime testRuntime, subjectType, subjectID, gate, suffix string) {
	t.Helper()
	typeItem := testMap(t, testRequest(t, runtime.handler, http.MethodPost, apiBase+"/qualification-types", map[string]any{
		"tenantId": "tenant-a", "organizationId": "org-a", "code": "PO-" + strings.ToUpper(suffix),
		"name": "Purchase " + suffix + " qualification", "subjectType": subjectType, "businessGate": gate,
		"description": "Required by purchase acceptance", "validityDays": 365, "alertDays": 30, "evidenceRequired": true, "businessRequired": true,
	}, "purchase-"+suffix+"-qualification-type", true, http.StatusCreated), "item")
	now := time.Now().UTC()
	qualification := testMap(t, testRequest(t, runtime.handler, http.MethodPost, apiBase+"/qualifications", map[string]any{
		"tenantId": "tenant-a", "organizationId": "org-a", "typeId": testString(t, typeItem, "id"),
		"subjectType": subjectType, "subjectId": subjectID, "certificateNumber": "CERT-" + strings.ToUpper(suffix),
		"issuer": "Acceptance Authority", "validFrom": now.AddDate(0, 0, -1).Format("2006-01-02"), "validTo": now.AddDate(1, 0, 0).Format("2006-01-02"),
		"evidence": map[string]any{"name": suffix + ".pdf", "contentBase64": base64.StdEncoding.EncodeToString([]byte("%PDF-1.4\n%%EOF"))},
	}, "purchase-"+suffix+"-qualification-create", true, http.StatusCreated), "item")
	id := testString(t, qualification, "id")
	testRequest(t, runtime.handler, http.MethodPost, apiBase+"/qualifications/"+id+"/submit", map[string]any{"version": 1}, "purchase-"+suffix+"-qualification-submit", true, http.StatusOK)
	testRequest(t, runtime.handler, http.MethodPost, apiBase+"/qualifications/"+id+"/approve", map[string]any{"comment": "valid", "version": 2}, "purchase-"+suffix+"-qualification-approve", true, http.StatusOK)
}

func testPendingWorkflowTaskID(t *testing.T, workflow map[string]any) string {
	t.Helper()
	for _, raw := range workflow["tasks"].([]any) {
		task := raw.(map[string]any)
		if task["status"] == string(pluginsdk.WorkflowTaskPending) {
			return testString(t, task, "id")
		}
	}
	t.Fatalf("workflow has no pending task: %v", workflow)
	return ""
}

func testAuditHasAction(entries []pluginsdk.AuditEntry, action string) bool {
	for _, entry := range entries {
		if entry.Action == action {
			return true
		}
	}
	return false
}

func TestFoundationRejectsIncompleteHostAndUnknownRoutes(t *testing.T) {
	if _, err := newHandler(pluginsdk.HostServices{PluginID: pluginID}); err == nil {
		t.Fatal("incomplete host services were accepted")
	}
	runtime := newTestRuntime(t)
	for _, path := range []string{"/v1/pharma-oa/api/meta", "/v1/plugins/other/api/meta", apiBase + "/not-found"} {
		testRequest(t, runtime.handler, http.MethodGet, path, nil, "", false, http.StatusNotFound)
	}
}

func testRequest(t *testing.T, handler http.Handler, method, path string, body any, requestKey string, authenticated bool, wantStatus int) map[string]any {
	t.Helper()
	var raw []byte
	if body != nil {
		var err error
		raw, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	request := httptest.NewRequest(method, path, bytes.NewReader(raw))
	request.Header.Set("Content-Type", "application/json")
	if authenticated {
		request.Header.Set("Authorization", "Bearer user-token")
	}
	if requestKey != "" {
		request.Header.Set("Idempotency-Key", requestKey)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != wantStatus {
		t.Fatalf("%s %s status=%d want=%d body=%s", method, path, recorder.Code, wantStatus, recorder.Body.String())
	}
	if wantStatus == http.StatusNoContent || recorder.Body.Len() == 0 {
		return nil
	}
	if !strings.Contains(recorder.Header().Get("Content-Type"), "application/json") {
		return nil
	}
	var envelope struct {
		Code string         `json:"code"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode %s %s: %v body=%s", method, path, err, recorder.Body.String())
	}
	return envelope.Data
}

func testFilterMatches(record pluginsdk.DataRecord, filter pluginsdk.DataFilter) bool {
	if len(filter.All) > 0 {
		for _, child := range filter.All {
			if !testFilterMatches(record, child) {
				return false
			}
		}
		return true
	}
	if len(filter.Any) > 0 {
		for _, child := range filter.Any {
			if testFilterMatches(record, child) {
				return true
			}
		}
		return false
	}
	actual := dataString(record, filter.Field)
	if filter.Operator != pluginsdk.DataOperatorIn && filter.Value == nil {
		return false
	}
	switch filter.Operator {
	case pluginsdk.DataOperatorEqual:
		return actual == filter.Value.Value
	case pluginsdk.DataOperatorContains:
		return strings.Contains(strings.ToLower(actual), strings.ToLower(filter.Value.Value))
	case pluginsdk.DataOperatorIn:
		for _, value := range filter.Values {
			if actual == value.Value {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func testProjectRecord(record pluginsdk.DataRecord, fields []string) pluginsdk.DataRecord {
	values := make(map[string]pluginsdk.DataValue, len(fields))
	for _, field := range fields {
		value, ok := record.Values[field]
		if !ok {
			value = pluginsdk.DataValue{Type: pluginsdk.DataValueNull}
		}
		values[field] = value
	}
	return pluginsdk.DataRecord{Values: values, Version: record.Version}
}

func testCloneValues(values map[string]pluginsdk.DataValue) map[string]pluginsdk.DataValue {
	copy := make(map[string]pluginsdk.DataValue, len(values)+6)
	for key, value := range values {
		copy[key] = value
	}
	return copy
}

func testMap(t *testing.T, values map[string]any, field string) map[string]any {
	t.Helper()
	value, ok := values[field].(map[string]any)
	if !ok {
		t.Fatalf("field %q is not an object: %v", field, values[field])
	}
	return value
}

func testString(t *testing.T, values map[string]any, field string) string {
	t.Helper()
	value, ok := values[field].(string)
	if !ok {
		t.Fatalf("field %q is not a string: %v", field, values[field])
	}
	return value
}

func testInt64(t *testing.T, values map[string]any, field string) int64 {
	t.Helper()
	value, ok := values[field].(float64)
	if !ok {
		t.Fatalf("field %q is not a number: %v", field, values[field])
	}
	return int64(value)
}
