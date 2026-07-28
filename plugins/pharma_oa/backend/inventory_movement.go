package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

const (
	inventoryMovementTable = "inventory_movements"
	stockReturnTotalTable  = "stock_return_totals"

	movementStockTransfer   = "stock_transfer"
	movementStocktake       = "stocktake"
	movementStockAdjustment = "stock_adjustment"
	movementStockReturn     = "stock_return"

	movementStatusPendingApproval = "pending_approval"
	movementStatusPosting         = "posting"
	movementStatusPosted          = "posted"
	movementStatusCompleted       = "completed"
	movementStatusRejected        = "rejected"
)

var (
	inventoryMovementFields = []string{
		"id", "number", "movement_type", "return_type", "reason",
		"source_warehouse_id", "source_area_id", "source_location_id",
		"destination_warehouse_id", "destination_area_id", "destination_location_id",
		"legs", "requester_id", "approver_id", "approver_name",
		"workflow_definition_id", "workflow_instance_id", "status",
		"create_operation_key", "request_hash", "last_operation_key", "last_operation_hash",
		"decision_comment", "decided_at", "posted_at", "requested_at",
		"tenant_id", "organization_id", "owner_id", "created_at", "updated_at",
	}
	stockReturnTotalFields = []string{
		"id", "reference_ledger_entry_id", "return_type", "quantity_micros",
		"tenant_id", "organization_id", "owner_id", "created_at", "updated_at",
	}
)

type inventoryMovement struct {
	ID                     string                 `json:"id"`
	Number                 string                 `json:"number"`
	MovementType           string                 `json:"movementType"`
	ReturnType             string                 `json:"returnType,omitempty"`
	Reason                 string                 `json:"reason"`
	SourceWarehouseID      string                 `json:"warehouseId"`
	SourceAreaID           string                 `json:"areaId"`
	SourceLocationID       string                 `json:"locationId"`
	DestinationWarehouseID string                 `json:"destinationWarehouseId,omitempty"`
	DestinationAreaID      string                 `json:"destinationAreaId,omitempty"`
	DestinationLocationID  string                 `json:"destinationLocationId,omitempty"`
	Legs                   []inventoryMovementLeg `json:"lines"`
	RequesterID            string                 `json:"requesterId"`
	ApproverID             string                 `json:"approverId,omitempty"`
	ApproverName           string                 `json:"approverName,omitempty"`
	WorkflowDefinitionID   string                 `json:"workflowDefinitionId,omitempty"`
	WorkflowInstanceID     string                 `json:"workflowInstanceId,omitempty"`
	Status                 string                 `json:"status"`
	CreateOperationKey     string                 `json:"-"`
	RequestHash            string                 `json:"-"`
	LastOperationKey       string                 `json:"-"`
	LastOperationHash      string                 `json:"-"`
	DecisionComment        string                 `json:"decisionComment,omitempty"`
	DecidedAt              string                 `json:"decidedAt,omitempty"`
	PostedAt               string                 `json:"postedAt,omitempty"`
	RequestedAt            string                 `json:"requestedAt"`
	Version                int64                  `json:"version"`
	CreatedAt              string                 `json:"createdAt"`
	UpdatedAt              string                 `json:"updatedAt"`
	scope                  employeeScope
}

type inventoryMovementLeg struct {
	ID                     string `json:"id"`
	GroupID                string `json:"groupId"`
	LegType                string `json:"legType"`
	ProductID              string `json:"productId"`
	LotID                  string `json:"lotId"`
	BatchNo                string `json:"batchNo"`
	WarehouseID            string `json:"warehouseId"`
	AreaID                 string `json:"areaId"`
	LocationID             string `json:"locationId"`
	Quantity               string `json:"quantity"`
	SystemQuantity         string `json:"systemQuantity,omitempty"`
	CountedQuantity        string `json:"countedQuantity,omitempty"`
	ExpectedBalanceVersion int64  `json:"expectedBalanceVersion"`
	ReferenceLedgerEntryID string `json:"sourceLedgerEntryId,omitempty"`
	quantityMicros         int64
	systemQuantityMicros   int64
	countedQuantityMicros  int64
}

type inventoryMovementCreateInput struct {
	TenantID               string                       `json:"tenantId"`
	OrganizationID         string                       `json:"organizationId"`
	WarehouseID            string                       `json:"warehouseId"`
	AreaID                 string                       `json:"areaId"`
	LocationID             string                       `json:"locationId"`
	DestinationWarehouseID string                       `json:"destinationWarehouseId"`
	DestinationAreaID      string                       `json:"destinationAreaId"`
	DestinationLocationID  string                       `json:"destinationLocationId"`
	ReturnType             string                       `json:"returnType"`
	Reason                 string                       `json:"reason"`
	ApproverID             string                       `json:"approverId"`
	ApproverName           string                       `json:"approverName"`
	Lines                  []inventoryMovementLineInput `json:"lines"`
}

type inventoryMovementLineInput struct {
	ProductID           string `json:"productId"`
	LotID               string `json:"lotId"`
	Quantity            string `json:"quantity"`
	CountedQuantity     string `json:"countedQuantity"`
	SourceLedgerEntryID string `json:"sourceLedgerEntryId"`
}

type inventoryMovementDecisionInput struct {
	TaskID  string `json:"taskId"`
	Version int64  `json:"version"`
	Comment string `json:"comment"`
}

type stockReturnTotal struct {
	ID                     string
	ReferenceLedgerEntryID string
	ReturnType             string
	quantityMicros         int64
	Version                int64
	scope                  employeeScope
}

type movementProjection struct {
	Balance         stockBalance
	DeltaMicros     int64
	ExpectedVersion *int64
}

type movementReturnReservation struct {
	Leg      inventoryMovementLeg
	Original stockLedgerEntry
}

func inventoryMovementKinds() []string {
	return []string{movementStockTransfer, movementStocktake, movementStockAdjustment, movementStockReturn}
}

func inventoryMovementRoute(kind string) string {
	switch kind {
	case movementStockTransfer:
		return "stock-transfers"
	case movementStocktake:
		return "stocktakes"
	case movementStockAdjustment:
		return "stock-adjustments"
	case movementStockReturn:
		return "stock-returns"
	default:
		return ""
	}
}

func inventoryMovementPrefix(kind string) string {
	switch kind {
	case movementStockTransfer:
		return "ST"
	case movementStocktake:
		return "SC"
	case movementStockAdjustment:
		return "SA"
	case movementStockReturn:
		return "SR"
	default:
		return ""
	}
}

func inventoryMovementRequiresApproval(kind string) bool {
	return kind == movementStocktake || kind == movementStockAdjustment || kind == movementStockReturn
}

