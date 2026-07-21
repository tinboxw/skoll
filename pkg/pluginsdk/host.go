package pluginsdk

import (
	"context"
	"fmt"
)

type Transaction interface {
	Context() context.Context
}

type TransactionService interface {
	Within(ctx context.Context, fn func(Transaction) error) error
}

type Permission struct {
	Resource string
	Action   string
}

type DataScopeService interface {
	Resolve(ctx context.Context, permission Permission) (ScopePredicate, error)
}

type HostServices struct {
	Transactions TransactionService
	DataScopes   DataScopeService
}

func (s HostServices) Validate() error {
	if s.Transactions == nil {
		return fmt.Errorf("plugin host transaction service is required")
	}
	if s.DataScopes == nil {
		return fmt.Errorf("plugin host data-scope service is required")
	}
	return nil
}
