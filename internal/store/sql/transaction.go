package sql

import (
	"context"

	"github.com/tinboxw/skoll/internal/repository"
	"gorm.io/gorm"
)

type tx struct {
	ctx context.Context
	db  *gorm.DB
}

func (t tx) Context() context.Context {
	return t.ctx
}

type UnitOfWork struct {
	db *gorm.DB
}

func NewUnitOfWork() *UnitOfWork {
	return &UnitOfWork{db: nil}
}

func NewUnitOfWorkWithDB(db *gorm.DB) *UnitOfWork {
	return &UnitOfWork{db: db}
}

func (u *UnitOfWork) Do(ctx context.Context, fn func(tx repository.Tx) error) error {
	if u == nil || u.db == nil {
		return fn(tx{ctx: ctx, db: nil})
	}

	txdb := u.db.WithContext(ctx).Begin()
	if txdb.Error != nil {
		return txdb.Error
	}

	if err := fn(tx{ctx: ctx, db: txdb}); err != nil {
		_ = txdb.Rollback().Error
		return err
	}

	return txdb.Commit().Error
}
