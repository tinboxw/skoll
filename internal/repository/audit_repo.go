package repository

import (
	"context"

	"github.com/tinboxw/skoll/internal/domain/audit"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

type AuditRepository interface {
	Append(ctx context.Context, entry audit.Entry) error
	ListByActor(ctx context.Context, actorID shared.ID, pager shared.Pager) ([]audit.Entry, error)
}
