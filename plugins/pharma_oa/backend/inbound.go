package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
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

const purchaseInboundTable = "purchase_inbounds"

var purchaseInboundFields = []string{
	"id", "number", "purchase_order_id", "purchase_order_number", "warehouse_id", "area_id", "location_id",
	"lines", "attachments", "status", "received_by", "received_at", "last_operation_key", "request_hash",
	"tenant_id", "organization_id", "owner_id", "created_at", "updated_at",
}

type inboundLine struct {
	ID             string `json:"id"`
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
	RequestHash         string              `json:"-"`
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
	var input purchaseInboundCreateInput
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := validateInboundCreateInput(&input); err != nil {
		writeServiceError(w, err)
		return
	}
	requestHash, err := purchaseInboundRequestHash(input)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	scope, err := s.exactWriteScope(ctx, inboundPermission("create"), input.TenantID, input.OrganizationID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	operationKey := purchaseInboundOperationKey(scope, requestKey)
	unlock := s.inboundLocks.lock(operationKey)
	defer unlock()
	if existing, found, findErr := s.findPurchaseInboundByOperationKey(ctx, inboundPermission("create"), operationKey); findErr != nil {
		writeServiceError(w, findErr)
		return
	} else if found {
		if existing.scope.TenantID != scope.TenantID || existing.scope.OrganizationID != scope.OrganizationID ||
			existing.RequestHash != requestHash {
			writeServiceError(w, newHTTPError(http.StatusConflict, "purchase_inbound_idempotency_conflict", "idempotency key was used for a different receipt request"))
			return
		}
		order, orderErr := s.getPurchaseOrder(ctx, inboundPermission("create"), existing.PurchaseOrderID)
		if orderErr != nil {
			writeServiceError(w, orderErr)
			return
		}
		writeOK(w, map[string]any{"item": existing, "order": order, "duplicate": true})
		return
	}
	inboundID := stableInventoryID("purchase_inbound", scope.TenantID, scope.OrganizationID, operationKey)
	attachmentAttemptID := ""
	if len(input.Attachments) > 0 {
		attachmentAttemptID, err = newInboundAttachmentAttemptID()
		if err != nil {
			writeServiceError(w, err)
			return
		}
	}
	attachments, err := s.storeInboundAttachments(ctx, inboundID, operationKey, attachmentAttemptID, input.Attachments)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	cleanup := func() {
		for _, attachment := range attachments {
			_ = s.host.Files.Delete(ctx, attachment.FileID)
		}
	}
	item := purchaseInbound{
		ID: inboundID, PurchaseOrderID: input.PurchaseOrderID,
		WarehouseID: input.WarehouseID, AreaID: input.AreaID, LocationID: input.LocationID,
		Attachments: attachments, Status: "completed", ReceivedBy: scope.OwnerID,
		LastOperationKey: operationKey, RequestHash: requestHash, scope: scope,
	}
	var created purchaseInbound
	var updatedOrder purchaseOrder
	var posting inventoryPostingSummary
	duplicate := false
	err = s.transaction(ctx, func(tx context.Context) error {
		now := s.now().UTC()
		number, issueErr := s.host.DocumentNumbers.Issue(tx, pluginsdk.DocumentNumberInput{
			Rule: purchaseDocumentNumberRule("purchase_inbound", "PI"), TenantID: scope.TenantID,
			Permission: inboundPermission("create"), OccurredAt: now, IdempotencyKey: operationKey,
		})
		if issueErr != nil {
			return issueErr
		}
		if number.Duplicate {
			existing, found, findErr := s.findPurchaseInboundByOperationKey(tx, inboundPermission("create"), operationKey)
			if findErr != nil {
				return findErr
			}
			if !found {
				return newHTTPError(http.StatusConflict, "purchase_inbound_idempotency_conflict", "receipt number was already issued without a matching inbound")
			}
			if existing.scope.TenantID != scope.TenantID || existing.scope.OrganizationID != scope.OrganizationID ||
				existing.RequestHash != requestHash {
				return newHTTPError(http.StatusConflict, "purchase_inbound_idempotency_conflict", "idempotency key was used for a different receipt request")
			}
			existingOrder, orderErr := s.getPurchaseOrder(tx, inboundPermission("create"), existing.PurchaseOrderID)
			if orderErr != nil {
				return orderErr
			}
			created, updatedOrder, duplicate = existing, existingOrder, true
			return nil
		}
		item.Number = number.Number
		if _, topologyErr := s.resolveMovementTopology(
			tx, inboundPermission("create"), scope, input.WarehouseID, input.AreaID, input.LocationID,
		); topologyErr != nil {
			return topologyErr
		}
		order, orderErr := s.getPurchaseOrder(tx, inboundPermission("create"), input.PurchaseOrderID)
		if orderErr != nil {
			return orderErr
		}
		if order.scope.TenantID != scope.TenantID || order.scope.OrganizationID != scope.OrganizationID || order.Version != input.OrderVersion {
			return newHTTPError(http.StatusConflict, "stale_purchase_order", "purchase order scope or version is stale")
		}
		if order.Status != "open" && order.Status != "partial" {
			return newHTTPError(http.StatusConflict, "purchase_order_not_receivable", "purchase order is not open for receiving")
		}
		lines, updatedLines, orderStatus, lineErr := applyInboundLines(order.Lines, input.Lines, now)
		if lineErr != nil {
			return lineErr
		}
		for index := range lines {
			lines[index].ID = stableInventoryID(
				"receipt_line", item.ID, lines[index].OrderLineID, lines[index].BatchNo,
			)
		}
		item.PurchaseOrderNumber = order.Number
		item.Lines = lines
		item.ReceivedAt = now.Format(time.RFC3339Nano)
		order.Lines, order.Status, order.LastOperationKey = updatedLines, orderStatus, operationKey
		inboundResult, mutationErr := s.host.DataStore.Mutate(tx, pluginsdk.DataMutation{
			Table: purchaseInboundTable, Operation: pluginsdk.DataMutationInsert, Scope: inboundIntent("create", scope),
			Key: map[string]pluginsdk.DataValue{"id": stringValue(item.ID)}, Values: purchaseInboundValues(item),
			Returning: purchaseInboundFields, IdempotencyKey: operationKey,
		})
		if mutationErr != nil {
			return mutationErr
		}
		created, mutationErr = purchaseInboundFromMutation(inboundResult)
		if mutationErr != nil {
			return mutationErr
		}
		orderResult, mutationErr := s.host.DataStore.Mutate(tx, pluginsdk.DataMutation{
			Table: purchaseOrderTable, Operation: pluginsdk.DataMutationUpdate, Scope: inboundIntent("create", scope),
			Key: map[string]pluginsdk.DataValue{"id": stringValue(order.ID)}, Values: purchaseOrderValues(order),
			Returning: purchaseOrderFields, IdempotencyKey: operationKey + ".order", ExpectedVersion: &input.OrderVersion,
		})
		if mutationErr != nil {
			return mutationErr
		}
		updatedOrder, mutationErr = purchaseOrderFromMutation(orderResult)
		if mutationErr != nil {
			return mutationErr
		}
		posting, mutationErr = s.postInboundInventory(tx, created)
		if mutationErr != nil {
			return mutationErr
		}
		if mutationErr = s.audit(tx, "pharma_oa.inventory.receive", created.ID, pluginsdk.AuditRiskHigh, map[string]any{
			"purchaseOrderId": created.PurchaseOrderID, "warehouseId": created.WarehouseID,
			"locationId": created.LocationID, "lotCount": posting.LotCount,
			"ledgerEntryCount": posting.LedgerEntryCount, "quantity": microsQuantity(posting.QuantityMicros),
		}); mutationErr != nil {
			return mutationErr
		}
		if mutationErr = s.publishInventoryChanged(tx, created, posting); mutationErr != nil {
			return mutationErr
		}
		return s.audit(tx, "pharma_oa.inbound.create", created.ID, pluginsdk.AuditRiskHigh, map[string]any{
			"number": created.Number, "purchaseOrderId": order.ID, "lineCount": len(created.Lines),
			"attachmentCount": len(created.Attachments), "orderStatus": updatedOrder.Status,
			"ledgerEntryCount": posting.LedgerEntryCount, "quantity": microsQuantity(posting.QuantityMicros),
		})
	})
	if err != nil {
		cleanup()
		writeServiceError(w, err)
		return
	}
	if duplicate {
		cleanup()
		writeOK(w, map[string]any{"item": created, "order": updatedOrder, "duplicate": true})
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

func purchaseInboundRequestHash(input purchaseInboundCreateInput) (string, error) {
	type attachmentFingerprint struct {
		Name        string `json:"name"`
		ContentHash string `json:"contentHash"`
		Size        int    `json:"size"`
	}
	type requestFingerprint struct {
		TenantID        string                  `json:"tenantId"`
		OrganizationID  string                  `json:"organizationId"`
		PurchaseOrderID string                  `json:"purchaseOrderId"`
		WarehouseID     string                  `json:"warehouseId"`
		AreaID          string                  `json:"areaId"`
		LocationID      string                  `json:"locationId"`
		OrderVersion    int64                   `json:"orderVersion"`
		Lines           []inboundLineInput      `json:"lines"`
		Attachments     []attachmentFingerprint `json:"attachments"`
	}
	fingerprint := requestFingerprint{
		TenantID: input.TenantID, OrganizationID: input.OrganizationID, PurchaseOrderID: input.PurchaseOrderID,
		WarehouseID: input.WarehouseID, AreaID: input.AreaID, LocationID: input.LocationID,
		OrderVersion: input.OrderVersion, Lines: make([]inboundLineInput, len(input.Lines)),
		Attachments: make([]attachmentFingerprint, len(input.Attachments)),
	}
	for index, line := range input.Lines {
		line.OrderLineID = strings.TrimSpace(line.OrderLineID)
		line.BatchNo = strings.ToUpper(strings.TrimSpace(line.BatchNo))
		if line.OrderLineID == "" || line.BatchNo == "" || len(line.BatchNo) > 64 {
			return "", newHTTPError(http.StatusBadRequest, "invalid_inbound_line", fmt.Sprintf("inbound line %d has an invalid order line or batch", index+1))
		}
		quantity, _, quantityErr := normalizePurchaseDecimal(line.Quantity, 6, "quantity")
		productionDate, productionErr := parseDate(line.ProductionDate)
		expiresAt, expiryErr := parseDate(line.ExpiresAt)
		if quantityErr != nil || productionErr != nil || expiryErr != nil || !expiresAt.After(productionDate) {
			return "", newHTTPError(http.StatusBadRequest, "invalid_inbound_lot", fmt.Sprintf("inbound line %d quantity or lot dates are invalid", index+1))
		}
		line.Quantity = quantity
		line.ProductionDate = productionDate.Format(time.RFC3339Nano)
		line.ExpiresAt = expiresAt.Format(time.RFC3339Nano)
		fingerprint.Lines[index] = line
	}
	for index, attachment := range input.Attachments {
		name := strings.TrimSpace(filepath.Base(attachment.Name))
		content, decodeErr := base64.StdEncoding.DecodeString(attachment.ContentBase64)
		if name == "" || name == "." || decodeErr != nil || len(content) == 0 || len(content) > 2<<20 {
			return "", newHTTPError(http.StatusBadRequest, "invalid_inbound_attachment", fmt.Sprintf("inbound attachment %d is invalid", index+1))
		}
		contentHash := sha256.Sum256(content)
		fingerprint.Attachments[index] = attachmentFingerprint{
			Name: name, ContentHash: hex.EncodeToString(contentHash[:]), Size: len(content),
		}
	}
	raw, err := json.Marshal(fingerprint)
	if err != nil {
		return "", fmt.Errorf("encode purchase inbound request fingerprint: %w", err)
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), nil
}

func purchaseInboundOperationKey(scope employeeScope, requestKey string) string {
	return stableInventoryID(
		"purchase_inbound_create",
		scope.TenantID,
		scope.OrganizationID,
		strings.TrimSpace(requestKey),
	)
}

func newInboundAttachmentAttemptID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate inbound attachment attempt id: %w", err)
	}
	return hex.EncodeToString(raw[:]), nil
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

func (s *server) storeInboundAttachments(
	ctx context.Context,
	inboundID string,
	operationKey string,
	attemptID string,
	inputs []inboundAttachmentInput,
) ([]inboundAttachment, error) {
	if len(inputs) > 0 && (strings.TrimSpace(inboundID) == "" || strings.TrimSpace(operationKey) == "" || strings.TrimSpace(attemptID) == "") {
		return nil, fmt.Errorf("inbound attachment ownership identifiers are required")
	}
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
			Key: "purchase-inbounds/" + inboundID + "/attempts/" + attemptID + "/" + fmt.Sprint(index+1), Name: name, Content: content,
			Visibility: pluginsdk.FileVisibilityPrivate,
			Metadata: map[string]string{
				"purchaseInboundId": inboundID,
				"operationKey":      operationKey,
				"uploadAttemptId":   attemptID,
			},
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
		"request_hash": stringValue(item.RequestHash),
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
		LastOperationKey: dataString(record, "last_operation_key"), RequestHash: dataString(record, "request_hash"), Version: record.Version,
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
