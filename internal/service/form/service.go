package form

import (
	"context"
	"time"

	domainform "github.com/tinboxw/skoll/internal/domain/form"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

type Repository interface {
	SaveSchema(ctx context.Context, schema domainform.Schema) error
	GetSchema(ctx context.Context, id shared.ID) (*domainform.Schema, error)
	GetSchemaByKey(ctx context.Context, key string, version int) (*domainform.Schema, error)
}

type Service interface {
	CreateSchema(ctx context.Context, in CreateSchemaInput) (*domainform.Schema, error)
	GetSchema(ctx context.Context, id shared.ID) (*domainform.Schema, error)
	GetSchemaByKey(ctx context.Context, key string, version int) (*domainform.Schema, error)
}

type CreateSchemaInput struct {
	ID           shared.ID
	Key          string
	Name         string
	Version      int
	BusinessType string
	Description  string
	Fields       []domainform.Field
	Now          time.Time
}
