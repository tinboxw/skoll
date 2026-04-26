package audit

import (
	"sync"
	"time"
)

type Record struct {
	ID        int64
	Actor     string
	Action    string
	Target    string
	CreatedAt time.Time
}

type Service struct {
	mu      sync.RWMutex
	nextID  int64
	records []Record
}

func NewService() *Service {
	return &Service{nextID: 1, records: make([]Record, 0, 64)}
}

func (s *Service) Append(actor, action, target string) Record {
	s.mu.Lock()
	defer s.mu.Unlock()

	r := Record{
		ID:        s.nextID,
		Actor:     actor,
		Action:    action,
		Target:    target,
		CreatedAt: time.Now().UTC(),
	}
	s.nextID++
	s.records = append(s.records, r)
	return r
}

func (s *Service) Recent(limit int) []Record {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || len(s.records) == 0 {
		return nil
	}
	if limit > len(s.records) {
		limit = len(s.records)
	}
	start := len(s.records) - limit
	out := make([]Record, limit)
	copy(out, s.records[start:])
	return out
}
