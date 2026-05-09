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
	ID          shared.ID
	Username    string
	DisplayName string
	Email       Email
	Status      Status
	Password    PasswordHash
	Meta        shared.AuditMeta
}

func New(id shared.ID, username, displayName, email string, now time.Time) (*User, error) {
	if id.IsZero() {
		return nil, fmt.Errorf("id is required")
	}
	if err := ValidateUsername(username); err != nil {
		return nil, err
	}
	if err := ValidateDisplayName(displayName); err != nil {
		return nil, err
	}
	e, err := NewEmail(email)
	if err != nil {
		return nil, err
	}

	u := &User{
		ID:          id,
		Username:    strings.TrimSpace(username),
		DisplayName: strings.TrimSpace(displayName),
		Email:       e,
		Status:      StatusActive,
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

func (u *User) Rename(displayName string, now time.Time) error {
	if err := ValidateDisplayName(displayName); err != nil {
		return err
	}
	u.DisplayName = strings.TrimSpace(displayName)
	u.Meta.Touch(now)
	return nil
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
