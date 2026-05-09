package persistent

import (
	"context"
	"errors"
	"testing"
)

func TestAdapterClose_NoCloseFn(t *testing.T) {
	a := &Adapter{}
	if err := a.Close(context.Background()); err != nil {
		t.Fatalf("close without closeFn should be no-op: %v", err)
	}
}

func TestAdapterClose_DelegatesToCloseFn(t *testing.T) {
	called := false
	expectedErr := errors.New("close failed")
	a := &Adapter{
		closeFn: func() error {
			called = true
			return expectedErr
		},
	}

	err := a.Close(context.Background())
	if !called {
		t.Fatalf("expected closeFn to be called")
	}
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected close error %v, got %v", expectedErr, err)
	}
}
