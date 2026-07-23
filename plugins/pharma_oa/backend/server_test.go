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
	if testString(t, meta, "pluginId") != pluginID || testString(t, meta, "contractVersion") != "0.5.0" || len(meta["modules"].([]any)) != 12 || len(meta["documentTypes"].([]any)) != 12 || len(meta["events"].([]any)) != 5 {
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
