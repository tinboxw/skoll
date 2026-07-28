package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"testing"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

type movementStockFixture struct {
	runtime              testRuntime
	source               inboundTopology
	productID            string
	lotID                string
	receiptLedgerEntryID string
}

func TestStockTransferPostsPairedLedgerAndIsIdempotent(t *testing.T) {
	fixture := newMovementStockFixture(t, "10", "LOT-MOVE-TRANSFER")
	destination := createMovementTopology(t, fixture.runtime, "TRANSFER")
	body := transferCreateBody(fixture, destination, "3")

	created := testRequest(
		t, fixture.runtime.handler, http.MethodPost, apiBase+"/stock-transfers",
		body, "movement-transfer-create", true, http.StatusCreated,
	)
	if created["duplicate"] != false {
		t.Fatalf("new transfer was reported as duplicate: %v", created)
	}
	item := testMap(t, created, "item")
	if testString(t, item, "status") != "posted" {
		t.Fatalf("transfer did not post immediately: %v", item)
	}
	transferID := testString(t, item, "id")
	listed := testRequest(
		t, fixture.runtime.handler, http.MethodGet, apiBase+"/stock-transfers",
		nil, "", true, http.StatusOK,
	)
	if testInt64(t, listed, "total") != 1 {
		t.Fatalf("posted transfer is not queryable: %v", listed)
	}
	got := testMap(t, testRequest(
		t, fixture.runtime.handler, http.MethodGet, apiBase+"/stock-transfers/"+transferID,
		nil, "", true, http.StatusOK,
	), "item")
	if testString(t, got, "id") != transferID || testString(t, got, "status") != "posted" {
		t.Fatalf("transfer detail lost its current state: %v", got)
	}

	entries := movementLedgerEntries(t, fixture.runtime.handler, "stock_transfer", transferID)
	assertMovementEntryQuantities(t, entries, map[string]string{
		"transfer_out": "-3",
		"transfer_in":  "3",
	})
	assertMovementBalance(t, fixture.runtime.handler, fixture.productID, fixture.lotID, fixture.source.LocationID, "7")
	assertMovementBalance(t, fixture.runtime.handler, fixture.productID, fixture.lotID, destination.LocationID, "3")
	assertMovementSourceReconciled(t, fixture.runtime.handler, "stock_transfer", transferID, 2)

	replayed := testRequest(
		t, fixture.runtime.handler, http.MethodPost, apiBase+"/stock-transfers",
		body, "movement-transfer-create", true, http.StatusOK,
	)
	if replayed["duplicate"] != true ||
		testString(t, testMap(t, replayed, "item"), "id") != transferID {
		t.Fatalf("transfer replay was not idempotent: %v", replayed)
	}
	conflict := testDeepClone(body)
	conflict["lines"].([]any)[0].(map[string]any)["quantity"] = "2"
	testRequest(
		t, fixture.runtime.handler, http.MethodPost, apiBase+"/stock-transfers",
		conflict, "movement-transfer-create", true, http.StatusConflict,
	)
	if got := len(movementLedgerEntries(t, fixture.runtime.handler, "stock_transfer", transferID)); got != 2 {
		t.Fatalf("conflicting transfer replay changed ledger count to %d", got)
	}
	assertMovementBalance(t, fixture.runtime.handler, fixture.productID, fixture.lotID, fixture.source.LocationID, "7")
	assertMovementBalance(t, fixture.runtime.handler, fixture.productID, fixture.lotID, destination.LocationID, "3")
}

func TestStocktakeZeroDifferenceAndApprovedDifference(t *testing.T) {
	t.Run("zero_difference_completes_without_workflow_or_ledger", func(t *testing.T) {
		fixture := newMovementStockFixture(t, "4", "LOT-MOVE-COUNT-ZERO")
		body := pendingMovementCreateBase(
			fixture, "manager-1", "Inventory Manager", "quarterly exact count",
		)
		body["lines"] = []map[string]any{{
			"productId": fixture.productID, "lotId": fixture.lotID, "countedQuantity": "4",
		}}
		created := testRequest(
			t, fixture.runtime.handler, http.MethodPost, apiBase+"/stocktakes",
			body, "movement-count-zero", true, http.StatusCreated,
		)
		item := testMap(t, created, "item")
		if testString(t, item, "status") != "completed" || created["workflow"] != nil {
			t.Fatalf("zero-difference stocktake did not complete directly: %v", created)
		}
		if entries := movementLedgerEntries(
			t, fixture.runtime.handler, "stocktake", testString(t, item, "id"),
		); len(entries) != 0 {
			t.Fatalf("zero-difference stocktake created ledger entries: %v", entries)
		}
		assertMovementBalance(t, fixture.runtime.handler, fixture.productID, fixture.lotID, fixture.source.LocationID, "4")
	})

	t.Run("difference_waits_for_approval_then_posts_once", func(t *testing.T) {
		fixture := newMovementStockFixture(t, "4", "LOT-MOVE-COUNT-DIFF")
		body := pendingMovementCreateBase(fixture, "manager-1", "Inventory Manager", "count variance")
		body["lines"] = []map[string]any{{
			"productId": fixture.productID, "lotId": fixture.lotID, "countedQuantity": "3",
		}}
		created := testRequest(
			t, fixture.runtime.handler, http.MethodPost, apiBase+"/stocktakes",
			body, "movement-count-difference", true, http.StatusCreated,
		)
		item := testMap(t, created, "item")
		assertPendingMovement(t, created)
		stocktakeID := testString(t, item, "id")
		if entries := movementLedgerEntries(t, fixture.runtime.handler, "stocktake", stocktakeID); len(entries) != 0 {
			t.Fatalf("unapproved stocktake changed ledger: %v", entries)
		}
		assertMovementBalance(t, fixture.runtime.handler, fixture.productID, fixture.lotID, fixture.source.LocationID, "4")

		decision := movementDecisionBody(t, created, "count verified")
		approved := testRequest(
			t, fixture.runtime.handler, http.MethodPost, apiBase+"/stocktakes/"+stocktakeID+"/approve",
			decision, "movement-count-approve", true, http.StatusOK,
		)
		if approved["duplicate"] != false ||
			testString(t, testMap(t, approved, "item"), "status") != "posted" {
			t.Fatalf("approved stocktake did not post: %v", approved)
		}
		entries := movementLedgerEntries(t, fixture.runtime.handler, "stocktake", stocktakeID)
		assertMovementEntryQuantities(t, entries, map[string]string{"stocktake_loss": "-1"})
		assertMovementBalance(t, fixture.runtime.handler, fixture.productID, fixture.lotID, fixture.source.LocationID, "3")
		assertMovementSourceReconciled(t, fixture.runtime.handler, "stocktake", stocktakeID, 1)

		replayed := testRequest(
			t, fixture.runtime.handler, http.MethodPost, apiBase+"/stocktakes/"+stocktakeID+"/approve",
			decision, "movement-count-approve", true, http.StatusOK,
		)
		if replayed["duplicate"] != true {
			t.Fatalf("stocktake decision replay was not idempotent: %v", replayed)
		}
		conflict := testDeepClone(decision)
		conflict["comment"] = "different decision payload"
		testRequest(
			t, fixture.runtime.handler, http.MethodPost, apiBase+"/stocktakes/"+stocktakeID+"/approve",
			conflict, "movement-count-approve", true, http.StatusConflict,
		)
		if got := len(movementLedgerEntries(t, fixture.runtime.handler, "stocktake", stocktakeID)); got != 1 {
			t.Fatalf("stocktake decision replay changed ledger count to %d", got)
		}
	})
}

