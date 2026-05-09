package user

import (
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

func TestNewUserAndChangeEmail(t *testing.T) {
	now := time.Now().UTC()
	u, err := New(shared.ID("u-1"), "alice", "Alice", "alice@example.com", now)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	if !u.IsActive() {
		t.Fatalf("expected active user")
	}

	if err := u.ChangeEmail("alice+new@example.com", now.Add(time.Minute)); err != nil {
		t.Fatalf("ChangeEmail error: %v", err)
	}
	if got := u.Email.String(); got != "alice+new@example.com" {
		t.Fatalf("unexpected email: %s", got)
	}
}

func TestNewUserRejectsInvalidUsername(t *testing.T) {
	_, err := New(shared.ID("u-1"), "@bad", "Alice", "alice@example.com", time.Now().UTC())
	if err == nil {
		t.Fatalf("expected username validation error")
	}
}
