package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

const (
	purchaseRequestTable = "purchase_requests"
	purchaseOrderTable   = "purchase_orders"
)

var (
	purchaseRequestFields = []string{"id", "number", "supplier_id", "supplier_code", "supplier_name", "requester_id", "approver_id", "approver_name", "reason", "currency", "lines", "total_amount", "status", "workflow_definition_id", "workflow_instance_id", "purchase_order_id", "decision_comment", "decided_at", "last_operation_key", "tenant_id", "organization_id", "owner_id", "created_at", "updated_at"}
	purchaseOrderFields   = []string{"id", "number", "purchase_request_id", "supplier_id", "supplier_code", "supplier_name", "currency", "lines", "total_amount", "status", "approved_by", "approved_at", "last_operation_key", "tenant_id", "organization_id", "owner_id", "created_at", "updated_at"}
	purchaseDecimal       = regexp.MustCompile(`^(0|[1-9][0-9]{0,11})(?:\.([0-9]+))?$`)
)

type purchaseLine struct {
	ID               string `json:"id"`
	ProductID        string `json:"productId"`
	ProductCode      string `json:"productCode"`
	ProductName      string `json:"productName"`
	Specification    string `json:"specification"`
	UnitID           string `json:"unitId"`
	Quantity         string `json:"quantity"`
	UnitPrice        string `json:"unitPrice"`
	Amount           string `json:"amount"`
	ReceivedQuantity string `json:"receivedQuantity"`
}

type purchaseLineInput struct {
	ProductID string `json:"productId"`
	Quantity  string `json:"quantity"`
	UnitPrice string `json:"unitPrice"`
}

type purchaseRequest struct {
	ID                   string         `json:"id"`
	Number               string         `json:"number"`
	SupplierID           string         `json:"supplierId"`
	SupplierCode         string         `json:"supplierCode"`
	SupplierName         string         `json:"supplierName"`
	RequesterID          string         `json:"requesterId"`
	ApproverID           string         `json:"approverId"`
	ApproverName         string         `json:"approverName"`
	Reason               string         `json:"reason"`
	Currency             string         `json:"currency"`
	Lines                []purchaseLine `json:"lines"`
	TotalAmount          string         `json:"totalAmount"`
	Status               string         `json:"status"`
	WorkflowDefinitionID string         `json:"workflowDefinitionId"`
	WorkflowInstanceID   string         `json:"workflowInstanceId"`
	PurchaseOrderID      string         `json:"purchaseOrderId,omitempty"`
	DecisionComment      string         `json:"decisionComment,omitempty"`
	DecidedAt            string         `json:"decidedAt,omitempty"`
	LastOperationKey     string         `json:"-"`
	Version              int64          `json:"version"`
	CreatedAt            string         `json:"createdAt"`
	UpdatedAt            string         `json:"updatedAt"`
	scope                employeeScope
}

type purchaseOrder struct {
	ID                string         `json:"id"`
	Number            string         `json:"number"`
	PurchaseRequestID string         `json:"purchaseRequestId"`
	SupplierID        string         `json:"supplierId"`
	SupplierCode      string         `json:"supplierCode"`
	SupplierName      string         `json:"supplierName"`
	Currency          string         `json:"currency"`
	Lines             []purchaseLine `json:"lines"`
	TotalAmount       string         `json:"totalAmount"`
	Status            string         `json:"status"`
	ApprovedBy        string         `json:"approvedBy"`
	ApprovedAt        string         `json:"approvedAt"`
	LastOperationKey  string         `json:"-"`
	Version           int64          `json:"version"`
	CreatedAt         string         `json:"createdAt"`
	UpdatedAt         string         `json:"updatedAt"`
	scope             employeeScope
}

type purchaseRequestCreateInput struct {
	TenantID       string              `json:"tenantId"`
	OrganizationID string              `json:"organizationId"`
	SupplierID     string              `json:"supplierId"`
	Reason         string              `json:"reason"`
	Currency       string              `json:"currency"`
	ApproverID     string              `json:"approverId"`
	ApproverName   string              `json:"approverName"`
	Lines          []purchaseLineInput `json:"lines"`
}

type purchaseDecisionInput struct {
	TaskID  string `json:"taskId"`
	Comment string `json:"comment"`
	Version int64  `json:"version"`
}

