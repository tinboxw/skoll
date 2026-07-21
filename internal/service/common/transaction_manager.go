package common

import (
	"context"
	"fmt"

	"github.com/tinboxw/skoll/internal/repository"
)

type TransactionManager struct {
	uow repository.UnitOfWork
}

func NewTransactionManager(uow repository.UnitOfWork) *TransactionManager {
	return &TransactionManager{uow: uow}
}

func (m *TransactionManager) InTx(ctx context.Context, fn func(tx repository.Tx) error) error {
	if m == nil || m.uow == nil {
		return fmt.Errorf("transaction unit of work is required")
	}
	if fn == nil {
		return fmt.Errorf("transaction callback is required")
	}
	return m.uow.Do(ctx, fn)
}
