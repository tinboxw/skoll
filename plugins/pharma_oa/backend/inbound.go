package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

const purchaseInboundTable = "pharma_oa_purchase_inbounds"

var purchaseInboundFields = []string{
	"id", "number", "purchase_order_id", "purchase_order_number", "warehouse_id", "area_id", "location_id",
	"lines", "attachments", "status", "received_by", "received_at", "last_operation_key",
	"tenant_id", "organization_id", "owner_id", "created_at", "updated_at",
}

type inboundLine struct {
	OrderLineID    string `json:"orderLineId"`
	ProductID      string `json:"productId"`
	ProductCode    string `json:"productCode"`
	ProductName    string `json:"productName"`
	Quantity       string `json:"quantity"`
	BatchNo        string `json:"batchNo"`
	ProductionDate string `json:"productionDate"`
	ExpiresAt      string `json:"expiresAt"`
}

type inboundAttachment struct {
	FileID string `json:"fileId"`
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	MIME   string `json:"mime"`
}

type purchaseInbound struct {
	ID                  string              `json:"id"`
	Number              string              `json:"number"`
	PurchaseOrderID     string              `json:"purchaseOrderId"`
	PurchaseOrderNumber string              `json:"purchaseOrderNumber"`
	WarehouseID         string              `json:"warehouseId"`
	AreaID              string              `json:"areaId"`
	LocationID          string              `json:"locationId"`
	Lines               []inboundLine       `json:"lines"`
	Attachments         []inboundAttachment `json:"attachments"`
	Status              string              `json:"status"`
	ReceivedBy          string              `json:"receivedBy"`
	ReceivedAt          string              `json:"receivedAt"`
	LastOperationKey    string              `json:"-"`
	Version             int64               `json:"version"`
	CreatedAt           string              `json:"createdAt"`
	UpdatedAt           string              `json:"updatedAt"`
	scope               employeeScope
}

type inboundLineInput struct {
	OrderLineID    string `json:"orderLineId"`
	Quantity       string `json:"quantity"`
	BatchNo        string `json:"batchNo"`
	ProductionDate string `json:"productionDate"`
	ExpiresAt      string `json:"expiresAt"`
}

type inboundAttachmentInput struct {
	Name          string `json:"name"`
	ContentBase64 string `json:"contentBase64"`
}

type purchaseInboundCreateInput struct {
	TenantID        string                   `json:"tenantId"`
	OrganizationID  string                   `json:"organizationId"`
	PurchaseOrderID string                   `json:"purchaseOrderId"`
	WarehouseID     string                   `json:"warehouseId"`
	AreaID          string                   `json:"areaId"`
	LocationID      string                   `json:"locationId"`
	OrderVersion    int64                    `json:"orderVersion"`
	Lines           []inboundLineInput       `json:"lines"`
	Attachments     []inboundAttachmentInput `json:"attachments"`
}

