package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

type testTransaction struct{ ctx context.Context }

func (t testTransaction) Context() context.Context { return t.ctx }

type testTransactions struct{}

func (testTransactions) Within(ctx context.Context, fn func(pluginsdk.Transaction) error) error {
	return fn(testTransaction{ctx: ctx})
}

type testScopes struct{ tenant, organization, subject string }

func (s testScopes) Resolve(context.Context, pluginsdk.Permission) (pluginsdk.ScopePredicate, error) {
	return pluginsdk.NewScopePredicate(pluginsdk.TrustedScope{SubjectID: s.subject, TenantIDs: []string{s.tenant}, OwnerIDs: []string{s.subject}, OrganizationIDs: []string{s.organization}})
}

type testFiles struct {
	items map[string]pluginsdk.FileObject
}

func (f *testFiles) Store(context.Context, pluginsdk.FileWrite) (pluginsdk.FileObject, error) {
	return pluginsdk.FileObject{}, errors.New("not used")
}
func (f *testFiles) List(context.Context, pluginsdk.FileQuery) ([]pluginsdk.FileObject, error) {
	return nil, nil
}
func (f *testFiles) Get(_ context.Context, id string) (pluginsdk.FileObject, error) {
	item, ok := f.items[id]
	if !ok {
		return pluginsdk.FileObject{}, errors.New("file not found")
	}
	return item, nil
}
func (f *testFiles) Download(context.Context, string) (pluginsdk.FileDownload, error) {
	return pluginsdk.FileDownload{}, errors.New("not used")
}
func (f *testFiles) Delete(context.Context, string) error { return errors.New("not used") }

type testAudit struct {
	mu      sync.Mutex
	entries []pluginsdk.AuditEntry
}

func (a *testAudit) Record(_ context.Context, entry pluginsdk.AuditEntry) (pluginsdk.AuditReceipt, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.entries = append(a.entries, entry)
	return pluginsdk.AuditReceipt{ID: "audit", OccurredAt: time.Now()}, nil
}

type testConfig struct{ values map[string]any }

func (c *testConfig) Get(context.Context) (map[string]any, error) { return c.values, nil }
func (c *testConfig) Replace(_ context.Context, values map[string]any) (map[string]any, error) {
	c.values = values
	return values, nil
}

type testWorkflows struct {
	mu         sync.Mutex
	definition *pluginsdk.WorkflowDefinition
	instances  map[string]pluginsdk.WorkflowInstance
}

func (w *testWorkflows) CreateDefinition(_ context.Context, in pluginsdk.WorkflowDefinitionInput) (pluginsdk.WorkflowDefinition, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.definition != nil {
		return pluginsdk.WorkflowDefinition{}, errors.New("definition exists")
	}
	item := pluginsdk.WorkflowDefinition{ID: in.ID, Key: in.Key, Name: in.Name, Version: in.Version, Status: pluginsdk.WorkflowDefinitionDraft, Nodes: in.Nodes, Transitions: in.Transitions}
	w.definition = &item
	return item, nil
}
func (w *testWorkflows) GetDefinition(_ context.Context, id string) (pluginsdk.WorkflowDefinition, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.definition == nil || w.definition.ID != id {
		return pluginsdk.WorkflowDefinition{}, errors.New("definition not found")
	}
	return *w.definition, nil
}
func (w *testWorkflows) PublishDefinition(_ context.Context, id string) (pluginsdk.WorkflowDefinition, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.definition == nil || w.definition.ID != id {
		return pluginsdk.WorkflowDefinition{}, errors.New("definition not found")
	}
	w.definition.Status = pluginsdk.WorkflowDefinitionPublished
	return *w.definition, nil
}
func (w *testWorkflows) Start(_ context.Context, in pluginsdk.WorkflowStartInput) (pluginsdk.WorkflowInstance, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if item, ok := w.instances[in.ID]; ok {
		return item, nil
	}
	item := pluginsdk.WorkflowInstance{ID: in.ID, DefinitionID: in.DefinitionID, BusinessType: in.BusinessType, BusinessID: in.BusinessID, Title: in.Title, Status: pluginsdk.WorkflowInstanceRunning}
	w.instances[in.ID] = item
	return item, nil
}
func (w *testWorkflows) GetInstance(_ context.Context, id string) (pluginsdk.WorkflowInstance, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	item, ok := w.instances[id]
	if !ok {
		return pluginsdk.WorkflowInstance{}, errors.New("instance not found")
	}
	return item, nil
}
func (w *testWorkflows) Approve(context.Context, pluginsdk.WorkflowTaskActionInput) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{}, errors.New("not used")
}
func (w *testWorkflows) Reject(context.Context, pluginsdk.WorkflowTaskActionInput) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{}, errors.New("not used")
}
func (w *testWorkflows) Withdraw(context.Context, pluginsdk.WorkflowInstanceActionInput) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{}, errors.New("not used")
}
func (w *testWorkflows) Transfer(context.Context, pluginsdk.WorkflowTargetActionInput) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{}, errors.New("not used")
}
func (w *testWorkflows) Copy(context.Context, pluginsdk.WorkflowTargetActionInput) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{}, errors.New("not used")
}

