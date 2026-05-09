package user

import (
	"fmt"
	"net/mail"
	"strings"
)

type Email string

func NewEmail(raw string) (Email, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return "", fmt.Errorf("email is required")
	}
	if _, err := mail.ParseAddress(normalized); err != nil {
		return "", fmt.Errorf("invalid email: %w", err)
	}
	return Email(normalized), nil
}

func (e Email) String() string {
	return string(e)
}

type PasswordHash string

func NewPasswordHash(raw string) (PasswordHash, error) {
	v := strings.TrimSpace(raw)
	if len(v) < 16 {
		return "", fmt.Errorf("password hash is too short")
	}
	return PasswordHash(v), nil
}

func (h PasswordHash) String() string {
	return string(h)
}
