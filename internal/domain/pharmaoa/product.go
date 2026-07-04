package pharmaoa

import (
	"fmt"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

type ProductStatus string

const (
	ProductStatusActive   ProductStatus = "active"
	ProductStatusDisabled ProductStatus = "disabled"
)

type ProductTemperature struct {
	Required   bool    `json:"required"`
	MinCelsius float64 `json:"minCelsius"`
	MaxCelsius float64 `json:"maxCelsius"`
}

type Product struct {
	ID             shared.ID          `json:"id"`
	Code           string             `json:"code"`
	Name           string             `json:"name"`
	Spec           string             `json:"spec"`
	DosageForm     string             `json:"dosageForm"`
	Manufacturer   string             `json:"manufacturer"`
	ApprovalNumber string             `json:"approvalNumber"`
	Temperature    ProductTemperature `json:"temperature"`
	Status         ProductStatus      `json:"status"`
	DisableReason  string             `json:"disableReason,omitempty"`
	Meta           shared.AuditMeta   `json:"meta"`
}

type ProductInput struct {
	Code           string
	Name           string
	Spec           string
	DosageForm     string
	Manufacturer   string
	ApprovalNumber string
	Temperature    ProductTemperature
}

func NewProduct(id shared.ID, in ProductInput, now time.Time) (*Product, error) {
	if id.IsZero() {
		return nil, fmt.Errorf("id is required")
	}
	if err := validateProductInput(in); err != nil {
		return nil, err
	}
	entity := &Product{
		ID:             id,
		Code:           strings.TrimSpace(in.Code),
		Name:           strings.TrimSpace(in.Name),
		Spec:           strings.TrimSpace(in.Spec),
		DosageForm:     strings.TrimSpace(in.DosageForm),
		Manufacturer:   strings.TrimSpace(in.Manufacturer),
		ApprovalNumber: strings.TrimSpace(in.ApprovalNumber),
		Temperature:    normalizeProductTemperature(in.Temperature),
		Status:         ProductStatusActive,
	}
	entity.Meta.Touch(now)
	return entity, nil
}

func (p *Product) Update(in ProductInput, now time.Time) error {
	if p == nil {
		return fmt.Errorf("product is required")
	}
	if err := validateProductInput(in); err != nil {
		return err
	}
	p.Code = strings.TrimSpace(in.Code)
	p.Name = strings.TrimSpace(in.Name)
	p.Spec = strings.TrimSpace(in.Spec)
	p.DosageForm = strings.TrimSpace(in.DosageForm)
	p.Manufacturer = strings.TrimSpace(in.Manufacturer)
	p.ApprovalNumber = strings.TrimSpace(in.ApprovalNumber)
	p.Temperature = normalizeProductTemperature(in.Temperature)
	p.Meta.Touch(now)
	return nil
}

func (p *Product) Disable(reason string, now time.Time) error {
	if p == nil {
		return fmt.Errorf("product is required")
	}
	if p.Status == ProductStatusDisabled {
		return nil
	}
	p.Status = ProductStatusDisabled
	p.DisableReason = strings.TrimSpace(reason)
	p.Meta.Touch(now)
	return nil
}

func validateProductInput(in ProductInput) error {
	for field, value := range map[string]string{
		"code":           in.Code,
		"name":           in.Name,
		"spec":           in.Spec,
		"dosageForm":     in.DosageForm,
		"manufacturer":   in.Manufacturer,
		"approvalNumber": in.ApprovalNumber,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", field)
		}
	}
	temperature := normalizeProductTemperature(in.Temperature)
	if temperature.Required && temperature.MinCelsius > temperature.MaxCelsius {
		return fmt.Errorf("temperature min cannot exceed max")
	}
	return nil
}

func normalizeProductTemperature(value ProductTemperature) ProductTemperature {
	if !value.Required {
		value.MinCelsius = 0
		value.MaxCelsius = 0
	}
	return value
}