func (s *server) listInventoryMovements(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, err := s.requestContext(r)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		kindValue := stringValue(kind)
		filters := []pluginsdk.DataFilter{{
			Field: "movement_type", Operator: pluginsdk.DataOperatorEqual, Value: &kindValue,
		}}
		if status := strings.TrimSpace(r.URL.Query().Get("status")); status != "" {
			if len(status) > 32 {
				writeServiceError(w, newHTTPError(http.StatusBadRequest, "invalid_inventory_movement_filter", "status exceeds its bounded length"))
				return
			}
			statusValue := stringValue(status)
			filters = append(filters, pluginsdk.DataFilter{
				Field: "status", Operator: pluginsdk.DataOperatorEqual, Value: &statusValue,
			})
		}
		items := make([]inventoryMovement, 0)
		err = s.queryInventoryMovements(ctx, movementPermission(kind, "read"), &pluginsdk.DataFilter{All: filters}, func(record pluginsdk.DataRecord) error {
			item, parseErr := inventoryMovementFromRecord(record)
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
}

func (s *server) getInventoryMovementHandler(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, err := s.requestContext(r)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		item, err := s.getInventoryMovement(ctx, movementPermission(kind, "read"), kind, r.PathValue("id"))
		if err != nil {
			writeServiceError(w, err)
			return
		}
		payload := map[string]any{"item": item}
		if item.WorkflowInstanceID != "" {
			workflow, workflowErr := s.host.Workflows.GetInstance(ctx, item.WorkflowInstanceID)
			if workflowErr != nil {
				writeServiceError(w, workflowErr)
				return
			}
			payload["workflow"] = oaWorkflowView(workflow)
		}
		writeOK(w, payload)
	}
}

func (s *server) createInventoryMovement(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		var input inventoryMovementCreateInput
		if !decodeJSON(w, r, &input) {
			return
		}
		requestHash, err := normalizeInventoryMovementInput(kind, &input)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		permission := movementPermission(kind, "create")
		requesterScope, err := s.exactWriteScope(ctx, permission, input.TenantID, input.OrganizationID)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		sourceTopology, err := s.findMovementTopologyWithIntent(
			ctx,
			pluginsdk.DataScopeIntent{Permission: permission},
			input.WarehouseID,
			input.AreaID,
			input.LocationID,
		)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		stockScope := sourceTopology.Warehouse.scope
		if stockScope.TenantID != requesterScope.TenantID ||
			stockScope.OrganizationID != requesterScope.OrganizationID {
			writeServiceError(w, newHTTPError(http.StatusForbidden, "scope_denied", "source inventory is outside the requested organization scope"))
			return
		}
		if err = s.authorizeInventoryMovementStockScope(ctx, permission, stockScope); err != nil {
			writeServiceError(w, err)
			return
		}
		operationKey := inventoryMovementOperationKey(
			kind, "create", stockScope, requesterScope.OwnerID, requestKey,
		)
		unlock := s.movementLocks.lock(operationKey)
		defer unlock()
		if existing, found, findErr := s.findInventoryMovementByCreateOperation(
			ctx, permission, kind, operationKey,
		); findErr != nil {
			writeServiceError(w, findErr)
			return
		} else if found {
			if existing.RequestHash != requestHash {
				writeServiceError(w, newHTTPError(http.StatusConflict, "inventory_movement_idempotency_conflict", "idempotency key was used for a different inventory movement"))
				return
			}
			s.writeInventoryMovementResponse(w, ctx, existing, true, false)
			return
		}

		item := inventoryMovement{
			ID: stableInventoryID(
				"movement", kind, stockScope.TenantID, stockScope.OrganizationID, stockScope.OwnerID, operationKey,
			),
			MovementType: kind, ReturnType: input.ReturnType, Reason: input.Reason,
			SourceWarehouseID: input.WarehouseID, SourceAreaID: input.AreaID, SourceLocationID: input.LocationID,
			DestinationWarehouseID: input.DestinationWarehouseID, DestinationAreaID: input.DestinationAreaID,
			DestinationLocationID: input.DestinationLocationID,
			RequesterID:           requesterScope.OwnerID, ApproverID: input.ApproverID, ApproverName: input.ApproverName,
			CreateOperationKey: operationKey, RequestHash: requestHash,
			LastOperationKey: operationKey, LastOperationHash: requestHash, scope: stockScope,
		}
		item.WorkflowDefinitionID = item.ID + "-definition"
		item.WorkflowInstanceID = item.ID + "-workflow"

		var created inventoryMovement
		duplicate := false
		err = s.transaction(ctx, func(tx context.Context) error {
			now := s.now().UTC()
			number, issueErr := s.host.DocumentNumbers.Issue(tx, pluginsdk.DocumentNumberInput{
				Rule: purchaseDocumentNumberRule(kind, inventoryMovementPrefix(kind)), TenantID: stockScope.TenantID,
				Permission: permission, OccurredAt: now, IdempotencyKey: operationKey,
			})
			if issueErr != nil {
				return issueErr
			}
			if number.Duplicate {
				existing, found, findErr := s.findInventoryMovementByCreateOperation(
					tx, permission, kind, operationKey,
				)
				if findErr != nil {
					return findErr
				}
				if !found || existing.RequestHash != requestHash {
					return newHTTPError(http.StatusConflict, "inventory_movement_idempotency_conflict", "movement number was already issued without a matching request")
				}
				created, duplicate = existing, true
				return nil
			}
			item.Number = number.Number
			item.RequestedAt = now.Format(time.RFC3339Nano)
			if topologyErr := s.resolveInventoryMovementTopologies(
				tx, permission, item,
			); topologyErr != nil {
				return topologyErr
			}
			legs, legErr := s.buildInventoryMovementLegs(
				tx, item, input.Lines, permission, now,
			)
			if legErr != nil {
				return legErr
			}
			item.Legs = legs
			switch kind {
			case movementStockTransfer:
				item.Status, item.PostedAt = movementStatusPosted, item.RequestedAt
				item.WorkflowDefinitionID, item.WorkflowInstanceID = "", ""
			case movementStocktake:
				item.Status = movementStatusCompleted
				for _, leg := range item.Legs {
					if leg.quantityMicros != 0 {
						item.Status = movementStatusPendingApproval
						break
					}
				}
				if item.Status == movementStatusCompleted {
					item.WorkflowDefinitionID, item.WorkflowInstanceID = "", ""
				}
			default:
				item.Status = movementStatusPendingApproval
			}
			if item.Status == movementStatusPendingApproval {
				if _, workflowErr := s.createInventoryMovementWorkflow(tx, item); workflowErr != nil {
					return workflowErr
				}
			}
			result, mutationErr := s.host.DataStore.Mutate(tx, pluginsdk.DataMutation{
				Table: inventoryMovementTable, Operation: pluginsdk.DataMutationInsert,
				Scope:  movementIntent(kind, "create", stockScope),
				Key:    map[string]pluginsdk.DataValue{"id": stringValue(item.ID)},
				Values: inventoryMovementInsertValues(item), Returning: inventoryMovementFields,
				IdempotencyKey: operationKey,
			})
			if mutationErr != nil {
				return mutationErr
			}
			created, mutationErr = inventoryMovementFromMutation(result)
			if mutationErr != nil {
				return mutationErr
			}
			if created.Status == movementStatusPosted {
				posting, postingErr := s.postInventoryMovement(tx, created, permission)
				if postingErr != nil {
					return postingErr
				}
				if eventErr := s.publishMovementInventoryChanged(tx, created, posting); eventErr != nil {
					return eventErr
				}
			}
			return s.audit(tx, "pharma_oa."+kind+".create", created.ID, pluginsdk.AuditRiskHigh, map[string]any{
				"number": created.Number, "status": created.Status, "legCount": len(created.Legs),
				"sourceLocationId": created.SourceLocationID, "destinationLocationId": created.DestinationLocationID,
			})
		})
		if err != nil {
			writeServiceError(w, err)
			return
		}
		s.writeInventoryMovementResponse(w, ctx, created, duplicate, !duplicate)
	}
}

