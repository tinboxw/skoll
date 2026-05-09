package audit

import (
	"context"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
)

type Service interface {
	Append(ctx context.Context, actorID, action, resource, resourceID string, detail map[string]any) (*domainaudit.Record, error)
	ListByActor(ctx context.Context, actorID string, limit int) ([]*domainaudit.Record, error)
}
