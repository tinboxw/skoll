package pharmaoa

import (
	"fmt"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

type EmployeeStatus string

const (
	EmployeeStatusActive  EmployeeStatus = "active"
	EmployeeStatusOnLeave EmployeeStatus = "on_leave"
	EmployeeStatusLeft    EmployeeStatus = "left"
)

type EmployeeCertificate struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Number    string    `json:"number"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type Employee struct {
	ID           shared.ID             `json:"id"`
	Code         string                `json:"code"`
	Name         string                `json:"name"`
	DepartmentID string                `json:"departmentId"`
	PositionID   string                `json:"positionId"`
	Phone        string                `json:"phone"`
	Email        string                `json:"email"`
	Status       EmployeeStatus        `json:"status"`
	LeaveReason  string                `json:"leaveReason,omitempty"`
	Certificates []EmployeeCertificate `json:"certificates"`
	Meta         shared.AuditMeta      `json:"meta"`
}

type EmployeeInput struct {
	Code         string
	Name         string
	DepartmentID string
	PositionID   string
	Phone        string
	Email        string
	Certificates []EmployeeCertificate
}

func NewEmployee(id shared.ID, in EmployeeInput, now time.Time) (*Employee, error) {
	if id.IsZero() {
		return nil, fmt.Errorf("id is required")
	}
	if err := validateEmployeeRequired(in.Code, "code"); err != nil {
		return nil, err
	}
	if err := validateEmployeeRequired(in.Name, "name"); err != nil {
		return nil, err
	}
	if err := validateEmployeeRequired(in.DepartmentID, "departmentId"); err != nil {
		return nil, err
	}
	if err := validateEmployeeRequired(in.PositionID, "positionId"); err != nil {
		return nil, err
	}
	certificates, err := NormalizeCertificates(in.Certificates)
	if err != nil {
		return nil, err
	}
	entity := &Employee{
		ID:           id,
		Code:         strings.TrimSpace(in.Code),
		Name:         strings.TrimSpace(in.Name),
		DepartmentID: strings.TrimSpace(in.DepartmentID),
		PositionID:   strings.TrimSpace(in.PositionID),
		Phone:        strings.TrimSpace(in.Phone),
		Email:        strings.TrimSpace(in.Email),
		Status:       EmployeeStatusActive,
		Certificates: certificates,
	}
	entity.Meta.Touch(now)
	return entity, nil
}

func (e *Employee) Update(in EmployeeInput, now time.Time) error {
	if e == nil {
		return fmt.Errorf("employee is required")
	}
	if err := validateEmployeeRequired(in.Code, "code"); err != nil {
		return err
	}
	if err := validateEmployeeRequired(in.Name, "name"); err != nil {
		return err
	}
	if err := validateEmployeeRequired(in.DepartmentID, "departmentId"); err != nil {
		return err
	}
	if err := validateEmployeeRequired(in.PositionID, "positionId"); err != nil {
		return err
	}
	certificates, err := NormalizeCertificates(in.Certificates)
	if err != nil {
		return err
	}
	e.Code = strings.TrimSpace(in.Code)
	e.Name = strings.TrimSpace(in.Name)
	e.DepartmentID = strings.TrimSpace(in.DepartmentID)
	e.PositionID = strings.TrimSpace(in.PositionID)
	e.Phone = strings.TrimSpace(in.Phone)
	e.Email = strings.TrimSpace(in.Email)
	e.Certificates = certificates
	e.Meta.Touch(now)
	return nil
}

func (e *Employee) MarkLeft(reason string, now time.Time) error {
	if e == nil {
		return fmt.Errorf("employee is required")
	}
	if e.Status == EmployeeStatusLeft {
		return nil
	}
	e.Status = EmployeeStatusLeft
	e.LeaveReason = strings.TrimSpace(reason)
	e.Meta.Touch(now)
	return nil
}

func (e *Employee) CertificateExpiringBefore(deadline time.Time) []EmployeeCertificate {
	if e == nil || e.Status == EmployeeStatusLeft {
		return []EmployeeCertificate{}
	}
	out := make([]EmployeeCertificate, 0)
	for _, certificate := range e.Certificates {
		if certificate.ExpiresAt.IsZero() {
			continue
		}
		if !certificate.ExpiresAt.After(deadline) {
			out = append(out, certificate)
		}
	}
	return out
}

func NormalizeCertificates(items []EmployeeCertificate) ([]EmployeeCertificate, error) {
	out := make([]EmployeeCertificate, 0, len(items))
	seen := map[string]struct{}{}
	for idx, item := range items {
		item.ID = strings.TrimSpace(item.ID)
		item.Name = strings.TrimSpace(item.Name)
		item.Number = strings.TrimSpace(item.Number)
		if item.ID == "" {
			item.ID = fmt.Sprintf("cert-%d", idx+1)
		}
		if item.Name == "" {
			return nil, fmt.Errorf("certificate name is required")
		}
		if _, ok := seen[item.ID]; ok {
			return nil, fmt.Errorf("duplicate certificate id: %s", item.ID)
		}
		seen[item.ID] = struct{}{}
		out = append(out, item)
	}
	return out, nil
}

func validateEmployeeRequired(value string, field string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", field)
	}
	return nil
}