func (s *server) decideInventoryMovement(kind string, approve bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		var input inventoryMovementDecisionInput
		if !decodeJSON(w, r, &input) {
			return
		}
		input.TaskID, input.Comment = strings.TrimSpace(input.TaskID), strings.TrimSpace(input.Comment)
		if input.TaskID == "" || input.Version < 1 || input.Comment == "" || len(input.Comment) > 1000 {
			writeServiceError(w, newHTTPError(http.StatusBadRequest, "invalid_inventory_movement_decision", "taskId, current version, and a bounded comment are required"))
			return
		}
		item, err := s.getInventoryMovement(ctx, movementPermission(kind, action), kind, r.PathValue("id"))
		if err != nil {
			writeServiceError(w, err)
			return
		}
		operationKey := inventoryMovementOperationKey(kind, action, item.scope, item.ID, requestKey)
		operationHash := inventoryMovementDecisionHash(kind, action, item.ID, input)
		unlock := s.movementLocks.lock(item.ID)
		defer unlock()

		var updated inventoryMovement
		duplicate := false
		err = s.transaction(ctx, func(tx context.Context) error {
			current, getErr := s.getInventoryMovement(tx, movementPermission(kind, action), kind, item.ID)
			if getErr != nil {
				return getErr
			}
			terminalStatus := movementStatusRejected
			if approve {
				terminalStatus = movementStatusPosted
			}
			if current.Status != movementStatusPendingApproval {
				if current.Status == terminalStatus && current.LastOperationKey == operationKey && current.LastOperationHash == operationHash {
					updated, duplicate = current, true
					return nil
				}
				return newHTTPError(http.StatusConflict, "inventory_movement_not_actionable", "inventory movement is no longer pending approval")
			}
			if current.Version != input.Version {
				return newHTTPError(http.StatusConflict, "stale_inventory_movement", "inventory movement version is stale")
			}
			if workflowErr := s.validateInventoryMovementWorkflow(tx, current, input.TaskID); workflowErr != nil {
				return workflowErr
			}
			now := s.now().UTC().Format(time.RFC3339Nano)
			if approve {
				claim := current
				claim.Status, claim.LastOperationKey, claim.LastOperationHash = movementStatusPosting, operationKey, operationHash
				claim.DecisionComment, claim.DecidedAt = input.Comment, now
				claimResult, claimErr := s.host.DataStore.Mutate(tx, pluginsdk.DataMutation{
					Table: inventoryMovementTable, Operation: pluginsdk.DataMutationUpdate,
					Scope:  movementIntent(kind, action, current.scope),
					Key:    map[string]pluginsdk.DataValue{"id": stringValue(current.ID)},
					Values: inventoryMovementStateValues(claim), Returning: inventoryMovementFields,
					IdempotencyKey: operationKey + ".claim", ExpectedVersion: &input.Version,
				})
				if claimErr != nil {
					return claimErr
				}
				claim, claimErr = inventoryMovementFromMutation(claimResult)
				if claimErr != nil {
					return claimErr
				}
				workflow, workflowErr := s.host.Workflows.Approve(tx, pluginsdk.WorkflowTaskActionInput{
					InstanceID: current.WorkflowInstanceID, TaskID: input.TaskID, Comment: input.Comment,
				})
				if workflowErr != nil {
					return workflowErr
				}
				if workflow.Status != pluginsdk.WorkflowInstanceApproved {
					return newHTTPError(http.StatusConflict, "inventory_movement_workflow_incomplete", "inventory movement workflow has not reached approval")
				}
				postingItem := current
				postingItem.PostedAt = now
				posting, postingErr := s.postInventoryMovement(tx, postingItem, movementPermission(kind, action))
				if postingErr != nil {
					return postingErr
				}
				claim.Status, claim.PostedAt = movementStatusPosted, now
				claimVersion := claim.Version
				result, mutationErr := s.host.DataStore.Mutate(tx, pluginsdk.DataMutation{
					Table: inventoryMovementTable, Operation: pluginsdk.DataMutationUpdate,
					Scope:  movementIntent(kind, action, current.scope),
					Key:    map[string]pluginsdk.DataValue{"id": stringValue(current.ID)},
					Values: inventoryMovementStateValues(claim), Returning: inventoryMovementFields,
					IdempotencyKey: operationKey + ".complete", ExpectedVersion: &claimVersion,
				})
				if mutationErr != nil {
					return mutationErr
				}
				updated, mutationErr = inventoryMovementFromMutation(result)
				if mutationErr != nil {
					return mutationErr
				}
				if eventErr := s.publishMovementInventoryChanged(tx, updated, posting); eventErr != nil {
					return eventErr
				}
			} else {
				workflow, workflowErr := s.host.Workflows.Reject(tx, pluginsdk.WorkflowTaskActionInput{
					InstanceID: current.WorkflowInstanceID, TaskID: input.TaskID, Comment: input.Comment,
				})
				if workflowErr != nil {
					return workflowErr
				}
				if workflow.Status != pluginsdk.WorkflowInstanceRejected {
					return newHTTPError(http.StatusConflict, "inventory_movement_workflow_incomplete", "inventory movement workflow has not reached rejection")
				}
				current.Status, current.LastOperationKey, current.LastOperationHash = movementStatusRejected, operationKey, operationHash
				current.DecisionComment, current.DecidedAt = input.Comment, now
				result, mutationErr := s.host.DataStore.Mutate(tx, pluginsdk.DataMutation{
					Table: inventoryMovementTable, Operation: pluginsdk.DataMutationUpdate,
					Scope:  movementIntent(kind, action, current.scope),
					Key:    map[string]pluginsdk.DataValue{"id": stringValue(current.ID)},
					Values: inventoryMovementStateValues(current), Returning: inventoryMovementFields,
					IdempotencyKey: operationKey, ExpectedVersion: &input.Version,
				})
				if mutationErr != nil {
					return mutationErr
				}
				updated, mutationErr = inventoryMovementFromMutation(result)
				if mutationErr != nil {
					return mutationErr
				}
			}
			return s.audit(tx, "pharma_oa."+kind+"."+action, current.ID, pluginsdk.AuditRiskHigh, map[string]any{
				"number": current.Number, "status": updated.Status, "comment": input.Comment,
			})
		})
		if err != nil {
			writeServiceError(w, err)
			return
		}
		s.writeInventoryMovementResponse(w, ctx, updated, duplicate, false)
	}
}

func normalizeInventoryMovementInput(kind string, input *inventoryMovementCreateInput) (string, error) {
	if inventoryMovementRoute(kind) == "" {
		return "", newHTTPError(http.StatusNotFound, "inventory_movement_type_not_found", "inventory movement type is not supported")
	}
	input.TenantID, input.OrganizationID = strings.TrimSpace(input.TenantID), strings.TrimSpace(input.OrganizationID)
	input.WarehouseID, input.AreaID, input.LocationID = strings.TrimSpace(input.WarehouseID), strings.TrimSpace(input.AreaID), strings.TrimSpace(input.LocationID)
	input.DestinationWarehouseID = strings.TrimSpace(input.DestinationWarehouseID)
	input.DestinationAreaID = strings.TrimSpace(input.DestinationAreaID)
	input.DestinationLocationID = strings.TrimSpace(input.DestinationLocationID)
	input.ReturnType, input.Reason = strings.ToLower(strings.TrimSpace(input.ReturnType)), strings.TrimSpace(input.Reason)
	input.ApproverID, input.ApproverName = strings.TrimSpace(input.ApproverID), strings.TrimSpace(input.ApproverName)
	if input.TenantID == "" || input.OrganizationID == "" || input.WarehouseID == "" || input.AreaID == "" ||
		input.LocationID == "" || input.Reason == "" || len(input.Reason) > 1000 || len(input.Lines) == 0 || len(input.Lines) > 100 {
		return "", newHTTPError(http.StatusBadRequest, "invalid_inventory_movement", "scope, source topology, bounded reason, and 1 to 100 lines are required")
	}
	if kind == movementStockTransfer {
		if input.DestinationWarehouseID == "" || input.DestinationAreaID == "" || input.DestinationLocationID == "" ||
			input.DestinationLocationID == input.LocationID || input.ReturnType != "" {
			return "", newHTTPError(http.StatusBadRequest, "invalid_stock_transfer", "a different complete destination topology is required")
		}
	} else if input.DestinationWarehouseID != "" || input.DestinationAreaID != "" || input.DestinationLocationID != "" {
		return "", newHTTPError(http.StatusBadRequest, "invalid_inventory_movement", "destination topology is only valid for stock transfers")
	}
	if inventoryMovementRequiresApproval(kind) &&
		(input.ApproverID == "" || input.ApproverName == "" || len(input.ApproverID) > 128 || len(input.ApproverName) > 128) {
		return "", newHTTPError(http.StatusBadRequest, "invalid_inventory_approver", "a bounded movement approver is required")
	}
	if kind == movementStockReturn {
		if input.ReturnType != "purchase" && input.ReturnType != "sales" {
			return "", newHTTPError(http.StatusBadRequest, "invalid_stock_return_type", "returnType must be purchase or sales")
		}
	} else if input.ReturnType != "" {
		return "", newHTTPError(http.StatusBadRequest, "invalid_inventory_movement", "returnType is only valid for stock returns")
	}
	for index := range input.Lines {
		line := &input.Lines[index]
		line.ProductID, line.LotID = strings.TrimSpace(line.ProductID), strings.TrimSpace(line.LotID)
		line.Quantity, line.CountedQuantity = strings.TrimSpace(line.Quantity), strings.TrimSpace(line.CountedQuantity)
		line.SourceLedgerEntryID = strings.TrimSpace(line.SourceLedgerEntryID)
		switch kind {
		case movementStockTransfer:
			if line.ProductID == "" || line.LotID == "" || line.CountedQuantity != "" || line.SourceLedgerEntryID != "" {
				return "", invalidMovementLine(index)
			}
			quantity, err := normalizeMovementQuantity(line.Quantity, false, false)
			if err != nil {
				return "", invalidMovementLine(index)
			}
			line.Quantity = quantity
		case movementStocktake:
			if line.ProductID == "" || line.LotID == "" || line.Quantity != "" || line.SourceLedgerEntryID != "" {
				return "", invalidMovementLine(index)
			}
			quantity, err := normalizeMovementQuantity(line.CountedQuantity, true, false)
			if err != nil {
				return "", invalidMovementLine(index)
			}
			line.CountedQuantity = quantity
		case movementStockAdjustment:
			if line.ProductID == "" || line.LotID == "" || line.CountedQuantity != "" || line.SourceLedgerEntryID != "" {
				return "", invalidMovementLine(index)
			}
			quantity, err := normalizeMovementQuantity(line.Quantity, false, true)
			if err != nil {
				return "", invalidMovementLine(index)
			}
			line.Quantity = quantity
		case movementStockReturn:
			if line.SourceLedgerEntryID == "" || line.ProductID != "" || line.LotID != "" || line.CountedQuantity != "" {
				return "", invalidMovementLine(index)
			}
			quantity, err := normalizeMovementQuantity(line.Quantity, false, false)
			if err != nil {
				return "", invalidMovementLine(index)
			}
			line.Quantity = quantity
		}
	}
	raw, err := json.Marshal(struct {
		Kind  string                       `json:"kind"`
		Input inventoryMovementCreateInput `json:"input"`
	}{Kind: kind, Input: *input})
	if err != nil {
		return "", fmt.Errorf("encode inventory movement fingerprint: %w", err)
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), nil
}

