package pharmaoa

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

type PaymentPlanStatus string
type InvoiceRecordStatus string
type PaymentReminderJobStatus string
type PaymentReminderJobLogLevel string

const (
	PaymentPlanPending PaymentPlanStatus = "pending"
	PaymentPlanPartial PaymentPlanStatus = "partial"
	PaymentPlanPaid    PaymentPlanStatus = "paid"
	PaymentPlanOverdue PaymentPlanStatus = "overdue"

	InvoiceRecordIssued InvoiceRecordStatus = "issued"
	InvoiceRecordVoided InvoiceRecordStatus = "voided"

	PaymentReminderJobPending   PaymentReminderJobStatus = "pending"
	PaymentReminderJobRunning   PaymentReminderJobStatus = "running"
	PaymentReminderJobSucceeded PaymentReminderJobStatus = "succeeded"
	PaymentReminderJobFailed    PaymentReminderJobStatus = "failed"

	PaymentReminderJobLogInfo  PaymentReminderJobLogLevel = "info"
	PaymentReminderJobLogError PaymentReminderJobLogLevel = "error"
)

type FinancialAttachment struct {
	FileID   string `json:"fileId"`
	FileName string `json:"fileName"`
	Size     int64  `json:"size"`
}

type PaymentReceipt struct {
	ID          shared.ID             `json:"id"`
	AmountCents int64                 `json:"amountCents"`
	PaidAt      time.Time             `json:"paidAt"`
	Reference   string                `json:"reference"`
	Attachments []FinancialAttachment `json:"attachments"`
	RecordedBy  string                `json:"recordedBy"`
	RecordedAt  time.Time             `json:"recordedAt"`
}

type PaymentPlan struct {
	ID                  shared.ID             `json:"id"`
	SalesOrderID        string                `json:"salesOrderId"`
	SalesOrderNumber    string                `json:"salesOrderNumber"`
	CustomerID          string                `json:"customerId"`
	OwnerID             string                `json:"ownerId"`
	OrderTotalCents     int64                 `json:"orderTotalCents"`
	AmountCents         int64                 `json:"amountCents"`
	PaidAmountCents     int64                 `json:"paidAmountCents"`
	DueAt               time.Time             `json:"dueAt"`
	Status              PaymentPlanStatus     `json:"status"`
	Note                string                `json:"note,omitempty"`
	Attachments         []FinancialAttachment `json:"attachments"`
	Receipts            []PaymentReceipt      `json:"receipts"`
	NotificationID      string                `json:"notificationId,omitempty"`
	ReminderRecipientID string                `json:"reminderRecipientId,omitempty"`
	RemindedAt          *time.Time            `json:"remindedAt,omitempty"`
	CreatedAt           time.Time             `json:"createdAt"`
	UpdatedAt           time.Time             `json:"updatedAt"`
}

type InvoiceRecord struct {
	ID               shared.ID             `json:"id"`
	Number           string                `json:"number"`
	SalesOrderID     string                `json:"salesOrderId"`
	SalesOrderNumber string                `json:"salesOrderNumber"`
	CustomerID       string                `json:"customerId"`
	OwnerID          string                `json:"ownerId"`
	OrderTotalCents  int64                 `json:"orderTotalCents"`
	AmountCents      int64                 `json:"amountCents"`
	IssuedAt         time.Time             `json:"issuedAt"`
	Status           InvoiceRecordStatus   `json:"status"`
	Note             string                `json:"note,omitempty"`
	Attachments      []FinancialAttachment `json:"attachments"`
	CreatedBy        string                `json:"createdBy"`
	CreatedAt        time.Time             `json:"createdAt"`
	VoidedBy         string                `json:"voidedBy,omitempty"`
	VoidReason       string                `json:"voidReason,omitempty"`
	VoidedAt         *time.Time            `json:"voidedAt,omitempty"`
}

type PaymentReminderJobLog struct {
	Level     PaymentReminderJobLogLevel `json:"level"`
	Message   string                     `json:"message"`
	CreatedAt time.Time                  `json:"createdAt"`
}

type PaymentReminderJob struct {
	ID           shared.ID                `json:"id"`
	Status       PaymentReminderJobStatus `json:"status"`
	RecipientID  string                   `json:"recipientId"`
	OwnerID      string                   `json:"ownerId,omitempty"`
	IncludeAll   bool                     `json:"includeAll"`
	InitiatedBy  string                   `json:"initiatedBy"`
	LastRunBy    string                   `json:"lastRunBy"`
	RetryCount   int                      `json:"retryCount"`
	MatchedCount int                      `json:"matchedCount"`
	CreatedCount int                      `json:"createdCount"`
	Error        string                   `json:"error,omitempty"`
	Logs         []PaymentReminderJobLog  `json:"logs"`
	CreatedAt    time.Time                `json:"createdAt"`
	StartedAt    *time.Time               `json:"startedAt,omitempty"`
	CompletedAt  *time.Time               `json:"completedAt,omitempty"`
}

