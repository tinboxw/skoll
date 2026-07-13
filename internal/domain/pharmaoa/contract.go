package pharmaoa

import (
	"fmt"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

type ContractPartyType string
type ContractStatus string

const (
	ContractPartySupplier ContractPartyType = "supplier"
	ContractPartyCustomer ContractPartyType = "customer"

	ContractPendingApproval ContractStatus = "pending_approval"
	ContractActive          ContractStatus = "active"
	ContractRejected        ContractStatus = "rejected"
	ContractExpired         ContractStatus = "expired"
)

type ContractAttachment struct {
	FileID   string `json:"fileId"`
	FileName string `json:"fileName"`
	MIME     string `json:"mime"`
	Size     int64  `json:"size"`
}

type Contract struct {
	ID                     shared.ID            `json:"id"`
	Number                 string               `json:"number"`
	Title                  string               `json:"title"`
	PartyType              ContractPartyType    `json:"partyType"`
	PartyID                string               `json:"partyId"`
	PartyName              string               `json:"partyName"`
	OwnerID                string               `json:"ownerId"`
	ApproverID             string               `json:"approverId"`
	Amount                 float64              `json:"amount"`
	Currency               string               `json:"currency"`
	EffectiveAt            time.Time            `json:"effectiveAt"`
	ExpiresAt              time.Time            `json:"expiresAt"`
	Attachments            []ContractAttachment `json:"attachments"`
	WorkflowInstanceID     string               `json:"workflowInstanceId"`
	Status                 ContractStatus       `json:"status"`
	ApprovedBy             string               `json:"approvedBy,omitempty"`
	ApprovedAt             *time.Time           `json:"approvedAt,omitempty"`
	RejectedBy             string               `json:"rejectedBy,omitempty"`
	RejectedAt             *time.Time           `json:"rejectedAt,omitempty"`
	ReminderNotificationID string               `json:"reminderNotificationId,omitempty"`
	Meta                   shared.AuditMeta     `json:"meta"`
}

type ContractInput struct {
	Number             string
	Title              string
	PartyType          ContractPartyType
	PartyID            string
	PartyName          string
	OwnerID            string
	ApproverID         string
	Amount             float64
	Currency           string
	EffectiveAt        time.Time
	ExpiresAt          time.Time
	Attachments        []ContractAttachment
	WorkflowInstanceID string
}

func NewContract(id shared.ID, in ContractInput, now time.Time) (*Contract, error) {
	if id.IsZero() {
		return nil, fmt.Errorf("contract id is required")
	}
	attachments, err := normalizeContractAttachments(in.Attachments)
	if err != nil {
		return nil, err
	}
	if err := validateContractInput(in); err != nil {
		return nil, err
	}
	item := &Contract{
		ID: id, Number: strings.TrimSpace(in.Number), Title: strings.TrimSpace(in.Title), PartyType: in.PartyType,
		PartyID: strings.TrimSpace(in.PartyID), PartyName: strings.TrimSpace(in.PartyName), OwnerID: strings.TrimSpace(in.OwnerID),
		ApproverID: strings.TrimSpace(in.ApproverID), Amount: in.Amount, Currency: strings.ToUpper(strings.TrimSpace(in.Currency)),
		EffectiveAt: in.EffectiveAt.UTC(), ExpiresAt: in.ExpiresAt.UTC(), Attachments: attachments,
		WorkflowInstanceID: strings.TrimSpace(in.WorkflowInstanceID), Status: ContractPendingApproval,
	}
	item.Meta.Touch(normalizeContractTime(now))
	return item, nil
}

func (c *Contract) Approve(actorID string, now time.Time) error {
	if c == nil || c.Status != ContractPendingApproval {
		return fmt.Errorf("contract is not pending approval")
	}
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return fmt.Errorf("contract approver is required")
	}
	now = normalizeContractTime(now)
	c.Status = ContractActive
	c.ApprovedBy = actorID
	c.ApprovedAt = &now
	c.Meta.Touch(now)
	return nil
}

func (c *Contract) Reject(actorID string, now time.Time) error {
	if c == nil || c.Status != ContractPendingApproval {
		return fmt.Errorf("contract is not pending approval")
	}
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return fmt.Errorf("contract approver is required")
	}
	now = normalizeContractTime(now)
	c.Status = ContractRejected
	c.RejectedBy = actorID
	c.RejectedAt = &now
	c.Meta.Touch(now)
	return nil
}

func (c *Contract) MarkExpired(now time.Time) bool {
	if c == nil || c.Status != ContractActive || c.ExpiresAt.After(now) {
		return false
	}
	c.Status = ContractExpired
	c.Meta.Touch(normalizeContractTime(now))
	return true
}

func validateContractInput(in ContractInput) error {
	if strings.TrimSpace(in.Number) == "" || strings.TrimSpace(in.Title) == "" || strings.TrimSpace(in.PartyID) == "" || strings.TrimSpace(in.PartyName) == "" || strings.TrimSpace(in.OwnerID) == "" || strings.TrimSpace(in.ApproverID) == "" || strings.TrimSpace(in.WorkflowInstanceID) == "" {
		return fmt.Errorf("contract input is incomplete")
	}
	if in.PartyType != ContractPartySupplier && in.PartyType != ContractPartyCustomer {
		return fmt.Errorf("contract party type is invalid")
	}
	if in.Amount < 0 || strings.TrimSpace(in.Currency) == "" {
		return fmt.Errorf("contract amount and currency are invalid")
	}
	if in.EffectiveAt.IsZero() || in.ExpiresAt.IsZero() || !in.ExpiresAt.After(in.EffectiveAt) {
		return fmt.Errorf("contract effective and expiry dates are invalid")
	}
	if len(in.Attachments) == 0 {
		return fmt.Errorf("contract requires at least one attachment")
	}
	return nil
}

func normalizeContractAttachments(items []ContractAttachment) ([]ContractAttachment, error) {
	out := make([]ContractAttachment, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		item.FileID = strings.TrimSpace(item.FileID)
		item.FileName = strings.TrimSpace(item.FileName)
		item.MIME = strings.TrimSpace(strings.ToLower(item.MIME))
		if item.FileID == "" || item.FileName == "" || item.Size < 0 {
			return nil, fmt.Errorf("contract attachment is invalid")
		}
		if _, ok := seen[item.FileID]; ok {
			continue
		}
		seen[item.FileID] = struct{}{}
		out = append(out, item)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("contract requires at least one attachment")
	}
	return out, nil
}

func normalizeContractTime(now time.Time) time.Time {
	if now.IsZero() {
		return time.Now().UTC()
	}
	return now.UTC()
}