func invalidMovementLine(index int) error {
	return newHTTPError(http.StatusBadRequest, "invalid_inventory_movement_line", fmt.Sprintf("inventory movement line %d is invalid", index+1))
}

func normalizeMovementQuantity(value string, allowZero bool, allowNegative bool) (string, error) {
	quantity, ok := new(big.Rat).SetString(strings.TrimSpace(value))
	if !ok || (!allowNegative && quantity.Sign() < 0) || (!allowZero && quantity.Sign() == 0) {
		return "", fmt.Errorf("quantity is invalid")
	}
	scaled := new(big.Rat).Mul(quantity, big.NewRat(1_000_000, 1))
	if !scaled.IsInt() || !scaled.Num().IsInt64() {
		return "", fmt.Errorf("quantity exceeds exact capacity")
	}
	return microsQuantity(scaled.Num().Int64()), nil
}

func movementQuantityMicros(value string, allowZero bool) (int64, error) {
	normalized, err := normalizeMovementQuantity(value, allowZero, true)
	if err != nil {
		return 0, newHTTPError(http.StatusBadRequest, "invalid_inventory_quantity", "inventory movement quantity is invalid")
	}
	quantity, _ := new(big.Rat).SetString(normalized)
	scaled := new(big.Rat).Mul(quantity, big.NewRat(1_000_000, 1))
	return scaled.Num().Int64(), nil
}

func (s *server) buildInventoryMovementLegs(
	ctx context.Context,
	item inventoryMovement,
	inputs []inventoryMovementLineInput,
	permission pluginsdk.Permission,
	now time.Time,
) ([]inventoryMovementLeg, error) {
	intent := pluginsdk.DataScopeIntent{Permission: permission, Filter: scopeFilter(item.scope)}
	legs := make([]inventoryMovementLeg, 0, len(inputs)*2)
	seen := make(map[string]struct{}, len(inputs))
	for index, input := range inputs {
		groupID := stableInventoryID("movement_group", item.ID, strconv.Itoa(index+1))
		switch item.MovementType {
		case movementStockTransfer, movementStocktake, movementStockAdjustment:
			key := input.ProductID + "\x00" + input.LotID
			if _, duplicate := seen[key]; duplicate {
				return nil, newHTTPError(http.StatusConflict, "duplicate_inventory_movement_line", "inventory movement repeats a product and lot")
			}
			seen[key] = struct{}{}
			lot, found, err := s.findInventoryLotWithIntent(ctx, intent, input.LotID)
			if err != nil {
				return nil, err
			}
			if !found || lot.ProductID != input.ProductID {
				return nil, newHTTPError(http.StatusConflict, "invalid_inventory_lot", "inventory lot does not match the movement product")
			}
			if item.MovementType == movementStockTransfer {
				if err := validateMovementLotExpiry(lot, now, true); err != nil {
					return nil, err
				}
				quantity, _ := movementQuantityMicros(input.Quantity, false)
				legs = append(legs,
					newMovementLeg(item, lot, groupID, "transfer_out", -quantity, item.SourceWarehouseID, item.SourceAreaID, item.SourceLocationID),
					newMovementLeg(item, lot, groupID, "transfer_in", quantity, item.DestinationWarehouseID, item.DestinationAreaID, item.DestinationLocationID),
				)
				continue
			}
			if item.MovementType == movementStocktake {
				counted, _ := movementQuantityMicros(input.CountedQuantity, true)
				balanceID := stableInventoryID("balance", item.scope.TenantID, item.scope.OrganizationID, lot.ProductID, lot.ID, item.SourceLocationID)
				current, found, err := s.findStockBalanceWithIntent(ctx, intent, balanceID)
				if err != nil {
					return nil, err
				}
				system, version := int64(0), int64(0)
				if found {
					system, version = current.quantityMicros, current.Version
				}
				delta, err := checkedAddMicros(counted, -system)
				if err != nil {
					return nil, err
				}
				legType := "stocktake_match"
				if delta > 0 {
					legType = "stocktake_gain"
				} else if delta < 0 {
					legType = "stocktake_loss"
				}
				leg := newMovementLeg(item, lot, groupID, legType, delta, item.SourceWarehouseID, item.SourceAreaID, item.SourceLocationID)
				leg.SystemQuantity, leg.CountedQuantity = microsQuantity(system), microsQuantity(counted)
				leg.ExpectedBalanceVersion = version
				leg.systemQuantityMicros, leg.countedQuantityMicros = system, counted
				legs = append(legs, leg)
				continue
			}
			quantity, _ := movementQuantityMicros(input.Quantity, false)
			if quantity > 0 {
				if err := validateMovementLotExpiry(lot, now, true); err != nil {
					return nil, err
				}
			}
			legType := "adjustment_gain"
			if quantity < 0 {
				legType = "adjustment_loss"
			}
			legs = append(legs, newMovementLeg(
				item, lot, groupID, legType, quantity,
				item.SourceWarehouseID, item.SourceAreaID, item.SourceLocationID,
			))
		case movementStockReturn:
			if _, duplicate := seen[input.SourceLedgerEntryID]; duplicate {
				return nil, newHTTPError(http.StatusConflict, "duplicate_inventory_return_source", "stock return repeats an original ledger entry")
			}
			seen[input.SourceLedgerEntryID] = struct{}{}
			original, found, err := s.findStockLedgerWithIntent(ctx, intent, input.SourceLedgerEntryID)
			if err != nil {
				return nil, err
			}
			if !found {
				return nil, newHTTPError(http.StatusConflict, "invalid_inventory_return_source", "original stock ledger entry was not found")
			}
			quantity, _ := movementQuantityMicros(input.Quantity, false)
			if err := validateReturnOriginal(item.ReturnType, original, quantity); err != nil {
				return nil, err
			}
			lot, found, err := s.findInventoryLotWithIntent(ctx, intent, original.LotID)
			if err != nil {
				return nil, err
			}
			if !found || lot.ProductID != original.ProductID || lot.BatchNo != original.BatchNo {
				return nil, newHTTPError(http.StatusConflict, "invalid_inventory_lot", "return source lot facts are invalid")
			}
			signed := -quantity
			legType := "purchase_return_out"
			if item.ReturnType == "sales" {
				signed, legType = quantity, "sales_return_in"
				if err := validateMovementLotExpiry(lot, now, true); err != nil {
					return nil, err
				}
			}
			leg := newMovementLeg(
				item, lot, groupID, legType, signed,
				item.SourceWarehouseID, item.SourceAreaID, item.SourceLocationID,
			)
			leg.ReferenceLedgerEntryID = original.ID
			legs = append(legs, leg)
		}
	}
	return legs, nil
}

func newMovementLeg(
	item inventoryMovement,
	lot inventoryLot,
	groupID string,
	legType string,
	quantityMicros int64,
	warehouseID string,
	areaID string,
	locationID string,
) inventoryMovementLeg {
	return inventoryMovementLeg{
		ID:      stableInventoryID("movement_leg", item.ID, groupID, legType),
		GroupID: groupID, LegType: legType, ProductID: lot.ProductID, LotID: lot.ID, BatchNo: lot.BatchNo,
		WarehouseID: warehouseID, AreaID: areaID, LocationID: locationID,
		Quantity: microsQuantity(quantityMicros), quantityMicros: quantityMicros,
	}
}

func validateMovementLotExpiry(lot inventoryLot, now time.Time, rejectExpired bool) error {
	expiresAt, err := time.Parse(time.RFC3339Nano, lot.ExpiresAt)
	if err != nil {
		return fmt.Errorf("decode inventory lot expiry: %w", err)
	}
	if rejectExpired && !expiresAt.After(now) {
		return newHTTPError(http.StatusConflict, "expired_inventory_lot", "expired inventory lot is not eligible for this movement")
	}
	return nil
}

func validateReturnOriginal(returnType string, original stockLedgerEntry, quantityMicros int64) error {
	limit, err := absoluteInventoryMicros(original.quantityMicros)
	if err != nil || quantityMicros > limit {
		return newHTTPError(http.StatusConflict, "inventory_return_exceeds_source", "stock return exceeds its original ledger quantity")
	}
	if returnType == "purchase" && (original.EntryType != "receipt" || original.quantityMicros <= 0) {
		return newHTTPError(http.StatusConflict, "invalid_inventory_return_source", "purchase return must reference a positive receipt ledger entry")
	}
	if returnType == "sales" && (original.EntryType != "sale_outbound" || original.quantityMicros >= 0) {
		return newHTTPError(http.StatusConflict, "invalid_inventory_return_source", "sales return must reference a sale outbound ledger entry")
	}
	return nil
}

