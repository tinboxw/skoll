package common

import (
	"context"

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
		return fn(nil)
	}
	return m.uow.Do(ctx, fn)
}
