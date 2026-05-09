package audit

import (
	"context"
	"strconv"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	"github.com/tinboxw/skoll/internal/domain/shared"
	"github.com/tinboxw/skoll/internal/repository"
)

type serviceImpl struct {
	repo  repository.AuditRepository
	nowFn func() time.Time
	idFn  func(prefix string) shared.ID
}

func NewService(repo repository.AuditRepository) Service {
	return &serviceImpl{
		repo:  repo,
		nowFn: func() time.Time { return time.Now().UTC() },
		idFn: func(prefix string) shared.ID {
			return shared.ID(prefix + "-" + strconv.FormatInt(time.Now().UTC().UnixNano(), 10))
		},
	}
}

func (s *serviceImpl) Append(ctx context.Context, actorID, action, resource, resourceID string, detail map[string]any) (*domainaudit.Record, error) {
	rec, err := domainaudit.NewRecord(s.idFn("audit"), shared.ID(actorID), action, resource, resourceID, detail, s.nowFn())
	if err != nil {
		return nil, err
	}
	if err := s.repo.Append(ctx, rec); err != nil {
		return nil, err
	}
	return rec, nil
}

func (s *serviceImpl) ListByActor(ctx context.Context, actorID string, limit int) ([]*domainaudit.Record, error) {
	return s.repo.ListByActor(ctx, shared.ID(actorID), limit)
}