func TestStocktakeRejectsStaleBalanceSnapshot(t *testing.T) {
	fixture := newMovementStockFixture(t, "4", "LOT-MOVE-COUNT-STALE")
	body := pendingMovementCreateBase(fixture, "manager-1", "Inventory Manager", "stale count")
	body["lines"] = []map[string]any{{
		"productId": fixture.productID, "lotId": fixture.lotID, "countedQuantity": "2",
	}}
	created := testRequest(
		t, fixture.runtime.handler, http.MethodPost, apiBase+"/stocktakes",
		body, "movement-count-stale", true, http.StatusCreated,
	)
	stocktakeID := testString(t, testMap(t, created, "item"), "id")
	destination := createMovementTopology(t, fixture.runtime, "COUNT-STALE")
	testRequest(
		t, fixture.runtime.handler, http.MethodPost, apiBase+"/stock-transfers",
		transferCreateBody(fixture, destination, "1"), "movement-count-stale-transfer", true, http.StatusCreated,
	)
	assertMovementBalance(t, fixture.runtime.handler, fixture.productID, fixture.lotID, fixture.source.LocationID, "3")

	testRequest(
		t, fixture.runtime.handler, http.MethodPost, apiBase+"/stocktakes/"+stocktakeID+"/approve",
		movementDecisionBody(t, created, "approve stale count"),
		"movement-count-stale-approve", true, http.StatusConflict,
	)
	current := testRequest(
		t, fixture.runtime.handler, http.MethodGet, apiBase+"/stocktakes/"+stocktakeID,
		nil, "", true, http.StatusOK,
	)
	assertPendingMovement(t, current)
	if entries := movementLedgerEntries(t, fixture.runtime.handler, "stocktake", stocktakeID); len(entries) != 0 {
		t.Fatalf("stale stocktake left ledger entries: %v", entries)
	}
	assertMovementBalance(t, fixture.runtime.handler, fixture.productID, fixture.lotID, fixture.source.LocationID, "3")
}

