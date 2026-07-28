package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

func TestInboundCreatesImmutableLedgerAndExactProjection(t *testing.T) {
	runtime := newTestRuntime(t)
	order := createApprovedPurchaseOrder(t, runtime, "2.5")
	topology := createInboundTopology(t, runtime)
	orderLine := order["lines"].([]any)[0].(map[string]any)
	body := inventoryInboundBody(order, orderLine, topology, "1.25", "LOT-INVENTORY-001", 1)

	created := testRequest(
		t, runtime.handler, http.MethodPost, apiBase+"/purchase-inbounds",
		body, "inventory-receipt-1", true, http.StatusCreated,
	)
	inbound := testMap(t, created, "item")
	inboundID := testString(t, inbound, "id")
	inboundLine := inbound["lines"].([]any)[0].(map[string]any)
	productID := testString(t, inboundLine, "productId")

	lots := testRequest(
		t, runtime.handler, http.MethodGet,
		apiBase+"/inventory-lots?productId="+productID+"&batchNo=lot-inventory-001",
		nil, "", true, http.StatusOK,
	)
	if testInt64(t, lots, "total") != 1 {
		t.Fatalf("receipt did not create one immutable lot: %v", lots)
	}
	lot := lots["items"].([]any)[0].(map[string]any)
	if testString(t, lot, "productionDate") != "2026-06-01T00:00:00Z" ||
		testString(t, lot, "expiresAt") != "2028-06-01T00:00:00Z" {
		t.Fatalf("immutable lot facts were not retained: %v", lot)
	}

	ledgerURL := fmt.Sprintf(
		"%s/stock-ledger?productId=%s&batchNo=LOT-INVENTORY-001&warehouseId=%s&locationId=%s&sourceDocumentId=%s",
		apiBase, productID, topology.WarehouseID, topology.LocationID, inboundID,
	)
	ledger := testRequest(t, runtime.handler, http.MethodGet, ledgerURL, nil, "", true, http.StatusOK)
	if testInt64(t, ledger, "total") != 1 {
		t.Fatalf("receipt detail did not create exactly one ledger entry: %v", ledger)
	}
	entry := ledger["items"].([]any)[0].(map[string]any)
	if testString(t, entry, "quantity") != "1.25" ||
		testString(t, entry, "sourceDocumentType") != "purchase_inbound" ||
		testString(t, entry, "sourceDocumentLineId") != testString(t, inboundLine, "id") {
		t.Fatalf("ledger trace is incomplete: %v", entry)
	}

	balances := testRequest(
		t, runtime.handler, http.MethodGet,
		apiBase+"/stock-balances?productId="+productID+"&batchNo=LOT-INVENTORY-001&locationId="+topology.LocationID,
		nil, "", true, http.StatusOK,
	)
	if testInt64(t, balances, "total") != 1 {
		t.Fatalf("receipt did not create one balance projection: %v", balances)
	}
	balance := balances["items"].([]any)[0].(map[string]any)
	if testString(t, balance, "quantity") != "1.25" {
		t.Fatalf("balance quantity is not exact: %v", balance)
	}
	reconciliation := testRequest(
		t, runtime.handler, http.MethodGet,
		apiBase+"/stock-reconciliation?productId="+productID+"&locationId="+topology.LocationID,
		nil, "", true, http.StatusOK,
	)
	if reconciled, ok := reconciliation["matched"].(bool); !ok || !reconciled ||
		testInt64(t, reconciliation, "total") != 1 {
		t.Fatalf("ledger and balance did not reconcile: %v", reconciliation)
	}
	reconciledItem := reconciliation["items"].([]any)[0].(map[string]any)
	if testString(t, reconciledItem, "ledgerQuantity") != "1.25" ||
		testString(t, reconciledItem, "balanceQuantity") != "1.25" {
		t.Fatalf("reconciliation quantities are not exact: %v", reconciledItem)
	}
	sourceReconciliation := testRequest(
		t, runtime.handler, http.MethodGet,
		apiBase+"/stock-reconciliation?sourceDocumentType=purchase_inbound&sourceDocumentId="+inboundID,
		nil, "", true, http.StatusOK,
	)
	if sourceReconciliation["matched"] != true || testInt64(t, sourceReconciliation, "total") != 1 {
		t.Fatalf("source-document reconciliation did not select its affected projection: %v", sourceReconciliation)
	}
	missingSourceReconciliation := testRequest(
		t, runtime.handler, http.MethodGet,
		apiBase+"/stock-reconciliation?sourceDocumentId=missing-inbound",
		nil, "", true, http.StatusOK,
	)
	if missingSourceReconciliation["matched"] != true || testInt64(t, missingSourceReconciliation, "total") != 0 {
		t.Fatalf("unknown source document was silently ignored by reconciliation: %v", missingSourceReconciliation)
	}

	duplicate := testRequest(
		t, runtime.handler, http.MethodPost, apiBase+"/purchase-inbounds",
		body, "inventory-receipt-1", true, http.StatusOK,
	)
	if duplicate["duplicate"] != true {
		t.Fatalf("receipt replay was not reported as duplicate: %v", duplicate)
	}
	conflictingReplay := inventoryInboundBody(order, orderLine, topology, "1.5", "LOT-INVENTORY-001", 1)
	testRequest(
		t, runtime.handler, http.MethodPost, apiBase+"/purchase-inbounds",
		conflictingReplay, "inventory-receipt-1", true, http.StatusConflict,
	)
	assertInventoryRouteTotal(t, runtime.handler, "/inventory-lots", 1)
	assertInventoryRouteTotal(t, runtime.handler, "/stock-ledger", 1)
	assertInventoryRouteTotal(t, runtime.handler, "/stock-balances", 1)

	runtime.events.mu.Lock()
	publications := append([]pluginsdk.EventPublication(nil), runtime.events.publications...)
	runtime.events.mu.Unlock()
	if len(publications) != 1 || publications[0].Name != "inventory-changed" ||
		publications[0].Subject.ID != inboundID ||
		publications[0].Payload["ledger_entry_count"].Value != "1" {
		t.Fatalf("inventory event is missing or duplicated: %+v", publications)
	}
	runtime.audit.mu.Lock()
	audits := append([]pluginsdk.AuditEntry(nil), runtime.audit.entries...)
	runtime.audit.mu.Unlock()
	if !testAuditHasAction(audits, "pharma_oa.inventory.receive") ||
		!testAuditHasAction(audits, "pharma_oa.inbound.create") {
		t.Fatalf("inventory audit evidence is incomplete: %+v", audits)
	}

	assertAppendOnlyMutationRejected(t, runtime, inventoryLotTable, testString(t, lot, "id"))
	assertAppendOnlyMutationRejected(t, runtime, stockLedgerTable, testString(t, entry, "id"))
	testRequest(t, runtime.handler, http.MethodPut, apiBase+"/stock-ledger/"+testString(t, entry, "id"), map[string]any{}, "forbidden-ledger-update", true, http.StatusNotFound)
	testRequest(t, runtime.handler, http.MethodDelete, apiBase+"/stock-ledger/"+testString(t, entry, "id"), nil, "", true, http.StatusNotFound)

	restarted, err := newHandler(pluginsdk.HostServices{
		PluginID: pluginID, Transactions: runtime.transactions, DataScopes: runtime.scopes,
		DataStore: runtime.store, Events: runtime.events, DocumentNumbers: runtime.numbers,
		Files: runtime.files, Audit: runtime.audit, Workflows: runtime.workflows, Jobs: runtime.jobs,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertInventoryRouteTotal(t, restarted, "/inventory-lots", 1)
	assertInventoryRouteTotal(t, restarted, "/stock-ledger", 1)
	assertInventoryRouteTotal(t, restarted, "/stock-balances", 1)
}

func TestInboundInventoryFailureRollsBackReceiptLedgerBalanceAuditAndEvent(t *testing.T) {
	runtime := newTestRuntime(t)
	order := createApprovedPurchaseOrder(t, runtime, "1")
	topology := createInboundTopology(t, runtime)
	orderLine := order["lines"].([]any)[0].(map[string]any)
	body := inventoryInboundBody(order, orderLine, topology, "1", "LOT-ROLLBACK-001", 1)

	runtime.events.mu.Lock()
	runtime.events.fail = true
	runtime.events.mu.Unlock()
	testRequest(
		t, runtime.handler, http.MethodPost, apiBase+"/purchase-inbounds",
		body, "inventory-rollback", true, http.StatusInternalServerError,
	)
	runtime.events.mu.Lock()
	runtime.events.fail = false
	runtime.events.mu.Unlock()

	assertInventoryRouteTotal(t, runtime.handler, "/inventory-lots", 0)
	assertInventoryRouteTotal(t, runtime.handler, "/stock-ledger", 0)
	assertInventoryRouteTotal(t, runtime.handler, "/stock-balances", 0)
	inbounds := testRequest(t, runtime.handler, http.MethodGet, apiBase+"/purchase-inbounds", nil, "", true, http.StatusOK)
	if testInt64(t, inbounds, "total") != 0 {
		t.Fatalf("failed transaction retained receipt: %v", inbounds)
	}
	currentOrder := testMap(t, testRequest(
		t, runtime.handler, http.MethodGet, apiBase+"/purchase-orders/"+testString(t, order, "id"),
		nil, "", true, http.StatusOK,
	), "item")
	if testInt64(t, currentOrder, "version") != 1 || testString(t, currentOrder, "status") != "open" {
		t.Fatalf("failed transaction changed purchase order: %v", currentOrder)
	}
	runtime.audit.mu.Lock()
	audits := append([]pluginsdk.AuditEntry(nil), runtime.audit.entries...)
	runtime.audit.mu.Unlock()
	if testAuditHasAction(audits, "pharma_oa.inventory.receive") ||
		testAuditHasAction(audits, "pharma_oa.inbound.create") {
		t.Fatalf("failed transaction retained business audit: %+v", audits)
	}

	created := testRequest(
		t, runtime.handler, http.MethodPost, apiBase+"/purchase-inbounds",
		body, "inventory-rollback", true, http.StatusCreated,
	)
	if testString(t, testMap(t, created, "item"), "number") != "PI-000001" {
		t.Fatalf("rollback consumed receipt number: %v", created)
	}
	assertInventoryRouteTotal(t, runtime.handler, "/stock-ledger", 1)
	assertInventoryRouteTotal(t, runtime.handler, "/stock-balances", 1)
}

func TestInventoryLotFactsRejectConflictAndPermitExactReuse(t *testing.T) {
	runtime := newTestRuntime(t)
	order := createApprovedPurchaseOrder(t, runtime, "2")
	topology := createInboundTopology(t, runtime)
	orderLine := order["lines"].([]any)[0].(map[string]any)
	first := inventoryInboundBody(order, orderLine, topology, "1", "LOT-IMMUTABLE-001", 1)
	testRequest(t, runtime.handler, http.MethodPost, apiBase+"/purchase-inbounds", first, "lot-first", true, http.StatusCreated)

	conflict := inventoryInboundBody(order, orderLine, topology, "1", "LOT-IMMUTABLE-001", 2)
	conflict["lines"].([]map[string]any)[0]["expiresAt"] = "2029-06-01"
	testRequest(t, runtime.handler, http.MethodPost, apiBase+"/purchase-inbounds", conflict, "lot-conflict", true, http.StatusConflict)
	assertInventoryRouteTotal(t, runtime.handler, "/inventory-lots", 1)
	assertInventoryRouteTotal(t, runtime.handler, "/stock-ledger", 1)
	balances := testRequest(t, runtime.handler, http.MethodGet, apiBase+"/stock-balances", nil, "", true, http.StatusOK)
	if testString(t, balances["items"].([]any)[0].(map[string]any), "quantity") != "1" {
		t.Fatalf("lot conflict changed balance: %v", balances)
	}

	conflict["lines"].([]map[string]any)[0]["expiresAt"] = "2028-06-01"
	testRequest(t, runtime.handler, http.MethodPost, apiBase+"/purchase-inbounds", conflict, "lot-conflict", true, http.StatusCreated)
	assertInventoryRouteTotal(t, runtime.handler, "/inventory-lots", 1)
	assertInventoryRouteTotal(t, runtime.handler, "/stock-ledger", 2)
	balances = testRequest(t, runtime.handler, http.MethodGet, apiBase+"/stock-balances", nil, "", true, http.StatusOK)
	if testString(t, balances["items"].([]any)[0].(map[string]any), "quantity") != "2" {
		t.Fatalf("exact lot reuse did not update balance: %v", balances)
	}
}

func TestInboundRejectsInactiveTopologyBeforeAnyInventoryPosting(t *testing.T) {
	runtime := newTestRuntime(t)
	order := createApprovedPurchaseOrder(t, runtime, "1")
	topology := createInboundTopology(t, runtime)
	orderLine := order["lines"].([]any)[0].(map[string]any)
	testRequest(
		t, runtime.handler, http.MethodPost,
		apiBase+"/warehouse-locations/"+topology.LocationID+"/disable",
		map[string]any{
			"tenantId": "tenant-a", "organizationId": "org-a",
			"reason": "receiving lane maintenance", "version": 1,
		},
		"inactive-topology-disable", true, http.StatusOK,
	)
	testRequest(
		t, runtime.handler, http.MethodPost, apiBase+"/purchase-inbounds",
		inventoryInboundBody(order, orderLine, topology, "1", "LOT-INACTIVE-001", 1),
		"inactive-topology-receipt", true, http.StatusConflict,
	)
	assertInventoryRouteTotal(t, runtime.handler, "/inventory-lots", 0)
	assertInventoryRouteTotal(t, runtime.handler, "/stock-ledger", 0)
	assertInventoryRouteTotal(t, runtime.handler, "/stock-balances", 0)
	currentOrder := testMap(t, testRequest(
		t, runtime.handler, http.MethodGet, apiBase+"/purchase-orders/"+testString(t, order, "id"),
		nil, "", true, http.StatusOK,
	), "item")
	if testInt64(t, currentOrder, "version") != 1 || testString(t, currentOrder, "status") != "open" {
		t.Fatalf("inactive topology changed the purchase order: %v", currentOrder)
	}
}

func TestInventoryRejectsCrossOrganizationReadAndReceipt(t *testing.T) {
	runtime := newTestRuntime(t)
	order := createApprovedPurchaseOrder(t, runtime, "1")
	topology := createInboundTopology(t, runtime)
	orderLine := order["lines"].([]any)[0].(map[string]any)
	body := inventoryInboundBody(order, orderLine, topology, "1", "LOT-SCOPE-001", 1)
	testRequest(
		t, runtime.handler, http.MethodPost, apiBase+"/purchase-inbounds",
		body, "inventory-scope-owner", true, http.StatusCreated,
	)

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
		PluginID: pluginID, Transactions: runtime.transactions,
		DataScopes: testScopes{predicate: crossPredicate}, DataStore: runtime.store,
		Events: runtime.events, DocumentNumbers: runtime.numbers, Files: runtime.files,
		Audit: runtime.audit, Workflows: runtime.workflows, Jobs: runtime.jobs,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertInventoryRouteTotal(t, crossHandler, "/inventory-lots", 0)
	assertInventoryRouteTotal(t, crossHandler, "/stock-ledger", 0)
	assertInventoryRouteTotal(t, crossHandler, "/stock-balances", 0)
	reconciliation := testRequest(
		t, crossHandler, http.MethodGet, apiBase+"/stock-reconciliation",
		nil, "", true, http.StatusOK,
	)
	if testInt64(t, reconciliation, "total") != 0 || reconciliation["matched"] != true {
		t.Fatalf("cross-organization reconciliation leaked stock: %v", reconciliation)
	}
	testRequest(
		t, crossHandler, http.MethodPost, apiBase+"/purchase-inbounds",
		body, "inventory-cross-organization", true, http.StatusForbidden,
	)
}

func TestConcurrentSameReceiptRequestPostsOnce(t *testing.T) {
	runtime := newTestRuntime(t)
	order := createApprovedPurchaseOrder(t, runtime, "1")
	topology := createInboundTopology(t, runtime)
	orderLine := order["lines"].([]any)[0].(map[string]any)
	body := inventoryInboundBody(order, orderLine, topology, "1", "LOT-IDEMPOTENT-001", 1)

	statuses := make(chan int, 2)
	var wait sync.WaitGroup
	for index := 0; index < 2; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			statuses <- rawTestRequestStatus(runtime.handler, http.MethodPost, apiBase+"/purchase-inbounds", body, "same-receipt")
		}()
	}
	wait.Wait()
	close(statuses)
	counts := map[int]int{}
	for status := range statuses {
		counts[status]++
	}
	if counts[http.StatusCreated] != 1 || counts[http.StatusOK] != 1 {
		t.Fatalf("same receipt replay statuses=%v want one created and one replay", counts)
	}
	assertInventoryRouteTotal(t, runtime.handler, "/inventory-lots", 1)
	assertInventoryRouteTotal(t, runtime.handler, "/stock-ledger", 1)
	assertInventoryRouteTotal(t, runtime.handler, "/stock-balances", 1)
	runtime.events.mu.Lock()
	eventCount := len(runtime.events.publications)
	runtime.events.mu.Unlock()
	if eventCount != 1 {
		t.Fatalf("same receipt replay published %d inventory events", eventCount)
	}
}

