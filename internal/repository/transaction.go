package repository

import "context"

// Tx defines a minimal transaction context passed through repository operations.
type Tx interface {
	Context() context.Context
}

// UnitOfWork executes operations in a transaction boundary.
type UnitOfWork interface {
	Do(ctx context.Context, fn func(tx Tx) error) error
}