func TestStockAdjustmentApprovalNegativeGuardAndRejection(t *testing.T) {
	fixture := newMovementStockFixture(t, "10", "LOT-MOVE-ADJUST")

	positive := adjustmentCreateBody(fixture, "2", "approved positive correction")
	positiveCreated := testRequest(
		t, fixture.runtime.handler, http.MethodPost, apiBase+"/stock-adjustments",
		positive, "movement-adjust-positive", true, http.StatusCreated,
	)
	assertPendingMovement(t, positiveCreated)
	positiveItem := testMap(t, positiveCreated, "item")
	testRequest(
		t, fixture.runtime.handler, http.MethodPost,
		apiBase+"/stock-adjustments/"+testString(t, positiveItem, "id")+"/approve",
		movementDecisionBody(t, positiveCreated, "positive evidence accepted"),
		"movement-adjust-positive-approve", true, http.StatusOK,
	)
	assertMovementEntryQuantities(
		t,
		movementLedgerEntries(t, fixture.runtime.handler, "stock_adjustment", testString(t, positiveItem, "id")),
		map[string]string{"adjustment_gain": "2"},
	)
	assertMovementBalance(t, fixture.runtime.handler, fixture.productID, fixture.lotID, fixture.source.LocationID, "12")

	negative := adjustmentCreateBody(fixture, "-3", "approved negative correction")
	negativeCreated := testRequest(
		t, fixture.runtime.handler, http.MethodPost, apiBase+"/stock-adjustments",
		negative, "movement-adjust-negative", true, http.StatusCreated,
	)
	negativeItem := testMap(t, negativeCreated, "item")
	testRequest(
		t, fixture.runtime.handler, http.MethodPost,
		apiBase+"/stock-adjustments/"+testString(t, negativeItem, "id")+"/approve",
		movementDecisionBody(t, negativeCreated, "negative evidence accepted"),
		"movement-adjust-negative-approve", true, http.StatusOK,
	)
	assertMovementEntryQuantities(
		t,
		movementLedgerEntries(t, fixture.runtime.handler, "stock_adjustment", testString(t, negativeItem, "id")),
		map[string]string{"adjustment_loss": "-3"},
	)
	assertMovementBalance(t, fixture.runtime.handler, fixture.productID, fixture.lotID, fixture.source.LocationID, "9")

	rejectedCreated := testRequest(
		t, fixture.runtime.handler, http.MethodPost, apiBase+"/stock-adjustments",
		adjustmentCreateBody(fixture, "1", "rejected correction"),
		"movement-adjust-reject", true, http.StatusCreated,
	)
	rejectedItem := testMap(t, rejectedCreated, "item")
	rejected := testRequest(
		t, fixture.runtime.handler, http.MethodPost,
		apiBase+"/stock-adjustments/"+testString(t, rejectedItem, "id")+"/reject",
		movementDecisionBody(t, rejectedCreated, "evidence rejected"),
		"movement-adjust-reject-decision", true, http.StatusOK,
	)
	if testString(t, testMap(t, rejected, "item"), "status") != "rejected" {
		t.Fatalf("rejected adjustment has wrong state: %v", rejected)
	}
	if entries := movementLedgerEntries(
		t, fixture.runtime.handler, "stock_adjustment", testString(t, rejectedItem, "id"),
	); len(entries) != 0 {
		t.Fatalf("rejected adjustment posted ledger entries: %v", entries)
	}

	overdrawCreated := testRequest(
		t, fixture.runtime.handler, http.MethodPost, apiBase+"/stock-adjustments",
		adjustmentCreateBody(fixture, "-10", "overdraw attempt"),
		"movement-adjust-overdraw", true, http.StatusCreated,
	)
	overdrawID := testString(t, testMap(t, overdrawCreated, "item"), "id")
	testRequest(
		t, fixture.runtime.handler, http.MethodPost, apiBase+"/stock-adjustments/"+overdrawID+"/approve",
		movementDecisionBody(t, overdrawCreated, "must not overdraw"),
		"movement-adjust-overdraw-approve", true, http.StatusConflict,
	)
	overdrawCurrent := testRequest(
		t, fixture.runtime.handler, http.MethodGet, apiBase+"/stock-adjustments/"+overdrawID,
		nil, "", true, http.StatusOK,
	)
	assertPendingMovement(t, overdrawCurrent)
	if entries := movementLedgerEntries(t, fixture.runtime.handler, "stock_adjustment", overdrawID); len(entries) != 0 {
		t.Fatalf("negative-stock rejection retained ledger entries: %v", entries)
	}
	assertMovementBalance(t, fixture.runtime.handler, fixture.productID, fixture.lotID, fixture.source.LocationID, "9")
	assertMovementReconciled(t, fixture.runtime.handler, fixture.productID)
}

func TestPurchaseReturnTracesReceiptAndEnforcesSourceLimit(t *testing.T) {
	fixture := newMovementStockFixture(t, "5", "LOT-MOVE-RETURN")
	first := returnCreateBody(fixture, "3", "damaged shipment return")
	firstCreated := testRequest(
		t, fixture.runtime.handler, http.MethodPost, apiBase+"/stock-returns",
		first, "movement-return-first", true, http.StatusCreated,
	)
	second := returnCreateBody(fixture, "3", "duplicate quantity return")
	secondCreated := testRequest(
		t, fixture.runtime.handler, http.MethodPost, apiBase+"/stock-returns",
		second, "movement-return-second", true, http.StatusCreated,
	)

	firstItem := testMap(t, firstCreated, "item")
	firstID := testString(t, firstItem, "id")
	testRequest(
		t, fixture.runtime.handler, http.MethodPost, apiBase+"/stock-returns/"+firstID+"/approve",
		movementDecisionBody(t, firstCreated, "supplier accepted return"),
		"movement-return-first-approve", true, http.StatusOK,
	)
	entries := movementLedgerEntries(t, fixture.runtime.handler, "stock_return", firstID)
	assertMovementEntryQuantities(t, entries, map[string]string{"purchase_return_out": "-3"})
	if lines, ok := firstItem["lines"].([]any); !ok || len(lines) != 1 ||
		testString(t, lines[0].(map[string]any), "sourceLedgerEntryId") != fixture.receiptLedgerEntryID {
		t.Fatalf("return lost its immutable receipt-ledger trace: %v", firstItem)
	}
	assertMovementBalance(t, fixture.runtime.handler, fixture.productID, fixture.lotID, fixture.source.LocationID, "2")

	secondID := testString(t, testMap(t, secondCreated, "item"), "id")
	testRequest(
		t, fixture.runtime.handler, http.MethodPost, apiBase+"/stock-returns/"+secondID+"/approve",
		movementDecisionBody(t, secondCreated, "must exceed original source"),
		"movement-return-second-approve", true, http.StatusConflict,
	)
	secondCurrent := testRequest(
		t, fixture.runtime.handler, http.MethodGet, apiBase+"/stock-returns/"+secondID,
		nil, "", true, http.StatusOK,
	)
	assertPendingMovement(t, secondCurrent)
	if entries := movementLedgerEntries(t, fixture.runtime.handler, "stock_return", secondID); len(entries) != 0 {
		t.Fatalf("over-return retained ledger entries: %v", entries)
	}
	assertMovementBalance(t, fixture.runtime.handler, fixture.productID, fixture.lotID, fixture.source.LocationID, "2")
	assertMovementSourceReconciled(t, fixture.runtime.handler, "stock_return", firstID, 1)
}

