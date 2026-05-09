package sql

import (
	"context"

	"github.com/tinboxw/skoll/internal/repository"
)

type tx struct {
	ctx context.Context
}

func (t tx) Context() context.Context {
	return t.ctx
}

type UnitOfWork struct{}

func NewUnitOfWork() *UnitOfWork {
	return &UnitOfWork{}
}

func (u *UnitOfWork) Do(ctx context.Context, fn func(tx repository.Tx) error) error {
	return fn(tx{ctx: ctx})
}
