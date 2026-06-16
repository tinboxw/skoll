package user

import (
	"fmt"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

type Status string

const (
	StatusActive   Status = "active"
	StatusDisabled Status = "disabled"
)

type User struct {
	ID           shared.ID
	Account      string
	Name         string
	Email        Email
	Status       Status
	Password     PasswordHash
	DepartmentID string
	PositionID   string
	Meta         shared.AuditMeta
}

func New(id shared.ID, account, name, email string, now time.Time) (*User, error) {
	if id.IsZero() {
		return nil, fmt.Errorf("id is required")
	}
	if err := ValidateAccount(account); err != nil {
		return nil, err
	}
	if err := ValidateName(name); err != nil {
		return nil, err
	}
	e, err := NewEmail(email)
	if err != nil {
		return nil, err
	}

	u := &User{
		ID:      id,
		Account: strings.TrimSpace(account),
		Name:    strings.TrimSpace(name),
		Email:   e,
		Status:  StatusActive,
	}
	u.Meta.Touch(now)
	return u, nil
}

func (u *User) SetPasswordHash(hash string) error {
	ph, err := NewPasswordHash(hash)
	if err != nil {
		return err
	}
	u.Password = ph
	return nil
}

func (u *User) ChangeEmail(email string, now time.Time) error {
	e, err := NewEmail(email)
	if err != nil {
		return err
	}
	u.Email = e
	u.Meta.Touch(now)
	return nil
}

func (u *User) Rename(name string, now time.Time) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	u.Name = strings.TrimSpace(name)
	u.Meta.Touch(now)
	return nil
}

func (u *User) SetOrganization(departmentID, positionID string, now time.Time) {
	u.DepartmentID = strings.TrimSpace(departmentID)
	u.PositionID = strings.TrimSpace(positionID)
	u.Meta.Touch(now)
}

func (u *User) Disable(now time.Time) {
	u.Status = StatusDisabled
	u.Meta.Touch(now)
}

func (u *User) Activate(now time.Time) {
	u.Status = StatusActive
	u.Meta.Touch(now)
}

func (u *User) IsActive() bool {
	return u.Status == StatusActive
}
