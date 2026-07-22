package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
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
		record := s.records[id]
		if dataString(record, "tenant_id") != s.scope.TenantID || dataString(record, "organization_id") != s.scope.OrganizationID || dataString(record, "owner_id") != s.scope.OwnerID {
			continue
		}
		if query.Filter != nil && !testFilterMatches(record, *query.Filter) {
			continue
		}
		records = append(records, testProjectRecord(record, query.Fields))
	}
	return pluginsdk.DataPage{Records: records}, nil
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
	case pluginsdk.DataMutationUpdate:
		record, exists := s.records[id]
		if !exists {
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
	files        *testFiles
	audit        *testAudit
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
	store := &testDataStore{records: make(map[string]pluginsdk.DataRecord), idempotency: make(map[string]string), scope: employeeScope{TenantID: "tenant-a", OrganizationID: "org-a", OwnerID: "actor-1"}}
	files := &testFiles{items: make(map[string]pluginsdk.FileObject)}
	audit := &testAudit{}
	handler, err := newHandler(pluginsdk.HostServices{
		PluginID: pluginID, Transactions: transactions, DataScopes: testScopes{predicate: predicate}, DataStore: store, Files: files, Audit: audit,
	})
	if err != nil {
		t.Fatal(err)
	}
	return testRuntime{handler: handler, transactions: transactions, store: store, files: files, audit: audit}
}

func TestFoundationEndpoints(t *testing.T) {
	runtime := newTestRuntime(t)
	health := testRequest(t, runtime.handler, http.MethodGet, "/health", nil, "", false, http.StatusOK)
	if testString(t, health, "status") != "ready" {
		t.Fatalf("unexpected health data: %v", health)
	}
	meta := testRequest(t, runtime.handler, http.MethodGet, apiBase+"/meta", nil, "", false, http.StatusOK)
	if testString(t, meta, "pluginId") != pluginID || testString(t, meta, "contractVersion") != "0.3.0" || len(meta["modules"].([]any)) != 12 || len(meta["documentTypes"].([]any)) != 12 || len(meta["events"].([]any)) != 5 {
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
		PluginID: pluginID, Transactions: runtime.transactions, DataScopes: testScopes{predicate: pluginsdk.NewDeniedScopePredicate("actor-1")}, DataStore: runtime.store, Files: runtime.files, Audit: runtime.audit,
	})
	if err != nil {
		t.Fatal(err)
	}
	testRequest(t, handler, http.MethodPost, apiBase+"/employees", map[string]any{"code": "EMP-002", "name": "王芳", "departmentId": "sales", "positionId": "staff"}, "employee-denied-1", true, http.StatusForbidden)
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
	if filter.Value == nil {
		return false
	}
	switch filter.Operator {
	case pluginsdk.DataOperatorEqual:
		return actual == filter.Value.Value
	case pluginsdk.DataOperatorContains:
		return strings.Contains(strings.ToLower(actual), strings.ToLower(filter.Value.Value))
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