func TestInventoryMovementRejectsDisabledTopologyAndCrossScope(t *testing.T) {
	fixture := newMovementStockFixture(t, "5", "LOT-MOVE-SCOPE")
	destination := createMovementTopology(t, fixture.runtime, "DISABLED")
	testRequest(
		t, fixture.runtime.handler, http.MethodPost,
		apiBase+"/warehouse-locations/"+destination.LocationID+"/disable",
		map[string]any{
			"tenantId": "tenant-a", "organizationId": "org-a",
			"reason": "movement acceptance maintenance", "version": 1,
		},
		"movement-disable-destination", true, http.StatusOK,
	)
	testRequest(
		t, fixture.runtime.handler, http.MethodPost, apiBase+"/stock-transfers",
		transferCreateBody(fixture, destination, "1"),
		"movement-disabled-transfer", true, http.StatusConflict,
	)
	assertMovementDocumentTotal(t, fixture.runtime.handler, "/stock-transfers", 0)
	assertMovementBalance(t, fixture.runtime.handler, fixture.productID, fixture.lotID, fixture.source.LocationID, "5")
	assertMovementBalance(t, fixture.runtime.handler, fixture.productID, fixture.lotID, destination.LocationID, "0")

	adjustment := testRequest(
		t, fixture.runtime.handler, http.MethodPost, apiBase+"/stock-adjustments",
		adjustmentCreateBody(fixture, "-1", "approval topology revalidation"),
		"movement-disabled-approval-create", true, http.StatusCreated,
	)
	adjustmentID := testString(t, testMap(t, adjustment, "item"), "id")
	testRequest(
		t, fixture.runtime.handler, http.MethodPost,
		apiBase+"/warehouse-locations/"+fixture.source.LocationID+"/disable",
		map[string]any{
			"tenantId": "tenant-a", "organizationId": "org-a",
			"reason": "source closed before approval", "version": 1,
		},
		"movement-disable-source", true, http.StatusOK,
	)
	testRequest(
		t, fixture.runtime.handler, http.MethodPost,
		apiBase+"/stock-adjustments/"+adjustmentID+"/approve",
		movementDecisionBody(t, adjustment, "must revalidate location"),
		"movement-disabled-approval", true, http.StatusConflict,
	)
	currentAdjustment := testRequest(
		t, fixture.runtime.handler, http.MethodGet, apiBase+"/stock-adjustments/"+adjustmentID,
		nil, "", true, http.StatusOK,
	)
	assertPendingMovement(t, currentAdjustment)
	if entries := movementLedgerEntries(
		t, fixture.runtime.handler, "stock_adjustment", adjustmentID,
	); len(entries) != 0 {
		t.Fatalf("disabled-location approval retained ledger entries: %v", entries)
	}
	assertMovementBalance(t, fixture.runtime.handler, fixture.productID, fixture.lotID, fixture.source.LocationID, "5")

	crossScope := employeeScope{TenantID: "tenant-a", OrganizationID: "org-b", OwnerID: "actor-b"}
	crossPredicate, err := pluginsdk.NewScopePredicate(pluginsdk.TrustedScope{
		SubjectID: crossScope.OwnerID, TenantIDs: []string{crossScope.TenantID},
		OrganizationIDs: []string{crossScope.OrganizationID}, OwnerIDs: []string{crossScope.OwnerID},
	})
	if err != nil {
		t.Fatal(err)
	}
	originalScope := fixture.runtime.store.scope
	fixture.runtime.store.scope = crossScope
	defer func() { fixture.runtime.store.scope = originalScope }()
	crossHandler, err := newHandler(pluginsdk.HostServices{
		PluginID: pluginID, Transactions: fixture.runtime.transactions,
		DataScopes: testScopes{predicate: crossPredicate}, DataStore: fixture.runtime.store,
		Events: fixture.runtime.events, DocumentNumbers: fixture.runtime.numbers,
		Files: fixture.runtime.files, Audit: fixture.runtime.audit,
		Workflows: fixture.runtime.workflows, Jobs: fixture.runtime.jobs,
	})
	if err != nil {
		t.Fatal(err)
	}
	crossBody := transferCreateBody(fixture, destination, "1")
	testRequest(
		t, crossHandler, http.MethodPost, apiBase+"/stock-transfers",
		crossBody, "movement-cross-scope-transfer", true, http.StatusForbidden,
	)
	for _, route := range []string{"/stock-transfers", "/stocktakes", "/stock-adjustments", "/stock-returns"} {
		assertMovementDocumentTotal(t, crossHandler, route, 0)
	}
}

func TestInventoryMovementRejectsInvalidAndExpiredLotsAtomically(t *testing.T) {
	fixture := newMovementStockFixture(t, "5", "LOT-MOVE-VALIDITY")
	destination := createMovementTopology(t, fixture.runtime, "VALIDITY")

	invalid := transferCreateBody(fixture, destination, "1")
	invalid["lines"].([]map[string]any)[0]["lotId"] = "missing-lot"
	testRequest(
		t, fixture.runtime.handler, http.MethodPost, apiBase+"/stock-transfers",
		invalid, "movement-invalid-lot", true, http.StatusConflict,
	)
	assertMovementDocumentTotal(t, fixture.runtime.handler, "/stock-transfers", 0)
	assertMovementBalance(
		t, fixture.runtime.handler, fixture.productID, fixture.lotID,
		fixture.source.LocationID, "5",
	)

	fixture.runtime.store.mu.Lock()
	lot := fixture.runtime.store.records[fixture.lotID]
	lot.Values["expires_at"] = timestampValue("2020-01-01T00:00:00Z")
	fixture.runtime.store.records[fixture.lotID] = lot
	fixture.runtime.store.mu.Unlock()
	testRequest(
		t, fixture.runtime.handler, http.MethodPost, apiBase+"/stock-transfers",
		transferCreateBody(fixture, destination, "1"),
		"movement-expired-lot", true, http.StatusConflict,
	)
	assertMovementDocumentTotal(t, fixture.runtime.handler, "/stock-transfers", 0)
	assertMovementBalance(
		t, fixture.runtime.handler, fixture.productID, fixture.lotID,
		fixture.source.LocationID, "5",
	)
	assertMovementReconciled(t, fixture.runtime.handler, fixture.productID)
}

