package user

import (
	"errors"
	"strings"
)

type Email string

func ParseEmail(raw string) (Email, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" || !strings.Contains(raw, "@") {
		return "", errors.New("invalid email")
	}
	return Email(raw), nil
}
