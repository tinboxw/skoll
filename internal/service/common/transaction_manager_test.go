package common

import (
	"context"
	"strings"
	"testing"

	"github.com/tinboxw/skoll/internal/repository"
)

type transactionManagerUnitOfWork struct {
	called bool
}

func (u *transactionManagerUnitOfWork) Do(_ context.Context, fn func(repository.Tx) error) error {
	u.called = true
	return fn(transactionManagerTx{})
}

type transactionManagerTx struct{}

func (transactionManagerTx) Context() context.Context { return context.Background() }

func TestTransactionManagerFailsClosedWithoutUnitOfWork(t *testing.T) {
	callbackCalled := false
	err := NewTransactionManager(nil).InTx(context.Background(), func(repository.Tx) error {
		callbackCalled = true
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "unit of work") {
		t.Fatalf("expected missing unit of work error, got %v", err)
	}
	if callbackCalled {
		t.Fatal("callback executed without a transaction boundary")
	}
}

func TestTransactionManagerRejectsNilCallback(t *testing.T) {
	err := NewTransactionManager(&transactionManagerUnitOfWork{}).InTx(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "callback") {
		t.Fatalf("expected missing callback error, got %v", err)
	}
}

func TestTransactionManagerDelegatesToUnitOfWork(t *testing.T) {
	uow := &transactionManagerUnitOfWork{}
	callbackCalled := false
	err := NewTransactionManager(uow).InTx(context.Background(), func(tx repository.Tx) error {
		callbackCalled = tx != nil && tx.Context() != nil
		return nil
	})
	if err != nil {
		t.Fatalf("run transaction: %v", err)
	}
	if !uow.called || !callbackCalled {
		t.Fatal("transaction manager did not delegate a valid transaction")
	}
}
