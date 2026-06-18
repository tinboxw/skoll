package audit

import (
	"context"
	"errors"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	"github.com/tinboxw/skoll/internal/domain/shared"
	auditrepo "github.com/tinboxw/skoll/internal/repository/audit"
)

var ErrEventRepositoryNotConfigured = errors.New("audit event repository is not configured")

type EventFilter struct {
	Type         domainaudit.EventType
	ActorID      string
	Action       domainaudit.AuditAction
	ResourceType string
	ResourceID   string
	Result       domainaudit.EventResult
	Risk         domainaudit.EventRisk
	From         time.Time
	To           time.Time
	Offset       int
	Limit        int
}

type EventSourceData struct {
	EventID    string
	SourceData map[string]any
}

type EventService interface {
	AppendEvent(ctx context.Context, event *domainaudit.Event) error
	GetEventByID(ctx context.Context, id string) (*domainaudit.Event, error)
	ListEvents(ctx context.Context, filter EventFilter) ([]*domainaudit.Event, error)
	ExportEventSourceData(ctx context.Context, filter EventFilter) ([]EventSourceData, error)
}

type eventServiceImpl struct {
	repo auditrepo.EventRepository
}

func NewEventService(repo auditrepo.EventRepository) EventService {
	return &eventServiceImpl{repo: repo}
}

func (s *eventServiceImpl) AppendEvent(ctx context.Context, event *domainaudit.Event) error {
	if err := s.ensureRepo(); err != nil {
		return err
	}
	if event == nil {
		return errors.New("audit event is required")
	}
	return s.repo.AppendEvent(ctx, event)
}

func (s *eventServiceImpl) GetEventByID(ctx context.Context, id string) (*domainaudit.Event, error) {
	if err := s.ensureRepo(); err != nil {
		return nil, err
	}
	return s.repo.GetEventByID(ctx, shared.ID(id))
}

func (s *eventServiceImpl) ListEvents(ctx context.Context, filter EventFilter) ([]*domainaudit.Event, error) {
	if err := s.ensureRepo(); err != nil {
		return nil, err
	}
	repoFilter, err := toRepoEventFilter(filter)
	if err != nil {
		return nil, err
	}
	return s.repo.ListEvents(ctx, repoFilter)
}

func (s *eventServiceImpl) ExportEventSourceData(ctx context.Context, filter EventFilter) ([]EventSourceData, error) {
	if err := s.ensureRepo(); err != nil {
		return nil, err
	}
	repoFilter, err := toRepoEventFilter(filter)
	if err != nil {
		return nil, err
	}
	items, err := s.repo.ExportEventSourceData(ctx, repoFilter)
	if err != nil {
		return nil, err
	}
	out := make([]EventSourceData, 0, len(items))
	for _, item := range items {
		out = append(out, EventSourceData{
			EventID:    item.EventID.String(),
			SourceData: cloneAnyMap(item.SourceData),
		})
	}
	return out, nil
}

func (s *eventServiceImpl) ensureRepo() error {
	if s == nil || s.repo == nil {
		return ErrEventRepositoryNotConfigured
	}
	return nil
}

func toRepoEventFilter(filter EventFilter) (auditrepo.EventFilter, error) {
	repoFilter := auditrepo.EventFilter{
		Type:         filter.Type,
		ActorID:      shared.ID(filter.ActorID),
		Action:       filter.Action,
		ResourceType: filter.ResourceType,
		ResourceID:   filter.ResourceID,
		Result:       filter.Result,
		Risk:         filter.Risk,
		Offset:       filter.Offset,
		Limit:        filter.Limit,
	}
	if !filter.From.IsZero() || !filter.To.IsZero() {
		tr := shared.TimeRange{From: filter.From, To: filter.To}
		if !tr.IsValid() {
			return auditrepo.EventFilter{}, domainaudit.ErrInvalidTimeRange
		}
		repoFilter.TimeRange = tr
	}
	return repoFilter, nil
}

func cloneAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
