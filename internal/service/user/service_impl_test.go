package user

import (
	"context"
	"testing"

	"github.com/tinboxw/skoll/internal/store"
)

func TestUserServiceCreateAndDisable(t *testing.T) {
	bundle, err := store.NewBundle(store.Options{Mode: store.ModeMemory})
	if err != nil {
		t.Fatalf("store.NewBundle error: %v", err)
	}

	svc := NewService(bundle.Users, bundle.Audit, bundle.UnitOfWork)
	created, err := svc.Create(context.Background(), CreateUserInput{
		Account:      "svc_user",
		Name:         "Service User",
		Email:        "svc@example.com",
		PasswordHash: "1234567890abcdef",
		ActorID:      "admin-1",
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("expected generated id")
	}

	if err := svc.Disable(context.Background(), created.ID.String(), "admin-1"); err != nil {
		t.Fatalf("Disable error: %v", err)
	}

	got, err := svc.Get(context.Background(), created.ID.String())
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got == nil || got.IsActive() {
		t.Fatalf("expected disabled user")
	}
}
