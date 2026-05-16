package system

import (
	"context"

	domainsystem "github.com/tinboxw/skoll/internal/domain/system"
)

type Service interface {
	Upsert(ctx context.Context, in UpsertInput) (*domainsystem.Setting, error)
	GetByKey(ctx context.Context, key string) (*domainsystem.Setting, error)
	List(ctx context.Context, in ListInput) ([]*domainsystem.Setting, error)
	Reset(ctx context.Context) (int, error)
}
