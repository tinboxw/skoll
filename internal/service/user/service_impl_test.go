package user

import (
	"context"
	"strings"
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

func TestUserServiceCreateBatchAtomicStopsOnFirstError(t *testing.T) {
	bundle, err := store.NewBundle(store.Options{Mode: store.ModeMemory})
	if err != nil {
		t.Fatalf("store.NewBundle error: %v", err)
	}

	svc := NewService(bundle.Users, bundle.Audit, bundle.UnitOfWork)
	results, err := svc.CreateBatch(context.Background(), BatchCreateInput{
		Atomic: true,
		Items: []CreateUserInput{
			{
				Account:      "batch_ok",
				Name:         "Batch OK",
				Email:        "batch_ok@example.com",
				PasswordHash: "1234567890abcdef",
				ActorID:      "admin-1",
			},
			{
				Account:      "batch_bad",
				Name:         "Batch Bad",
				Email:        "batch_bad@example.com",
				PasswordHash: "short",
				ActorID:      "admin-1",
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateBatch returned unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if !results[0].Success {
		t.Fatalf("expected first item success, got %+v", results[0])
	}
	if results[1].Success {
		t.Fatalf("expected second item failed, got %+v", results[1])
	}
	if !strings.Contains(strings.ToLower(results[1].Message), "password") {
		t.Fatalf("expected password validation failure message, got %q", results[1].Message)
	}

	list, err := svc.List(context.Background(), ListInput{Offset: 0, Limit: 20})
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected one persisted user before stop, got %d", len(list))
	}
	if list[0].Account != "batch_ok" {
		t.Fatalf("expected first user to persist, got account=%q", list[0].Account)
	}
}

func TestUserServiceUpdateRejectsUnsupportedStatus(t *testing.T) {
	bundle, err := store.NewBundle(store.Options{Mode: store.ModeMemory})
	if err != nil {
		t.Fatalf("store.NewBundle error: %v", err)
	}

	svc := NewService(bundle.Users, bundle.Audit, bundle.UnitOfWork)
	created, err := svc.Create(context.Background(), CreateUserInput{
		Account:      "update_status_user",
		Name:         "Update Status User",
		Email:        "update_status@example.com",
		PasswordHash: "1234567890abcdef",
		ActorID:      "admin-1",
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}

	_, err = svc.Update(context.Background(), UpdateUserInput{ID: created.ID.String(), Status: "paused"})
	if err == nil {
		t.Fatalf("expected unsupported status error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "unsupported status") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUserServiceDeleteRequiresID(t *testing.T) {
	bundle, err := store.NewBundle(store.Options{Mode: store.ModeMemory})
	if err != nil {
		t.Fatalf("store.NewBundle error: %v", err)
	}

	svc := NewService(bundle.Users, bundle.Audit, bundle.UnitOfWork)
	err = svc.Delete(context.Background(), "   ")
	if err == nil {
		t.Fatalf("expected id required error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "id is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUserServiceUpdateNotFound(t *testing.T) {
	bundle, err := store.NewBundle(store.Options{Mode: store.ModeMemory})
	if err != nil {
		t.Fatalf("store.NewBundle error: %v", err)
	}

	svc := NewService(bundle.Users, bundle.Audit, bundle.UnitOfWork)
	_, err = svc.Update(context.Background(), UpdateUserInput{ID: "missing-id", Name: "N"})
	if err == nil {
		t.Fatalf("expected user not found error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "user not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}