func TestInventoryMovementUsesSourceOwnerForAuthorizedOrganizationActor(t *testing.T) {
	fixture := newMovementStockFixture(t, "5", "LOT-MOVE-SHARED-OWNER")
	destination := createMovementTopology(t, fixture.runtime, "SHARED-OWNER")
	predicate, err := pluginsdk.NewScopePredicate(pluginsdk.TrustedScope{
		SubjectID:       "actor-2",
		TenantIDs:       []string{"tenant-a"},
		OrganizationIDs: []string{"org-a"},
		AllOwners:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	handler, err := newHandler(pluginsdk.HostServices{
		PluginID: pluginID, Transactions: fixture.runtime.transactions,
		DataScopes: testScopes{predicate: predicate}, DataStore: fixture.runtime.store,
		Events: fixture.runtime.events, DocumentNumbers: fixture.runtime.numbers,
		Files: fixture.runtime.files, Audit: fixture.runtime.audit,
		Workflows: fixture.runtime.workflows, Jobs: fixture.runtime.jobs,
	})
	if err != nil {
		t.Fatal(err)
	}

	created := testRequest(
		t, handler, http.MethodPost, apiBase+"/stock-transfers",
		transferCreateBody(fixture, destination, "2"),
		"movement-shared-owner-transfer", true, http.StatusCreated,
	)
	item := testMap(t, created, "item")
	if testString(t, item, "requesterId") != "actor-2" ||
		testString(t, item, "status") != movementStatusPosted {
		t.Fatalf("organization actor or movement state was not preserved: %v", item)
	}
	assertMovementBalance(
		t, handler, fixture.productID, fixture.lotID,
		fixture.source.LocationID, "3",
	)
	assertMovementBalance(
		t, handler, fixture.productID, fixture.lotID,
		destination.LocationID, "2",
	)
	assertMovementReconciled(t, handler, fixture.productID)
}

func TestInventoryMovementRejectsOrganizationActorWithoutSourceOwnerScope(t *testing.T) {
	fixture := newMovementStockFixture(t, "5", "LOT-MOVE-OWNER-DENIED")
	destination := createMovementTopology(t, fixture.runtime, "OWNER-DENIED")
	predicate, err := pluginsdk.NewScopePredicate(pluginsdk.TrustedScope{
		SubjectID:       "actor-2",
		TenantIDs:       []string{"tenant-a"},
		OrganizationIDs: []string{"org-a"},
		OwnerIDs:        []string{"actor-2"},
	})
	if err != nil {
		t.Fatal(err)
	}
	handler, err := newHandler(pluginsdk.HostServices{
		PluginID: pluginID, Transactions: fixture.runtime.transactions,
		DataScopes: testScopes{predicate: predicate}, DataStore: fixture.runtime.store,
		Events: fixture.runtime.events, DocumentNumbers: fixture.runtime.numbers,
		Files: fixture.runtime.files, Audit: fixture.runtime.audit,
		Workflows: fixture.runtime.workflows, Jobs: fixture.runtime.jobs,
	})
	if err != nil {
		t.Fatal(err)
	}

	testRequest(
		t, handler, http.MethodPost, apiBase+"/stock-transfers",
		transferCreateBody(fixture, destination, "1"),
		"movement-owner-denied", true, http.StatusForbidden,
	)
	assertMovementDocumentTotal(t, fixture.runtime.handler, "/stock-transfers", 0)
	assertMovementBalance(
		t, fixture.runtime.handler, fixture.productID, fixture.lotID,
		fixture.source.LocationID, "5",
	)
	assertMovementReconciled(t, fixture.runtime.handler, fixture.productID)
}

func TestConcurrentTransfersPreventNegativeStock(t *testing.T) {
	fixture := newMovementStockFixture(t, "10", "LOT-MOVE-CONCURRENT")
	destinationA := createMovementTopology(t, fixture.runtime, "RACE-A")
	destinationB := createMovementTopology(t, fixture.runtime, "RACE-B")
	bodies := []map[string]any{
		transferCreateBody(fixture, destinationA, "6"),
		transferCreateBody(fixture, destinationB, "6"),
	}
	statuses := make(chan int, len(bodies))
	var wait sync.WaitGroup
	for index, body := range bodies {
		wait.Add(1)
		go func(index int, body map[string]any) {
			defer wait.Done()
			statuses <- rawTestRequestStatus(
				fixture.runtime.handler, http.MethodPost, apiBase+"/stock-transfers",
				body, fmt.Sprintf("movement-transfer-race-%d", index),
			)
		}(index, body)
	}
	wait.Wait()
	close(statuses)
	counts := make(map[int]int)
	for status := range statuses {
		counts[status]++
	}
	if counts[http.StatusCreated] != 1 || counts[http.StatusConflict] != 1 {
		t.Fatalf("concurrent transfer statuses=%v want one posted and one conflict", counts)
	}
	assertMovementDocumentTotal(t, fixture.runtime.handler, "/stock-transfers", 1)
	assertMovementBalance(t, fixture.runtime.handler, fixture.productID, fixture.lotID, fixture.source.LocationID, "4")
	destinationTotal := movementBalanceMicros(
		t, fixture.runtime.handler, fixture.productID, fixture.lotID, destinationA.LocationID,
	) + movementBalanceMicros(
		t, fixture.runtime.handler, fixture.productID, fixture.lotID, destinationB.LocationID,
	)
	if destinationTotal != 6_000_000 {
		t.Fatalf("concurrent transfer destination total=%d want=6000000", destinationTotal)
	}
	ledger := testRequest(
		t, fixture.runtime.handler, http.MethodGet,
		apiBase+"/stock-ledger?sourceDocumentType=stock_transfer",
		nil, "", true, http.StatusOK,
	)
	if testInt64(t, ledger, "total") != 2 {
		t.Fatalf("concurrent transfers retained partial or duplicate legs: %v", ledger)
	}
	assertMovementReconciled(t, fixture.runtime.handler, fixture.productID)
}

func TestStockTransferSequenceProperty(t *testing.T) {
	fixture := newMovementStockFixture(t, "101", "LOT-MOVE-PROPERTY")
	destination := createMovementTopology(t, fixture.runtime, "PROPERTY")
	random := rand.New(rand.NewSource(505))
	sourceQuantity, destinationQuantity := 101, 0

	for step := 0; step < 50; step++ {
		from, to := fixture.source, destination
		available := sourceQuantity
		reverse := random.Intn(2) == 1
		if sourceQuantity == 0 {
			reverse = true
		} else if destinationQuantity == 0 {
			reverse = false
		}
		if reverse {
			from, to = destination, fixture.source
			available = destinationQuantity
		}
		limit := available
		if limit > 9 {
			limit = 9
		}
		quantity := random.Intn(limit) + 1
		body := map[string]any{
			"tenantId": "tenant-a", "organizationId": "org-a",
			"warehouseId": from.WarehouseID, "areaId": from.AreaID, "locationId": from.LocationID,
			"destinationWarehouseId": to.WarehouseID, "destinationAreaId": to.AreaID,
			"destinationLocationId": to.LocationID, "reason": "deterministic movement sequence property",
			"lines": []map[string]any{{
				"productId": fixture.productID, "lotId": fixture.lotID, "quantity": fmt.Sprintf("%d", quantity),
			}},
		}
		created := testRequest(
			t, fixture.runtime.handler, http.MethodPost, apiBase+"/stock-transfers",
			body, fmt.Sprintf("movement-property-%02d", step), true, http.StatusCreated,
		)
		item := testMap(t, created, "item")
		entries := movementLedgerEntries(
			t, fixture.runtime.handler, movementStockTransfer, testString(t, item, "id"),
		)
		assertMovementEntryQuantities(t, entries, map[string]string{
			"transfer_out": fmt.Sprintf("-%d", quantity),
			"transfer_in":  fmt.Sprintf("%d", quantity),
		})
		if reverse {
			destinationQuantity -= quantity
			sourceQuantity += quantity
		} else {
			sourceQuantity -= quantity
			destinationQuantity += quantity
		}
	}

	if sourceQuantity+destinationQuantity != 101 {
		t.Fatalf("movement model total=%d want=101", sourceQuantity+destinationQuantity)
	}
	assertMovementBalance(
		t, fixture.runtime.handler, fixture.productID, fixture.lotID,
		fixture.source.LocationID, fmt.Sprintf("%d", sourceQuantity),
	)
	assertMovementBalance(
		t, fixture.runtime.handler, fixture.productID, fixture.lotID,
		destination.LocationID, fmt.Sprintf("%d", destinationQuantity),
	)
	assertMovementReconciled(t, fixture.runtime.handler, fixture.productID)
}

func TestMovementEventFailureRollsBackApprovalWorkflowLedgerAndBalance(t *testing.T) {
	fixture := newMovementStockFixture(t, "3", "LOT-MOVE-ROLLBACK")
	created := testRequest(
		t, fixture.runtime.handler, http.MethodPost, apiBase+"/stock-adjustments",
		adjustmentCreateBody(fixture, "1", "event rollback adjustment"),
		"movement-event-failure-create", true, http.StatusCreated,
	)
	item := testMap(t, created, "item")
	adjustmentID := testString(t, item, "id")
	decision := movementDecisionBody(t, created, "approval must roll back")
	initialEventCount := testEventPublicationCount(fixture.runtime.events)
	initialAuditCount := testAuditEntryCount(fixture.runtime.audit)

	fixture.runtime.events.mu.Lock()
	fixture.runtime.events.fail = true
	fixture.runtime.events.mu.Unlock()
	testRequest(
		t, fixture.runtime.handler, http.MethodPost,
		apiBase+"/stock-adjustments/"+adjustmentID+"/approve",
		decision, "movement-event-failure-approve", true, http.StatusInternalServerError,
	)
	fixture.runtime.events.mu.Lock()
	fixture.runtime.events.fail = false
	fixture.runtime.events.mu.Unlock()

	current := testRequest(
		t, fixture.runtime.handler, http.MethodGet, apiBase+"/stock-adjustments/"+adjustmentID,
		nil, "", true, http.StatusOK,
	)
	assertPendingMovement(t, current)
	if entries := movementLedgerEntries(t, fixture.runtime.handler, "stock_adjustment", adjustmentID); len(entries) != 0 {
		t.Fatalf("event failure retained adjustment ledger: %v", entries)
	}
	assertMovementBalance(t, fixture.runtime.handler, fixture.productID, fixture.lotID, fixture.source.LocationID, "3")
	if got := testEventPublicationCount(fixture.runtime.events); got != initialEventCount {
		t.Fatalf("event failure changed publication count from %d to %d", initialEventCount, got)
	}
	if got := testAuditEntryCount(fixture.runtime.audit); got != initialAuditCount {
		t.Fatalf("event failure changed audit count from %d to %d", initialAuditCount, got)
	}

	retried := testRequest(
		t, fixture.runtime.handler, http.MethodPost,
		apiBase+"/stock-adjustments/"+adjustmentID+"/approve",
		decision, "movement-event-failure-approve", true, http.StatusOK,
	)
	if testString(t, testMap(t, retried, "item"), "status") != "posted" {
		t.Fatalf("rolled-back decision could not be retried: %v", retried)
	}
	assertMovementBalance(t, fixture.runtime.handler, fixture.productID, fixture.lotID, fixture.source.LocationID, "4")
	if got := len(movementLedgerEntries(t, fixture.runtime.handler, "stock_adjustment", adjustmentID)); got != 1 {
		t.Fatalf("retried adjustment ledger count=%d want=1", got)
	}
	assertMovementReconciled(t, fixture.runtime.handler, fixture.productID)
}

func newMovementStockFixture(t *testing.T, quantity, batch string) movementStockFixture {
	t.Helper()
	runtime := newTestRuntime(t)
	order := createApprovedPurchaseOrder(t, runtime, quantity)
	source := createInboundTopology(t, runtime)
	orderLine := order["lines"].([]any)[0].(map[string]any)
	received := testRequest(
		t, runtime.handler, http.MethodPost, apiBase+"/purchase-inbounds",
		inventoryInboundBody(order, orderLine, source, quantity, batch, 1),
		"movement-seed-receipt", true, http.StatusCreated,
	)
	inbound := testMap(t, received, "item")
	inboundLine := inbound["lines"].([]any)[0].(map[string]any)
	productID := testString(t, inboundLine, "productId")
	lots := testRequest(
		t, runtime.handler, http.MethodGet,
		apiBase+"/inventory-lots?productId="+productID+"&batchNo="+batch,
		nil, "", true, http.StatusOK,
	)
	if testInt64(t, lots, "total") != 1 {
		t.Fatalf("movement fixture lot=%v", lots)
	}
	lot := lots["items"].([]any)[0].(map[string]any)
	receiptLedger := movementLedgerEntries(
		t, runtime.handler, "purchase_inbound", testString(t, inbound, "id"),
	)
	if len(receiptLedger) != 1 {
		t.Fatalf("movement fixture receipt ledger=%v", receiptLedger)
	}
	return movementStockFixture{
		runtime: runtime, source: source, productID: productID,
		lotID:                testString(t, lot, "id"),
		receiptLedgerEntryID: testString(t, receiptLedger[0], "id"),
	}
}

func createMovementTopology(t *testing.T, runtime testRuntime, suffix string) inboundTopology {
	t.Helper()
	warehouse := testMap(t, testRequest(
		t, runtime.handler, http.MethodPost, apiBase+"/warehouses",
		map[string]any{
			"tenantId": "tenant-a", "organizationId": "org-a",
			"code": "MOVE-WH-" + suffix, "name": "Movement Warehouse " + suffix,
			"address": "Movement Road " + suffix, "contactName": "Movement Owner",
			"contactPhone": "13800000000",
		},
		"movement-topology-warehouse-"+suffix, true, http.StatusCreated,
	), "item")
	warehouseID := testString(t, warehouse, "id")
	area := testMap(t, testRequest(
		t, runtime.handler, http.MethodPost, apiBase+"/warehouse-areas",
		map[string]any{
			"tenantId": "tenant-a", "organizationId": "org-a", "warehouseId": warehouseID,
			"code": "MOVE-AREA-" + suffix, "name": "Movement Area " + suffix,
			"temperatureMin": "2", "temperatureMax": "25",
		},
		"movement-topology-area-"+suffix, true, http.StatusCreated,
	), "item")
	areaID := testString(t, area, "id")
	location := testMap(t, testRequest(
		t, runtime.handler, http.MethodPost, apiBase+"/warehouse-locations",
		map[string]any{
			"tenantId": "tenant-a", "organizationId": "org-a",
			"warehouseId": warehouseID, "areaId": areaID,
			"code": "MOVE-LOC-" + suffix, "name": "Movement Location " + suffix,
			"locationType": "standard",
		},
		"movement-topology-location-"+suffix, true, http.StatusCreated,
	), "item")
	return inboundTopology{
		WarehouseID: warehouseID, AreaID: areaID, LocationID: testString(t, location, "id"),
	}
}

func movementCreateBase(fixture movementStockFixture, reason string) map[string]any {
	return map[string]any{
		"tenantId": "tenant-a", "organizationId": "org-a",
		"warehouseId": fixture.source.WarehouseID, "areaId": fixture.source.AreaID,
		"locationId": fixture.source.LocationID, "reason": reason,
	}
}

func pendingMovementCreateBase(
	fixture movementStockFixture,
	approverID string,
	approverName string,
	reason string,
) map[string]any {
	body := movementCreateBase(fixture, reason)
	body["approverId"] = approverID
	body["approverName"] = approverName
	return body
}

func transferCreateBody(
	fixture movementStockFixture,
	destination inboundTopology,
	quantity string,
) map[string]any {
	body := movementCreateBase(fixture, "authorized stock relocation")
	body["destinationWarehouseId"] = destination.WarehouseID
	body["destinationAreaId"] = destination.AreaID
	body["destinationLocationId"] = destination.LocationID
	body["lines"] = []map[string]any{{
		"productId": fixture.productID, "lotId": fixture.lotID, "quantity": quantity,
	}}
	return body
}

func adjustmentCreateBody(
	fixture movementStockFixture,
	quantity string,
	reason string,
) map[string]any {
	body := pendingMovementCreateBase(fixture, "manager-1", "Inventory Manager", reason)
	body["lines"] = []map[string]any{{
		"productId": fixture.productID, "lotId": fixture.lotID, "quantity": quantity,
	}}
	return body
}

func returnCreateBody(
	fixture movementStockFixture,
	quantity string,
	reason string,
) map[string]any {
	body := pendingMovementCreateBase(fixture, "manager-1", "Inventory Manager", reason)
	body["returnType"] = "purchase"
	body["lines"] = []map[string]any{{
		"sourceLedgerEntryId": fixture.receiptLedgerEntryID, "quantity": quantity,
	}}
	return body
}

func movementDecisionBody(t *testing.T, created map[string]any, comment string) map[string]any {
	t.Helper()
	item := testMap(t, created, "item")
	return map[string]any{
		"taskId":  testPendingWorkflowTaskID(t, testMap(t, created, "workflow")),
		"version": testInt64(t, item, "version"),
		"comment": comment,
	}
}

func assertPendingMovement(t *testing.T, response map[string]any) {
	t.Helper()
	item := testMap(t, response, "item")
	if testString(t, item, "status") != "pending_approval" {
		t.Fatalf("movement is not pending approval: %v", response)
	}
	workflow := testMap(t, response, "workflow")
	if testString(t, workflow, "status") != string(pluginsdk.WorkflowInstanceRunning) {
		t.Fatalf("pending movement workflow is not running: %v", workflow)
	}
	_ = testPendingWorkflowTaskID(t, workflow)
}

func movementLedgerEntries(
	t *testing.T,
	handler http.Handler,
	sourceDocumentType string,
	sourceDocumentID string,
) []map[string]any {
	t.Helper()
	page := testRequest(
		t, handler, http.MethodGet,
		apiBase+"/stock-ledger?sourceDocumentType="+sourceDocumentType+"&sourceDocumentId="+sourceDocumentID,
		nil, "", true, http.StatusOK,
	)
	raw, ok := page["items"].([]any)
	if !ok {
		t.Fatalf("stock ledger items=%T want array: %v", page["items"], page)
	}
	items := make([]map[string]any, 0, len(raw))
	for _, value := range raw {
		item, ok := value.(map[string]any)
		if !ok {
			t.Fatalf("stock ledger item=%T want object", value)
		}
		items = append(items, item)
	}
	return items
}

func assertMovementEntryQuantities(
	t *testing.T,
	entries []map[string]any,
	want map[string]string,
) {
	t.Helper()
	if len(entries) != len(want) {
		t.Fatalf("movement ledger entries=%v want entry types=%v", entries, want)
	}
	got := make(map[string]string, len(entries))
	for _, entry := range entries {
		entryType := testString(t, entry, "entryType")
		if _, duplicate := got[entryType]; duplicate {
			t.Fatalf("movement ledger repeated entry type %q: %v", entryType, entries)
		}
		got[entryType] = testString(t, entry, "quantity")
		if testString(t, entry, "sourceDocumentLineId") == "" {
			t.Fatalf("movement ledger lost source line trace: %v", entry)
		}
	}
	for entryType, quantity := range want {
		if got[entryType] != quantity {
			t.Fatalf("movement ledger quantities=%v want=%v", got, want)
		}
	}
}

func assertMovementBalance(
	t *testing.T,
	handler http.Handler,
	productID string,
	lotID string,
	locationID string,
	want string,
) {
	t.Helper()
	page := testRequest(
		t, handler, http.MethodGet,
		apiBase+"/stock-balances?productId="+productID+"&lotId="+lotID+"&locationId="+locationID,
		nil, "", true, http.StatusOK,
	)
	total := testInt64(t, page, "total")
	if want == "0" && total == 0 {
		return
	}
	if total != 1 {
		t.Fatalf("balance projection total=%d want=1 page=%v", total, page)
	}
	item := page["items"].([]any)[0].(map[string]any)
	if got := testString(t, item, "quantity"); got != want {
		t.Fatalf("balance quantity=%q want=%q item=%v", got, want, item)
	}
}

func movementBalanceMicros(
	t *testing.T,
	handler http.Handler,
	productID string,
	lotID string,
	locationID string,
) int64 {
	t.Helper()
	page := testRequest(
		t, handler, http.MethodGet,
		apiBase+"/stock-balances?productId="+productID+"&lotId="+lotID+"&locationId="+locationID,
		nil, "", true, http.StatusOK,
	)
	if testInt64(t, page, "total") == 0 {
		return 0
	}
	item := page["items"].([]any)[0].(map[string]any)
	value, err := purchaseQuantityMicros(testString(t, item, "quantity"))
	if err != nil {
		t.Fatalf("decode movement balance: %v", err)
	}
	return value
}

func assertMovementSourceReconciled(
	t *testing.T,
	handler http.Handler,
	sourceDocumentType string,
	sourceDocumentID string,
	wantTotal int64,
) {
	t.Helper()
	page := testRequest(
		t, handler, http.MethodGet,
		apiBase+"/stock-reconciliation?sourceDocumentType="+sourceDocumentType+"&sourceDocumentId="+sourceDocumentID,
		nil, "", true, http.StatusOK,
	)
	if page["matched"] != true || testInt64(t, page, "total") != wantTotal {
		t.Fatalf("movement source reconciliation=%v want matched total=%d", page, wantTotal)
	}
}

func assertMovementReconciled(t *testing.T, handler http.Handler, productID string) {
	t.Helper()
	page := testRequest(
		t, handler, http.MethodGet,
		apiBase+"/stock-reconciliation?productId="+productID,
		nil, "", true, http.StatusOK,
	)
	if page["matched"] != true {
		t.Fatalf("movement ledger and balance diverged: %v", page)
	}
}

func assertMovementDocumentTotal(t *testing.T, handler http.Handler, route string, want int64) {
	t.Helper()
	page := testRequest(t, handler, http.MethodGet, apiBase+route, nil, "", true, http.StatusOK)
	if got := testInt64(t, page, "total"); got != want {
		t.Fatalf("%s total=%d want=%d page=%v", route, got, want, page)
	}
}

func testEventPublicationCount(events *testEvents) int {
	events.mu.Lock()
	defer events.mu.Unlock()
	return len(events.publications)
}

func testAuditEntryCount(audit *testAudit) int {
	audit.mu.Lock()
	defer audit.mu.Unlock()
	return len(audit.entries)
}
