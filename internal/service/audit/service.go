package audit

import (
	"context"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
)

type Service interface {
	Append(ctx context.Context, actorID, action, resource, resourceID string, detail map[string]any) (*domainaudit.Record, error)
	GetByID(ctx context.Context, id string) (*domainaudit.Record, error)
	ListByActor(ctx context.Context, actorID string, limit int) ([]*domainaudit.Record, error)
	ListByTimeRange(ctx context.Context, from, to time.Time, limit int) ([]*domainaudit.Record, error)
	ClearByTimeRange(ctx context.Context, from, to time.Time) (int, error)
}