func absoluteInventoryMicros(value int64) (int64, error) {
	if value == math.MinInt64 {
		return 0, newHTTPError(http.StatusConflict, "inventory_quantity_overflow", "inventory quantity exceeds integer capacity")
	}
	if value < 0 {
		return -value, nil
	}
	return value, nil
}

func (s *server) postInventoryMovement(
	ctx context.Context,
	item inventoryMovement,
	permission pluginsdk.Permission,
) (inventoryPostingSummary, error) {
	if err := s.resolveInventoryMovementTopologies(ctx, permission, item); err != nil {
		return inventoryPostingSummary{}, err
	}
	intent := pluginsdk.DataScopeIntent{Permission: permission, Filter: scopeFilter(item.scope)}
	if err := validateMovementLegContract(item); err != nil {
		return inventoryPostingSummary{}, err
	}

	summary := inventoryPostingSummary{}
	entries := make([]stockLedgerEntry, 0, len(item.Legs))
	projections := make(map[string]movementProjection)
	reservations := make([]movementReturnReservation, 0)
	lots := make(map[string]struct{})
	for _, leg := range item.Legs {
		lot, found, err := s.findInventoryLotWithIntent(ctx, intent, leg.LotID)
		if err != nil {
			return inventoryPostingSummary{}, err
		}
		if !found || lot.ProductID != leg.ProductID || lot.BatchNo != leg.BatchNo {
			return inventoryPostingSummary{}, newHTTPError(http.StatusConflict, "invalid_inventory_lot", "movement leg no longer matches its immutable lot")
		}
		if (item.MovementType == movementStockTransfer || leg.quantityMicros > 0) &&
			item.MovementType != movementStocktake {
			if err := validateMovementLotExpiry(lot, s.now().UTC(), true); err != nil {
				return inventoryPostingSummary{}, err
			}
		}
		balance := stockBalance{
			ID:        stableInventoryID("balance", item.scope.TenantID, item.scope.OrganizationID, leg.ProductID, leg.LotID, leg.LocationID),
			ProductID: leg.ProductID, LotID: leg.LotID, BatchNo: leg.BatchNo,
			WarehouseID: leg.WarehouseID, AreaID: leg.AreaID, LocationID: leg.LocationID, scope: item.scope,
		}
		if item.MovementType == movementStocktake {
			current, found, findErr := s.findStockBalanceWithIntent(ctx, intent, balance.ID)
			if findErr != nil {
				return inventoryPostingSummary{}, findErr
			}
			currentQuantity, currentVersion := int64(0), int64(0)
			if found {
				currentQuantity, currentVersion = current.quantityMicros, current.Version
			}
			if currentQuantity != leg.systemQuantityMicros || currentVersion != leg.ExpectedBalanceVersion {
				return inventoryPostingSummary{}, newHTTPError(http.StatusConflict, "stale_stocktake", "stock balance changed after the count snapshot")
			}
		}
		if item.MovementType == movementStockReturn {
			original, found, findErr := s.findStockLedgerWithIntent(ctx, intent, leg.ReferenceLedgerEntryID)
			if findErr != nil {
				return inventoryPostingSummary{}, findErr
			}
			quantity, quantityErr := absoluteInventoryMicros(leg.quantityMicros)
			if !found || quantityErr != nil {
				return inventoryPostingSummary{}, newHTTPError(http.StatusConflict, "invalid_inventory_return_source", "return source ledger entry is invalid")
			}
			if returnErr := validateReturnOriginal(item.ReturnType, original, quantity); returnErr != nil {
				return inventoryPostingSummary{}, returnErr
			}
			if original.ProductID != leg.ProductID ||
				original.LotID != leg.LotID ||
				original.BatchNo != leg.BatchNo {
				return inventoryPostingSummary{}, newHTTPError(http.StatusConflict, "invalid_inventory_return_source", "return posting facts do not match the original ledger entry")
			}
			reservations = append(reservations, movementReturnReservation{Leg: leg, Original: original})
		}
		lots[leg.LotID] = struct{}{}
		if leg.quantityMicros == 0 {
			continue
		}
		entry := stockLedgerEntry{
			ID:        stableInventoryID("ledger", item.scope.TenantID, item.scope.OrganizationID, item.ID, leg.ID),
			EntryType: leg.LegType, ProductID: leg.ProductID, LotID: leg.LotID, BatchNo: leg.BatchNo,
			WarehouseID: leg.WarehouseID, AreaID: leg.AreaID, LocationID: leg.LocationID,
			SourceDocumentType: item.MovementType, SourceDocumentID: item.ID, SourceDocumentNumber: item.Number,
			SourceDocumentLineID: leg.ID, OccurredAt: item.PostedAt, quantityMicros: leg.quantityMicros, scope: item.scope,
		}
		entries = append(entries, entry)
		projection := projections[balance.ID]
		projection.Balance = balance
		projection.DeltaMicros, err = checkedAddMicros(projection.DeltaMicros, leg.quantityMicros)
		if err != nil {
			return inventoryPostingSummary{}, err
		}
		if item.MovementType == movementStocktake {
			version := leg.ExpectedBalanceVersion
			projection.ExpectedVersion = &version
		}
		projections[balance.ID] = projection
		summary.QuantityMicros, err = checkedAddMicros(summary.QuantityMicros, leg.quantityMicros)
		if err != nil {
			return inventoryPostingSummary{}, err
		}
	}
	sort.Slice(reservations, func(left, right int) bool {
		return reservations[left].Leg.ReferenceLedgerEntryID < reservations[right].Leg.ReferenceLedgerEntryID
	})
	for _, reservation := range reservations {
		if err := s.reserveStockReturn(ctx, intent, item, reservation); err != nil {
			return inventoryPostingSummary{}, err
		}
	}

	projectionIDs := make([]string, 0, len(projections))
	for id, projection := range projections {
		if projection.DeltaMicros != 0 {
			projectionIDs = append(projectionIDs, id)
		}
	}
	sort.Strings(projectionIDs)
	for _, id := range projectionIDs {
		projection := projections[id]
		if projection.DeltaMicros < 0 {
			current, found, err := s.findStockBalanceWithIntent(ctx, intent, id)
			if err != nil {
				return inventoryPostingSummary{}, err
			}
			required, quantityErr := absoluteInventoryMicros(projection.DeltaMicros)
			if quantityErr != nil || !found || current.quantityMicros < required {
				return inventoryPostingSummary{}, newHTTPError(http.StatusConflict, "insufficient_stock", "inventory movement would create negative stock")
			}
		}
		if _, err := s.projectStockBalanceWithIntent(
			ctx, intent, projection.Balance, projection.DeltaMicros,
			stableInventoryID("movement_projection", item.ID, id), projection.ExpectedVersion,
		); err != nil {
			return inventoryPostingSummary{}, inventoryProjectionError(err, projection)
		}
	}
	sort.Slice(entries, func(left, right int) bool { return entries[left].ID < entries[right].ID })
	for _, entry := range entries {
		if _, err := s.appendStockLedgerWithIntent(ctx, intent, entry); err != nil {
			return inventoryPostingSummary{}, err
		}
	}
	summary.LotCount, summary.LedgerEntryCount = len(lots), len(entries)
	return summary, nil
}

