package clickhouse

import (
	"context"
	"sort"
	"sync"

	"github.com/tinboxw/skoll/internal/domain/audit"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

type AuditStore struct {
	mu    sync.RWMutex
	items map[shared.ID]*audit.Record
}

func NewAuditStore() *AuditStore {
	return &AuditStore{items: map[shared.ID]*audit.Record{}}
}

func (s *AuditStore) Append(_ context.Context, record *audit.Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[record.ID] = record
	return nil
}

func (s *AuditStore) GetByID(_ context.Context, id shared.ID) (*audit.Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.items[id], nil
}

func (s *AuditStore) ListByActor(_ context.Context, actorID shared.ID, limit int) ([]*audit.Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*audit.Record, 0)
	for _, r := range s.items {
		if r.ActorID == actorID {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].OccurredAt.After(out[j].OccurredAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *AuditStore) ListByTimeRange(_ context.Context, tr shared.TimeRange, limit int) ([]*audit.Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*audit.Record, 0)
	for _, r := range s.items {
		if (r.OccurredAt.Equal(tr.From) || r.OccurredAt.After(tr.From)) && (r.OccurredAt.Equal(tr.To) || r.OccurredAt.Before(tr.To)) {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].OccurredAt.After(out[j].OccurredAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *AuditStore) DeleteByTimeRange(_ context.Context, tr shared.TimeRange) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	deleted := 0
	for id, r := range s.items {
		if (r.OccurredAt.Equal(tr.From) || r.OccurredAt.After(tr.From)) && (r.OccurredAt.Equal(tr.To) || r.OccurredAt.Before(tr.To)) {
			delete(s.items, id)
			deleted++
		}
	}
	return deleted, nil
}
