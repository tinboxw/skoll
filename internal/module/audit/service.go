package audit

import (
	"strings"
	"sync"
	"time"
)

const (
	DefaultPage = 1
	DefaultSize = 20
	MaxSize     = 200
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

type Query struct {
	Page   int
	Size   int
	Actor  string
	Action string
	Target string
	Q      string
}

type QueryResult struct {
	Page  int      `json:"page"`
	Size  int      `json:"size"`
	Total int      `json:"total"`
	Items []Record `json:"items"`
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
	result := s.Query(Query{Page: 1, Size: limit})
	return result.Items
}

func (s *Service) Query(raw Query) QueryResult {
	q := normalizeQuery(raw)

	s.mu.RLock()
	defer s.mu.RUnlock()

	matched := make([]Record, 0, len(s.records))
	for i := len(s.records) - 1; i >= 0; i-- {
		record := s.records[i]
		if !matches(record, q) {
			continue
		}
		matched = append(matched, record)
	}

	total := len(matched)
	if total == 0 {
		return QueryResult{Page: q.Page, Size: q.Size, Total: 0, Items: nil}
	}

	offset := (q.Page - 1) * q.Size
	if offset >= total {
		return QueryResult{Page: q.Page, Size: q.Size, Total: total, Items: nil}
	}

	end := offset + q.Size
	if end > total {
		end = total
	}
	items := make([]Record, end-offset)
	copy(items, matched[offset:end])

	return QueryResult{Page: q.Page, Size: q.Size, Total: total, Items: items}
}

func normalizeQuery(raw Query) Query {
	q := raw
	if q.Page <= 0 {
		q.Page = DefaultPage
	}
	if q.Size <= 0 {
		q.Size = DefaultSize
	}
	if q.Size > MaxSize {
		q.Size = MaxSize
	}
	q.Actor = strings.TrimSpace(strings.ToLower(q.Actor))
	q.Action = strings.TrimSpace(strings.ToLower(q.Action))
	q.Target = strings.TrimSpace(strings.ToLower(q.Target))
	q.Q = strings.TrimSpace(strings.ToLower(q.Q))
	return q
}

func matches(record Record, q Query) bool {
	actor := strings.ToLower(record.Actor)
	action := strings.ToLower(record.Action)
	target := strings.ToLower(record.Target)
	if q.Actor != "" && actor != q.Actor {
		return false
	}
	if q.Action != "" && action != q.Action {
		return false
	}
	if q.Target != "" && target != q.Target {
		return false
	}
	if q.Q != "" && !strings.Contains(actor, q.Q) && !strings.Contains(action, q.Q) && !strings.Contains(target, q.Q) {
		return false
	}
	return true
}