func (s *server) listPurchaseInbounds(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	items, err := s.queryPurchaseInbounds(ctx, r.URL.Query().Get("purchaseOrderId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, map[string]any{"items": items, "total": len(items)})
}

func (s *server) getPurchaseInboundHandler(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	item, err := s.getPurchaseInbound(ctx, inboundPermission("read"), r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, map[string]any{"item": item})
}

func (s *server) createPurchaseInbound(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	requestKey, err := mutationKey(r, "create")
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if existing, found, findErr := s.findPurchaseInboundByOperationKey(ctx, inboundPermission("create"), requestKey); findErr != nil {
		writeServiceError(w, findErr)
		return
	} else if found {
		order, orderErr := s.getPurchaseOrder(ctx, inboundPermission("create"), existing.PurchaseOrderID)
		if orderErr != nil {
			writeServiceError(w, orderErr)
			return
		}
		writeOK(w, map[string]any{"item": existing, "order": order, "duplicate": true})
		return
	}
	var input purchaseInboundCreateInput
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := validateInboundCreateInput(&input); err != nil {
		writeServiceError(w, err)
		return
	}
	scope, err := s.exactWriteScope(ctx, inboundPermission("create"), input.TenantID, input.OrganizationID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	order, err := s.getPurchaseOrder(ctx, inboundPermission("create"), input.PurchaseOrderID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if order.scope.TenantID != scope.TenantID || order.scope.OrganizationID != scope.OrganizationID || order.Version != input.OrderVersion {
		writeServiceError(w, newHTTPError(http.StatusConflict, "stale_purchase_order", "purchase order scope or version is stale"))
		return
	}
	if order.Status != "open" && order.Status != "partial" {
		writeServiceError(w, newHTTPError(http.StatusConflict, "purchase_order_not_receivable", "purchase order is not open for receiving"))
		return
	}
	lines, updatedLines, orderStatus, err := applyInboundLines(order.Lines, input.Lines, s.now().UTC())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	inboundID := newEntityID("purchase-inbound")
	attachments, err := s.storeInboundAttachments(ctx, inboundID, requestKey, input.Attachments)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	cleanup := func() {
		for _, attachment := range attachments {
			_ = s.host.Files.Delete(ctx, attachment.FileID)
		}
	}
	now := s.now().UTC()
	item := purchaseInbound{
		ID: inboundID, PurchaseOrderID: order.ID, PurchaseOrderNumber: order.Number,
		WarehouseID: input.WarehouseID, AreaID: input.AreaID, LocationID: input.LocationID,
		Lines: lines, Attachments: attachments, Status: "completed", ReceivedBy: scope.OwnerID,
		ReceivedAt: now.Format(time.RFC3339Nano), LastOperationKey: requestKey, scope: scope,
	}
	order.Lines, order.Status, order.LastOperationKey = updatedLines, orderStatus, requestKey
	var created purchaseInbound
	var updatedOrder purchaseOrder
	err = s.transaction(ctx, func(tx context.Context) error {
		number, issueErr := s.host.DocumentNumbers.Issue(tx, pluginsdk.DocumentNumberInput{
			Rule: purchaseDocumentNumberRule("purchase_inbound", "PI"), TenantID: scope.TenantID,
			Permission: inboundPermission("create"), OccurredAt: now, IdempotencyKey: requestKey,
		})
		if issueErr != nil {
			return issueErr
		}
		item.Number = number.Number
		orderResult, mutationErr := s.host.DataStore.Mutate(tx, pluginsdk.DataMutation{
			Table: purchaseOrderTable, Operation: pluginsdk.DataMutationUpdate, Scope: inboundIntent("create", scope),
			Key: map[string]pluginsdk.DataValue{"id": stringValue(order.ID)}, Values: purchaseOrderValues(order),
			Returning: purchaseOrderFields, IdempotencyKey: requestKey + ".order", ExpectedVersion: &input.OrderVersion,
		})
		if mutationErr != nil {
			return mutationErr
		}
		updatedOrder, mutationErr = purchaseOrderFromMutation(orderResult)
		if mutationErr != nil {
			return mutationErr
		}
		inboundResult, mutationErr := s.host.DataStore.Mutate(tx, pluginsdk.DataMutation{
			Table: purchaseInboundTable, Operation: pluginsdk.DataMutationInsert, Scope: inboundIntent("create", scope),
			Key: map[string]pluginsdk.DataValue{"id": stringValue(item.ID)}, Values: purchaseInboundValues(item),
			Returning: purchaseInboundFields, IdempotencyKey: requestKey,
		})
		if mutationErr != nil {
			return mutationErr
		}
		created, mutationErr = purchaseInboundFromMutation(inboundResult)
		if mutationErr != nil {
			return mutationErr
		}
		return s.audit(tx, "pharma_oa.inbound.create", created.ID, pluginsdk.AuditRiskHigh, map[string]any{
			"number": created.Number, "purchaseOrderId": order.ID, "lineCount": len(created.Lines),
			"attachmentCount": len(created.Attachments), "orderStatus": updatedOrder.Status,
		})
	})
	if err != nil {
		cleanup()
		writeServiceError(w, err)
		return
	}
	writeCreated(w, map[string]any{"item": created, "order": updatedOrder, "duplicate": false})
}

func validateInboundCreateInput(input *purchaseInboundCreateInput) error {
	input.TenantID, input.OrganizationID = strings.TrimSpace(input.TenantID), strings.TrimSpace(input.OrganizationID)
	input.PurchaseOrderID, input.WarehouseID = strings.TrimSpace(input.PurchaseOrderID), strings.TrimSpace(input.WarehouseID)
	input.AreaID, input.LocationID = strings.TrimSpace(input.AreaID), strings.TrimSpace(input.LocationID)
	if input.TenantID == "" || input.OrganizationID == "" || input.PurchaseOrderID == "" || input.WarehouseID == "" ||
		input.AreaID == "" || input.LocationID == "" || input.OrderVersion < 1 || len(input.Lines) == 0 ||
		len(input.Lines) > 200 || len(input.Attachments) > 10 {
		return newHTTPError(http.StatusBadRequest, "invalid_purchase_inbound", "scope, order, location, current version, 1 to 200 lines, and at most 10 attachments are required")
	}
	return nil
}

func applyInboundLines(orderLines []purchaseLine, inputs []inboundLineInput, now time.Time) ([]inboundLine, []purchaseLine, string, error) {
	updated := append([]purchaseLine(nil), orderLines...)
	byID := make(map[string]int, len(updated))
	for index, line := range updated {
		byID[line.ID] = index
	}
	inboundLines := make([]inboundLine, 0, len(inputs))
	seen := make(map[string]struct{}, len(inputs))
	for index, input := range inputs {
		input.OrderLineID, input.BatchNo = strings.TrimSpace(input.OrderLineID), strings.ToUpper(strings.TrimSpace(input.BatchNo))
		key := input.OrderLineID + "\x00" + input.BatchNo
		if _, duplicate := seen[key]; duplicate {
			return nil, nil, "", newHTTPError(http.StatusConflict, "duplicate_inbound_batch", fmt.Sprintf("inbound line %d repeats an order line and batch", index+1))
		}
		seen[key] = struct{}{}
		orderIndex, exists := byID[input.OrderLineID]
		if !exists || input.BatchNo == "" || len(input.BatchNo) > 64 {
			return nil, nil, "", newHTTPError(http.StatusBadRequest, "invalid_inbound_line", fmt.Sprintf("inbound line %d has an invalid order line or batch", index+1))
		}
		quantity, quantityValue, valueErr := normalizePurchaseDecimal(input.Quantity, 6, "quantity")
		productionDate, productionErr := parseDate(input.ProductionDate)
		expiresAt, expiryErr := parseDate(input.ExpiresAt)
		if valueErr != nil || productionErr != nil || expiryErr != nil || !expiresAt.After(productionDate) ||
			!expiresAt.After(now) || productionDate.After(now.Add(24*time.Hour)) {
			return nil, nil, "", newHTTPError(http.StatusBadRequest, "invalid_inbound_lot", fmt.Sprintf("inbound line %d quantity or lot dates are invalid", index+1))
		}
		ordered, _ := new(big.Rat).SetString(updated[orderIndex].Quantity)
		received, _ := new(big.Rat).SetString(updated[orderIndex].ReceivedQuantity)
		next := new(big.Rat).Add(received, quantityValue)
		if next.Cmp(ordered) > 0 {
			return nil, nil, "", newHTTPError(http.StatusConflict, "purchase_over_receipt", fmt.Sprintf("inbound line %d exceeds the remaining order quantity", index+1))
		}
		updated[orderIndex].ReceivedQuantity = canonicalRat(next, 6)
		inboundLines = append(inboundLines, inboundLine{
			OrderLineID: input.OrderLineID, ProductID: updated[orderIndex].ProductID, ProductCode: updated[orderIndex].ProductCode,
			ProductName: updated[orderIndex].ProductName, Quantity: quantity, BatchNo: input.BatchNo,
			ProductionDate: productionDate.Format(time.RFC3339Nano), ExpiresAt: expiresAt.Format(time.RFC3339Nano),
		})
	}
	status := "received"
	for _, line := range updated {
		ordered, _ := new(big.Rat).SetString(line.Quantity)
		received, _ := new(big.Rat).SetString(line.ReceivedQuantity)
		if received.Cmp(ordered) < 0 {
			status = "partial"
			break
		}
	}
	return inboundLines, updated, status, nil
}

func canonicalRat(value *big.Rat, scale int) string {
	return strings.TrimRight(strings.TrimRight(value.FloatString(scale), "0"), ".")
}

func (s *server) storeInboundAttachments(ctx context.Context, inboundID, requestKey string, inputs []inboundAttachmentInput) ([]inboundAttachment, error) {
	items := make([]inboundAttachment, 0, len(inputs))
	for index, input := range inputs {
		name := strings.TrimSpace(filepath.Base(input.Name))
		content, err := base64.StdEncoding.DecodeString(input.ContentBase64)
		if name == "" || name == "." || err != nil || len(content) == 0 || len(content) > 2<<20 {
			for _, item := range items {
				_ = s.host.Files.Delete(ctx, item.FileID)
			}
			return nil, newHTTPError(http.StatusBadRequest, "invalid_inbound_attachment", fmt.Sprintf("inbound attachment %d is invalid", index+1))
		}
		file, err := s.host.Files.Store(ctx, pluginsdk.FileWrite{
			Key: "purchase-inbounds/" + inboundID + "/" + requestKey + "/" + fmt.Sprint(index+1), Name: name, Content: content,
			Visibility: pluginsdk.FileVisibilityPrivate, Metadata: map[string]string{"purchaseInboundId": inboundID, "requestKey": requestKey},
		})
		if err != nil {
			for _, item := range items {
				_ = s.host.Files.Delete(ctx, item.FileID)
			}
			return nil, err
		}
		items = append(items, inboundAttachment{FileID: file.ID, Name: file.Name, Size: file.Size, MIME: file.MIME})
	}
	return items, nil
}

func (s *server) getPurchaseInbound(ctx context.Context, permission pluginsdk.Permission, id string) (purchaseInbound, error) {
	idValue := stringValue(strings.TrimSpace(id))
	filter := pluginsdk.DataFilter{Field: "id", Operator: pluginsdk.DataOperatorEqual, Value: &idValue}
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{
		Table: purchaseInboundTable, Fields: purchaseInboundFields, Scope: pluginsdk.DataScopeIntent{Permission: permission},
		Filter: &filter, Sort: []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 1},
	})
	if err != nil {
		return purchaseInbound{}, err
	}
	if len(page.Records) != 1 {
		return purchaseInbound{}, newHTTPError(http.StatusNotFound, "purchase_inbound_not_found", "purchase inbound was not found")
	}
	return purchaseInboundFromRecord(page.Records[0])
}

func (s *server) findPurchaseInboundByOperationKey(ctx context.Context, permission pluginsdk.Permission, key string) (purchaseInbound, bool, error) {
	value := stringValue(key)
	filter := pluginsdk.DataFilter{Field: "last_operation_key", Operator: pluginsdk.DataOperatorEqual, Value: &value}
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{
		Table: purchaseInboundTable, Fields: purchaseInboundFields, Scope: pluginsdk.DataScopeIntent{Permission: permission},
		Filter: &filter, Sort: []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 1},
	})
	if err != nil || len(page.Records) == 0 {
		return purchaseInbound{}, false, err
	}
	item, err := purchaseInboundFromRecord(page.Records[0])
	return item, err == nil, err
}