func (s *server) listPurchaseRequests(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	items, err := s.queryPurchaseRequests(ctx, r.URL.Query().Get("keyword"), r.URL.Query().Get("status"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, map[string]any{"items": items, "total": len(items)})
}

func (s *server) getPurchaseRequestHandler(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	item, err := s.getPurchaseRequest(ctx, purchasePermission("read"), r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	workflow, err := s.host.Workflows.GetInstance(ctx, item.WorkflowInstanceID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, map[string]any{"item": item, "workflow": oaWorkflowView(workflow)})
}

func (s *server) createPurchaseRequest(w http.ResponseWriter, r *http.Request) {
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
	if existing, found, findErr := s.findPurchaseRequestByOperationKey(ctx, purchasePermission("create"), requestKey); findErr != nil {
		writeServiceError(w, findErr)
		return
	} else if found {
		workflow, workflowErr := s.host.Workflows.GetInstance(ctx, existing.WorkflowInstanceID)
		if workflowErr != nil {
			writeServiceError(w, workflowErr)
			return
		}
		writeOK(w, map[string]any{"item": existing, "workflow": oaWorkflowView(workflow), "duplicate": true})
		return
	}
	var input purchaseRequestCreateInput
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := validatePurchaseRequestInput(&input); err != nil {
		writeServiceError(w, err)
		return
	}
	scope, err := s.exactWriteScope(ctx, purchasePermission("create"), input.TenantID, input.OrganizationID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	supplier, lines, total, err := s.resolvePurchaseReferences(ctx, purchasePermission("create"), input.SupplierID, input.Lines)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	now := s.now().UTC()
	item := purchaseRequest{
		ID: newEntityID("purchase-request"), SupplierID: supplier.ID, SupplierCode: supplier.Code, SupplierName: supplier.Name,
		RequesterID: scope.OwnerID, ApproverID: input.ApproverID, ApproverName: input.ApproverName, Reason: input.Reason,
		Currency: input.Currency, Lines: lines, TotalAmount: total, Status: "pending", LastOperationKey: requestKey, scope: scope,
	}
	item.WorkflowDefinitionID, item.WorkflowInstanceID = item.ID+"-definition", item.ID+"-workflow"
	var created purchaseRequest
	var workflow pluginsdk.WorkflowInstance
	err = s.transaction(ctx, func(tx context.Context) error {
		number, issueErr := s.host.DocumentNumbers.Issue(tx, pluginsdk.DocumentNumberInput{
			Rule: purchaseDocumentNumberRule("purchase_request", "PR"), TenantID: scope.TenantID,
			Permission: purchasePermission("create"), OccurredAt: now, IdempotencyKey: requestKey,
		})
		if issueErr != nil {
			return issueErr
		}
		item.Number = number.Number
		if _, workflowErr := s.host.Workflows.CreateDefinition(tx, pluginsdk.WorkflowDefinitionInput{
			ID: item.WorkflowDefinitionID, Key: "purchase-request-" + item.ID, Name: item.Number + " purchase approval", Version: 1,
			Nodes: []pluginsdk.WorkflowNode{
				{ID: "start", Key: "start", Name: "Start", Type: pluginsdk.WorkflowNodeStart},
				{ID: "approval", Key: "approval", Name: "Purchase approval", Type: pluginsdk.WorkflowNodeApproval, AssigneeIDs: []string{item.ApproverID}, Decision: &pluginsdk.WorkflowDecisionRule{Strategy: pluginsdk.WorkflowDecisionAny, Quorum: 1}},
				{ID: "end", Key: "end", Name: "Complete", Type: pluginsdk.WorkflowNodeEnd},
			},
			Transitions: []pluginsdk.WorkflowTransition{{From: "start", To: "approval"}, {From: "approval", To: "end"}},
		}); workflowErr != nil {
			return workflowErr
		}
		if _, workflowErr := s.host.Workflows.PublishDefinition(tx, item.WorkflowDefinitionID); workflowErr != nil {
			return workflowErr
		}
		var workflowErr error
		workflow, workflowErr = s.host.Workflows.Start(tx, pluginsdk.WorkflowStartInput{
			ID: item.WorkflowInstanceID, DefinitionID: item.WorkflowDefinitionID, BusinessType: "purchase_request", BusinessID: item.ID, Title: item.Number,
		})
		if workflowErr != nil {
			return workflowErr
		}
		result, mutationErr := s.host.DataStore.Mutate(tx, pluginsdk.DataMutation{
			Table: purchaseRequestTable, Operation: pluginsdk.DataMutationInsert, Scope: purchaseIntent("create", scope),
			Key: map[string]pluginsdk.DataValue{"id": stringValue(item.ID)}, Values: purchaseRequestValues(item),
			Returning: purchaseRequestFields, IdempotencyKey: requestKey,
		})
		if mutationErr != nil {
			return mutationErr
		}
		created, mutationErr = purchaseRequestFromMutation(result)
		if mutationErr != nil {
			return mutationErr
		}
		return s.audit(tx, "pharma_oa.purchase.create", created.ID, pluginsdk.AuditRiskHigh, map[string]any{
			"number": created.Number, "supplierId": created.SupplierID, "lineCount": len(created.Lines), "totalAmount": created.TotalAmount,
		})
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeCreated(w, map[string]any{"item": created, "workflow": oaWorkflowView(workflow), "duplicate": false})
}

func (s *server) approvePurchaseRequest(w http.ResponseWriter, r *http.Request) {
	s.decidePurchaseRequest(w, r, true)
}

func (s *server) rejectPurchaseRequest(w http.ResponseWriter, r *http.Request) {
	s.decidePurchaseRequest(w, r, false)
}

func (s *server) decidePurchaseRequest(w http.ResponseWriter, r *http.Request, approve bool) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	action := "reject"
	if approve {
		action = "approve"
	}
	requestKey, err := mutationKey(r, action)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	var input purchaseDecisionInput
	if !decodeJSON(w, r, &input) {
		return
	}
	input.TaskID, input.Comment = strings.TrimSpace(input.TaskID), strings.TrimSpace(input.Comment)
	if input.TaskID == "" || input.Version < 1 || len(input.Comment) > 1000 {
		writeServiceError(w, newHTTPError(http.StatusBadRequest, "invalid_purchase_decision", "taskId, current version, and a bounded comment are required"))
		return
	}
	item, err := s.getPurchaseRequest(ctx, purchasePermission(action), r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if item.Status != "pending" {
		if item.LastOperationKey == requestKey && (item.Status == "approved" || item.Status == "rejected") {
			s.writeDuplicatePurchaseDecision(w, ctx, item)
			return
		}
		writeServiceError(w, newHTTPError(http.StatusConflict, "purchase_not_actionable", "purchase request is no longer pending"))
		return
	}
	if item.Version != input.Version {
		writeServiceError(w, newHTTPError(http.StatusConflict, "stale_purchase_request", "purchase request version is stale"))
		return
	}
	if approve {
		lineInputs := make([]purchaseLineInput, 0, len(item.Lines))
		for _, line := range item.Lines {
			lineInputs = append(lineInputs, purchaseLineInput{ProductID: line.ProductID, Quantity: line.Quantity, UnitPrice: line.UnitPrice})
		}
		if _, _, _, err := s.resolvePurchaseReferences(ctx, purchasePermission("approve"), item.SupplierID, lineInputs); err != nil {
			writeServiceError(w, err)
			return
		}
	}
	var workflow pluginsdk.WorkflowInstance
	var updated purchaseRequest
	var order purchaseOrder
	err = s.transaction(ctx, func(tx context.Context) error {
		workflowInput := pluginsdk.WorkflowTaskActionInput{InstanceID: item.WorkflowInstanceID, TaskID: input.TaskID, Comment: input.Comment}
		var workflowErr error
		if approve {
			workflow, workflowErr = s.host.Workflows.Approve(tx, workflowInput)
		} else {
			workflow, workflowErr = s.host.Workflows.Reject(tx, workflowInput)
		}
		if workflowErr != nil {
			return workflowErr
		}
		item.Status, item.DecisionComment, item.DecidedAt, item.LastOperationKey = string(workflow.Status), input.Comment, s.now().UTC().Format(time.RFC3339Nano), requestKey
		if approve {
			orderID := newEntityID("purchase-order")
			number, issueErr := s.host.DocumentNumbers.Issue(tx, pluginsdk.DocumentNumberInput{
				Rule: purchaseDocumentNumberRule("purchase_order", "PO"), TenantID: item.scope.TenantID,
				Permission: purchasePermission("approve"), OccurredAt: s.now().UTC(), IdempotencyKey: requestKey,
			})
			if issueErr != nil {
				return issueErr
			}
			item.PurchaseOrderID = orderID
			order = purchaseOrder{
				ID: orderID, Number: number.Number, PurchaseRequestID: item.ID, SupplierID: item.SupplierID, SupplierCode: item.SupplierCode,
				SupplierName: item.SupplierName, Currency: item.Currency, Lines: item.Lines, TotalAmount: item.TotalAmount, Status: "open",
				ApprovedBy: item.ApproverID, ApprovedAt: item.DecidedAt, LastOperationKey: requestKey, scope: item.scope,
			}
		}
		result, mutationErr := s.host.DataStore.Mutate(tx, pluginsdk.DataMutation{
			Table: purchaseRequestTable, Operation: pluginsdk.DataMutationUpdate, Scope: purchaseIntent(action, item.scope),
			Key: map[string]pluginsdk.DataValue{"id": stringValue(item.ID)}, Values: purchaseRequestValues(item),
			Returning: purchaseRequestFields, IdempotencyKey: requestKey, ExpectedVersion: &input.Version,
		})
		if mutationErr != nil {
			return mutationErr
		}
		updated, mutationErr = purchaseRequestFromMutation(result)
		if mutationErr != nil {
			return mutationErr
		}
		if approve {
			orderResult, orderErr := s.host.DataStore.Mutate(tx, pluginsdk.DataMutation{
				Table: purchaseOrderTable, Operation: pluginsdk.DataMutationInsert, Scope: purchaseIntent("approve", item.scope),
				Key: map[string]pluginsdk.DataValue{"id": stringValue(order.ID)}, Values: purchaseOrderValues(order),
				Returning: purchaseOrderFields, IdempotencyKey: requestKey + ".order",
			})
			if orderErr != nil {
				return orderErr
			}
			order, orderErr = purchaseOrderFromMutation(orderResult)
			if orderErr != nil {
				return orderErr
			}
			if auditErr := s.audit(tx, "pharma_oa.purchase.order.create", order.ID, pluginsdk.AuditRiskHigh, map[string]any{"requestId": item.ID, "number": order.Number}); auditErr != nil {
				return auditErr
			}
		}
		return s.audit(tx, "pharma_oa.purchase."+action, item.ID, pluginsdk.AuditRiskHigh, map[string]any{"comment": input.Comment, "orderId": item.PurchaseOrderID})
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	payload := map[string]any{"item": updated, "workflow": oaWorkflowView(workflow), "duplicate": false}
	if approve {
		payload["order"] = order
	}
	writeOK(w, payload)
}

func (s *server) writeDuplicatePurchaseDecision(w http.ResponseWriter, ctx context.Context, item purchaseRequest) {
	workflow, err := s.host.Workflows.GetInstance(ctx, item.WorkflowInstanceID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	payload := map[string]any{"item": item, "workflow": oaWorkflowView(workflow), "duplicate": true}
	if item.PurchaseOrderID != "" {
		order, orderErr := s.getPurchaseOrder(ctx, orderPermission("read"), item.PurchaseOrderID)
		if orderErr != nil {
			writeServiceError(w, orderErr)
			return
		}
		payload["order"] = order
	}
	writeOK(w, payload)
}

func (s *server) listPurchaseOrders(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	items, err := s.queryPurchaseOrders(ctx, r.URL.Query().Get("keyword"), r.URL.Query().Get("status"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, map[string]any{"items": items, "total": len(items)})
}

func (s *server) getPurchaseOrderHandler(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	item, err := s.getPurchaseOrder(ctx, orderPermission("read"), r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, map[string]any{"item": item})
}

func (s *server) resolvePurchaseReferences(ctx context.Context, permission pluginsdk.Permission, supplierID string, inputs []purchaseLineInput) (party, []purchaseLine, string, error) {
	supplier, err := s.getParty(ctx, "supplier", permission, supplierID)
	if err != nil || supplier.Status != "active" {
		return party{}, nil, "", newHTTPError(http.StatusUnprocessableEntity, "invalid_purchase_supplier", "supplier is missing, disabled, or outside the trusted scope")
	}
	if err := s.requireQualificationGate(ctx, permission, "supplier", supplier.ID, "purchase"); err != nil {
		return party{}, nil, "", err
	}
	lines := make([]purchaseLine, 0, len(inputs))
	seen := make(map[string]struct{}, len(inputs))
	total := new(big.Rat)
	for index, input := range inputs {
		input.ProductID = strings.TrimSpace(input.ProductID)
		if _, exists := seen[input.ProductID]; exists {
			return party{}, nil, "", newHTTPError(http.StatusConflict, "duplicate_purchase_product", fmt.Sprintf("purchase line %d repeats a product", index+1))
		}
		seen[input.ProductID] = struct{}{}
		product, productErr := s.getProduct(ctx, permission, input.ProductID)
		if productErr != nil || product.Status != "active" {
			return party{}, nil, "", newHTTPError(http.StatusUnprocessableEntity, "invalid_purchase_product", fmt.Sprintf("purchase line %d product is missing, disabled, or outside the trusted scope", index+1))
		}
		if gateErr := s.requireQualificationGate(ctx, permission, "product", product.ID, "purchase"); gateErr != nil {
			return party{}, nil, "", gateErr
		}
		if gateErr := s.requireQualificationGate(ctx, permission, "manufacturer", product.ManufacturerID, "supply"); gateErr != nil {
			return party{}, nil, "", gateErr
		}
		quantity, quantityValue, valueErr := normalizePurchaseDecimal(input.Quantity, 6, "quantity")
		if valueErr != nil {
			return party{}, nil, "", newHTTPError(http.StatusBadRequest, "invalid_purchase_quantity", fmt.Sprintf("purchase line %d quantity is invalid", index+1))
		}
		unitPrice, priceValue, valueErr := normalizePurchaseDecimal(input.UnitPrice, 2, "unit price")
		if valueErr != nil {
			return party{}, nil, "", newHTTPError(http.StatusBadRequest, "invalid_purchase_price", fmt.Sprintf("purchase line %d unit price is invalid", index+1))
		}
		amount := new(big.Rat).Mul(quantityValue, priceValue).FloatString(2)
		amountValue, _ := new(big.Rat).SetString(amount)
		total.Add(total, amountValue)
		lines = append(lines, purchaseLine{
			ID: newEntityID("purchase-line"), ProductID: product.ID, ProductCode: product.Code, ProductName: product.Name,
			Specification: product.Specification, UnitID: product.UnitID, Quantity: quantity, UnitPrice: unitPrice,
			Amount: amount, ReceivedQuantity: "0",
		})
	}
	return supplier, lines, total.FloatString(2), nil
}

func (s *server) requireQualificationGate(ctx context.Context, permission pluginsdk.Permission, subjectType, subjectID, gate string) error {
	required, missing, err := s.qualificationEligibility(ctx, permission, subjectType, subjectID, gate)
	if err != nil {
		return err
	}
	if len(required) == 0 || len(missing) > 0 {
		return newHTTPError(http.StatusUnprocessableEntity, "qualification_gate_failed", fmt.Sprintf("%s %s does not satisfy the %s qualification gate", subjectType, subjectID, gate))
	}
	return nil
}

func validatePurchaseRequestInput(input *purchaseRequestCreateInput) error {
	input.TenantID, input.OrganizationID = strings.TrimSpace(input.TenantID), strings.TrimSpace(input.OrganizationID)
	input.SupplierID, input.Reason = strings.TrimSpace(input.SupplierID), strings.TrimSpace(input.Reason)
	input.Currency, input.ApproverID, input.ApproverName = strings.ToUpper(strings.TrimSpace(input.Currency)), strings.TrimSpace(input.ApproverID), strings.TrimSpace(input.ApproverName)
	if input.TenantID == "" || input.OrganizationID == "" || input.SupplierID == "" || input.Reason == "" || len(input.Reason) > 2000 ||
		input.Currency != "CNY" || input.ApproverID == "" || input.ApproverName == "" || len(input.ApproverName) > 200 ||
		len(input.Lines) == 0 || len(input.Lines) > 200 {
		return newHTTPError(http.StatusBadRequest, "invalid_purchase_request", "scope, supplier, reason, CNY currency, approver, and 1 to 200 lines are required")
	}
	return nil
}

func normalizePurchaseDecimal(raw string, maxScale int, field string) (string, *big.Rat, error) {
	raw = strings.TrimSpace(raw)
	match := purchaseDecimal.FindStringSubmatch(raw)
	if match == nil || len(match[2]) > maxScale {
		return "", nil, fmt.Errorf("%s must be a bounded positive decimal with at most %d fractional digits", field, maxScale)
	}
	value, ok := new(big.Rat).SetString(raw)
	if !ok || value.Sign() <= 0 {
		return "", nil, fmt.Errorf("%s must be positive", field)
	}
	if maxScale == 2 {
		return value.FloatString(2), value, nil
	}
	canonical := raw
	if strings.Contains(canonical, ".") {
		canonical = strings.TrimRight(strings.TrimRight(canonical, "0"), ".")
	}
	if canonical == "" {
		canonical = "0"
	}
	return canonical, value, nil
}

func purchaseDocumentNumberRule(documentType, prefix string) pluginsdk.DocumentNumberRule {
	return pluginsdk.DocumentNumberRule{
		DocumentType: documentType, Prefix: prefix, Separator: "-", Period: pluginsdk.DocumentNumberPeriodMonth,
		Width: 6, Start: 1, GapPolicy: pluginsdk.DocumentNumberGapTransactional,
	}
}

func (s *server) getPurchaseRequest(ctx context.Context, permission pluginsdk.Permission, id string) (purchaseRequest, error) {
	idValue := stringValue(strings.TrimSpace(id))
	filter := pluginsdk.DataFilter{Field: "id", Operator: pluginsdk.DataOperatorEqual, Value: &idValue}
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{
		Table: purchaseRequestTable, Fields: purchaseRequestFields, Scope: pluginsdk.DataScopeIntent{Permission: permission},
		Filter: &filter, Sort: []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 1},
	})
	if err != nil {
		return purchaseRequest{}, err
	}
	if len(page.Records) != 1 {
		return purchaseRequest{}, newHTTPError(http.StatusNotFound, "purchase_request_not_found", "purchase request was not found")
	}
	return purchaseRequestFromRecord(page.Records[0])
}

func (s *server) findPurchaseRequestByOperationKey(ctx context.Context, permission pluginsdk.Permission, key string) (purchaseRequest, bool, error) {
	keyValue := stringValue(key)
	filter := pluginsdk.DataFilter{Field: "last_operation_key", Operator: pluginsdk.DataOperatorEqual, Value: &keyValue}
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{
		Table: purchaseRequestTable, Fields: purchaseRequestFields, Scope: pluginsdk.DataScopeIntent{Permission: permission},
		Filter: &filter, Sort: []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 1},
	})
	if err != nil {
		return purchaseRequest{}, false, err
	}
	if len(page.Records) == 0 {
		return purchaseRequest{}, false, nil
	}
	item, err := purchaseRequestFromRecord(page.Records[0])
	return item, err == nil, err
}

func (s *server) queryPurchaseRequests(ctx context.Context, keyword, status string) ([]purchaseRequest, error) {
	filter, err := purchaseListFilter(keyword, status, []string{"number", "supplier_name", "reason"})
	if err != nil {
		return nil, err
	}
	items := make([]purchaseRequest, 0)
	err = s.queryPurchaseRecords(ctx, purchaseRequestTable, purchaseRequestFields, purchasePermission("read"), filter, func(record pluginsdk.DataRecord) error {
		item, parseErr := purchaseRequestFromRecord(record)
		if parseErr == nil {
			items = append(items, item)
		}
		return parseErr
	})
	sort.SliceStable(items, func(left, right int) bool { return items[left].UpdatedAt > items[right].UpdatedAt })
	return items, err
}

func (s *server) getPurchaseOrder(ctx context.Context, permission pluginsdk.Permission, id string) (purchaseOrder, error) {
	idValue := stringValue(strings.TrimSpace(id))
	filter := pluginsdk.DataFilter{Field: "id", Operator: pluginsdk.DataOperatorEqual, Value: &idValue}
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{
		Table: purchaseOrderTable, Fields: purchaseOrderFields, Scope: pluginsdk.DataScopeIntent{Permission: permission},
		Filter: &filter, Sort: []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 1},
	})
	if err != nil {
		return purchaseOrder{}, err
	}
	if len(page.Records) != 1 {
		return purchaseOrder{}, newHTTPError(http.StatusNotFound, "purchase_order_not_found", "purchase order was not found")
	}
	return purchaseOrderFromRecord(page.Records[0])
}

func (s *server) queryPurchaseOrders(ctx context.Context, keyword, status string) ([]purchaseOrder, error) {
	filter, err := purchaseListFilter(keyword, status, []string{"number", "supplier_name"})
	if err != nil {
		return nil, err
	}
	items := make([]purchaseOrder, 0)
	err = s.queryPurchaseRecords(ctx, purchaseOrderTable, purchaseOrderFields, orderPermission("read"), filter, func(record pluginsdk.DataRecord) error {
		item, parseErr := purchaseOrderFromRecord(record)
		if parseErr == nil {
			items = append(items, item)
		}
		return parseErr
	})
	sort.SliceStable(items, func(left, right int) bool { return items[left].UpdatedAt > items[right].UpdatedAt })
	return items, err
}

func (s *server) queryPurchaseRecords(ctx context.Context, table string, fields []string, permission pluginsdk.Permission, filter *pluginsdk.DataFilter, appendRecord func(pluginsdk.DataRecord) error) error {
	cursor := ""
	for pageNumber := 0; pageNumber < 25; pageNumber++ {
		page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{
			Table: table, Fields: fields, Scope: pluginsdk.DataScopeIntent{Permission: permission}, Filter: filter,
			Sort: []pluginsdk.DataSort{{Field: "updated_at", Direction: pluginsdk.DataSortDescending}}, Page: pluginsdk.DataPageRequest{Cursor: cursor, Limit: 200},
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
	return pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorLimitExceeded, table, "purchase query exceeds 5000 records", false)
}

func purchaseListFilter(keyword, status string, keywordFields []string) (*pluginsdk.DataFilter, error) {
	keyword, status = strings.TrimSpace(keyword), strings.ToLower(strings.TrimSpace(status))
	if len(keyword) > 100 {
		return nil, newHTTPError(http.StatusBadRequest, "invalid_purchase_filter", "purchase keyword exceeds 100 characters")
	}
	if status != "" {
		valid := map[string]bool{"pending": true, "approved": true, "rejected": true, "open": true, "partial": true, "received": true}
		if !valid[status] {
			return nil, newHTTPError(http.StatusBadRequest, "invalid_purchase_filter", "purchase status is invalid")
		}
	}
	all := make([]pluginsdk.DataFilter, 0, 2)
	if status != "" {
		value := stringValue(status)
		all = append(all, pluginsdk.DataFilter{Field: "status", Operator: pluginsdk.DataOperatorEqual, Value: &value})
	}
	if keyword != "" {
		value := stringValue(keyword)
		any := make([]pluginsdk.DataFilter, 0, len(keywordFields))
		for _, field := range keywordFields {
			any = append(any, pluginsdk.DataFilter{Field: field, Operator: pluginsdk.DataOperatorContains, Value: &value})
		}
		all = append(all, pluginsdk.DataFilter{Any: any})
	}
	if len(all) == 0 {
		return nil, nil
	}
	return &pluginsdk.DataFilter{All: all}, nil
}

func purchaseRequestValues(item purchaseRequest) map[string]pluginsdk.DataValue {
	return map[string]pluginsdk.DataValue{
		"number": stringValue(item.Number), "supplier_id": stringValue(item.SupplierID), "supplier_code": stringValue(item.SupplierCode),
		"supplier_name": stringValue(item.SupplierName), "requester_id": stringValue(item.RequesterID), "approver_id": stringValue(item.ApproverID),
		"approver_name": stringValue(item.ApproverName), "reason": stringValue(item.Reason), "currency": stringValue(item.Currency),
		"lines": jsonValue(item.Lines), "total_amount": decimalValue(item.TotalAmount), "status": stringValue(item.Status),
		"workflow_definition_id": stringValue(item.WorkflowDefinitionID), "workflow_instance_id": stringValue(item.WorkflowInstanceID),
		"purchase_order_id": nullableStringValue(item.PurchaseOrderID), "decision_comment": nullableStringValue(item.DecisionComment),
		"decided_at": nullableTimestampValue(item.DecidedAt), "last_operation_key": stringValue(item.LastOperationKey),
	}
}

func purchaseOrderValues(item purchaseOrder) map[string]pluginsdk.DataValue {
	return map[string]pluginsdk.DataValue{
		"number": stringValue(item.Number), "purchase_request_id": stringValue(item.PurchaseRequestID),
		"supplier_id": stringValue(item.SupplierID), "supplier_code": stringValue(item.SupplierCode), "supplier_name": stringValue(item.SupplierName),
		"currency": stringValue(item.Currency), "lines": jsonValue(item.Lines), "total_amount": decimalValue(item.TotalAmount),
		"status": stringValue(item.Status), "approved_by": stringValue(item.ApprovedBy), "approved_at": timestampValue(item.ApprovedAt),
		"last_operation_key": stringValue(item.LastOperationKey),
	}
}

func decimalValue(value string) pluginsdk.DataValue {
	return pluginsdk.DataValue{Type: pluginsdk.DataValueDecimal, Value: value}
}

func purchaseRequestFromMutation(result pluginsdk.DataMutationResult) (purchaseRequest, error) {
	if result.Record == nil {
		return purchaseRequest{}, errorsNewDataRecord("purchase request")
	}
	return purchaseRequestFromRecord(*result.Record)
}

func purchaseOrderFromMutation(result pluginsdk.DataMutationResult) (purchaseOrder, error) {
	if result.Record == nil {
		return purchaseOrder{}, errorsNewDataRecord("purchase order")
	}
	return purchaseOrderFromRecord(*result.Record)
}

func errorsNewDataRecord(name string) error {
	return fmt.Errorf("%s mutation did not return a record", name)
}

func purchaseRequestFromRecord(record pluginsdk.DataRecord) (purchaseRequest, error) {
	item := purchaseRequest{
		ID: dataString(record, "id"), Number: dataString(record, "number"), SupplierID: dataString(record, "supplier_id"),
		SupplierCode: dataString(record, "supplier_code"), SupplierName: dataString(record, "supplier_name"), RequesterID: dataString(record, "requester_id"),
		ApproverID: dataString(record, "approver_id"), ApproverName: dataString(record, "approver_name"), Reason: dataString(record, "reason"),
		Currency: dataString(record, "currency"), TotalAmount: dataString(record, "total_amount"), Status: dataString(record, "status"),
		WorkflowDefinitionID: dataString(record, "workflow_definition_id"), WorkflowInstanceID: dataString(record, "workflow_instance_id"),
		PurchaseOrderID: dataString(record, "purchase_order_id"), DecisionComment: dataString(record, "decision_comment"),
		DecidedAt: dataString(record, "decided_at"), LastOperationKey: dataString(record, "last_operation_key"),
		Version: record.Version, CreatedAt: dataString(record, "created_at"), UpdatedAt: dataString(record, "updated_at"),
		scope: recordScope(record), Lines: []purchaseLine{},
	}
	if raw := dataString(record, "lines"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &item.Lines); err != nil {
			return purchaseRequest{}, fmt.Errorf("decode purchase request lines: %w", err)
		}
	}
	return item, nil
}

func purchaseOrderFromRecord(record pluginsdk.DataRecord) (purchaseOrder, error) {
	item := purchaseOrder{
		ID: dataString(record, "id"), Number: dataString(record, "number"), PurchaseRequestID: dataString(record, "purchase_request_id"),
		SupplierID: dataString(record, "supplier_id"), SupplierCode: dataString(record, "supplier_code"), SupplierName: dataString(record, "supplier_name"),
		Currency: dataString(record, "currency"), TotalAmount: dataString(record, "total_amount"), Status: dataString(record, "status"),
		ApprovedBy: dataString(record, "approved_by"), ApprovedAt: dataString(record, "approved_at"),
		LastOperationKey: dataString(record, "last_operation_key"), Version: record.Version,
		CreatedAt: dataString(record, "created_at"), UpdatedAt: dataString(record, "updated_at"), scope: recordScope(record), Lines: []purchaseLine{},
	}
	if raw := dataString(record, "lines"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &item.Lines); err != nil {
			return purchaseOrder{}, fmt.Errorf("decode purchase order lines: %w", err)
		}
	}
	return item, nil
}

func purchasePermission(action string) pluginsdk.Permission {
	return pluginsdk.Permission{Resource: "pharma_oa.purchase", Action: action}
}

func orderPermission(action string) pluginsdk.Permission {
	return pluginsdk.Permission{Resource: "pharma_oa.order", Action: action}
}

func purchaseIntent(action string, scope employeeScope) pluginsdk.DataScopeIntent {
	return pluginsdk.DataScopeIntent{Permission: purchasePermission(action), Filter: scopeFilter(scope)}
}
