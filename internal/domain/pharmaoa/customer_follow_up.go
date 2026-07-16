package pharmaoa

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

type CustomerFollowUpStatus string
type CustomerFollowUpChannel string

const (
	CustomerFollowUpPlanned   CustomerFollowUpStatus = "planned"
	CustomerFollowUpCompleted CustomerFollowUpStatus = "completed"
	CustomerFollowUpCancelled CustomerFollowUpStatus = "cancelled"

	CustomerFollowUpOnsite CustomerFollowUpChannel = "onsite"
	CustomerFollowUpPhone  CustomerFollowUpChannel = "phone"
	CustomerFollowUpOnline CustomerFollowUpChannel = "online"
	CustomerFollowUpEmail  CustomerFollowUpChannel = "email"
)

type CustomerFollowUpAttachment struct {
	FileID   string `json:"fileId"`
	FileName string `json:"fileName"`
	Size     int64  `json:"size"`
}

type CustomerFollowUp struct {
	ID             shared.ID                    `json:"id"`
	CustomerID     string                       `json:"customerId"`
	CustomerCode   string                       `json:"customerCode"`
	CustomerName   string                       `json:"customerName"`
	OrganizationID string                       `json:"organizationId"`
	OwnerID        string                       `json:"ownerId"`
	ContactName    string                       `json:"contactName,omitempty"`
	Channel        CustomerFollowUpChannel      `json:"channel"`
	Status         CustomerFollowUpStatus       `json:"status"`
	ScheduledAt    time.Time                    `json:"scheduledAt"`
	CompletedAt    *time.Time                   `json:"completedAt,omitempty"`
	Summary        string                       `json:"summary,omitempty"`
	NextAction     string                       `json:"nextAction,omitempty"`
	CancelReason   string                       `json:"cancelReason,omitempty"`
	Attachments    []CustomerFollowUpAttachment `json:"attachments"`
	CreatedAt      time.Time                    `json:"createdAt"`
	UpdatedAt      time.Time                    `json:"updatedAt"`
}

type CustomerFollowUpInput struct {
	CustomerID     string
	CustomerCode   string
	CustomerName   string
	OrganizationID string
	OwnerID        string
	ContactName    string
	Channel        CustomerFollowUpChannel
	ScheduledAt    time.Time
	NextAction     string
	Attachments    []CustomerFollowUpAttachment
}

func NewCustomerFollowUp(id shared.ID, in CustomerFollowUpInput, now time.Time) (*CustomerFollowUp, error) {
	if id.IsZero() || strings.TrimSpace(in.CustomerID) == "" || strings.TrimSpace(in.CustomerCode) == "" || strings.TrimSpace(in.CustomerName) == "" || strings.TrimSpace(in.OrganizationID) == "" || strings.TrimSpace(in.OwnerID) == "" {
		return nil, fmt.Errorf("customer follow-up identity is incomplete")
	}
	if in.ScheduledAt.IsZero() {
		return nil, fmt.Errorf("scheduledAt is required")
	}
	if !validCustomerFollowUpChannel(in.Channel) {
		return nil, fmt.Errorf("customer follow-up channel is invalid")
	}
	attachments, err := normalizeCustomerFollowUpAttachments(in.Attachments)
	if err != nil {
		return nil, err
	}
	now = now.UTC()
	return &CustomerFollowUp{
		ID: id, CustomerID: strings.TrimSpace(in.CustomerID), CustomerCode: strings.TrimSpace(in.CustomerCode), CustomerName: strings.TrimSpace(in.CustomerName),
		OrganizationID: strings.TrimSpace(in.OrganizationID), OwnerID: strings.TrimSpace(in.OwnerID), ContactName: strings.TrimSpace(in.ContactName), Channel: in.Channel,
		Status: CustomerFollowUpPlanned, ScheduledAt: in.ScheduledAt.UTC(), NextAction: strings.TrimSpace(in.NextAction), Attachments: attachments, CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (f *CustomerFollowUp) UpdatePlan(in CustomerFollowUpInput, now time.Time) error {
	if f == nil || f.Status != CustomerFollowUpPlanned {
		return fmt.Errorf("only planned customer follow-ups can be updated")
	}
	if in.ScheduledAt.IsZero() || !validCustomerFollowUpChannel(in.Channel) {
		return fmt.Errorf("customer follow-up plan is invalid")
	}
	attachments, err := normalizeCustomerFollowUpAttachments(in.Attachments)
	if err != nil {
		return err
	}
	f.ContactName = strings.TrimSpace(in.ContactName)
	f.Channel = in.Channel
	f.ScheduledAt = in.ScheduledAt.UTC()
	f.NextAction = strings.TrimSpace(in.NextAction)
	f.Attachments = attachments
	f.UpdatedAt = now.UTC()
	return nil
}

func (f *CustomerFollowUp) Complete(summary, nextAction string, attachments []CustomerFollowUpAttachment, now time.Time) error {
	if f == nil || f.Status != CustomerFollowUpPlanned {
		return fmt.Errorf("only planned customer follow-ups can be completed")
	}
	if strings.TrimSpace(summary) == "" {
		return fmt.Errorf("follow-up summary is required")
	}
	normalized, err := normalizeCustomerFollowUpAttachments(attachments)
	if err != nil {
		return err
	}
	completedAt := now.UTC()
	f.Status = CustomerFollowUpCompleted
	f.Summary = strings.TrimSpace(summary)
	f.NextAction = strings.TrimSpace(nextAction)
	f.Attachments = normalized
	f.CompletedAt = &completedAt
	f.UpdatedAt = completedAt
	return nil
}

func (f *CustomerFollowUp) Cancel(reason string, now time.Time) error {
	if f == nil || f.Status != CustomerFollowUpPlanned {
		return fmt.Errorf("only planned customer follow-ups can be cancelled")
	}
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("cancel reason is required")
	}
	f.Status = CustomerFollowUpCancelled
	f.CancelReason = strings.TrimSpace(reason)
	f.UpdatedAt = now.UTC()
	return nil
}

func validCustomerFollowUpChannel(channel CustomerFollowUpChannel) bool {
	switch channel {
	case CustomerFollowUpOnsite, CustomerFollowUpPhone, CustomerFollowUpOnline, CustomerFollowUpEmail:
		return true
	default:
		return false
	}
}

func normalizeCustomerFollowUpAttachments(values []CustomerFollowUpAttachment) ([]CustomerFollowUpAttachment, error) {
	seen := map[string]struct{}{}
	out := make([]CustomerFollowUpAttachment, 0, len(values))
	for _, value := range values {
		value.FileID = strings.TrimSpace(value.FileID)
		value.FileName = strings.TrimSpace(value.FileName)
		if value.FileID == "" || value.FileName == "" || value.Size < 0 {
			return nil, fmt.Errorf("customer follow-up attachment is invalid")
		}
		if filepath.Base(value.FileName) != value.FileName || strings.Contains(value.FileName, "..") {
			return nil, fmt.Errorf("customer follow-up attachment name is unsafe")
		}
		ext := strings.ToLower(filepath.Ext(value.FileName))
		if ext == ".exe" || ext == ".bat" || ext == ".cmd" || ext == ".com" || ext == ".ps1" || ext == ".sh" {
			return nil, fmt.Errorf("customer follow-up attachment type is not allowed")
		}
		if _, ok := seen[value.FileID]; ok {
			continue
		}
		seen[value.FileID] = struct{}{}
		out = append(out, value)
	}
	return out, nil
}
