package pharmaoa

import (
	"fmt"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

type PurchaseRequestStatus string

const (
	PurchaseRequestPending  PurchaseRequestStatus = "pending_approval"
	PurchaseRequestApproved PurchaseRequestStatus = "approved"
	PurchaseRequestRejected PurchaseRequestStatus = "rejected"
)

type PurchaseOrderStatus string

const PurchaseOrderOpen PurchaseOrderStatus = "open"

type PurchaseLine struct {
	ProductID string  `json:"productId"`
	Quantity  float64 `json:"quantity"`
	UnitPrice float64 `json:"unitPrice"`
	Amount    float64 `json:"amount"`
}

type PurchaseRequest struct {
	ID                 shared.ID             `json:"id"`
	Number             string                `json:"number"`
	SupplierID         string                `json:"supplierId"`
	RequesterID        string                `json:"requesterId"`
	ApproverID         string                `json:"approverId"`
	Reason             string                `json:"reason"`
	Lines              []PurchaseLine        `json:"lines"`
	TotalAmount        float64               `json:"totalAmount"`
	Status             PurchaseRequestStatus `json:"status"`
	WorkflowInstanceID string                `json:"workflowInstanceId"`
	PurchaseOrderID    string                `json:"purchaseOrderId,omitempty"`
	Meta               shared.AuditMeta      `json:"meta"`
}

type PurchaseOrder struct {
	ID                shared.ID           `json:"id"`
	Number            string              `json:"number"`
	PurchaseRequestID string              `json:"purchaseRequestId"`
	SupplierID        string              `json:"supplierId"`
	Lines             []PurchaseLine      `json:"lines"`
	TotalAmount       float64             `json:"totalAmount"`
	Status            PurchaseOrderStatus `json:"status"`
	ApprovedBy        string              `json:"approvedBy"`
	ApprovedAt        time.Time           `json:"approvedAt"`
	Meta              shared.AuditMeta    `json:"meta"`
}

type PurchaseRequestInput struct {
	Number      string
	SupplierID  string
	RequesterID string
	ApproverID  string
	Reason      string
	Lines       []PurchaseLine
}

func NewPurchaseRequest(id shared.ID, in PurchaseRequestInput, workflowInstanceID string, now time.Time) (*PurchaseRequest, error) {
	if id.IsZero() {
		return nil, fmt.Errorf("purchase request id is required")
	}
	if strings.TrimSpace(in.Number) == "" || strings.TrimSpace(in.SupplierID) == "" || strings.TrimSpace(in.RequesterID) == "" || strings.TrimSpace(in.ApproverID) == "" || strings.TrimSpace(workflowInstanceID) == "" {
		return nil, fmt.Errorf("purchase request input is incomplete")
	}
	lines, total, err := normalizePurchaseLines(in.Lines)
	if err != nil {
		return nil, err
	}
	item := &PurchaseRequest{
		ID: id, Number: strings.TrimSpace(in.Number), SupplierID: strings.TrimSpace(in.SupplierID),
		RequesterID: strings.TrimSpace(in.RequesterID), ApproverID: strings.TrimSpace(in.ApproverID),
		Reason: strings.TrimSpace(in.Reason), Lines: lines, TotalAmount: total,
		Status: PurchaseRequestPending, WorkflowInstanceID: strings.TrimSpace(workflowInstanceID),
	}
	item.Meta.Touch(normalizePurchaseTime(now))
	return item, nil
}

func NewPurchaseOrder(id shared.ID, number string, request PurchaseRequest, approvedBy string, approvedAt time.Time) (*PurchaseOrder, error) {
	if id.IsZero() || strings.TrimSpace(number) == "" || request.ID.IsZero() || request.Status != PurchaseRequestApproved || strings.TrimSpace(approvedBy) == "" {
		return nil, fmt.Errorf("purchase order input is incomplete")
	}
	item := &PurchaseOrder{
		ID: id, Number: strings.TrimSpace(number), PurchaseRequestID: request.ID.String(), SupplierID: request.SupplierID,
		Lines: append([]PurchaseLine(nil), request.Lines...), TotalAmount: request.TotalAmount,
		Status: PurchaseOrderOpen, ApprovedBy: strings.TrimSpace(approvedBy), ApprovedAt: normalizePurchaseTime(approvedAt),
	}
	item.Meta.Touch(item.ApprovedAt)
	return item, nil
}

func (r *PurchaseRequest) Approve(orderID string, now time.Time) error {
	if r == nil || r.Status != PurchaseRequestPending {
		return fmt.Errorf("purchase request is not pending approval")
	}
	if strings.TrimSpace(orderID) == "" {
		return fmt.Errorf("purchase order id is required")
	}
	r.Status = PurchaseRequestApproved
	r.PurchaseOrderID = strings.TrimSpace(orderID)
	r.Meta.Touch(normalizePurchaseTime(now))
	return nil
}

func (r *PurchaseRequest) Reject(now time.Time) error {
	if r == nil || r.Status != PurchaseRequestPending {
		return fmt.Errorf("purchase request is not pending approval")
	}
	r.Status = PurchaseRequestRejected
	r.Meta.Touch(normalizePurchaseTime(now))
	return nil
}

func normalizePurchaseLines(lines []PurchaseLine) ([]PurchaseLine, float64, error) {
	if len(lines) == 0 {
		return nil, 0, fmt.Errorf("purchase request requires at least one line")
	}
	out := make([]PurchaseLine, 0, len(lines))
	total := 0.0
	for _, line := range lines {
		line.ProductID = strings.TrimSpace(line.ProductID)
		if line.ProductID == "" || line.Quantity <= 0 || line.UnitPrice < 0 {
			return nil, 0, fmt.Errorf("purchase line is invalid")
		}
		line.Amount = line.Quantity * line.UnitPrice
		total += line.Amount
		out = append(out, line)
	}
	return out, total, nil
}

func normalizePurchaseTime(now time.Time) time.Time {
	if now.IsZero() {
		return time.Now().UTC()
	}
	return now.UTC()
}