func validateMovementLegContract(item inventoryMovement) error {
	if len(item.Legs) == 0 {
		return newHTTPError(http.StatusConflict, "invalid_inventory_movement_legs", "inventory movement has no posting legs")
	}
	type transferGroup struct {
		ProductID string
		LotID     string
		Count     int
		OutCount  int
		InCount   int
		Net       int64
	}
	groups := make(map[string]transferGroup)
	for _, leg := range item.Legs {
		if leg.ID == "" || leg.GroupID == "" || leg.ProductID == "" || leg.LotID == "" || leg.BatchNo == "" ||
			leg.WarehouseID == "" || leg.AreaID == "" || leg.LocationID == "" {
			return newHTTPError(http.StatusConflict, "invalid_inventory_movement_legs", "inventory movement leg identity is incomplete")
		}
		if item.MovementType != movementStockTransfer &&
			(leg.WarehouseID != item.SourceWarehouseID ||
				leg.AreaID != item.SourceAreaID ||
				leg.LocationID != item.SourceLocationID) {
			return newHTTPError(http.StatusConflict, "invalid_inventory_movement_legs", "inventory movement leg is outside its source topology")
		}
		switch item.MovementType {
		case movementStockTransfer:
			if leg.LegType != "transfer_out" && leg.LegType != "transfer_in" {
				return newHTTPError(http.StatusConflict, "invalid_inventory_movement_legs", "stock transfer has an invalid leg type")
			}
			if leg.LegType == "transfer_out" {
				if leg.quantityMicros >= 0 ||
					leg.WarehouseID != item.SourceWarehouseID ||
					leg.AreaID != item.SourceAreaID ||
					leg.LocationID != item.SourceLocationID {
					return newHTTPError(http.StatusConflict, "unbalanced_stock_transfer", "stock transfer source leg is invalid")
				}
			} else if leg.quantityMicros <= 0 ||
				leg.WarehouseID != item.DestinationWarehouseID ||
				leg.AreaID != item.DestinationAreaID ||
				leg.LocationID != item.DestinationLocationID {
				return newHTTPError(http.StatusConflict, "unbalanced_stock_transfer", "stock transfer destination leg is invalid")
			}
			group := groups[leg.GroupID]
			if group.Count > 0 && (group.ProductID != leg.ProductID || group.LotID != leg.LotID) {
				return newHTTPError(http.StatusConflict, "unbalanced_stock_transfer", "stock transfer legs do not share one product and lot")
			}
			group.ProductID, group.LotID, group.Count = leg.ProductID, leg.LotID, group.Count+1
			if leg.LegType == "transfer_out" {
				group.OutCount++
			} else {
				group.InCount++
			}
			var err error
			group.Net, err = checkedAddMicros(group.Net, leg.quantityMicros)
			if err != nil {
				return err
			}
			groups[leg.GroupID] = group
		case movementStocktake:
			if leg.LegType != "stocktake_match" && leg.LegType != "stocktake_gain" && leg.LegType != "stocktake_loss" {
				return newHTTPError(http.StatusConflict, "invalid_inventory_movement_legs", "stocktake has an invalid leg type")
			}
			delta, err := checkedAddMicros(leg.countedQuantityMicros, -leg.systemQuantityMicros)
			if err != nil ||
				leg.SystemQuantity == "" ||
				leg.CountedQuantity == "" ||
				leg.systemQuantityMicros < 0 ||
				leg.countedQuantityMicros < 0 ||
				leg.ExpectedBalanceVersion < 0 ||
				delta != leg.quantityMicros {
				return newHTTPError(http.StatusConflict, "invalid_inventory_movement_legs", "stocktake snapshot facts are invalid")
			}
			if (leg.LegType == "stocktake_match" && leg.quantityMicros != 0) ||
				(leg.LegType == "stocktake_gain" && leg.quantityMicros <= 0) ||
				(leg.LegType == "stocktake_loss" && leg.quantityMicros >= 0) {
				return newHTTPError(http.StatusConflict, "invalid_inventory_movement_legs", "stocktake leg quantity does not match its type")
			}
		case movementStockAdjustment:
			if leg.LegType != "adjustment_gain" && leg.LegType != "adjustment_loss" {
				return newHTTPError(http.StatusConflict, "invalid_inventory_movement_legs", "stock adjustment has an invalid leg type")
			}
			if (leg.LegType == "adjustment_gain" && leg.quantityMicros <= 0) ||
				(leg.LegType == "adjustment_loss" && leg.quantityMicros >= 0) {
				return newHTTPError(http.StatusConflict, "invalid_inventory_movement_legs", "stock adjustment leg quantity does not match its type")
			}
		case movementStockReturn:
			if (item.ReturnType != "purchase" && item.ReturnType != "sales") ||
				(item.ReturnType == "purchase" && leg.LegType != "purchase_return_out") ||
				(item.ReturnType == "sales" && leg.LegType != "sales_return_in") ||
				leg.ReferenceLedgerEntryID == "" {
				return newHTTPError(http.StatusConflict, "invalid_inventory_movement_legs", "stock return has an invalid source leg")
			}
			if (leg.LegType == "purchase_return_out" && leg.quantityMicros >= 0) ||
				(leg.LegType == "sales_return_in" && leg.quantityMicros <= 0) {
				return newHTTPError(http.StatusConflict, "invalid_inventory_movement_legs", "stock return leg quantity does not match its type")
			}
		}
	}
	for _, group := range groups {
		if group.Count != 2 || group.OutCount != 1 || group.InCount != 1 || group.Net != 0 {
			return newHTTPError(http.StatusConflict, "unbalanced_stock_transfer", "each stock transfer group must have one balanced source and destination leg")
		}
	}
	return nil
}

func (s *server) reserveStockReturn(
	ctx context.Context,
	intent pluginsdk.DataScopeIntent,
	item inventoryMovement,
	reservation movementReturnReservation,
) error {
	id := stableInventoryID(
		"return_total", item.scope.TenantID, item.scope.OrganizationID,
		reservation.Leg.ReferenceLedgerEntryID, item.ReturnType,
	)
	total, found, err := s.findStockReturnTotal(ctx, intent, id)
	if err != nil {
		return err
	}
	if !found {
		result, insertErr := s.host.DataStore.Mutate(ctx, pluginsdk.DataMutation{
			Table: stockReturnTotalTable, Operation: pluginsdk.DataMutationInsert, Scope: intent,
			Key: map[string]pluginsdk.DataValue{"id": stringValue(id)},
			Values: map[string]pluginsdk.DataValue{
				"reference_ledger_entry_id": stringValue(reservation.Leg.ReferenceLedgerEntryID),
				"return_type":               stringValue(item.ReturnType),
				"quantity_micros":           integerValue(0),
			},
			Returning: stockReturnTotalFields, IdempotencyKey: id + ".open",
		})
		if insertErr != nil {
			return insertErr
		}
		total, err = stockReturnTotalFromMutation(result)
		if err != nil {
			return err
		}
	}
	if total.ReferenceLedgerEntryID != reservation.Leg.ReferenceLedgerEntryID || total.ReturnType != item.ReturnType {
		return newHTTPError(http.StatusConflict, "stock_return_total_identity_conflict", "stock return total identity is invalid")
	}
	maximumMicros, err := absoluteInventoryMicros(reservation.Original.quantityMicros)
	if err != nil {
		return err
	}
	quantityMicros, err := absoluteInventoryMicros(reservation.Leg.quantityMicros)
	if err != nil {
		return err
	}
	minimum, maximum := integerValue(0), integerValue(maximumMicros)
	_, err = s.host.DataStore.Mutate(ctx, pluginsdk.DataMutation{
		Table: stockReturnTotalTable, Operation: pluginsdk.DataMutationAdjust, Scope: intent,
		Key: map[string]pluginsdk.DataValue{"id": stringValue(id)},
		Adjustment: &pluginsdk.DataAdjustment{
			Field: "quantity_micros", Delta: integerValue(quantityMicros), Minimum: &minimum, Maximum: &maximum,
		},
		Returning:      stockReturnTotalFields,
		IdempotencyKey: stableInventoryID("return_reservation", item.ID, reservation.Leg.ID),
	})
	if dataStoreConflictIs(err, pluginsdk.DataAdjustmentGuardConflictField) {
		return newHTTPError(http.StatusConflict, "inventory_return_exceeds_source", "stock return exceeds its original ledger quantity")
	}
	return err
}

func inventoryProjectionError(err error, projection movementProjection) error {
	var transport *httpError
	if projection.ExpectedVersion != nil &&
		errors.As(err, &transport) &&
		transport.code == "stale_stock_balance" {
		return newHTTPError(http.StatusConflict, "stale_stocktake", "stock balance changed after the count snapshot")
	}
	if projection.ExpectedVersion != nil && dataStoreConflictIs(err, "expectedVersion") {
		return newHTTPError(http.StatusConflict, "stale_stocktake", "stock balance changed after the count snapshot")
	}
	if dataStoreConflictIs(err, pluginsdk.DataAdjustmentGuardConflictField) {
		if projection.DeltaMicros < 0 {
			return newHTTPError(http.StatusConflict, "insufficient_stock", "inventory movement would create negative stock")
		}
		return newHTTPError(http.StatusConflict, "inventory_quantity_overflow", "inventory movement exceeds quantity capacity")
	}
	return err
}

func dataStoreConflictIs(err error, field string) bool {
	var dataErr *pluginsdk.DataStoreError
	return errors.As(err, &dataErr) &&
		dataErr.Code == pluginsdk.DataStoreErrorConflict &&
		dataErr.Field == field
}

func (s *server) resolveInventoryMovementTopologies(
	ctx context.Context,
	permission pluginsdk.Permission,
	item inventoryMovement,
) error {
	if _, err := s.resolveMovementTopology(
		ctx, permission, item.scope, item.SourceWarehouseID, item.SourceAreaID, item.SourceLocationID,
	); err != nil {
		return err
	}
	if item.MovementType == movementStockTransfer {
		if item.DestinationLocationID == item.SourceLocationID {
			return newHTTPError(http.StatusConflict, "invalid_stock_transfer_destination", "stock transfer destination must differ from its source")
		}
		if _, err := s.resolveMovementTopology(
			ctx, permission, item.scope,
			item.DestinationWarehouseID, item.DestinationAreaID, item.DestinationLocationID,
		); err != nil {
			return err
		}
	}
	return nil
}

