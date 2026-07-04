package pharmaoa

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

type CustomerStatus string

const (
	CustomerStatusActive   CustomerStatus = "active"
	CustomerStatusDisabled CustomerStatus = "disabled"
)

type CustomerContact struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Phone string `json:"phone"`
	Email string `json:"email"`
	Title string `json:"title"`
}

type CustomerAttachment struct {
	FileID   string `json:"fileId"`
	FileName string `json:"fileName"`
	Size     int64  `json:"size"`
}

type CustomerQualification struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Number      string               `json:"number"`
	ExpiresAt   time.Time            `json:"expiresAt"`
	Attachments []CustomerAttachment `json:"attachments"`
}

type Customer struct {
	ID             shared.ID               `json:"id"`
	Code           string                  `json:"code"`
	Name           string                  `json:"name"`
	Region         string                  `json:"region"`
	OrganizationID string                  `json:"organizationId"`
	OwnerID        string                  `json:"ownerId"`
	Rating         int                     `json:"rating"`
	Status         CustomerStatus          `json:"status"`
	DisableReason  string                  `json:"disableReason,omitempty"`
	Contacts       []CustomerContact       `json:"contacts"`
	Qualifications []CustomerQualification `json:"qualifications"`
	Meta           shared.AuditMeta        `json:"meta"`
}

type CustomerInput struct {
	Code           string
	Name           string
	Region         string
	OrganizationID string
	OwnerID        string
	Rating         int
	Contacts       []CustomerContact
	Qualifications []CustomerQualification
}

func NewCustomer(id shared.ID, in CustomerInput, now time.Time) (*Customer, error) {
	if id.IsZero() {
		return nil, fmt.Errorf("id is required")
	}
	if err := validateCustomerInput(in); err != nil {
		return nil, err
	}
	contacts, err := NormalizeCustomerContacts(in.Contacts)
	if err != nil {
		return nil, err
	}
	qualifications, err := NormalizeCustomerQualifications(in.Qualifications)
	if err != nil {
		return nil, err
	}
	entity := &Customer{
		ID:             id,
		Code:           strings.TrimSpace(in.Code),
		Name:           strings.TrimSpace(in.Name),
		Region:         strings.TrimSpace(in.Region),
		OrganizationID: strings.TrimSpace(in.OrganizationID),
		OwnerID:        strings.TrimSpace(in.OwnerID),
		Rating:         normalizeCustomerRating(in.Rating),
		Status:         CustomerStatusActive,
		Contacts:       contacts,
		Qualifications: qualifications,
	}
	entity.Meta.Touch(now)
	return entity, nil
}

func (c *Customer) Update(in CustomerInput, now time.Time) error {
	if c == nil {
		return fmt.Errorf("customer is required")
	}
	if err := validateCustomerInput(in); err != nil {
		return err
	}
	contacts, err := NormalizeCustomerContacts(in.Contacts)
	if err != nil {
		return err
	}
	qualifications, err := NormalizeCustomerQualifications(in.Qualifications)
	if err != nil {
		return err
	}
	c.Code = strings.TrimSpace(in.Code)
	c.Name = strings.TrimSpace(in.Name)
	c.Region = strings.TrimSpace(in.Region)
	c.OrganizationID = strings.TrimSpace(in.OrganizationID)
	c.OwnerID = strings.TrimSpace(in.OwnerID)
	c.Rating = normalizeCustomerRating(in.Rating)
	c.Contacts = contacts
	c.Qualifications = qualifications
	c.Meta.Touch(now)
	return nil
}

func (c *Customer) Disable(reason string, now time.Time) error {
	if c == nil {
		return fmt.Errorf("customer is required")
	}
	if c.Status == CustomerStatusDisabled {
		return nil
	}
	c.Status = CustomerStatusDisabled
	c.DisableReason = strings.TrimSpace(reason)
	c.Meta.Touch(now)
	return nil
}

func (c *Customer) QualificationExpiringBefore(deadline time.Time) []CustomerQualification {
	if c == nil || c.Status == CustomerStatusDisabled {
		return []CustomerQualification{}
	}
	out := make([]CustomerQualification, 0)
	for _, qualification := range c.Qualifications {
		if qualification.ExpiresAt.IsZero() {
			continue
		}
		if !qualification.ExpiresAt.After(deadline) {
			out = append(out, qualification)
		}
	}
	return out
}

