package pharmaoa

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

type SupplierStatus string

const (
	SupplierStatusActive   SupplierStatus = "active"
	SupplierStatusDisabled SupplierStatus = "disabled"
)

type SupplierContact struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Email   string `json:"email"`
	Title   string `json:"title"`
	Primary bool   `json:"primary"`
}

type SupplierAttachment struct {
	FileID   string `json:"fileId"`
	FileName string `json:"fileName"`
	MimeType string `json:"mimeType"`
	Size     int64  `json:"size"`
}

type SupplierQualification struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Number      string               `json:"number"`
	ExpiresAt   time.Time            `json:"expiresAt"`
	Attachments []SupplierAttachment `json:"attachments"`
}

type Supplier struct {
	ID             shared.ID               `json:"id"`
	Code           string                  `json:"code"`
	Name           string                  `json:"name"`
	Rating         int                     `json:"rating"`
	Status         SupplierStatus          `json:"status"`
	DisableReason  string                  `json:"disableReason,omitempty"`
	Contacts       []SupplierContact       `json:"contacts"`
	Qualifications []SupplierQualification `json:"qualifications"`
	Meta           shared.AuditMeta        `json:"meta"`
}

type SupplierInput struct {
	Code           string
	Name           string
	Rating         int
	Contacts       []SupplierContact
	Qualifications []SupplierQualification
}

func NewSupplier(id shared.ID, in SupplierInput, now time.Time) (*Supplier, error) {
	if id.IsZero() {
		return nil, fmt.Errorf("id is required")
	}
	if err := validateSupplierInput(in); err != nil {
		return nil, err
	}
	contacts, err := NormalizeSupplierContacts(in.Contacts)
	if err != nil {
		return nil, err
	}
	qualifications, err := NormalizeSupplierQualifications(in.Qualifications)
	if err != nil {
		return nil, err
	}
	entity := &Supplier{
		ID:             id,
		Code:           strings.TrimSpace(in.Code),
		Name:           strings.TrimSpace(in.Name),
		Rating:         normalizeSupplierRating(in.Rating),
		Status:         SupplierStatusActive,
		Contacts:       contacts,
		Qualifications: qualifications,
	}
	entity.Meta.Touch(now)
	return entity, nil
}

func (s *Supplier) Update(in SupplierInput, now time.Time) error {
	if s == nil {
		return fmt.Errorf("supplier is required")
	}
	if err := validateSupplierInput(in); err != nil {
		return err
	}
	contacts, err := NormalizeSupplierContacts(in.Contacts)
	if err != nil {
		return err
	}
	qualifications, err := NormalizeSupplierQualifications(in.Qualifications)
	if err != nil {
		return err
	}
	s.Code = strings.TrimSpace(in.Code)
	s.Name = strings.TrimSpace(in.Name)
	s.Rating = normalizeSupplierRating(in.Rating)
	s.Contacts = contacts
	s.Qualifications = qualifications
	s.Meta.Touch(now)
	return nil
}

func (s *Supplier) Disable(reason string, now time.Time) error {
	if s == nil {
		return fmt.Errorf("supplier is required")
	}
	if s.Status == SupplierStatusDisabled {
		return nil
	}
	s.Status = SupplierStatusDisabled
	s.DisableReason = strings.TrimSpace(reason)
	s.Meta.Touch(now)
	return nil
}

func (s *Supplier) QualificationExpiringBefore(deadline time.Time) []SupplierQualification {
	if s == nil || s.Status == SupplierStatusDisabled {
		return []SupplierQualification{}
	}
	out := make([]SupplierQualification, 0)
	for _, qualification := range s.Qualifications {
		if qualification.ExpiresAt.IsZero() {
			continue
		}
		if !qualification.ExpiresAt.After(deadline) {
			out = append(out, qualification)
		}
	}
	return out
}

func (s *Supplier) CanUseForPurchase(now time.Time) error {
	if s == nil {
		return fmt.Errorf("supplier is required")
	}
	if s.Status == SupplierStatusDisabled {
		return fmt.Errorf("supplier is disabled")
	}
	for _, qualification := range s.Qualifications {
		if !qualification.ExpiresAt.IsZero() && qualification.ExpiresAt.Before(now) {
			return fmt.Errorf("supplier qualification expired: %s", qualification.Name)
		}
	}
	return nil
}

func NormalizeSupplierContacts(items []SupplierContact) ([]SupplierContact, error) {
	out := make([]SupplierContact, 0, len(items))
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

func NormalizeSupplierQualifications(items []SupplierQualification) ([]SupplierQualification, error) {
	out := make([]SupplierQualification, 0, len(items))
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
		attachments, err := NormalizeSupplierAttachments(item.Attachments)
		if err != nil {
			return nil, err
		}
		item.Attachments = attachments
		seen[item.ID] = struct{}{}
		out = append(out, item)
	}
	return out, nil
}

func NormalizeSupplierAttachments(items []SupplierAttachment) ([]SupplierAttachment, error) {
	out := make([]SupplierAttachment, 0, len(items))
	for _, item := range items {
		item.FileID = strings.TrimSpace(item.FileID)
		item.FileName = strings.TrimSpace(item.FileName)
		item.MimeType = strings.TrimSpace(item.MimeType)
		if item.FileID == "" {
			return nil, fmt.Errorf("attachment fileId is required")
		}
		if item.FileName == "" {
			return nil, fmt.Errorf("attachment fileName is required")
		}
		if item.Size < 0 {
			return nil, fmt.Errorf("attachment size is invalid")
		}
		if strings.Contains(item.FileName, "..") || strings.ContainsAny(item.FileName, `/\`) {
			return nil, fmt.Errorf("attachment fileName is unsafe")
		}
		ext := strings.ToLower(filepath.Ext(item.FileName))
		if ext == ".exe" || ext == ".bat" || ext == ".cmd" || ext == ".ps1" || ext == ".sh" {
			return nil, fmt.Errorf("attachment extension is not allowed")
		}
		out = append(out, item)
	}
	return out, nil
}

func validateSupplierInput(in SupplierInput) error {
	if strings.TrimSpace(in.Code) == "" {
		return fmt.Errorf("code is required")
	}
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if in.Rating < 0 || in.Rating > 5 {
		return fmt.Errorf("rating must be between 0 and 5")
	}
	return nil
}

func normalizeSupplierRating(rating int) int {
	if rating == 0 {
		return 3
	}
	return rating
}