func TestPurchaseInboundOperationKeyIsResourceAndScopeNamespaced(t *testing.T) {
	const clientKey = "shared-client-key"
	baseScope := employeeScope{TenantID: "tenant-a", OrganizationID: "org-a", OwnerID: "actor-a"}
	base := purchaseInboundOperationKey(baseScope, clientKey)
	if base != purchaseInboundOperationKey(baseScope, clientKey) {
		t.Fatal("purchase inbound operation key is not deterministic")
	}
	if base == purchaseInboundOperationKey(
		employeeScope{TenantID: "tenant-a", OrganizationID: "org-b", OwnerID: "actor-b"},
		clientKey,
	) {
		t.Fatal("purchase inbound operation key collided across organizations")
	}
	if base == purchaseInboundOperationKey(
		employeeScope{TenantID: "tenant-b", OrganizationID: "org-a", OwnerID: "actor-b"},
		clientKey,
	) {
		t.Fatal("purchase inbound operation key collided across tenants")
	}
	if !strings.HasPrefix(base, "purchase-inbound-create-") || strings.Contains(base, clientKey) || len(base) > 128 {
		t.Fatalf("purchase inbound operation key is not bounded and opaque: %q", base)
	}
}

func TestConcurrentReceiptAttachmentAttemptsCannotDeleteCommittedObject(t *testing.T) {
	runtime := newTestRuntime(t)
	order := createApprovedPurchaseOrder(t, runtime, "1")
	topology := createInboundTopology(t, runtime)
	orderLine := order["lines"].([]any)[0].(map[string]any)
	body := inventoryInboundBody(order, orderLine, topology, "1", "LOT-ATTACHMENT-RACE-001", 1)
	body["attachments"] = []map[string]any{{
		"name": "receipt-proof.txt", "contentBase64": "cmVjZWlwdC1wcm9vZg==",
	}}

	parallelHandler, err := newHandler(pluginsdk.HostServices{
		PluginID: pluginID, Transactions: runtime.transactions,
		DataScopes: runtime.scopes, DataStore: runtime.store,
		Events: runtime.events, DocumentNumbers: runtime.numbers, Files: runtime.files,
		Audit: runtime.audit, Workflows: runtime.workflows, Jobs: runtime.jobs,
	})
	if err != nil {
		t.Fatal(err)
	}

	runtime.files.mu.Lock()
	initialStores := runtime.files.stores
	initialDeletes := runtime.files.deletes
	initialLive := len(runtime.files.items)
	initialKeys := len(runtime.files.keys)
	runtime.files.mu.Unlock()
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	runtime.files.beforeStore = func() {
		started <- struct{}{}
		<-release
	}
	statuses := make(chan int, 2)
	var wait sync.WaitGroup
	for _, handler := range []http.Handler{runtime.handler, parallelHandler} {
		wait.Add(1)
		go func(handler http.Handler) {
			defer wait.Done()
			statuses <- rawTestRequestStatus(
				handler, http.MethodPost, apiBase+"/purchase-inbounds", body, "attachment-race",
			)
		}(handler)
	}
	for index := 0; index < 2; index++ {
		select {
		case <-started:
		case <-time.After(5 * time.Second):
			close(release)
			wait.Wait()
			t.Fatal("concurrent receipt requests did not both reach attachment storage")
		}
	}
	close(release)
	wait.Wait()
	runtime.files.beforeStore = nil
	close(statuses)
	counts := map[int]int{}
	for status := range statuses {
		counts[status]++
	}
	if counts[http.StatusCreated] != 1 || counts[http.StatusOK] != 1 {
		t.Fatalf("concurrent attachment receipt statuses=%v want one created and one replay", counts)
	}

	inbounds := testRequest(
		t, runtime.handler, http.MethodGet, apiBase+"/purchase-inbounds",
		nil, "", true, http.StatusOK,
	)
	if testInt64(t, inbounds, "total") != 1 {
		t.Fatalf("concurrent attachment receipt created duplicate vouchers: %v", inbounds)
	}
	item := inbounds["items"].([]any)[0].(map[string]any)
	attachments := item["attachments"].([]any)
	if len(attachments) != 1 {
		t.Fatalf("committed receipt attachment count=%d want=1", len(attachments))
	}
	committedFileID := testString(t, attachments[0].(map[string]any), "fileId")

	runtime.files.mu.Lock()
	keys := append([]string(nil), runtime.files.keys[initialKeys:]...)
	storeCount := runtime.files.stores - initialStores
	deleteCount := runtime.files.deletes - initialDeletes
	liveCount := len(runtime.files.items) - initialLive
	_, committedObjectExists := runtime.files.items[committedFileID]
	runtime.files.mu.Unlock()
	if storeCount != 2 || deleteCount != 1 || liveCount != 1 || !committedObjectExists {
		t.Fatalf(
			"attachment ownership cleanup stores=%d deletes=%d live=%d committed=%v",
			storeCount, deleteCount, liveCount, committedObjectExists,
		)
	}
	if len(keys) != 2 || keys[0] == keys[1] {
		t.Fatalf("concurrent attachment attempts reused an object key: %v", keys)
	}
}