type testJobs struct {
	mu    sync.Mutex
	items map[string]pluginsdk.Job
}

func (j *testJobs) Schedule(_ context.Context, in pluginsdk.JobScheduleInput) (pluginsdk.Job, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if item, ok := j.items[in.IdempotencyKey]; ok {
		return item, nil
	}
	item := pluginsdk.Job{ID: in.ID, Kind: in.Kind, IdempotencyKey: in.IdempotencyKey, Payload: in.Payload, Status: pluginsdk.JobStatusScheduled, RunAt: in.RunAt, MaxAttempts: in.MaxAttempts}
	j.items[in.IdempotencyKey] = item
	return item, nil
}
func (j *testJobs) LeaseDue(context.Context, pluginsdk.JobLeaseInput) ([]pluginsdk.Job, error) {
	return nil, errors.New("not used")
}
func (j *testJobs) Complete(context.Context, pluginsdk.JobCompleteInput) (pluginsdk.Job, error) {
	return pluginsdk.Job{}, errors.New("not used")
}
func (j *testJobs) Fail(context.Context, pluginsdk.JobFailInput) (pluginsdk.Job, error) {
	return pluginsdk.Job{}, errors.New("not used")
}
func (j *testJobs) Get(_ context.Context, id string) (pluginsdk.Job, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	for _, item := range j.items {
		if item.ID == id {
			return item, nil
		}
	}
	return pluginsdk.Job{}, errors.New("job not found")
}
func (j *testJobs) List(_ context.Context, query pluginsdk.JobQuery) ([]pluginsdk.Job, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	out := make([]pluginsdk.Job, 0)
	for _, item := range j.items {
		if query.Kind == "" || item.Kind == query.Kind {
			out = append(out, item)
		}
	}
	return out, nil
}

type testRuntime struct {
	handler http.Handler
	store   *Store
	audit   *testAudit
	jobs    *testJobs
	dataDir string
}

func newTestRuntime(t *testing.T) testRuntime {
	t.Helper()
	dataDir := t.TempDir()
	store, err := OpenStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	audit := &testAudit{}
	jobs := &testJobs{items: make(map[string]pluginsdk.Job)}
	server, err := NewServer(store, Host{
		Transactions: testTransactions{}, DataScopes: testScopes{tenant: "tenant-a", organization: "org-a", subject: "user-a"},
		Files: &testFiles{items: map[string]pluginsdk.FileObject{"file-1": {ID: "file-1"}}}, Audit: audit,
		Config:    &testConfig{values: map[string]any{"equipment_maintenance.approval_cost_limit": 1000.0, "equipment_maintenance.low_stock_threshold": 2.0}},
		Workflows: &testWorkflows{instances: make(map[string]pluginsdk.WorkflowInstance)}, Jobs: jobs,
	})
	if err != nil {
		t.Fatal(err)
	}
	server.nowFn = func() time.Time { return time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC) }
	return testRuntime{handler: server.Handler(), store: store, audit: audit, jobs: jobs, dataDir: dataDir}
}