func (s *server) queryPurchaseInbounds(ctx context.Context, orderID string) ([]purchaseInbound, error) {
	var filter *pluginsdk.DataFilter
	if orderID = strings.TrimSpace(orderID); orderID != "" {
		value := stringValue(orderID)
		filter = &pluginsdk.DataFilter{Field: "purchase_order_id", Operator: pluginsdk.DataOperatorEqual, Value: &value}
	}
	items := make([]purchaseInbound, 0)
	err := s.queryPurchaseRecords(ctx, purchaseInboundTable, purchaseInboundFields, inboundPermission("read"), filter, func(record pluginsdk.DataRecord) error {
		item, parseErr := purchaseInboundFromRecord(record)
		if parseErr == nil {
			items = append(items, item)
		}
		return parseErr
	})
	sort.SliceStable(items, func(left, right int) bool { return items[left].ReceivedAt > items[right].ReceivedAt })
	return items, err
}

func purchaseInboundValues(item purchaseInbound) map[string]pluginsdk.DataValue {
	return map[string]pluginsdk.DataValue{
		"number": stringValue(item.Number), "purchase_order_id": stringValue(item.PurchaseOrderID),
		"purchase_order_number": stringValue(item.PurchaseOrderNumber), "warehouse_id": stringValue(item.WarehouseID),
		"area_id": stringValue(item.AreaID), "location_id": stringValue(item.LocationID), "lines": jsonValue(item.Lines),
		"attachments": jsonValue(item.Attachments), "status": stringValue(item.Status), "received_by": stringValue(item.ReceivedBy),
		"received_at": timestampValue(item.ReceivedAt), "last_operation_key": stringValue(item.LastOperationKey),
	}
}

