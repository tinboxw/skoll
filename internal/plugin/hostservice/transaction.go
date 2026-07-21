package hostservice

import (
	"context"
	"fmt"

	"github.com/tinboxw/skoll/internal/repository"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

type transactionService struct {
	uow repository.UnitOfWork
}

type transaction struct {
	ctx context.Context
}

func NewTransactionService(uow repository.UnitOfWork) (pluginsdk.TransactionService, error) {
	if uow == nil {
		return nil, fmt.Errorf("plugin host unit of work is required")
	}
	return &transactionService{uow: uow}, nil
}

func (s *transactionService) Within(ctx context.Context, fn func(pluginsdk.Transaction) error) error {
	if fn == nil {
		return fmt.Errorf("plugin transaction callback is required")
	}
	return s.uow.Do(ctx, func(tx repository.Tx) error {
		if tx == nil || tx.Context() == nil {
			return fmt.Errorf("plugin transaction context is unavailable")
		}
		return fn(transaction{ctx: tx.Context()})
	})
}

func (t transaction) Context() context.Context { return t.ctx }

var _ pluginsdk.TransactionService = (*transactionService)(nil)
