package sql

import (
	"context"
	"fmt"

	"github.com/tinboxw/skoll/internal/repository"
	"gorm.io/gorm"
)

type tx struct {
	ctx context.Context
	db  *gorm.DB
}

type transactionDBContextKey struct{}

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
	if db == nil {
		panic("sql unit of work database is required")
	}
	return &UnitOfWork{db: db}
}

func (u *UnitOfWork) Do(ctx context.Context, fn func(tx repository.Tx) error) error {
	if u == nil {
		return fmt.Errorf("unit of work is required")
	}
	if fn == nil {
		return fmt.Errorf("unit of work callback is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if scoped := DBFromContext(ctx); scoped != nil {
		return fn(tx{ctx: ctx, db: scoped})
	}
	if u.db == nil {
		return fn(tx{ctx: ctx, db: nil})
	}
	return u.db.WithContext(ctx).Transaction(func(txdb *gorm.DB) error {
		txContext := context.WithValue(ctx, transactionDBContextKey{}, txdb)
		return fn(tx{ctx: txContext, db: txdb})
	})
}

func DBFromContext(ctx context.Context) *gorm.DB {
	if ctx == nil {
		return nil
	}
	db, _ := ctx.Value(transactionDBContextKey{}).(*gorm.DB)
	return db
}

func ResolveDB(ctx context.Context, fallback *gorm.DB) *gorm.DB {
	if scoped := DBFromContext(ctx); scoped != nil {
		return scoped.WithContext(ctx)
	}
	if fallback == nil {
		return nil
	}
	return fallback.WithContext(ctx)
}
