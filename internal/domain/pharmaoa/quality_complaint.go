package pharmaoa

import (
	"fmt"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

type QualityComplaintStatus string

const (
	QualityComplaintPending  QualityComplaintStatus = "pending"
	QualityComplaintResolved QualityComplaintStatus = "resolved"
	QualityComplaintRejected QualityComplaintStatus = "rejected"
)

type QualityComplaint struct {
	ID                 shared.ID              `json:"id"`
	Number             string                 `json:"number"`
	Title              string                 `json:"title"`
	Description        string                 `json:"description"`
	CustomerID         string                 `json:"customerId"`
	CustomerName       string                 `json:"customerName"`
	ProductID          string                 `json:"productId"`
	ProductName        string                 `json:"productName"`
	BatchID            string                 `json:"batchId"`
	BatchNo            string                 `json:"batchNo"`
	ReporterID         string                 `json:"reporterId"`
	HandlerID          string                 `json:"handlerId"`
	Attachments        []ContractAttachment   `json:"attachments"`
	WorkflowInstanceID string                 `json:"workflowInstanceId"`
	Status             QualityComplaintStatus `json:"status"`
	Conclusion         string                 `json:"conclusion,omitempty"`
	ResolvedBy         string                 `json:"resolvedBy,omitempty"`
	ResolvedAt         *time.Time             `json:"resolvedAt,omitempty"`
	RejectedBy         string                 `json:"rejectedBy,omitempty"`
	RejectedAt         *time.Time             `json:"rejectedAt,omitempty"`
	Meta               shared.AuditMeta       `json:"meta"`
}

type QualityComplaintInput struct {
	Number             string
	Title              string
	Description        string
	CustomerID         string
	CustomerName       string
	ProductID          string
	ProductName        string
	BatchID            string
	BatchNo            string
	ReporterID         string
	HandlerID          string
	Attachments        []ContractAttachment
	WorkflowInstanceID string
}

func NewQualityComplaint(id shared.ID, in QualityComplaintInput, now time.Time) (*QualityComplaint, error) {
	if id.IsZero() {
		return nil, fmt.Errorf("quality complaint id is required")
	}
	attachments, err := normalizeContractAttachments(in.Attachments)
	if err != nil {
		return nil, fmt.Errorf("quality complaint attachments: %w", err)
	}
	if strings.TrimSpace(in.Number) == "" || strings.TrimSpace(in.Title) == "" || strings.TrimSpace(in.Description) == "" || strings.TrimSpace(in.CustomerID) == "" || strings.TrimSpace(in.CustomerName) == "" || strings.TrimSpace(in.ProductID) == "" || strings.TrimSpace(in.ProductName) == "" || strings.TrimSpace(in.BatchID) == "" || strings.TrimSpace(in.BatchNo) == "" || strings.TrimSpace(in.ReporterID) == "" || strings.TrimSpace(in.HandlerID) == "" || strings.TrimSpace(in.WorkflowInstanceID) == "" {
		return nil, fmt.Errorf("quality complaint input is incomplete")
	}
	item := &QualityComplaint{
		ID: id, Number: strings.TrimSpace(in.Number), Title: strings.TrimSpace(in.Title), Description: strings.TrimSpace(in.Description),
		CustomerID: strings.TrimSpace(in.CustomerID), CustomerName: strings.TrimSpace(in.CustomerName), ProductID: strings.TrimSpace(in.ProductID), ProductName: strings.TrimSpace(in.ProductName),
		BatchID: strings.TrimSpace(in.BatchID), BatchNo: strings.TrimSpace(in.BatchNo), ReporterID: strings.TrimSpace(in.ReporterID), HandlerID: strings.TrimSpace(in.HandlerID),
		Attachments: attachments, WorkflowInstanceID: strings.TrimSpace(in.WorkflowInstanceID), Status: QualityComplaintPending,
	}
	item.Meta.Touch(normalizeContractTime(now))
	return item, nil
}

func (c *QualityComplaint) Resolve(actorID, conclusion string, now time.Time) error {
	if c == nil || c.Status != QualityComplaintPending {
		return fmt.Errorf("quality complaint is not pending")
	}
	actorID, conclusion = strings.TrimSpace(actorID), strings.TrimSpace(conclusion)
	if actorID == "" || conclusion == "" {
		return fmt.Errorf("quality complaint handler and conclusion are required")
	}
	now = normalizeContractTime(now)
	c.Status, c.Conclusion, c.ResolvedBy, c.ResolvedAt = QualityComplaintResolved, conclusion, actorID, &now
	c.Meta.Touch(now)
	return nil
}

func (c *QualityComplaint) Reject(actorID, conclusion string, now time.Time) error {
	if c == nil || c.Status != QualityComplaintPending {
		return fmt.Errorf("quality complaint is not pending")
	}
	actorID, conclusion = strings.TrimSpace(actorID), strings.TrimSpace(conclusion)
	if actorID == "" || conclusion == "" {
		return fmt.Errorf("quality complaint handler and rejection conclusion are required")
	}
	now = normalizeContractTime(now)
	c.Status, c.Conclusion, c.RejectedBy, c.RejectedAt = QualityComplaintRejected, conclusion, actorID, &now
	c.Meta.Touch(now)
	return nil
}
