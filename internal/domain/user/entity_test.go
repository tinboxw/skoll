package user

import (
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

func TestNewUser(t *testing.T) {
	email, err := ParseEmail("dev@example.com")
	if err != nil {
		t.Fatalf("ParseEmail() error = %v", err)
	}

	now := time.Now()
	u, err := New(shared.ID("u-1"), "devuser", email, "hashed", now)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if u.Status != StatusActive {
		t.Fatalf("status = %v, want %v", u.Status, StatusActive)
	}
}

func TestValidateUsername(t *testing.T) {
	if err := ValidateUsername("ab"); err == nil {
		t.Fatalf("expected validation error for short username")
	}
	if err := ValidateUsername("valid-user"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