func TestEquipmentMaintenancePrimaryWorkflowsAndPersistence(t *testing.T) {
	runtime := newTestRuntime(t)
	asset := createRecord(t, runtime.handler, apiBase+"/assets", map[string]any{"tenantId": "tenant-a", "organizationId": "org-a", "code": "EQ-001", "name": "Tablet Press", "category": "production", "location": "Line A"})
	assetID := stringField(t, asset, "id")
	order := createRecord(t, runtime.handler, apiBase+"/work-orders", map[string]any{"tenantId": "tenant-a", "organizationId": "org-a", "assetId": assetID, "number": "WO-001", "title": "Replace drive", "priority": "high", "estimatedCost": 2500})
	orderID := stringField(t, order, "id")
	perform(t, runtime.handler, http.MethodPost, apiBase+"/work-orders/"+orderID+"/dispatch", map[string]any{"assigneeId": "engineer-a"}, http.StatusOK)
	perform(t, runtime.handler, http.MethodPost, apiBase+"/work-orders/"+orderID+"/start", map[string]any{}, http.StatusOK)
	submitted := perform(t, runtime.handler, http.MethodPost, apiBase+"/work-orders/"+orderID+"/submit", map[string]any{}, http.StatusOK)
	if nestedString(t, submitted, "item", "status") != "awaiting_approval" {
		t.Fatalf("submit response=%v", submitted)
	}
	event := map[string]any{"deliveryId": "delivery-1", "pluginId": pluginID, "handler": "onApprovalCompleted", "eventName": "approval-completed", "subject": map[string]any{"id": orderID}, "payload": map[string]any{"businessId": orderID, "status": "approved"}}
	performWithoutAuth(t, runtime.handler, http.MethodPost, "/_skoll/events", event, http.StatusNoContent)
	performWithoutAuth(t, runtime.handler, http.MethodPost, "/_skoll/events", event, http.StatusNoContent)
	perform(t, runtime.handler, http.MethodPost, apiBase+"/work-orders/"+orderID+"/complete", map[string]any{}, http.StatusOK)
	perform(t, runtime.handler, http.MethodPost, apiBase+"/work-orders/"+orderID+"/close", map[string]any{}, http.StatusOK)
	perform(t, runtime.handler, http.MethodPost, apiBase+"/assets/"+assetID+"/retire", map[string]any{}, http.StatusOK)

	reopened, err := OpenStore(runtime.dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := reopened.View(func(state State) error {
		if state.Assets[assetID].Status != "retired" || state.WorkOrders[orderID].Status != "closed" || len(state.EventDeliveries) != 1 {
			t.Fatalf("reopened state=%+v", state)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(runtime.audit.entries) < 8 {
		t.Fatalf("audit entries=%d", len(runtime.audit.entries))
	}
}

func TestEquipmentMaintenanceInspectionSpareAndJobFlows(t *testing.T) {
	runtime := newTestRuntime(t)
	assetID := stringField(t, createRecord(t, runtime.handler, apiBase+"/assets", map[string]any{"tenantId": "tenant-a", "organizationId": "org-a", "code": "EQ-002", "name": "Coater"}), "id")
	plan := createRecord(t, runtime.handler, apiBase+"/maintenance-plans", map[string]any{"tenantId": "tenant-a", "organizationId": "org-a", "assetId": assetID, "name": "Monthly inspection", "intervalDays": 30, "nextRunAt": "2026-07-20T08:00:00Z"})
	inspection := createRecord(t, runtime.handler, apiBase+"/inspections", map[string]any{"tenantId": "tenant-a", "organizationId": "org-a", "planId": stringField(t, plan, "id")})
	completed := perform(t, runtime.handler, http.MethodPost, apiBase+"/inspections/"+stringField(t, inspection, "id")+"/complete", map[string]any{"result": "pass", "finding": "No abnormal vibration", "attachmentIds": []string{"file-1"}}, http.StatusOK)
	if nestedString(t, completed, "item", "status") != "completed" {
		t.Fatalf("inspection response=%v", completed)
	}

	part := createRecord(t, runtime.handler, apiBase+"/spare-parts", map[string]any{"tenantId": "tenant-a", "organizationId": "org-a", "sku": "BRG-01", "name": "Bearing", "unit": "piece", "minimumQuantity": 2})
	partID := stringField(t, part, "id")
	inbound := map[string]any{"tenantId": "tenant-a", "organizationId": "org-a", "sparePartId": partID, "movementType": "inbound", "quantity": 5, "idempotencyKey": "receipt-1"}
	perform(t, runtime.handler, http.MethodPost, apiBase+"/spare-movements", inbound, http.StatusCreated)
	duplicate := perform(t, runtime.handler, http.MethodPost, apiBase+"/spare-movements", inbound, http.StatusOK)
	if duplicate["duplicate"] != true {
		t.Fatalf("duplicate response=%v", duplicate)
	}
	perform(t, runtime.handler, http.MethodPost, apiBase+"/spare-movements", map[string]any{"tenantId": "tenant-a", "organizationId": "org-a", "sparePartId": partID, "movementType": "outbound", "quantity": 6, "idempotencyKey": "issue-too-many"}, http.StatusBadRequest)
	perform(t, runtime.handler, http.MethodPost, apiBase+"/spare-movements", map[string]any{"tenantId": "tenant-a", "organizationId": "org-a", "sparePartId": partID, "movementType": "outbound", "quantity": 3, "idempotencyKey": "issue-1"}, http.StatusCreated)
	perform(t, runtime.handler, http.MethodPost, apiBase+"/jobs/spare-stock-scan", map[string]any{"tenantId": "tenant-a", "organizationId": "org-a"}, http.StatusAccepted)
	perform(t, runtime.handler, http.MethodPost, apiBase+"/jobs/spare-stock-scan", map[string]any{"tenantId": "tenant-a", "organizationId": "org-a"}, http.StatusAccepted)
	if len(runtime.jobs.items) != 1 {
		t.Fatalf("scheduled jobs=%d", len(runtime.jobs.items))
	}
	dashboard := perform(t, runtime.handler, http.MethodGet, apiBase+"/dashboard", nil, http.StatusOK)
	metrics := dashboard["metrics"].(map[string]any)
	if metrics["lowStockParts"].(float64) != 1 {
		t.Fatalf("dashboard=%v", dashboard)
	}
}

func TestEquipmentMaintenanceRejectsUntrustedScope(t *testing.T) {
	runtime := newTestRuntime(t)
	perform(t, runtime.handler, http.MethodPost, apiBase+"/assets", map[string]any{"tenantId": "tenant-b", "organizationId": "org-a", "code": "EQ-X", "name": "Denied"}, http.StatusForbidden)
	performWithoutAuth(t, runtime.handler, http.MethodGet, apiBase+"/assets", nil, http.StatusForbidden)
}

func createRecord(t *testing.T, handler http.Handler, path string, body any) map[string]any {
	t.Helper()
	response := perform(t, handler, http.MethodPost, path, body, http.StatusCreated)
	item, ok := response["item"].(map[string]any)
	if !ok {
		t.Fatalf("missing item: %v", response)
	}
	return item
}

func perform(t *testing.T, handler http.Handler, method, path string, body any, want int) map[string]any {
	t.Helper()
	recorder := performRequest(t, handler, method, path, body, true)
	return decodeResponse(t, recorder, want)
}
func performWithoutAuth(t *testing.T, handler http.Handler, method, path string, body any, want int) map[string]any {
	t.Helper()
	recorder := performRequest(t, handler, method, path, body, false)
	return decodeResponse(t, recorder, want)
}
func performRequest(t *testing.T, handler http.Handler, method, path string, body any, auth bool) *httptest.ResponseRecorder {
	t.Helper()
	var raw []byte
	if body != nil {
		raw, _ = json.Marshal(body)
	}
	request := httptest.NewRequest(method, path, bytes.NewReader(raw))
	if auth {
		request.Header.Set("Authorization", "Bearer user-token")
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}
func decodeResponse(t *testing.T, recorder *httptest.ResponseRecorder, want int) map[string]any {
	t.Helper()
	if recorder.Code != want {
		t.Fatalf("status=%d want=%d body=%s", recorder.Code, want, recorder.Body.String())
	}
	if want == http.StatusNoContent {
		return map[string]any{}
	}
	var payload struct {
		Code    string         `json:"code"`
		Message string         `json:"message"`
		Data    map[string]any `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if want < 300 && payload.Code != "ok" {
		t.Fatalf("payload=%s", recorder.Body.String())
	}
	return payload.Data
}
func stringField(t *testing.T, item map[string]any, key string) string {
	t.Helper()
	value := strings.TrimSpace(item[key].(string))
	if value == "" {
		t.Fatalf("missing %s in %v", key, item)
	}
	return value
}
func nestedString(t *testing.T, item map[string]any, object, key string) string {
	t.Helper()
	return stringField(t, item[object].(map[string]any), key)
}

func TestStorePathIsOwnedByPluginDataDirectory(t *testing.T) {
	root := t.TempDir()
	store, err := OpenStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(store.path) != filepath.Clean(root) {
		t.Fatalf("store path=%s root=%s", store.path, root)
	}
}
