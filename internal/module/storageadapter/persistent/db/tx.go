package db

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// WithTx executes fn inside a transaction. If fn returns an error, the
// transaction is rolled back and the error is propagated. Nested calls reuse
// the existing transaction when one is already active on db.
func WithTx(ctx context.Context, db *gorm.DB, fn func(tx *gorm.DB) error) error {
	if db == nil {
		return fmt.Errorf("db.WithTx: nil db")
	}
	if fn == nil {
		return fmt.Errorf("db.WithTx: nil fn")
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})
}