func NewPaymentPlan(id shared.ID, orderID, orderNumber, customerID, ownerID string, orderTotalCents, amountCents int64, dueAt time.Time, note string, attachments []FinancialAttachment, now time.Time) (*PaymentPlan, error) {
	if id.IsZero() || strings.TrimSpace(orderID) == "" || strings.TrimSpace(orderNumber) == "" || strings.TrimSpace(customerID) == "" || strings.TrimSpace(ownerID) == "" {
		return nil, fmt.Errorf("payment plan identity is incomplete")
	}
	if orderTotalCents <= 0 || amountCents <= 0 || amountCents > orderTotalCents || dueAt.IsZero() {
		return nil, fmt.Errorf("payment plan amount or due date is invalid")
	}
	files, err := normalizeFinancialAttachments(attachments)
	if err != nil {
		return nil, err
	}
	now = normalizeFinancialTime(now)
	return &PaymentPlan{
		ID: id, SalesOrderID: strings.TrimSpace(orderID), SalesOrderNumber: strings.TrimSpace(orderNumber), CustomerID: strings.TrimSpace(customerID), OwnerID: strings.TrimSpace(ownerID),
		OrderTotalCents: orderTotalCents, AmountCents: amountCents, DueAt: dueAt.UTC(), Status: PaymentPlanPending, Note: strings.TrimSpace(note), Attachments: files,
		Receipts: []PaymentReceipt{}, CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (p *PaymentPlan) Receive(receiptID shared.ID, amountCents int64, paidAt time.Time, reference, actorID string, attachments []FinancialAttachment, now time.Time) error {
	if p == nil || receiptID.IsZero() || strings.TrimSpace(actorID) == "" || paidAt.IsZero() {
		return fmt.Errorf("payment receipt identity is incomplete")
	}
	remaining := p.AmountCents - p.PaidAmountCents
	if p.Status == PaymentPlanPaid || amountCents <= 0 || amountCents > remaining {
		return fmt.Errorf("payment receipt amount exceeds the remaining plan balance")
	}
	files, err := normalizeFinancialAttachments(attachments)
	if err != nil {
		return err
	}
	now = normalizeFinancialTime(now)
	p.Receipts = append(p.Receipts, PaymentReceipt{ID: receiptID, AmountCents: amountCents, PaidAt: paidAt.UTC(), Reference: strings.TrimSpace(reference), Attachments: files, RecordedBy: strings.TrimSpace(actorID), RecordedAt: now})
	p.PaidAmountCents += amountCents
	if p.PaidAmountCents == p.AmountCents {
		p.Status = PaymentPlanPaid
	} else if p.Status != PaymentPlanOverdue {
		p.Status = PaymentPlanPartial
	}
	p.UpdatedAt = now
	return nil
}

func (p *PaymentPlan) MarkOverdue(now time.Time) bool {
	if p == nil || p.Status == PaymentPlanPaid || !p.DueAt.Before(now.UTC()) {
		return false
	}
	p.Status = PaymentPlanOverdue
	p.UpdatedAt = now.UTC()
	return true
}

func NewInvoiceRecord(id shared.ID, number, orderID, orderNumber, customerID, ownerID, actorID string, orderTotalCents, amountCents int64, issuedAt time.Time, note string, attachments []FinancialAttachment, now time.Time) (*InvoiceRecord, error) {
	if id.IsZero() || strings.TrimSpace(number) == "" || strings.TrimSpace(orderID) == "" || strings.TrimSpace(orderNumber) == "" || strings.TrimSpace(customerID) == "" || strings.TrimSpace(ownerID) == "" || strings.TrimSpace(actorID) == "" {
		return nil, fmt.Errorf("invoice record identity is incomplete")
	}
	if orderTotalCents <= 0 || amountCents <= 0 || amountCents > orderTotalCents || issuedAt.IsZero() {
		return nil, fmt.Errorf("invoice amount or issue date is invalid")
	}
	files, err := normalizeFinancialAttachments(attachments)
	if err != nil {
		return nil, err
	}
	now = normalizeFinancialTime(now)
	return &InvoiceRecord{ID: id, Number: strings.TrimSpace(number), SalesOrderID: strings.TrimSpace(orderID), SalesOrderNumber: strings.TrimSpace(orderNumber), CustomerID: strings.TrimSpace(customerID), OwnerID: strings.TrimSpace(ownerID), OrderTotalCents: orderTotalCents, AmountCents: amountCents, IssuedAt: issuedAt.UTC(), Status: InvoiceRecordIssued, Note: strings.TrimSpace(note), Attachments: files, CreatedBy: strings.TrimSpace(actorID), CreatedAt: now}, nil
}

func (i *InvoiceRecord) Void(actorID, reason string, now time.Time) error {
	if i == nil || i.Status != InvoiceRecordIssued {
		return fmt.Errorf("only issued invoices can be voided")
	}
	if strings.TrimSpace(actorID) == "" || strings.TrimSpace(reason) == "" {
		return fmt.Errorf("invoice void actor and reason are required")
	}
	now = normalizeFinancialTime(now)
	i.Status = InvoiceRecordVoided
	i.VoidedBy = strings.TrimSpace(actorID)
	i.VoidReason = strings.TrimSpace(reason)
	i.VoidedAt = &now
	return nil
}

func NewPaymentReminderJob(id shared.ID, recipientID, ownerID, actorID string, includeAll bool, now time.Time) (*PaymentReminderJob, error) {
	if id.IsZero() || strings.TrimSpace(recipientID) == "" || strings.TrimSpace(actorID) == "" || (!includeAll && strings.TrimSpace(ownerID) == "") {
		return nil, fmt.Errorf("payment reminder job input is incomplete")
	}
	now = normalizeFinancialTime(now)
	return &PaymentReminderJob{ID: id, Status: PaymentReminderJobPending, RecipientID: strings.TrimSpace(recipientID), OwnerID: strings.TrimSpace(ownerID), IncludeAll: includeAll, InitiatedBy: strings.TrimSpace(actorID), LastRunBy: strings.TrimSpace(actorID), Logs: []PaymentReminderJobLog{}, CreatedAt: now}, nil
}

func (j *PaymentReminderJob) Start(actorID string, retry bool, now time.Time) error {
	if j == nil || strings.TrimSpace(actorID) == "" || retry && j.Status != PaymentReminderJobFailed || !retry && j.Status != PaymentReminderJobPending {
		return fmt.Errorf("payment reminder job cannot start from its current state")
	}
	now = normalizeFinancialTime(now)
	if retry {
		j.RetryCount++
	}
	j.Status = PaymentReminderJobRunning
	j.LastRunBy = strings.TrimSpace(actorID)
	j.Error = ""
	j.MatchedCount = 0
	j.CreatedCount = 0
	j.StartedAt = &now
	j.CompletedAt = nil
	j.Logs = append(j.Logs, PaymentReminderJobLog{Level: PaymentReminderJobLogInfo, Message: "overdue payment scan started", CreatedAt: now})
	return nil
}

func (j *PaymentReminderJob) Succeed(matched, created int, now time.Time) {
	now = normalizeFinancialTime(now)
	j.Status = PaymentReminderJobSucceeded
	j.MatchedCount = matched
	j.CreatedCount = created
	j.CompletedAt = &now
	j.Logs = append(j.Logs, PaymentReminderJobLog{Level: PaymentReminderJobLogInfo, Message: fmt.Sprintf("scan completed: matched=%d created=%d", matched, created), CreatedAt: now})
}

func (j *PaymentReminderJob) Fail(cause error, now time.Time) {
	now = normalizeFinancialTime(now)
	j.Status = PaymentReminderJobFailed
	if cause != nil {
		j.Error = cause.Error()
	}
	j.CompletedAt = &now
	j.Logs = append(j.Logs, PaymentReminderJobLog{Level: PaymentReminderJobLogError, Message: j.Error, CreatedAt: now})
}

func normalizeFinancialAttachments(values []FinancialAttachment) ([]FinancialAttachment, error) {
	seen := map[string]struct{}{}
	out := make([]FinancialAttachment, 0, len(values))
	for _, value := range values {
		value.FileID = strings.TrimSpace(value.FileID)
		value.FileName = strings.TrimSpace(value.FileName)
		if value.FileID == "" || value.FileName == "" || value.Size < 0 {
			return nil, fmt.Errorf("financial attachment is invalid")
		}
		if filepath.Base(value.FileName) != value.FileName || strings.Contains(value.FileName, "..") {
			return nil, fmt.Errorf("financial attachment name is unsafe")
		}
		switch strings.ToLower(filepath.Ext(value.FileName)) {
		case ".exe", ".bat", ".cmd", ".com", ".ps1", ".sh":
			return nil, fmt.Errorf("financial attachment type is not allowed")
		}
		if _, ok := seen[value.FileID]; ok {
			continue
		}
		seen[value.FileID] = struct{}{}
		out = append(out, value)
	}
	return out, nil
}

func normalizeFinancialTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value.UTC()
}
