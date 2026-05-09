package repository

import (
	"context"

	"github.com/tinboxw/skoll/internal/domain/audit"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

type AuditRepository interface {
	Append(ctx context.Context, record *audit.Record) error
	GetByID(ctx context.Context, id shared.ID) (*audit.Record, error)
	ListByActor(ctx context.Context, actorID shared.ID, limit int) ([]*audit.Record, error)
	ListByTimeRange(ctx context.Context, tr shared.TimeRange, limit int) ([]*audit.Record, error)
}