func (s *server) createInventoryMovementWorkflow(
	ctx context.Context,
	item inventoryMovement,
) (pluginsdk.WorkflowInstance, error) {
	if _, err := s.host.Workflows.CreateDefinition(ctx, pluginsdk.WorkflowDefinitionInput{
		ID: item.WorkflowDefinitionID, Key: item.MovementType + "-" + item.ID,
		Name: item.Number + " inventory movement approval", Version: 1,
		Nodes: []pluginsdk.WorkflowNode{
			{ID: "start", Key: "start", Name: "Start", Type: pluginsdk.WorkflowNodeStart},
			{
				ID: "approval", Key: "approval", Name: "Inventory movement approval",
				Type: pluginsdk.WorkflowNodeApproval, AssigneeIDs: []string{item.ApproverID},
				Decision: &pluginsdk.WorkflowDecisionRule{Strategy: pluginsdk.WorkflowDecisionAny, Quorum: 1},
			},
			{ID: "end", Key: "end", Name: "Complete", Type: pluginsdk.WorkflowNodeEnd},
		},
		Transitions: []pluginsdk.WorkflowTransition{{From: "start", To: "approval"}, {From: "approval", To: "end"}},
	}); err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	if _, err := s.host.Workflows.PublishDefinition(ctx, item.WorkflowDefinitionID); err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	return s.host.Workflows.Start(ctx, pluginsdk.WorkflowStartInput{
		ID: item.WorkflowInstanceID, DefinitionID: item.WorkflowDefinitionID,
		BusinessType: item.MovementType, BusinessID: item.ID, Title: item.Number,
	})
}

func (s *server) validateInventoryMovementWorkflow(
	ctx context.Context,
	item inventoryMovement,
	taskID string,
) error {
	instance, err := s.host.Workflows.GetInstance(ctx, item.WorkflowInstanceID)
	if err != nil {
		return err
	}
	if instance.ID != item.WorkflowInstanceID ||
		instance.DefinitionID != item.WorkflowDefinitionID ||
		instance.BusinessType != item.MovementType ||
		instance.BusinessID != item.ID ||
		instance.Status != pluginsdk.WorkflowInstanceRunning {
		return newHTTPError(http.StatusConflict, "invalid_inventory_movement_workflow", "inventory movement workflow binding is invalid")
	}
	for _, task := range instance.Tasks {
		if task.ID == taskID &&
			task.InstanceID == instance.ID &&
			task.Status == pluginsdk.WorkflowTaskPending {
			return nil
		}
	}
	return newHTTPError(http.StatusConflict, "invalid_inventory_movement_workflow_task", "inventory movement workflow task is not pending")
}

func (s *server) publishMovementInventoryChanged(
	ctx context.Context,
	item inventoryMovement,
	summary inventoryPostingSummary,
) error {
	_, err := s.host.Events.Publish(ctx, pluginsdk.EventPublication{
		IdempotencyKey: item.ID + ".inventory", Name: "inventory-changed", SchemaVersion: 1,
		Scope: pluginsdk.EventScope{
			TenantID: item.scope.TenantID, OrganizationID: item.scope.OrganizationID, OwnerID: item.scope.OwnerID,
		},
		CorrelationID: item.ID,
		Subject:       pluginsdk.EventSubject{Type: item.MovementType, ID: item.ID},
		Payload: pluginsdk.EventPayload{
			"source_document_type": {Type: pluginsdk.DataValueString, Value: item.MovementType},
			"source_document_id":   {Type: pluginsdk.DataValueString, Value: item.ID},
			"warehouse_id":         {Type: pluginsdk.DataValueString, Value: item.SourceWarehouseID},
			"location_id":          {Type: pluginsdk.DataValueString, Value: item.SourceLocationID},
			"ledger_entry_count":   integerValue(int64(summary.LedgerEntryCount)),
			"quantity_micros":      integerValue(summary.QuantityMicros),
		},
	})
	return err
}

func (s *server) writeInventoryMovementResponse(
	w http.ResponseWriter,
	ctx context.Context,
	item inventoryMovement,
	duplicate bool,
	created bool,
) {
	payload := map[string]any{"item": item, "duplicate": duplicate}
	if item.WorkflowInstanceID != "" {
		workflow, err := s.host.Workflows.GetInstance(ctx, item.WorkflowInstanceID)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		payload["workflow"] = oaWorkflowView(workflow)
	}
	if created {
		writeCreated(w, payload)
		return
	}
	writeOK(w, payload)
}

func (s *server) queryInventoryMovements(
	ctx context.Context,
	permission pluginsdk.Permission,
	filter *pluginsdk.DataFilter,
	appendRecord func(pluginsdk.DataRecord) error,
) error {
	cursor := ""
	for pageNumber := 0; pageNumber < 25; pageNumber++ {
		page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{
			Table: inventoryMovementTable, Fields: inventoryMovementFields,
			Scope: pluginsdk.DataScopeIntent{Permission: permission}, Filter: filter,
			Sort: []pluginsdk.DataSort{{Field: "requested_at", Direction: pluginsdk.DataSortDescending}},
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
		if page.NextCursor == "" || page.NextCursor == cursor {
			return fmt.Errorf("inventory movement pagination did not advance")
		}
		cursor = page.NextCursor
	}
	return pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorLimitExceeded, inventoryMovementTable, "inventory movement query exceeds 5000 records", false)
}

func (s *server) getInventoryMovement(
	ctx context.Context,
	permission pluginsdk.Permission,
	kind string,
	id string,
) (inventoryMovement, error) {
	idValue, kindValue := stringValue(strings.TrimSpace(id)), stringValue(kind)
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{
		Table: inventoryMovementTable, Fields: inventoryMovementFields,
		Scope: pluginsdk.DataScopeIntent{Permission: permission},
		Filter: &pluginsdk.DataFilter{All: []pluginsdk.DataFilter{
			{Field: "id", Operator: pluginsdk.DataOperatorEqual, Value: &idValue},
			{Field: "movement_type", Operator: pluginsdk.DataOperatorEqual, Value: &kindValue},
		}},
		Sort: []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}},
		Page: pluginsdk.DataPageRequest{Limit: 1},
	})
	if err != nil {
		return inventoryMovement{}, err
	}
	if len(page.Records) == 0 {
		return inventoryMovement{}, newHTTPError(http.StatusNotFound, "inventory_movement_not_found", "inventory movement was not found")
	}
	return inventoryMovementFromRecord(page.Records[0])
}

func (s *server) findInventoryMovementByCreateOperation(
	ctx context.Context,
	permission pluginsdk.Permission,
	kind string,
	operationKey string,
) (inventoryMovement, bool, error) {
	keyValue, kindValue := stringValue(operationKey), stringValue(kind)
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{
		Table: inventoryMovementTable, Fields: inventoryMovementFields,
		Scope: pluginsdk.DataScopeIntent{Permission: permission},
		Filter: &pluginsdk.DataFilter{All: []pluginsdk.DataFilter{
			{Field: "create_operation_key", Operator: pluginsdk.DataOperatorEqual, Value: &keyValue},
			{Field: "movement_type", Operator: pluginsdk.DataOperatorEqual, Value: &kindValue},
		}},
		Sort: []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}},
		Page: pluginsdk.DataPageRequest{Limit: 1},
	})
	if err != nil || len(page.Records) == 0 {
		return inventoryMovement{}, false, err
	}
	item, err := inventoryMovementFromRecord(page.Records[0])
	return item, err == nil, err
}

func (s *server) findStockLedgerWithIntent(
	ctx context.Context,
	intent pluginsdk.DataScopeIntent,
	id string,
) (stockLedgerEntry, bool, error) {
	value := stringValue(strings.TrimSpace(id))
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{
		Table: stockLedgerTable, Fields: stockLedgerFields, Scope: intent,
		Filter: &pluginsdk.DataFilter{Field: "id", Operator: pluginsdk.DataOperatorEqual, Value: &value},
		Sort:   []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}},
		Page:   pluginsdk.DataPageRequest{Limit: 1},
	})
	if err != nil || len(page.Records) == 0 {
		return stockLedgerEntry{}, false, err
	}
	item, err := stockLedgerFromRecord(page.Records[0])
	return item, err == nil, err
}