func purchaseInboundFromMutation(result pluginsdk.DataMutationResult) (purchaseInbound, error) {
	if result.Record == nil {
		return purchaseInbound{}, errorsNewDataRecord("purchase inbound")
	}
	return purchaseInboundFromRecord(*result.Record)
}

func purchaseInboundFromRecord(record pluginsdk.DataRecord) (purchaseInbound, error) {
	item := purchaseInbound{
		ID: dataString(record, "id"), Number: dataString(record, "number"), PurchaseOrderID: dataString(record, "purchase_order_id"),
		PurchaseOrderNumber: dataString(record, "purchase_order_number"), WarehouseID: dataString(record, "warehouse_id"),
		AreaID: dataString(record, "area_id"), LocationID: dataString(record, "location_id"), Status: dataString(record, "status"),
		ReceivedBy: dataString(record, "received_by"), ReceivedAt: dataString(record, "received_at"),
		LastOperationKey: dataString(record, "last_operation_key"), Version: record.Version,
		CreatedAt: dataString(record, "created_at"), UpdatedAt: dataString(record, "updated_at"), scope: recordScope(record),
		Lines: []inboundLine{}, Attachments: []inboundAttachment{},
	}
	if raw := dataString(record, "lines"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &item.Lines); err != nil {
			return purchaseInbound{}, fmt.Errorf("decode inbound lines: %w", err)
		}
	}
	if raw := dataString(record, "attachments"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &item.Attachments); err != nil {
			return purchaseInbound{}, fmt.Errorf("decode inbound attachments: %w", err)
		}
	}
	return item, nil
}

func inboundPermission(action string) pluginsdk.Permission {
	return pluginsdk.Permission{Resource: "pharma_oa.inbound", Action: action}
}

func inboundIntent(action string, scope employeeScope) pluginsdk.DataScopeIntent {
	return pluginsdk.DataScopeIntent{Permission: inboundPermission(action), Filter: scopeFilter(scope)}
}
