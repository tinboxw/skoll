package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	"github.com/tinboxw/skoll/internal/domain/shared"
	auditrepo "github.com/tinboxw/skoll/internal/repository/audit"
)

type AuditEventStore struct {
	mu    sync.RWMutex
	items map[string]*domainaudit.Event
}

func NewAuditEventStore() *AuditEventStore {
	return &AuditEventStore{items: make(map[string]*domainaudit.Event)}
}

func (s *AuditEventStore) AppendEvent(_ context.Context, event *domainaudit.Event) error {
	if event == nil || event.ID.IsZero() {
		return nil
	}
	cloned := cloneAuditEvent(event)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[cloned.ID.String()] = cloned
	return nil
}

func (s *AuditEventStore) GetEventByID(_ context.Context, id shared.ID) (*domainaudit.Event, error) {
	key := strings.TrimSpace(id.String())
	if key == "" {
		return nil, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.items[key]
	if !ok {
		return nil, nil
	}
	return cloneAuditEvent(item), nil
}

func (s *AuditEventStore) ListEvents(_ context.Context, filter auditrepo.EventFilter) ([]*domainaudit.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]*domainaudit.Event, 0, len(s.items))
	for _, item := range s.items {
		if matchesAuditEventFilter(item, filter) {
			items = append(items, cloneAuditEvent(item))
		}
	}
	sortAuditEvents(items)
	return paginateAuditEvents(items, filter.Offset, filter.Limit), nil
}

func (s *AuditEventStore) ExportEventSourceData(_ context.Context, filter auditrepo.EventFilter) ([]auditrepo.EventSourceData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]*domainaudit.Event, 0, len(s.items))
	for _, item := range s.items {
		if matchesAuditEventFilter(item, filter) {
			items = append(items, cloneAuditEvent(item))
		}
	}
	sortAuditEvents(items)
	items = paginateAuditEvents(items, filter.Offset, filter.Limit)

	out := make([]auditrepo.EventSourceData, 0, len(items))
	for _, item := range items {
		out = append(out, auditrepo.EventSourceData{
			EventID:    item.ID,
			SourceData: cloneAuditMap(item.SourceData),
		})
	}
	return out, nil
}

func matchesAuditEventFilter(event *domainaudit.Event, filter auditrepo.EventFilter) bool {
	if event == nil {
		return false
	}
	if filter.Type != "" && event.Type != filter.Type {
		return false
	}
	if !filter.ActorID.IsZero() && event.Actor.ID != filter.ActorID {
		return false
	}
	if filter.Action != "" && event.Action != filter.Action {
		return false
	}
	if strings.TrimSpace(filter.ResourceType) != "" && event.Resource.Type != strings.TrimSpace(strings.ToLower(filter.ResourceType)) {
		return false
	}
	if strings.TrimSpace(filter.ResourceID) != "" && event.Resource.ID != strings.TrimSpace(filter.ResourceID) {
		return false
	}
	if filter.Result != "" && event.Result != filter.Result {
		return false
	}
	if filter.Risk != "" && event.Risk != filter.Risk {
		return false
	}
	if filter.TimeRange.IsValid() && (event.OccurredAt.Before(filter.TimeRange.From) || event.OccurredAt.After(filter.TimeRange.To)) {
		return false
	}
	return true
}

func sortAuditEvents(items []*domainaudit.Event) {
	sort.SliceStable(items, func(i, j int) bool {
		if !items[i].OccurredAt.Equal(items[j].OccurredAt) {
			return items[i].OccurredAt.After(items[j].OccurredAt)
		}
		return items[i].ID.String() < items[j].ID.String()
	})
}

func paginateAuditEvents(items []*domainaudit.Event, offset, limit int) []*domainaudit.Event {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = len(items)
	}
	if offset > len(items) {
		return []*domainaudit.Event{}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return append([]*domainaudit.Event(nil), items[offset:end]...)
}

func cloneAuditEvent(event *domainaudit.Event) *domainaudit.Event {
	if event == nil {
		return nil
	}
	cloned := *event
	cloned.Metadata = cloneAuditMap(event.Metadata)
	cloned.SourceData = cloneAuditMap(event.SourceData)
	return &cloned
}

func cloneAuditMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}