func TestStockReconciliationTraversesEveryAggregatePage(t *testing.T) {
	runtime := newTestRuntime(t)
	err := runtime.transactions.Within(context.Background(), func(tx pluginsdk.Transaction) error {
		for index := 0; index < 101; index++ {
			suffix := fmt.Sprintf("%03d", index)
			entry := stockLedgerEntry{
				ID: "ledger-page-" + suffix, EntryType: "receipt",
				ProductID: "product-page-" + suffix, LotID: "lot-page-" + suffix, BatchNo: "BATCH-PAGE-" + suffix,
				WarehouseID: "warehouse-page", AreaID: "area-page", LocationID: "location-page-" + suffix,
				SourceDocumentType: "purchase_inbound", SourceDocumentID: "inbound-page-" + suffix,
				SourceDocumentNumber: "PI-PAGE-" + suffix, SourceDocumentLineID: "line-page-" + suffix,
				OccurredAt: "2026-07-28T08:00:00Z", quantityMicros: 1_000_000, scope: runtime.store.scope,
			}
			if _, mutationErr := runtime.store.Mutate(tx.Context(), pluginsdk.DataMutation{
				Table: stockLedgerTable, Operation: pluginsdk.DataMutationInsert,
				Scope:  inventoryIntent("receive", runtime.store.scope),
				Key:    map[string]pluginsdk.DataValue{"id": stringValue(entry.ID)},
				Values: stockLedgerValues(entry), IdempotencyKey: entry.ID + ".append",
			}); mutationErr != nil {
				return mutationErr
			}
			balanceID := "balance-page-" + suffix
			if _, mutationErr := runtime.store.Mutate(tx.Context(), pluginsdk.DataMutation{
				Table: stockBalanceTable, Operation: pluginsdk.DataMutationInsert,
				Scope: inventoryIntent("receive", runtime.store.scope),
				Key:   map[string]pluginsdk.DataValue{"id": stringValue(balanceID)},
				Values: map[string]pluginsdk.DataValue{
					"product_id": stringValue(entry.ProductID), "lot_id": stringValue(entry.LotID),
					"batch_no": stringValue(entry.BatchNo), "warehouse_id": stringValue(entry.WarehouseID),
					"area_id": stringValue(entry.AreaID), "location_id": stringValue(entry.LocationID),
					"quantity_micros": integerValue(entry.quantityMicros),
				},
				IdempotencyKey: balanceID + ".open",
			}); mutationErr != nil {
				return mutationErr
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	reconciliation := testRequest(
		t, runtime.handler, http.MethodGet, apiBase+"/stock-reconciliation",
		nil, "", true, http.StatusOK,
	)
	if testInt64(t, reconciliation, "total") != 101 || reconciliation["matched"] != true {
		t.Fatalf("reconciliation did not traverse all aggregate pages: %v", reconciliation)
	}
}

func inventoryInboundBody(
	order map[string]any,
	orderLine map[string]any,
	topology inboundTopology,
	quantity string,
	batchNo string,
	orderVersion int64,
) map[string]any {
	return map[string]any{
		"tenantId": "tenant-a", "organizationId": "org-a", "purchaseOrderId": order["id"],
		"warehouseId": topology.WarehouseID, "areaId": topology.AreaID, "locationId": topology.LocationID,
		"orderVersion": orderVersion,
		"lines": []map[string]any{{
			"orderLineId": orderLine["id"], "quantity": quantity, "batchNo": batchNo,
			"productionDate": "2026-06-01", "expiresAt": "2028-06-01",
		}},
		"attachments": []map[string]any{},
	}
}

func assertInventoryRouteTotal(t *testing.T, handler http.Handler, route string, want int64) {
	t.Helper()
	response := testRequest(t, handler, http.MethodGet, apiBase+route, nil, "", true, http.StatusOK)
	if got := testInt64(t, response, "total"); got != want {
		t.Fatalf("%s total=%d want=%d response=%v", route, got, want, response)
	}
}

func assertAppendOnlyMutationRejected(t *testing.T, runtime testRuntime, table, id string) {
	t.Helper()
	for _, mutation := range []pluginsdk.DataMutation{
		{
			Table: table, Operation: pluginsdk.DataMutationUpdate, Scope: inventoryIntent("receive", runtime.store.scope),
			Key:            map[string]pluginsdk.DataValue{"id": stringValue(id)},
			Values:         map[string]pluginsdk.DataValue{"product_id": stringValue("tampered")},
			IdempotencyKey: id + ".forbidden-update", ExpectedVersion: int64Pointer(1),
		},
		{
			Table: table, Operation: pluginsdk.DataMutationDelete, Scope: inventoryIntent("receive", runtime.store.scope),
			Key:            map[string]pluginsdk.DataValue{"id": stringValue(id)},
			IdempotencyKey: id + ".forbidden-delete",
		},
	} {
		err := runtime.transactions.Within(context.Background(), func(tx pluginsdk.Transaction) error {
			_, mutationErr := runtime.store.Mutate(tx.Context(), mutation)
			return mutationErr
		})
		var datastoreErr *pluginsdk.DataStoreError
		if !errors.As(err, &datastoreErr) || datastoreErr.Code != pluginsdk.DataStoreErrorUnsupported {
			t.Fatalf("%s %s error=%v want append-only denial", table, mutation.Operation, err)
		}
	}
}

func int64Pointer(value int64) *int64 {
	return &value
}
