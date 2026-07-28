package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

const (
	inventoryLotTable = "inventory_lots"
	stockLedgerTable  = "stock_ledger"
	stockBalanceTable = "stock_balances"
)

var (
	inventoryLotFields = []string{
		"id", "product_id", "batch_no", "production_date", "expires_at",
		"tenant_id", "organization_id", "owner_id", "created_at", "updated_at",
	}
	stockLedgerFields = []string{
		"id", "entry_type", "product_id", "lot_id", "batch_no", "warehouse_id", "area_id", "location_id",
		"quantity_micros", "source_document_type", "source_document_id", "source_document_number",
		"source_document_line_id", "occurred_at",
		"tenant_id", "organization_id", "owner_id", "created_at", "updated_at",
	}
	stockBalanceFields = []string{
		"id", "product_id", "lot_id", "batch_no", "warehouse_id", "area_id", "location_id", "quantity_micros",
		"tenant_id", "organization_id", "owner_id", "created_at", "updated_at",
	}
)

type inventoryLot struct {
	ID             string `json:"id"`
	ProductID      string `json:"productId"`
	BatchNo        string `json:"batchNo"`
	ProductionDate string `json:"productionDate"`
	ExpiresAt      string `json:"expiresAt"`
	Version        int64  `json:"version"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
	scope          employeeScope
}

type stockLedgerEntry struct {
	ID                   string `json:"id"`
	EntryType            string `json:"entryType"`
	ProductID            string `json:"productId"`
	LotID                string `json:"lotId"`
	BatchNo              string `json:"batchNo"`
	WarehouseID          string `json:"warehouseId"`
	AreaID               string `json:"areaId"`
	LocationID           string `json:"locationId"`
	Quantity             string `json:"quantity"`
	SourceDocumentType   string `json:"sourceDocumentType"`
	SourceDocumentID     string `json:"sourceDocumentId"`
	SourceDocumentNumber string `json:"sourceDocumentNumber"`
	SourceDocumentLineID string `json:"sourceDocumentLineId"`
	OccurredAt           string `json:"occurredAt"`
	Version              int64  `json:"version"`
	CreatedAt            string `json:"createdAt"`
	UpdatedAt            string `json:"updatedAt"`
	quantityMicros       int64
	scope                employeeScope
}

type stockBalance struct {
	ID             string `json:"id"`
	ProductID      string `json:"productId"`
	LotID          string `json:"lotId"`
	BatchNo        string `json:"batchNo"`
	WarehouseID    string `json:"warehouseId"`
	AreaID         string `json:"areaId"`
	LocationID     string `json:"locationId"`
	Quantity       string `json:"quantity"`
	Version        int64  `json:"version"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
	quantityMicros int64
	scope          employeeScope
}

type inventoryPostingSummary struct {
	LotCount         int
	LedgerEntryCount int
	QuantityMicros   int64
}

type stockReconciliationItem struct {
	ProductID       string `json:"productId"`
	LotID           string `json:"lotId"`
	BatchNo         string `json:"batchNo"`
	LocationID      string `json:"locationId"`
	LedgerQuantity  string `json:"ledgerQuantity"`
	BalanceQuantity string `json:"balanceQuantity"`
	Matched         bool   `json:"matched"`
	ledgerMicros    int64
	balanceMicros   int64
}