func (c *Customer) CanUseForSales(now time.Time) error {
	if c == nil {
		return fmt.Errorf("customer is required")
	}
	if c.Status == CustomerStatusDisabled {
		return fmt.Errorf("customer is disabled")
	}
	for _, qualification := range c.Qualifications {
		if !qualification.ExpiresAt.IsZero() && qualification.ExpiresAt.Before(now) {
			return fmt.Errorf("customer qualification expired: %s", qualification.Name)
		}
	}
	return nil
}

func NormalizeCustomerContacts(items []CustomerContact) ([]CustomerContact, error) {
	out := make([]CustomerContact, 0, len(items))
	seen := map[string]struct{}{}
	for idx, item := range items {
		item.ID = strings.TrimSpace(item.ID)
		item.Name = strings.TrimSpace(item.Name)
		item.Phone = strings.TrimSpace(item.Phone)
		item.Email = strings.TrimSpace(item.Email)
		item.Title = strings.TrimSpace(item.Title)
		if item.ID == "" {
			item.ID = fmt.Sprintf("contact-%d", idx+1)
		}
		if item.Name == "" {
			return nil, fmt.Errorf("contact name is required")
		}
		if _, ok := seen[item.ID]; ok {
			return nil, fmt.Errorf("duplicate contact id: %s", item.ID)
		}
		seen[item.ID] = struct{}{}
		out = append(out, item)
	}
	return out, nil
}

func NormalizeCustomerQualifications(items []CustomerQualification) ([]CustomerQualification, error) {
	out := make([]CustomerQualification, 0, len(items))
	seen := map[string]struct{}{}
	for idx, item := range items {
		item.ID = strings.TrimSpace(item.ID)
		item.Name = strings.TrimSpace(item.Name)
		item.Number = strings.TrimSpace(item.Number)
		if item.ID == "" {
			item.ID = fmt.Sprintf("qualification-%d", idx+1)
		}
		if item.Name == "" {
			return nil, fmt.Errorf("qualification name is required")
		}
		if _, ok := seen[item.ID]; ok {
			return nil, fmt.Errorf("duplicate qualification id: %s", item.ID)
		}
		attachments, err := NormalizeCustomerAttachments(item.Attachments)
		if err != nil {
			return nil, err
		}
		item.Attachments = attachments
		seen[item.ID] = struct{}{}
		out = append(out, item)
	}
	return out, nil
}

func NormalizeCustomerAttachments(items []CustomerAttachment) ([]CustomerAttachment, error) {
	out := make([]CustomerAttachment, 0, len(items))
	for _, item := range items {
		item.FileID = strings.TrimSpace(item.FileID)
		item.FileName = strings.TrimSpace(item.FileName)
		if item.FileID == "" {
			return nil, fmt.Errorf("attachment fileId is required")
		}
		if item.FileName == "" {
			return nil, fmt.Errorf("attachment fileName is required")
		}
		if item.Size < 0 {
			return nil, fmt.Errorf("attachment size cannot be negative")
		}
		if strings.Contains(item.FileName, "..") || strings.ContainsAny(item.FileName, `/\`) {
			return nil, fmt.Errorf("attachment fileName is unsafe")
		}
		switch strings.ToLower(filepath.Ext(item.FileName)) {
		case ".exe", ".bat", ".cmd", ".ps1", ".sh":
			return nil, fmt.Errorf("attachment fileName extension is not allowed")
		}
		out = append(out, item)
	}
	return out, nil
}

func validateCustomerInput(in CustomerInput) error {
	for field, value := range map[string]string{
		"code":           in.Code,
		"name":           in.Name,
		"region":         in.Region,
		"organizationId": in.OrganizationID,
		"ownerId":        in.OwnerID,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", field)
		}
	}
	if in.Rating < 0 || in.Rating > 5 {
		return fmt.Errorf("rating must be between 0 and 5")
	}
	return nil
}

func normalizeCustomerRating(rating int) int {
	if rating == 0 {
		return 3
	}
	return rating
}
