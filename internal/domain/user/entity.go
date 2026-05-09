package user

import (
	"errors"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

var ErrUserIDRequired = errors.New("user id is required")

type Status string

const (
	StatusActive   Status = "active"
	StatusDisabled Status = "disabled"
)

type User struct {
	ID           shared.ID
	Username     string
	Email        Email
	PasswordHash string
	Status       Status
	Audit        shared.AuditInfo
}

func New(id shared.ID, username string, email Email, passwordHash string, now time.Time) (User, error) {
	username = strings.TrimSpace(username)
	if id == "" {
		return User{}, ErrUserIDRequired
	}
	if err := ValidateUsername(username); err != nil {
		return User{}, err
	}
	if passwordHash == "" {
		return User{}, errors.New("password hash is required")
	}
	return User{
		ID:           id,
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		Status:       StatusActive,
		Audit: shared.AuditInfo{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}, nil
}

func (u *User) Disable(now time.Time) {
	u.Status = StatusDisabled
	u.Audit.UpdatedAt = now
}

func (u *User) Activate(now time.Time) {
	u.Status = StatusActive
	u.Audit.UpdatedAt = now
}
