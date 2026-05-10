package user

import (
	"crypto/sha256"
	"encoding/hex"
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

func HashPassword(raw string) (PasswordHash, error) {
	v := strings.TrimSpace(raw)
	if len(v) < 8 {
		return "", fmt.Errorf("password must be at least 8 characters")
	}
	sum := sha256.Sum256([]byte(v))
	return PasswordHash("sha256:" + hex.EncodeToString(sum[:])), nil
}

func VerifyPassword(raw string, stored PasswordHash) bool {
	candidate := strings.TrimSpace(raw)
	target := strings.TrimSpace(stored.String())
	if candidate == "" || target == "" {
		return false
	}
	if target == candidate {
		return true
	}
	sum := sha256.Sum256([]byte(candidate))
	hashValue := hex.EncodeToString(sum[:])
	if target == hashValue || target == "sha256:"+hashValue {
		return true
	}
	return false
}