func (s *server) listInventoryLots(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	filter, err := inventoryQueryFilter(r, map[string]string{
		"productId": "product_id", "batchNo": "batch_no",
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	items := make([]inventoryLot, 0)
	err = s.queryInventoryRecords(ctx, inventoryLotTable, inventoryLotFields, filter, func(record pluginsdk.DataRecord) error {
		item, parseErr := inventoryLotFromRecord(record)
		if parseErr == nil {
			items = append(items, item)
		}
		return parseErr
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, map[string]any{"items": items, "total": len(items)})
}

func (s *server) listStockLedger(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	filter, err := inventoryQueryFilter(r, map[string]string{
		"productId": "product_id", "lotId": "lot_id", "batchNo": "batch_no",
		"warehouseId": "warehouse_id", "locationId": "location_id",
		"sourceDocumentType": "source_document_type", "sourceDocumentId": "source_document_id",
		"sourceDocumentNumber": "source_document_number", "sourceDocumentLineId": "source_document_line_id",
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	items := make([]stockLedgerEntry, 0)
	err = s.queryInventoryRecords(ctx, stockLedgerTable, stockLedgerFields, filter, func(record pluginsdk.DataRecord) error {
		item, parseErr := stockLedgerFromRecord(record)
		if parseErr == nil {
			items = append(items, item)
		}
		return parseErr
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, map[string]any{"items": items, "total": len(items)})
}

func (s *server) listStockBalances(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	filter, err := inventoryQueryFilter(r, map[string]string{
		"productId": "product_id", "lotId": "lot_id", "batchNo": "batch_no",
		"warehouseId": "warehouse_id", "locationId": "location_id",
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	items := make([]stockBalance, 0)
	err = s.queryInventoryRecords(ctx, stockBalanceTable, stockBalanceFields, filter, func(record pluginsdk.DataRecord) error {
		item, parseErr := stockBalanceFromRecord(record)
		if parseErr == nil {
			items = append(items, item)
		}
		return parseErr
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, map[string]any{"items": items, "total": len(items)})
}

func (s *server) reconcileStock(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	projectionFilter, err := inventoryQueryFilter(r, map[string]string{
		"productId": "product_id", "lotId": "lot_id", "batchNo": "batch_no",
		"warehouseId": "warehouse_id", "locationId": "location_id",
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	sourceFilter, err := inventoryQueryFilter(r, map[string]string{
		"sourceDocumentType": "source_document_type", "sourceDocumentId": "source_document_id",
		"sourceDocumentNumber": "source_document_number", "sourceDocumentLineId": "source_document_line_id",
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	var selected map[string]stockQuantityAggregate
	if sourceFilter != nil {
		selected, err = s.aggregateStockQuantity(
			ctx, stockLedgerTable, combineInventoryFilters(projectionFilter, sourceFilter),
		)
		if err != nil {
			writeServiceError(w, err)
			return
		}
	}
	ledger, err := s.aggregateStockQuantity(ctx, stockLedgerTable, projectionFilter)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	balances, err := s.aggregateStockQuantity(ctx, stockBalanceTable, projectionFilter)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if sourceFilter != nil {
		for key := range ledger {
			if _, ok := selected[key]; !ok {
				delete(ledger, key)
			}
		}
		for key := range balances {
			if _, ok := selected[key]; !ok {
				delete(balances, key)
			}
		}
	}
	items := make(map[string]*stockReconciliationItem, len(ledger)+len(balances))
	for key, aggregate := range ledger {
		items[key] = &stockReconciliationItem{
			ProductID: aggregate.ProductID, LotID: aggregate.LotID, BatchNo: aggregate.BatchNo,
			LocationID: aggregate.LocationID, ledgerMicros: aggregate.QuantityMicros,
		}
	}
	for key, aggregate := range balances {
		item := items[key]
		if item == nil {
			item = &stockReconciliationItem{
				ProductID: aggregate.ProductID, LotID: aggregate.LotID, BatchNo: aggregate.BatchNo,
				LocationID: aggregate.LocationID,
			}
			items[key] = item
		}
		item.balanceMicros = aggregate.QuantityMicros
	}
	result := make([]stockReconciliationItem, 0, len(items))
	matched := true
	for _, item := range items {
		item.LedgerQuantity = microsQuantity(item.ledgerMicros)
		item.BalanceQuantity = microsQuantity(item.balanceMicros)
		item.Matched = item.ledgerMicros == item.balanceMicros
		matched = matched && item.Matched
		result = append(result, *item)
	}
	sort.Slice(result, func(left, right int) bool {
		leftKey := stockProjectionKey(result[left].ProductID, result[left].LotID, result[left].LocationID)
		rightKey := stockProjectionKey(result[right].ProductID, result[right].LotID, result[right].LocationID)
		return leftKey < rightKey
	})
	writeOK(w, map[string]any{"items": result, "total": len(result), "matched": matched})
}

func combineInventoryFilters(filters ...*pluginsdk.DataFilter) *pluginsdk.DataFilter {
	combined := make([]pluginsdk.DataFilter, 0, len(filters))
	for _, filter := range filters {
		if filter != nil {
			combined = append(combined, *filter)
		}
	}
	if len(combined) == 0 {
		return nil
	}
	if len(combined) == 1 {
		return &combined[0]
	}
	return &pluginsdk.DataFilter{All: combined}
}

type stockQuantityAggregate struct {
	ProductID      string
	LotID          string
	BatchNo        string
	LocationID     string
	QuantityMicros int64
}

func (s *server) aggregateStockQuantity(
	ctx context.Context,
	table string,
	filter *pluginsdk.DataFilter,
) (map[string]stockQuantityAggregate, error) {
	const aggregatePageSize = 100
	groupBy := []string{"product_id", "lot_id", "batch_no", "location_id"}
	metrics := []pluginsdk.DataAggregateMetric{{Operation: pluginsdk.DataAggregateSum, Field: "quantity_micros"}}
	items := make(map[string]stockQuantityAggregate)
	cursor := ""
	seenCursors := make(map[string]struct{})
	for {
		page, err := s.host.DataStore.Aggregate(ctx, pluginsdk.DataAggregateQuery{
			Table: table, Scope: pluginsdk.DataScopeIntent{Permission: inventoryPermission("read")},
			Filter: filter, Metrics: metrics, GroupBy: groupBy,
			Page: pluginsdk.DataPageRequest{Cursor: cursor, Limit: aggregatePageSize},
		})
		if err != nil {
			return nil, err
		}
		for rowIndex, row := range page.Rows {
			if len(row.Values) != 1 {
				return nil, fmt.Errorf("%s aggregate row %d returned an invalid metric count", table, rowIndex)
			}
			quantityMicros, parseErr := strconv.ParseInt(row.Values[0].Value, 10, 64)
			if parseErr != nil {
				return nil, fmt.Errorf("decode %s aggregate quantity: %w", table, parseErr)
			}
			item := stockQuantityAggregate{
				ProductID: dataAggregateGroupValue(row, "product_id"),
				LotID:     dataAggregateGroupValue(row, "lot_id"),
				BatchNo:   dataAggregateGroupValue(row, "batch_no"),
				LocationID: dataAggregateGroupValue(row,
					"location_id"),
				QuantityMicros: quantityMicros,
			}
			if item.ProductID == "" || item.LotID == "" || item.BatchNo == "" || item.LocationID == "" {
				return nil, fmt.Errorf("%s aggregate row %d returned an incomplete projection identity", table, rowIndex)
			}
			key := stockProjectionKey(item.ProductID, item.LotID, item.LocationID)
			if existing, found := items[key]; found &&
				(existing.BatchNo != item.BatchNo || existing.QuantityMicros != item.QuantityMicros) {
				return nil, fmt.Errorf("%s aggregate returned conflicting projection groups", table)
			}
			items[key] = item
		}
		if !page.HasMore {
			return items, nil
		}
		if page.NextCursor == "" || page.NextCursor == cursor {
			return nil, fmt.Errorf("%s aggregate pagination did not advance", table)
		}
		if _, duplicate := seenCursors[page.NextCursor]; duplicate {
			return nil, fmt.Errorf("%s aggregate pagination repeated a cursor", table)
		}
		seenCursors[page.NextCursor] = struct{}{}
		cursor = page.NextCursor
	}
}

func dataAggregateGroupValue(row pluginsdk.DataAggregateRow, field string) string {
	value, ok := row.Group[field]
	if !ok {
		return ""
	}
	return value.Value
}

func (s *server) postInboundInventory(ctx context.Context, item purchaseInbound) (inventoryPostingSummary, error) {
	summary := inventoryPostingSummary{}
	lots := make(map[string]inventoryLot)
	type balanceProjection struct {
		Balance        stockBalance
		QuantityMicros int64
	}
	projections := make(map[string]balanceProjection)
	for _, line := range item.Lines {
		quantityMicros, err := purchaseQuantityMicros(line.Quantity)
		if err != nil {
			return inventoryPostingSummary{}, err
		}
		lotID := stableInventoryID("lot", item.scope.TenantID, item.scope.OrganizationID, line.ProductID, line.BatchNo)
		lot, exists := lots[lotID]
		if !exists {
			lot, err = s.ensureInventoryLot(ctx, item.scope, inventoryLot{
				ID: lotID, ProductID: line.ProductID, BatchNo: line.BatchNo,
				ProductionDate: line.ProductionDate, ExpiresAt: line.ExpiresAt, scope: item.scope,
			})
			if err != nil {
				return inventoryPostingSummary{}, err
			}
			lots[lotID] = lot
		} else if lot.ProductionDate != line.ProductionDate || lot.ExpiresAt != line.ExpiresAt {
			return inventoryPostingSummary{}, newHTTPError(http.StatusConflict, "inventory_lot_fact_conflict", "batch production or expiry facts conflict with the immutable lot")
		}

		entry := stockLedgerEntry{
			ID:        stableInventoryID("ledger", item.scope.TenantID, item.scope.OrganizationID, item.ID, line.ID),
			EntryType: "receipt", ProductID: line.ProductID, LotID: lot.ID, BatchNo: lot.BatchNo,
			WarehouseID: item.WarehouseID, AreaID: item.AreaID, LocationID: item.LocationID,
			SourceDocumentType: "purchase_inbound", SourceDocumentID: item.ID, SourceDocumentNumber: item.Number,
			SourceDocumentLineID: line.ID, OccurredAt: item.ReceivedAt, quantityMicros: quantityMicros, scope: item.scope,
		}
		if _, err = s.appendStockLedger(ctx, entry); err != nil {
			return inventoryPostingSummary{}, err
		}
		balance := stockBalance{
			ID:        stableInventoryID("balance", item.scope.TenantID, item.scope.OrganizationID, line.ProductID, lot.ID, item.LocationID),
			ProductID: line.ProductID, LotID: lot.ID, BatchNo: lot.BatchNo,
			WarehouseID: item.WarehouseID, AreaID: item.AreaID, LocationID: item.LocationID, scope: item.scope,
		}
		projection := projections[balance.ID]
		projection.Balance = balance
		projection.QuantityMicros, err = checkedAddMicros(projection.QuantityMicros, quantityMicros)
		if err != nil {
			return inventoryPostingSummary{}, err
		}
		projections[balance.ID] = projection
		summary.LedgerEntryCount++
		summary.QuantityMicros, err = checkedAddMicros(summary.QuantityMicros, quantityMicros)
		if err != nil {
			return inventoryPostingSummary{}, err
		}
	}
	balanceIDs := make([]string, 0, len(projections))
	for balanceID := range projections {
		balanceIDs = append(balanceIDs, balanceID)
	}
	sort.Strings(balanceIDs)
	for _, balanceID := range balanceIDs {
		projection := projections[balanceID]
		if _, err := s.projectStockBalance(
			ctx, projection.Balance, projection.QuantityMicros,
			stableInventoryID("projection", item.ID, balanceID),
		); err != nil {
			return inventoryPostingSummary{}, err
		}
	}
	summary.LotCount = len(lots)
	return summary, nil
}

func (s *server) ensureInventoryLot(ctx context.Context, scope employeeScope, candidate inventoryLot) (inventoryLot, error) {
	current, found, err := s.findInventoryLot(ctx, scope, candidate.ID)
	if err != nil {
		return inventoryLot{}, err
	}
	if found {
		if current.ProductID != candidate.ProductID || current.BatchNo != candidate.BatchNo ||
			current.ProductionDate != candidate.ProductionDate || current.ExpiresAt != candidate.ExpiresAt {
			return inventoryLot{}, newHTTPError(http.StatusConflict, "inventory_lot_fact_conflict", "batch production or expiry facts conflict with the immutable lot")
		}
		return current, nil
	}
	result, err := s.host.DataStore.Mutate(ctx, pluginsdk.DataMutation{
		Table: inventoryLotTable, Operation: pluginsdk.DataMutationInsert,
		Scope: inventoryIntent("receive", scope), Key: map[string]pluginsdk.DataValue{"id": stringValue(candidate.ID)},
		Values: map[string]pluginsdk.DataValue{
			"product_id": stringValue(candidate.ProductID), "batch_no": stringValue(candidate.BatchNo),
			"production_date": timestampValue(candidate.ProductionDate), "expires_at": timestampValue(candidate.ExpiresAt),
		},
		Returning: inventoryLotFields, IdempotencyKey: candidate.ID + ".append",
	})
	if err != nil {
		return inventoryLot{}, err
	}
	if result.Record == nil {
		return inventoryLot{}, fmt.Errorf("inventory lot mutation returned no record")
	}
	created, err := inventoryLotFromRecord(*result.Record)
	if err != nil {
		return inventoryLot{}, err
	}
	if created.ProductID != candidate.ProductID || created.BatchNo != candidate.BatchNo ||
		created.ProductionDate != candidate.ProductionDate || created.ExpiresAt != candidate.ExpiresAt {
		return inventoryLot{}, newHTTPError(http.StatusConflict, "inventory_lot_fact_conflict", "batch production or expiry facts conflict with the immutable lot")
	}
	return created, nil
}

func (s *server) findInventoryLot(ctx context.Context, scope employeeScope, id string) (inventoryLot, bool, error) {
	value := stringValue(id)
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{
		Table: inventoryLotTable, Fields: inventoryLotFields, Scope: inventoryIntent("receive", scope),
		Filter: &pluginsdk.DataFilter{Field: "id", Operator: pluginsdk.DataOperatorEqual, Value: &value},
		Sort:   []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 1},
	})
	if err != nil || len(page.Records) == 0 {
		return inventoryLot{}, false, err
	}
	item, err := inventoryLotFromRecord(page.Records[0])
	return item, err == nil, err
}

func (s *server) appendStockLedger(ctx context.Context, entry stockLedgerEntry) (stockLedgerEntry, error) {
	result, err := s.host.DataStore.Mutate(ctx, pluginsdk.DataMutation{
		Table: stockLedgerTable, Operation: pluginsdk.DataMutationInsert,
		Scope: inventoryIntent("receive", entry.scope), Key: map[string]pluginsdk.DataValue{"id": stringValue(entry.ID)},
		Values: stockLedgerValues(entry), Returning: stockLedgerFields, IdempotencyKey: entry.ID + ".append",
	})
	if err != nil {
		return stockLedgerEntry{}, err
	}
	if result.Record == nil {
		return stockLedgerEntry{}, fmt.Errorf("stock ledger mutation returned no record")
	}
	return stockLedgerFromRecord(*result.Record)
}

func (s *server) projectStockBalance(
	ctx context.Context,
	balance stockBalance,
	quantityMicros int64,
	idempotencyKey string,
) (stockBalance, error) {
	current, found, err := s.findStockBalance(ctx, balance.scope, balance.ID)
	if err != nil {
		return stockBalance{}, err
	}
	if !found {
		result, insertErr := s.host.DataStore.Mutate(ctx, pluginsdk.DataMutation{
			Table: stockBalanceTable, Operation: pluginsdk.DataMutationInsert,
			Scope: inventoryIntent("receive", balance.scope), Key: map[string]pluginsdk.DataValue{"id": stringValue(balance.ID)},
			Values: map[string]pluginsdk.DataValue{
				"product_id": stringValue(balance.ProductID), "lot_id": stringValue(balance.LotID), "batch_no": stringValue(balance.BatchNo),
				"warehouse_id": stringValue(balance.WarehouseID), "area_id": stringValue(balance.AreaID), "location_id": stringValue(balance.LocationID),
				"quantity_micros": integerValue(0),
			},
			Returning: stockBalanceFields, IdempotencyKey: balance.ID + ".open",
		})
		if insertErr != nil {
			return stockBalance{}, insertErr
		}
		if result.Record == nil {
			return stockBalance{}, fmt.Errorf("stock balance creation returned no record")
		}
		current, err = stockBalanceFromRecord(*result.Record)
		if err != nil {
			return stockBalance{}, err
		}
	}
	if current.ProductID != balance.ProductID || current.LotID != balance.LotID || current.BatchNo != balance.BatchNo ||
		current.WarehouseID != balance.WarehouseID || current.AreaID != balance.AreaID || current.LocationID != balance.LocationID {
		return stockBalance{}, newHTTPError(http.StatusConflict, "stock_balance_identity_conflict", "stock balance identity does not match its projection key")
	}
	minimum := integerValue(0)
	maximum := integerValue(math.MaxInt64)
	result, err := s.host.DataStore.Mutate(ctx, pluginsdk.DataMutation{
		Table: stockBalanceTable, Operation: pluginsdk.DataMutationAdjust,
		Scope: inventoryIntent("receive", balance.scope), Key: map[string]pluginsdk.DataValue{"id": stringValue(balance.ID)},
		Adjustment: &pluginsdk.DataAdjustment{
			Field: "quantity_micros", Delta: integerValue(quantityMicros), Minimum: &minimum, Maximum: &maximum,
		},
		Returning: stockBalanceFields, IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		return stockBalance{}, err
	}
	if result.Record == nil {
		return stockBalance{}, fmt.Errorf("stock balance adjustment returned no record")
	}
	return stockBalanceFromRecord(*result.Record)
}

func (s *server) findStockBalance(ctx context.Context, scope employeeScope, id string) (stockBalance, bool, error) {
	value := stringValue(id)
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{
		Table: stockBalanceTable, Fields: stockBalanceFields, Scope: inventoryIntent("receive", scope),
		Filter: &pluginsdk.DataFilter{Field: "id", Operator: pluginsdk.DataOperatorEqual, Value: &value},
		Sort:   []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 1},
	})
	if err != nil || len(page.Records) == 0 {
		return stockBalance{}, false, err
	}
	item, err := stockBalanceFromRecord(page.Records[0])
	return item, err == nil, err
}

func (s *server) publishInventoryChanged(ctx context.Context, item purchaseInbound, summary inventoryPostingSummary) error {
	_, err := s.host.Events.Publish(ctx, pluginsdk.EventPublication{
		IdempotencyKey: item.ID + ".inventory", Name: "inventory-changed", SchemaVersion: 1,
		Scope: pluginsdk.EventScope{
			TenantID: item.scope.TenantID, OrganizationID: item.scope.OrganizationID, OwnerID: item.scope.OwnerID,
		},
		CorrelationID: item.ID,
		Subject:       pluginsdk.EventSubject{Type: "purchase_inbound", ID: item.ID},
		Payload: pluginsdk.EventPayload{
			"source_document_type": {Type: pluginsdk.DataValueString, Value: "purchase_inbound"},
			"source_document_id":   {Type: pluginsdk.DataValueString, Value: item.ID},
			"warehouse_id":         {Type: pluginsdk.DataValueString, Value: item.WarehouseID},
			"location_id":          {Type: pluginsdk.DataValueString, Value: item.LocationID},
			"ledger_entry_count":   integerValue(int64(summary.LedgerEntryCount)),
			"quantity_micros":      integerValue(summary.QuantityMicros),
		},
	})
	return err
}

func (s *server) queryInventoryRecords(
	ctx context.Context,
	table string,
	fields []string,
	filter *pluginsdk.DataFilter,
	appendRecord func(pluginsdk.DataRecord) error,
) error {
	cursor := ""
	for pageNumber := 0; pageNumber < 25; pageNumber++ {
		page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{
			Table: table, Fields: fields, Scope: pluginsdk.DataScopeIntent{Permission: inventoryPermission("read")},
			Filter: filter, Sort: []pluginsdk.DataSort{{Field: "created_at", Direction: pluginsdk.DataSortDescending}},
			Page: pluginsdk.DataPageRequest{Cursor: cursor, Limit: 200},
		})
		if err != nil {
			return err
		}
		for _, record := range page.Records {
			if err := appendRecord(record); err != nil {
				return err
			}
		}
		if !page.HasMore {
			return nil
		}
		cursor = page.NextCursor
	}
	return pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorLimitExceeded, table, "inventory query exceeds 5000 records", false)
}

func inventoryQueryFilter(r *http.Request, allowed map[string]string) (*pluginsdk.DataFilter, error) {
	filters := make([]pluginsdk.DataFilter, 0, len(allowed))
	for parameter, field := range allowed {
		value := strings.TrimSpace(r.URL.Query().Get(parameter))
		if value == "" {
			continue
		}
		if len(value) > 128 {
			return nil, newHTTPError(http.StatusBadRequest, "invalid_inventory_filter", parameter+" exceeds its bounded length")
		}
		if parameter == "batchNo" {
			value = strings.ToUpper(value)
		}
		typed := stringValue(value)
		filters = append(filters, pluginsdk.DataFilter{Field: field, Operator: pluginsdk.DataOperatorEqual, Value: &typed})
	}
	if len(filters) == 0 {
		return nil, nil
	}
	sort.Slice(filters, func(left, right int) bool { return filters[left].Field < filters[right].Field })
	if len(filters) == 1 {
		return &filters[0], nil
	}
	return &pluginsdk.DataFilter{All: filters}, nil
}

func inventoryLotFromRecord(record pluginsdk.DataRecord) (inventoryLot, error) {
	item := inventoryLot{
		ID: dataString(record, "id"), ProductID: dataString(record, "product_id"), BatchNo: dataString(record, "batch_no"),
		ProductionDate: dataString(record, "production_date"), ExpiresAt: dataString(record, "expires_at"),
		Version: record.Version, CreatedAt: dataString(record, "created_at"), UpdatedAt: dataString(record, "updated_at"),
		scope: recordScope(record),
	}
	if item.ID == "" || item.ProductID == "" || item.BatchNo == "" || item.ProductionDate == "" || item.ExpiresAt == "" {
		return inventoryLot{}, fmt.Errorf("inventory lot record is incomplete")
	}
	return item, nil
}

func stockLedgerValues(entry stockLedgerEntry) map[string]pluginsdk.DataValue {
	return map[string]pluginsdk.DataValue{
		"entry_type": stringValue(entry.EntryType), "product_id": stringValue(entry.ProductID),
		"lot_id": stringValue(entry.LotID), "batch_no": stringValue(entry.BatchNo),
		"warehouse_id": stringValue(entry.WarehouseID), "area_id": stringValue(entry.AreaID), "location_id": stringValue(entry.LocationID),
		"quantity_micros": integerValue(entry.quantityMicros), "source_document_type": stringValue(entry.SourceDocumentType),
		"source_document_id": stringValue(entry.SourceDocumentID), "source_document_number": stringValue(entry.SourceDocumentNumber),
		"source_document_line_id": stringValue(entry.SourceDocumentLineID), "occurred_at": timestampValue(entry.OccurredAt),
	}
}

func stockLedgerFromRecord(record pluginsdk.DataRecord) (stockLedgerEntry, error) {
	quantityMicros, err := recordMicros(record, "quantity_micros")
	if err != nil {
		return stockLedgerEntry{}, err
	}
	item := stockLedgerEntry{
		ID: dataString(record, "id"), EntryType: dataString(record, "entry_type"), ProductID: dataString(record, "product_id"),
		LotID: dataString(record, "lot_id"), BatchNo: dataString(record, "batch_no"),
		WarehouseID: dataString(record, "warehouse_id"), AreaID: dataString(record, "area_id"), LocationID: dataString(record, "location_id"),
		Quantity: microsQuantity(quantityMicros), SourceDocumentType: dataString(record, "source_document_type"),
		SourceDocumentID: dataString(record, "source_document_id"), SourceDocumentNumber: dataString(record, "source_document_number"),
		SourceDocumentLineID: dataString(record, "source_document_line_id"), OccurredAt: dataString(record, "occurred_at"),
		Version: record.Version, CreatedAt: dataString(record, "created_at"), UpdatedAt: dataString(record, "updated_at"),
		quantityMicros: quantityMicros, scope: recordScope(record),
	}
	if item.ID == "" || item.EntryType == "" || item.ProductID == "" || item.LotID == "" || item.BatchNo == "" ||
		item.WarehouseID == "" || item.AreaID == "" || item.LocationID == "" || item.SourceDocumentID == "" ||
		item.SourceDocumentLineID == "" || item.OccurredAt == "" || quantityMicros == 0 {
		return stockLedgerEntry{}, fmt.Errorf("stock ledger record is incomplete")
	}
	return item, nil
}

func stockBalanceFromRecord(record pluginsdk.DataRecord) (stockBalance, error) {
	quantityMicros, err := recordMicros(record, "quantity_micros")
	if err != nil {
		return stockBalance{}, err
	}
	item := stockBalance{
		ID: dataString(record, "id"), ProductID: dataString(record, "product_id"), LotID: dataString(record, "lot_id"),
		BatchNo: dataString(record, "batch_no"), WarehouseID: dataString(record, "warehouse_id"),
		AreaID: dataString(record, "area_id"), LocationID: dataString(record, "location_id"),
		Quantity: microsQuantity(quantityMicros), Version: record.Version,
		CreatedAt: dataString(record, "created_at"), UpdatedAt: dataString(record, "updated_at"),
		quantityMicros: quantityMicros, scope: recordScope(record),
	}
	if item.ID == "" || item.ProductID == "" || item.LotID == "" || item.BatchNo == "" ||
		item.WarehouseID == "" || item.AreaID == "" || item.LocationID == "" || quantityMicros < 0 {
		return stockBalance{}, fmt.Errorf("stock balance record is incomplete")
	}
	return item, nil
}

func purchaseQuantityMicros(value string) (int64, error) {
	quantity, ok := new(big.Rat).SetString(strings.TrimSpace(value))
	if !ok || quantity.Sign() <= 0 {
		return 0, newHTTPError(http.StatusBadRequest, "invalid_inventory_quantity", "inventory quantity is invalid")
	}
	scaled := new(big.Rat).Mul(quantity, big.NewRat(1_000_000, 1))
	if !scaled.IsInt() || !scaled.Num().IsInt64() {
		return 0, newHTTPError(http.StatusBadRequest, "invalid_inventory_quantity", "inventory quantity exceeds six exact decimal places or integer capacity")
	}
	return scaled.Num().Int64(), nil
}

func recordMicros(record pluginsdk.DataRecord, field string) (int64, error) {
	value, err := strconv.ParseInt(dataString(record, field), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("decode %s: %w", field, err)
	}
	return value, nil
}

func microsQuantity(value int64) string {
	return canonicalRat(big.NewRat(value, 1_000_000), 6)
}

func checkedAddMicros(left, right int64) (int64, error) {
	if (right > 0 && left > math.MaxInt64-right) || (right < 0 && left < math.MinInt64-right) {
		return 0, newHTTPError(http.StatusConflict, "inventory_quantity_overflow", "inventory quantity exceeds integer capacity")
	}
	return left + right, nil
}

func inventoryPermission(action string) pluginsdk.Permission {
	return pluginsdk.Permission{Resource: "pharma_oa.inventory", Action: action}
}

func inventoryIntent(action string, scope employeeScope) pluginsdk.DataScopeIntent {
	return pluginsdk.DataScopeIntent{Permission: inventoryPermission(action), Filter: scopeFilter(scope)}
}

func stockProjectionKey(productID, lotID, locationID string) string {
	return productID + "\x00" + lotID + "\x00" + locationID
}

func stableInventoryID(kind string, parts ...string) string {
	material := pluginID + "\x00inventory\x00" + kind + "\x00" + strings.Join(parts, "\x00")
	digest := sha256.Sum256([]byte(material))
	return strings.ReplaceAll(kind, "_", "-") + "-" + hex.EncodeToString(digest[:12])
}
