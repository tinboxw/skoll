package event

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	BusinessEventApprovalCompleted      = "approval-completed"
	BusinessEventInboundCompleted       = "inbound-completed"
	BusinessEventQualificationExpiring  = "qualification-expiring"
	BusinessRetryStatusFailed           = "failed"
	BusinessRetryStatusRunning          = "running"
	BusinessRetryStatusSucceeded        = "succeeded"
	BusinessRetryStatusDeadLetter       = "dead_letter"
	defaultBusinessEventRetryDelay      = time.Second
	defaultBusinessEventMaxRetryAttempt = 3
)

var businessEventNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]{2,127}$`)

type BusinessEvent struct {
	ID          string
	EventName   string
	Source      string
	SubjectType string
	SubjectID   string
	Payload     map[string]any
	Metadata    map[string]string
	OccurredAt  time.Time
}

func (e BusinessEvent) Name() string {
	return strings.TrimSpace(e.EventName)
}

func (e BusinessEvent) Validate() error {
	if !businessEventNamePattern.MatchString(strings.TrimSpace(e.EventName)) {
		return fmt.Errorf("invalid business event name: %s", e.EventName)
	}
	return nil
}

type BusinessEventHandler func(context.Context, BusinessEvent) error

type BusinessSubscription struct {
	EventName   string
	HandlerName string
	Handler     BusinessEventHandler
}

type BusinessRetryRecord struct {
	ID             string
	Event          BusinessEvent
	HandlerName    string
	Attempt        int
	Status         string
	Error          string
	NextRunAt      time.Time
	DeadLetteredAt time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type BusinessRetryStore interface {
	Save(record BusinessRetryRecord) (BusinessRetryRecord, error)
	Due(now time.Time) []BusinessRetryRecord
	Get(id string) (BusinessRetryRecord, bool)
	Snapshot() []BusinessRetryRecord
}

type MemoryBusinessRetryStore struct {
	mu      sync.RWMutex
	nextID  int64
	records map[string]BusinessRetryRecord
}

func NewMemoryBusinessRetryStore() *MemoryBusinessRetryStore {
	return &MemoryBusinessRetryStore{records: map[string]BusinessRetryRecord{}}
}

func (s *MemoryBusinessRetryStore) Save(record BusinessRetryRecord) (BusinessRetryRecord, error) {
	if s == nil {
		return record, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	record.UpdatedAt = now
	if strings.TrimSpace(record.ID) == "" {
		s.nextID++
		record.ID = fmt.Sprintf("business-retry-%d", s.nextID)
	}
	s.records[record.ID] = cloneBusinessRetryRecord(record)
	return record, nil
}

func (s *MemoryBusinessRetryStore) Due(now time.Time) []BusinessRetryRecord {
	if s == nil {
		return nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]BusinessRetryRecord, 0, len(s.records))
	for _, record := range s.records {
		if record.Status != BusinessRetryStatusFailed {
			continue
		}
		if !record.NextRunAt.IsZero() && record.NextRunAt.After(now) {
			continue
		}
		out = append(out, cloneBusinessRetryRecord(record))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out
}

func (s *MemoryBusinessRetryStore) Get(id string) (BusinessRetryRecord, bool) {
	if s == nil {
		return BusinessRetryRecord{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.records[id]
	return cloneBusinessRetryRecord(record), ok
}

func (s *MemoryBusinessRetryStore) Snapshot() []BusinessRetryRecord {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]BusinessRetryRecord, 0, len(s.records))
	for _, record := range s.records {
		out = append(out, cloneBusinessRetryRecord(record))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

type BusinessEventBus struct {
	mu          sync.RWMutex
	handlers    map[string]map[int64]BusinessSubscription
	nextID      int64
	retries     BusinessRetryStore
	retryDelay  time.Duration
	maxAttempts int
	now         func() time.Time
}

func NewBusinessEventBus(retries BusinessRetryStore) *BusinessEventBus {
	if retries == nil {
		retries = NewMemoryBusinessRetryStore()
	}
	return &BusinessEventBus{
		handlers:    map[string]map[int64]BusinessSubscription{},
		retries:     retries,
		retryDelay:  defaultBusinessEventRetryDelay,
		maxAttempts: defaultBusinessEventMaxRetryAttempt,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (b *BusinessEventBus) Subscribe(eventName, handlerName string, handler BusinessEventHandler) (func(), error) {
	if b == nil {
		return func() {}, nil
	}
	eventName = strings.TrimSpace(eventName)
	handlerName = strings.TrimSpace(handlerName)
	if !businessEventNamePattern.MatchString(eventName) || handlerName == "" || handler == nil {
		return nil, errors.New("invalid business event subscription")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.handlers[eventName] == nil {
		b.handlers[eventName] = map[int64]BusinessSubscription{}
	}
	b.nextID++
	id := b.nextID
	b.handlers[eventName][id] = BusinessSubscription{EventName: eventName, HandlerName: handlerName, Handler: handler}
	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if b.handlers[eventName] != nil {
			delete(b.handlers[eventName], id)
		}
	}, nil
}

func (b *BusinessEventBus) Publish(ctx context.Context, evt BusinessEvent) error {
	if b == nil {
		return nil
	}
	evt = normalizeBusinessEvent(evt, b.now())
	if err := evt.Validate(); err != nil {
		return err
	}
	handlers := b.handlersFor(evt.EventName)
	var failed int
	for _, subscription := range handlers {
		if err := subscription.Handler(ctx, cloneBusinessEvent(evt)); err != nil {
			failed++
			b.recordFailure(evt, subscription.HandlerName, 1, err)
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d business event handler(s) failed for %s", failed, evt.EventName)
	}
	return nil
}

func (b *BusinessEventBus) RetryDue(ctx context.Context, now time.Time) (int, error) {
	if b == nil || b.retries == nil {
		return 0, nil
	}
	due := b.retries.Due(now)
	processed := 0
	var failed int
	for _, record := range due {
		if b.maxAttempts > 0 && record.Attempt >= b.maxAttempts {
			b.deadLetter(record, errors.New("business event retry attempts exhausted"))
			processed++
			failed++
			continue
		}
		subscription, ok := b.findHandler(record.Event.EventName, record.HandlerName)
		if !ok {
			b.deadLetter(record, errors.New("business event retry handler is unavailable"))
			processed++
			failed++
			continue
		}
		record.Attempt++
		record.Status = BusinessRetryStatusRunning
		record.Error = ""
		record.NextRunAt = time.Time{}
		record, _ = b.retries.Save(record)
		if err := subscription.Handler(ctx, cloneBusinessEvent(record.Event)); err != nil {
			failed++
			if b.maxAttempts > 0 && record.Attempt >= b.maxAttempts {
				b.deadLetter(record, err)
			} else {
				record.Status = BusinessRetryStatusFailed
				record.Error = err.Error()
				record.NextRunAt = b.now().Add(b.retryDelay)
				_, _ = b.retries.Save(record)
			}
			processed++
			continue
		}
		record.Status = BusinessRetryStatusSucceeded
		record.Error = ""
		record.NextRunAt = time.Time{}
		_, _ = b.retries.Save(record)
		processed++
	}
	if failed > 0 {
		return processed, fmt.Errorf("%d business event retry handler(s) failed", failed)
	}
	return processed, nil
}

func (b *BusinessEventBus) RetryRecords() []BusinessRetryRecord {
	if b == nil || b.retries == nil {
		return nil
	}
	return b.retries.Snapshot()
}

func (b *BusinessEventBus) handlersFor(eventName string) []BusinessSubscription {
	b.mu.RLock()
	defer b.mu.RUnlock()
	set := b.handlers[eventName]
	out := make([]BusinessSubscription, 0, len(set))
	for _, subscription := range set {
		out = append(out, subscription)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].HandlerName < out[j].HandlerName
	})
	return out
}

func (b *BusinessEventBus) findHandler(eventName, handlerName string) (BusinessSubscription, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, subscription := range b.handlers[eventName] {
		if subscription.HandlerName == handlerName {
			return subscription, true
		}
	}
	return BusinessSubscription{}, false
}

func (b *BusinessEventBus) recordFailure(evt BusinessEvent, handlerName string, attempt int, err error) {
	if b == nil || b.retries == nil {
		return
	}
	if attempt <= 0 {
		attempt = 1
	}
	record := BusinessRetryRecord{
		Event:       cloneBusinessEvent(evt),
		HandlerName: handlerName,
		Attempt:     attempt,
		Status:      BusinessRetryStatusFailed,
		Error:       err.Error(),
		NextRunAt:   b.now().Add(b.retryDelay),
	}
	if b.maxAttempts > 0 && attempt >= b.maxAttempts {
		record.Status = BusinessRetryStatusDeadLetter
		record.NextRunAt = time.Time{}
		record.DeadLetteredAt = b.now()
	}
	_, _ = b.retries.Save(record)
}

func (b *BusinessEventBus) deadLetter(record BusinessRetryRecord, cause error) {
	record.Status = BusinessRetryStatusDeadLetter
	if cause != nil {
		record.Error = cause.Error()
	}
	record.NextRunAt = time.Time{}
	record.DeadLetteredAt = b.now()
	_, _ = b.retries.Save(record)
}

type AfterCommitQueue struct {
	publisher interface {
		Publish(context.Context, BusinessEvent) error
	}
	events []BusinessEvent
	closed bool
}

func NewAfterCommitQueue(publisher interface {
	Publish(context.Context, BusinessEvent) error
}) *AfterCommitQueue {
	return &AfterCommitQueue{publisher: publisher}
}

func (q *AfterCommitQueue) Enqueue(evt BusinessEvent) error {
	if q == nil {
		return nil
	}
	if q.closed {
		return errors.New("after-commit queue is closed")
	}
	if err := evt.Validate(); err != nil {
		return err
	}
	q.events = append(q.events, cloneBusinessEvent(evt))
	return nil
}

func (q *AfterCommitQueue) Commit(ctx context.Context) error {
	if q == nil || q.closed {
		return nil
	}
	q.closed = true
	if q.publisher == nil {
		return nil
	}
	var failed int
	for _, evt := range q.events {
		if err := q.publisher.Publish(ctx, evt); err != nil {
			failed++
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d after-commit event(s) failed", failed)
	}
	return nil
}

func (q *AfterCommitQueue) Rollback() {
	if q == nil {
		return
	}
	q.closed = true
	q.events = nil
}

func normalizeBusinessEvent(evt BusinessEvent, now time.Time) BusinessEvent {
	evt.EventName = strings.TrimSpace(evt.EventName)
	evt.Source = strings.TrimSpace(evt.Source)
	evt.SubjectType = strings.TrimSpace(evt.SubjectType)
	evt.SubjectID = strings.TrimSpace(evt.SubjectID)
	if evt.OccurredAt.IsZero() {
		evt.OccurredAt = now
	}
	if evt.Payload == nil {
		evt.Payload = map[string]any{}
	}
	if evt.Metadata == nil {
		evt.Metadata = map[string]string{}
	}
	return evt
}

func cloneBusinessEvent(evt BusinessEvent) BusinessEvent {
	evt.Payload = cloneAnyMap(evt.Payload)
	evt.Metadata = cloneStringMap(evt.Metadata)
	return evt
}

func cloneBusinessRetryRecord(record BusinessRetryRecord) BusinessRetryRecord {
	record.Event = cloneBusinessEvent(record.Event)
	return record
}

func cloneAnyMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