func (s *server) findStockReturnTotal(
	ctx context.Context,
	intent pluginsdk.DataScopeIntent,
	id string,
) (stockReturnTotal, bool, error) {
	value := stringValue(id)
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{
		Table: stockReturnTotalTable, Fields: stockReturnTotalFields, Scope: intent,
		Filter: &pluginsdk.DataFilter{Field: "id", Operator: pluginsdk.DataOperatorEqual, Value: &value},
		Sort:   []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}},
		Page:   pluginsdk.DataPageRequest{Limit: 1},
	})
	if err != nil || len(page.Records) == 0 {
		return stockReturnTotal{}, false, err
	}
	item, err := stockReturnTotalFromRecord(page.Records[0])
	return item, err == nil, err
}

func inventoryMovementInsertValues(item inventoryMovement) map[string]pluginsdk.DataValue {
	return map[string]pluginsdk.DataValue{
		"number": stringValue(item.Number), "movement_type": stringValue(item.MovementType),
		"return_type": nullableStringValue(item.ReturnType), "reason": stringValue(item.Reason),
		"source_warehouse_id": stringValue(item.SourceWarehouseID), "source_area_id": stringValue(item.SourceAreaID),
		"source_location_id":       stringValue(item.SourceLocationID),
		"destination_warehouse_id": nullableStringValue(item.DestinationWarehouseID),
		"destination_area_id":      nullableStringValue(item.DestinationAreaID),
		"destination_location_id":  nullableStringValue(item.DestinationLocationID),
		"legs":                     jsonValue(item.Legs), "requester_id": stringValue(item.RequesterID),
		"approver_id": nullableStringValue(item.ApproverID), "approver_name": nullableStringValue(item.ApproverName),
		"workflow_definition_id": nullableStringValue(item.WorkflowDefinitionID),
		"workflow_instance_id":   nullableStringValue(item.WorkflowInstanceID), "status": stringValue(item.Status),
		"create_operation_key": stringValue(item.CreateOperationKey), "request_hash": stringValue(item.RequestHash),
		"last_operation_key": stringValue(item.LastOperationKey), "last_operation_hash": stringValue(item.LastOperationHash),
		"decision_comment": nullableStringValue(item.DecisionComment), "decided_at": nullableTimestampValue(item.DecidedAt),
		"posted_at": nullableTimestampValue(item.PostedAt), "requested_at": timestampValue(item.RequestedAt),
	}
}

func inventoryMovementStateValues(item inventoryMovement) map[string]pluginsdk.DataValue {
	return map[string]pluginsdk.DataValue{
		"status": stringValue(item.Status), "last_operation_key": stringValue(item.LastOperationKey),
		"last_operation_hash": stringValue(item.LastOperationHash),
		"decision_comment":    nullableStringValue(item.DecisionComment),
		"decided_at":          nullableTimestampValue(item.DecidedAt), "posted_at": nullableTimestampValue(item.PostedAt),
	}
}

func inventoryMovementFromMutation(result pluginsdk.DataMutationResult) (inventoryMovement, error) {
	if result.Record == nil {
		return inventoryMovement{}, fmt.Errorf("inventory movement mutation returned no record")
	}
	return inventoryMovementFromRecord(*result.Record)
}

func inventoryMovementFromRecord(record pluginsdk.DataRecord) (inventoryMovement, error) {
	item := inventoryMovement{
		ID: dataString(record, "id"), Number: dataString(record, "number"),
		MovementType: dataString(record, "movement_type"), ReturnType: dataString(record, "return_type"),
		Reason:            dataString(record, "reason"),
		SourceWarehouseID: dataString(record, "source_warehouse_id"), SourceAreaID: dataString(record, "source_area_id"),
		SourceLocationID:       dataString(record, "source_location_id"),
		DestinationWarehouseID: dataString(record, "destination_warehouse_id"),
		DestinationAreaID:      dataString(record, "destination_area_id"),
		DestinationLocationID:  dataString(record, "destination_location_id"),
		RequesterID:            dataString(record, "requester_id"), ApproverID: dataString(record, "approver_id"),
		ApproverName:         dataString(record, "approver_name"),
		WorkflowDefinitionID: dataString(record, "workflow_definition_id"),
		WorkflowInstanceID:   dataString(record, "workflow_instance_id"), Status: dataString(record, "status"),
		CreateOperationKey: dataString(record, "create_operation_key"), RequestHash: dataString(record, "request_hash"),
		LastOperationKey: dataString(record, "last_operation_key"), LastOperationHash: dataString(record, "last_operation_hash"),
		DecisionComment: dataString(record, "decision_comment"), DecidedAt: dataString(record, "decided_at"),
		PostedAt: dataString(record, "posted_at"), RequestedAt: dataString(record, "requested_at"),
		Version: record.Version, CreatedAt: dataString(record, "created_at"), UpdatedAt: dataString(record, "updated_at"),
		scope: recordScope(record), Legs: []inventoryMovementLeg{},
	}
	if raw := dataString(record, "legs"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &item.Legs); err != nil {
			return inventoryMovement{}, fmt.Errorf("decode inventory movement legs: %w", err)
		}
	}
	for index := range item.Legs {
		quantity, err := movementQuantityMicros(item.Legs[index].Quantity, true)
		if err != nil {
			return inventoryMovement{}, err
		}
		item.Legs[index].quantityMicros = quantity
		if item.Legs[index].SystemQuantity != "" {
			system, parseErr := movementQuantityMicros(item.Legs[index].SystemQuantity, true)
			if parseErr != nil {
				return inventoryMovement{}, parseErr
			}
			item.Legs[index].systemQuantityMicros = system
		}
		if item.Legs[index].CountedQuantity != "" {
			counted, parseErr := movementQuantityMicros(item.Legs[index].CountedQuantity, true)
			if parseErr != nil {
				return inventoryMovement{}, parseErr
			}
			item.Legs[index].countedQuantityMicros = counted
		}
	}
	if item.ID == "" || item.Number == "" || inventoryMovementRoute(item.MovementType) == "" || item.Reason == "" ||
		item.SourceWarehouseID == "" || item.SourceAreaID == "" || item.SourceLocationID == "" ||
		item.RequesterID == "" || item.Status == "" || item.CreateOperationKey == "" || item.RequestHash == "" ||
		item.RequestedAt == "" || len(item.Legs) == 0 {
		return inventoryMovement{}, fmt.Errorf("inventory movement record is incomplete")
	}
	return item, nil
}

func stockReturnTotalFromMutation(result pluginsdk.DataMutationResult) (stockReturnTotal, error) {
	if result.Record == nil {
		return stockReturnTotal{}, fmt.Errorf("stock return total mutation returned no record")
	}
	return stockReturnTotalFromRecord(*result.Record)
}

func stockReturnTotalFromRecord(record pluginsdk.DataRecord) (stockReturnTotal, error) {
	quantity, err := recordMicros(record, "quantity_micros")
	if err != nil {
		return stockReturnTotal{}, err
	}
	item := stockReturnTotal{
		ID: dataString(record, "id"), ReferenceLedgerEntryID: dataString(record, "reference_ledger_entry_id"),
		ReturnType: dataString(record, "return_type"), quantityMicros: quantity,
		Version: record.Version, scope: recordScope(record),
	}
	if item.ID == "" || item.ReferenceLedgerEntryID == "" || (item.ReturnType != "purchase" && item.ReturnType != "sales") || quantity < 0 {
		return stockReturnTotal{}, fmt.Errorf("stock return total record is incomplete")
	}
	return item, nil
}

func inventoryMovementOperationKey(
	kind string,
	action string,
	scope employeeScope,
	id string,
	requestKey string,
) string {
	return stableInventoryID(
		"movement_"+kind+"_"+action,
		scope.TenantID, scope.OrganizationID, id, strings.TrimSpace(requestKey),
	)
}

func inventoryMovementDecisionHash(
	kind string,
	action string,
	id string,
	input inventoryMovementDecisionInput,
) string {
	raw, _ := json.Marshal(struct {
		Kind   string                         `json:"kind"`
		Action string                         `json:"action"`
		ID     string                         `json:"id"`
		Input  inventoryMovementDecisionInput `json:"input"`
	}{Kind: kind, Action: action, ID: id, Input: input})
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

func movementPermission(kind string, action string) pluginsdk.Permission {
	return pluginsdk.Permission{Resource: "pharma_oa." + kind, Action: action}
}

func (s *server) authorizeInventoryMovementStockScope(
	ctx context.Context,
	permission pluginsdk.Permission,
	scope employeeScope,
) error {
	predicate, err := s.host.DataScopes.Resolve(ctx, permission)
	if err != nil {
		return err
	}
	if predicate.Constrain(scopeFilter(scope)).Denied() {
		return newHTTPError(http.StatusForbidden, "scope_denied", "source inventory owner scope is not allowed")
	}
	return nil
}

func movementIntent(kind string, action string, scope employeeScope) pluginsdk.DataScopeIntent {
	return pluginsdk.DataScopeIntent{Permission: movementPermission(kind, action), Filter: scopeFilter(scope)}
}
