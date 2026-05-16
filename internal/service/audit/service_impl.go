package audit

import (
	"context"
	"strconv"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	"github.com/tinboxw/skoll/internal/domain/shared"
	auditrepo "github.com/tinboxw/skoll/internal/repository/audit"
)

type serviceImpl struct {
	repo  auditrepo.AuditRepository
	nowFn func() time.Time
	idFn  func(prefix string) shared.ID
}

func NewService(repo auditrepo.AuditRepository) Service {
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

func (s *serviceImpl) GetByID(ctx context.Context, id string) (*domainaudit.Record, error) {
	return s.repo.GetByID(ctx, shared.ID(id))
}

func (s *serviceImpl) ListByTimeRange(ctx context.Context, from, to time.Time, limit int) ([]*domainaudit.Record, error) {
	tr := shared.TimeRange{From: from, To: to}
	if !tr.IsValid() {
		return nil, domainaudit.ErrInvalidTimeRange
	}
	return s.repo.ListByTimeRange(ctx, tr, limit)
}

func (s *serviceImpl) ClearByTimeRange(ctx context.Context, from, to time.Time) (int, error) {
	tr := shared.TimeRange{From: from, To: to}
	if !tr.IsValid() {
		return 0, domainaudit.ErrInvalidTimeRange
	}
	return s.repo.DeleteByTimeRange(ctx, tr)
}
