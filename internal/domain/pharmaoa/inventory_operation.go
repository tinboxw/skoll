package pharmaoa

import (
	"fmt"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

type StocktakeOrderStatus string
type TransferOrderStatus string

const (
	StocktakePendingApproval StocktakeOrderStatus = "pending_approval"
	StocktakeApproved        StocktakeOrderStatus = "approved"
	StocktakeRejected        StocktakeOrderStatus = "rejected"
	TransferCompleted        TransferOrderStatus  = "completed"
)

type StocktakeOrder struct {
	ID                 shared.ID            `json:"id"`
	Number             string               `json:"number"`
	ProductID          string               `json:"productId"`
	WarehouseID        string               `json:"warehouseId"`
	AreaID             string               `json:"areaId"`
	LocationID         string               `json:"locationId"`
	BatchID            string               `json:"batchId"`
	SystemQuantity     int                  `json:"systemQuantity"`
	ActualQuantity     int                  `json:"actualQuantity"`
	Difference         int                  `json:"difference"`
	Reason             string               `json:"reason"`
	ApproverID         string               `json:"approverId"`
	WorkflowInstanceID string               `json:"workflowInstanceId"`
	LedgerID           string               `json:"ledgerId,omitempty"`
	Status             StocktakeOrderStatus `json:"status"`
	CreatedBy          string               `json:"createdBy"`
	ApprovedBy         string               `json:"approvedBy,omitempty"`
	CreatedAt          time.Time            `json:"createdAt"`
	CompletedAt        *time.Time           `json:"completedAt,omitempty"`
	Meta               shared.AuditMeta     `json:"meta"`
}

type TransferOrder struct {
	ID              shared.ID           `json:"id"`
	Number          string              `json:"number"`
	ProductID       string              `json:"productId"`
	BatchID         string              `json:"batchId"`
	Quantity        int                 `json:"quantity"`
	FromWarehouseID string              `json:"fromWarehouseId"`
	FromAreaID      string              `json:"fromAreaId"`
	FromLocationID  string              `json:"fromLocationId"`
	ToWarehouseID   string              `json:"toWarehouseId"`
	ToAreaID        string              `json:"toAreaId"`
	ToLocationID    string              `json:"toLocationId"`
	OutLedgerID     string              `json:"outLedgerId"`
	InLedgerID      string              `json:"inLedgerId"`
	Status          TransferOrderStatus `json:"status"`
	TransferredBy   string              `json:"transferredBy"`
	TransferredAt   time.Time           `json:"transferredAt"`
	Meta            shared.AuditMeta    `json:"meta"`
}

func NewStocktakeOrder(id shared.ID, number string, position StockPosition, systemQuantity, actualQuantity int, reason, creatorID, approverID, workflowID string, now time.Time) (*StocktakeOrder, error) {
	if id.IsZero() || strings.TrimSpace(number) == "" || strings.TrimSpace(position.ProductID) == "" || strings.TrimSpace(position.WarehouseID) == "" || strings.TrimSpace(position.AreaID) == "" || strings.TrimSpace(position.LocationID) == "" || strings.TrimSpace(position.BatchID) == "" || strings.TrimSpace(creatorID) == "" || strings.TrimSpace(approverID) == "" || strings.TrimSpace(workflowID) == "" {
		return nil, fmt.Errorf("stocktake order input is incomplete")
	}
	if actualQuantity < 0 || actualQuantity == systemQuantity {
		return nil, fmt.Errorf("stocktake order requires a non-zero difference")
	}
	now = normalizeInventoryOperationTime(now)
	item := &StocktakeOrder{ID: id, Number: strings.TrimSpace(number), ProductID: strings.TrimSpace(position.ProductID), WarehouseID: strings.TrimSpace(position.WarehouseID), AreaID: strings.TrimSpace(position.AreaID), LocationID: strings.TrimSpace(position.LocationID), BatchID: strings.TrimSpace(position.BatchID), SystemQuantity: systemQuantity, ActualQuantity: actualQuantity, Difference: actualQuantity - systemQuantity, Reason: strings.TrimSpace(reason), ApproverID: strings.TrimSpace(approverID), WorkflowInstanceID: strings.TrimSpace(workflowID), Status: StocktakePendingApproval, CreatedBy: strings.TrimSpace(creatorID), CreatedAt: now}
	item.Meta.Touch(now)
	return item, nil
}

func (o *StocktakeOrder) Approve(actorID, ledgerID string, now time.Time) error {
	if o == nil || o.Status != StocktakePendingApproval || strings.TrimSpace(actorID) == "" || strings.TrimSpace(ledgerID) == "" {
		return fmt.Errorf("stocktake approval input is invalid")
	}
	now = normalizeInventoryOperationTime(now)
	o.Status = StocktakeApproved
	o.ApprovedBy = strings.TrimSpace(actorID)
	o.LedgerID = strings.TrimSpace(ledgerID)
	o.CompletedAt = &now
	o.Meta.Touch(now)
	return nil
}

func (o *StocktakeOrder) Reject(actorID string, now time.Time) error {
	if o == nil || o.Status != StocktakePendingApproval || strings.TrimSpace(actorID) == "" {
		return fmt.Errorf("stocktake rejection input is invalid")
	}
	now = normalizeInventoryOperationTime(now)
	o.Status = StocktakeRejected
	o.ApprovedBy = strings.TrimSpace(actorID)
	o.CompletedAt = &now
	o.Meta.Touch(now)
	return nil
}

func NewTransferOrder(id shared.ID, number string, in StockTransferInputSnapshot, outLedgerID, inLedgerID, actorID string, now time.Time) (*TransferOrder, error) {
	if id.IsZero() || strings.TrimSpace(number) == "" || strings.TrimSpace(in.ProductID) == "" || strings.TrimSpace(in.BatchID) == "" || in.Quantity <= 0 || strings.TrimSpace(in.FromWarehouseID) == "" || strings.TrimSpace(in.FromAreaID) == "" || strings.TrimSpace(in.FromLocationID) == "" || strings.TrimSpace(in.ToWarehouseID) == "" || strings.TrimSpace(in.ToAreaID) == "" || strings.TrimSpace(in.ToLocationID) == "" || strings.TrimSpace(outLedgerID) == "" || strings.TrimSpace(inLedgerID) == "" || strings.TrimSpace(actorID) == "" {
		return nil, fmt.Errorf("transfer order input is incomplete")
	}
	now = normalizeInventoryOperationTime(now)
	item := &TransferOrder{ID: id, Number: strings.TrimSpace(number), ProductID: strings.TrimSpace(in.ProductID), BatchID: strings.TrimSpace(in.BatchID), Quantity: in.Quantity, FromWarehouseID: strings.TrimSpace(in.FromWarehouseID), FromAreaID: strings.TrimSpace(in.FromAreaID), FromLocationID: strings.TrimSpace(in.FromLocationID), ToWarehouseID: strings.TrimSpace(in.ToWarehouseID), ToAreaID: strings.TrimSpace(in.ToAreaID), ToLocationID: strings.TrimSpace(in.ToLocationID), OutLedgerID: strings.TrimSpace(outLedgerID), InLedgerID: strings.TrimSpace(inLedgerID), Status: TransferCompleted, TransferredBy: strings.TrimSpace(actorID), TransferredAt: now}
	item.Meta.Touch(now)
	return item, nil
}

type StockTransferInputSnapshot struct {
	ProductID, BatchID                                                                 string
	Quantity                                                                           int
	FromWarehouseID, FromAreaID, FromLocationID, ToWarehouseID, ToAreaID, ToLocationID string
}

func normalizeInventoryOperationTime(now time.Time) time.Time {
	if now.IsZero() {
		return time.Now().UTC()
	}
	return now.UTC()
}
