package sql

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/tinboxw/skoll/internal/repository"
)

func TestUnitOfWorkDoWithoutDB(t *testing.T) {
	uow := NewUnitOfWork()
	ctx := context.WithValue(context.Background(), "k", "v")
	called := false

	err := uow.Do(ctx, func(txCtx repository.Tx) error {
		called = true
		if txCtx.Context().Value("k") != "v" {
			t.Fatalf("expected context value to be preserved")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Do unexpected error: %v", err)
	}
	if !called {
		t.Fatalf("expected callback called")
	}
}

func TestUnitOfWorkDoPropagatesCallbackError(t *testing.T) {
	uow := NewUnitOfWork()
	expectedErr := errors.New("callback failed")

	err := uow.Do(context.Background(), func(_ repository.Tx) error {
		return expectedErr
	})
	if err == nil {
		t.Fatalf("expected callback error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "callback failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnitOfWorkDoWithNilReceiver(t *testing.T) {
	var uow *UnitOfWork
	called := false

	err := uow.Do(context.Background(), func(_ repository.Tx) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !called {
		t.Fatalf("expected callback to be called")
	}
}

func TestNewUnitOfWorkWithNilDBFallsBackToNoDBPath(t *testing.T) {
	uow := NewUnitOfWorkWithDB(nil)
	called := false

	err := uow.Do(context.Background(), func(_ repository.Tx) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !called {
		t.Fatalf("expected callback to be called")
	}
}
