package event

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	BusinessEventApprovalCompleted     = "approval-completed"
	BusinessEventInboundCompleted      = "inbound-completed"
	BusinessEventQualificationExpiring = "qualification-expiring"
	BusinessRetryStatusFailed          = "failed"
	BusinessRetryStatusRunning         = "running"
	BusinessRetryStatusSucceeded       = "succeeded"
	BusinessRetryStatusDeadLetter      = "dead_letter"
	BusinessRetryPolicyNone            = "none"
	BusinessRetryPolicyStandard        = "standard"
	BusinessRetryPolicyAggressive      = "aggressive"
	BusinessDeliveryStatusRunning      = "running"
	BusinessDeliveryStatusFailed       = "failed"
	BusinessDeliveryStatusSucceeded    = "succeeded"
	BusinessDeliveryStatusDeadLetter   = "dead_letter"
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
	if strings.TrimSpace(e.ID) == "" {
		return errors.New("business event id is required")
	}
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
	RetryPolicy BusinessRetryPolicy
}

type BusinessRetryPolicy struct {
	Name        string
	MaxAttempts int
	Delay       time.Duration
}

func ParseBusinessRetryPolicy(name string) (BusinessRetryPolicy, error) {
	switch strings.TrimSpace(strings.ToLower(name)) {
	case BusinessRetryPolicyNone:
		return BusinessRetryPolicy{Name: BusinessRetryPolicyNone, MaxAttempts: 1}, nil
	case "", BusinessRetryPolicyStandard:
		return BusinessRetryPolicy{Name: BusinessRetryPolicyStandard, MaxAttempts: 3, Delay: time.Second}, nil
	case BusinessRetryPolicyAggressive:
		return BusinessRetryPolicy{Name: BusinessRetryPolicyAggressive, MaxAttempts: 5, Delay: 250 * time.Millisecond}, nil
	default:
		return BusinessRetryPolicy{}, fmt.Errorf("invalid business retry policy: %s", name)
	}
}

type BusinessRetryRecord struct {
	ID             string
	DeliveryID     string
	Event          BusinessEvent
	HandlerName    string
	Attempt        int
	MaxAttempts    int
	RetryDelay     time.Duration
	Status         string
	Error          string
	NextRunAt      time.Time
	DeadLetteredAt time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type BusinessDeliveryRecord struct {
	ID          string
	EventID     string
	HandlerName string
	Status      string
	Attempt     int
	Error       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type BusinessDeliveryStore interface {
	ClaimInitial(record BusinessDeliveryRecord) (bool, error)
	ClaimRetry(deliveryID string, attempt int, now time.Time) (bool, error)
	Mark(deliveryID, status, failure string, now time.Time) error
	Get(deliveryID string) (BusinessDeliveryRecord, bool)
	Snapshot() []BusinessDeliveryRecord
}

type MemoryBusinessDeliveryStore struct {
	mu      sync.RWMutex
	records map[string]BusinessDeliveryRecord
}

func NewMemoryBusinessDeliveryStore() *MemoryBusinessDeliveryStore {
	return &MemoryBusinessDeliveryStore{records: map[string]BusinessDeliveryRecord{}}
}

func (s *MemoryBusinessDeliveryStore) ClaimInitial(record BusinessDeliveryRecord) (bool, error) {
	if s == nil {
		return false, errors.New("business delivery store is not configured")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.records[record.ID]; exists {
		return false, nil
	}
	now := record.CreatedAt.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	record.Status = BusinessDeliveryStatusRunning
	record.Attempt = 1
	record.CreatedAt = now
	record.UpdatedAt = now
	s.records[record.ID] = record
	return true, nil
}

func (s *MemoryBusinessDeliveryStore) ClaimRetry(deliveryID string, attempt int, now time.Time) (bool, error) {
	if s == nil {
		return false, errors.New("business delivery store is not configured")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, exists := s.records[deliveryID]
	if !exists || record.Status != BusinessDeliveryStatusFailed {
		return false, nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	record.Status = BusinessDeliveryStatusRunning
	record.Attempt = attempt
	record.Error = ""
	record.UpdatedAt = now.UTC()
	s.records[deliveryID] = record
	return true, nil
}

func (s *MemoryBusinessDeliveryStore) Mark(deliveryID, status, failure string, now time.Time) error {
	if s == nil {
		return errors.New("business delivery store is not configured")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, exists := s.records[deliveryID]
	if !exists {
		return fmt.Errorf("business delivery not found: %s", deliveryID)
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	record.Status = status
	record.Error = failure
	record.UpdatedAt = now.UTC()
	s.records[deliveryID] = record
	return nil
}

func (s *MemoryBusinessDeliveryStore) Get(deliveryID string) (BusinessDeliveryRecord, bool) {
	if s == nil {
		return BusinessDeliveryRecord{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.records[deliveryID]
	return record, ok
}

func (s *MemoryBusinessDeliveryStore) Snapshot() []BusinessDeliveryRecord {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]BusinessDeliveryRecord, 0, len(s.records))
	for _, record := range s.records {
		out = append(out, record)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
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
	mu         sync.RWMutex
	handlers   map[string]map[int64]BusinessSubscription
	nextID     int64
	retries    BusinessRetryStore
	deliveries BusinessDeliveryStore
	now        func() time.Time
}

func NewBusinessEventBus(retries BusinessRetryStore) *BusinessEventBus {
	return NewBusinessEventBusWithStores(retries, nil)
}

func NewBusinessEventBusWithStores(retries BusinessRetryStore, deliveries BusinessDeliveryStore) *BusinessEventBus {
	if retries == nil {
		retries = NewMemoryBusinessRetryStore()
	}
	if deliveries == nil {
		deliveries = NewMemoryBusinessDeliveryStore()
	}
	return &BusinessEventBus{
		handlers:   map[string]map[int64]BusinessSubscription{},
		retries:    retries,
		deliveries: deliveries,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (b *BusinessEventBus) Subscribe(eventName, handlerName string, handler BusinessEventHandler) (func(), error) {
	policy, _ := ParseBusinessRetryPolicy(BusinessRetryPolicyStandard)
	return b.SubscribeWithPolicy(eventName, handlerName, policy, handler)
}

func (b *BusinessEventBus) SubscribeWithPolicy(eventName, handlerName string, policy BusinessRetryPolicy, handler BusinessEventHandler) (func(), error) {
	if b == nil {
		return func() {}, nil
	}
	eventName = strings.TrimSpace(eventName)
	handlerName = strings.TrimSpace(handlerName)
	parsedPolicy, err := ParseBusinessRetryPolicy(policy.Name)
	if !businessEventNamePattern.MatchString(eventName) || handlerName == "" || handler == nil || err != nil {
		return nil, errors.New("invalid business event subscription")
	}
	if policy.MaxAttempts > 0 {
		parsedPolicy.MaxAttempts = policy.MaxAttempts
	}
	if policy.Delay > 0 {
		parsedPolicy.Delay = policy.Delay
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.handlers[eventName] == nil {
		b.handlers[eventName] = map[int64]BusinessSubscription{}
	}
	b.nextID++
	id := b.nextID
	b.handlers[eventName][id] = BusinessSubscription{EventName: eventName, HandlerName: handlerName, Handler: handler, RetryPolicy: parsedPolicy}
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
		deliveryID := BusinessDeliveryID(evt.ID, subscription.HandlerName)
		claimed, err := b.deliveries.ClaimInitial(BusinessDeliveryRecord{
			ID:          deliveryID,
			EventID:     evt.ID,
			HandlerName: subscription.HandlerName,
			CreatedAt:   b.now(),
		})
		if err != nil {
			failed++
			continue
		}
		if !claimed {
			continue
		}
		if err := subscription.Handler(ctx, cloneBusinessEvent(evt)); err != nil {
			failed++
			b.recordFailure(evt, subscription, deliveryID, 1, err)
			continue
		}
		_ = b.deliveries.Mark(deliveryID, BusinessDeliveryStatusSucceeded, "", b.now())
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
		if record.MaxAttempts > 0 && record.Attempt >= record.MaxAttempts {
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
		nextAttempt := record.Attempt + 1
		claimed, claimErr := b.deliveries.ClaimRetry(record.DeliveryID, nextAttempt, b.now())
		if claimErr != nil {
			failed++
			continue
		}
		if !claimed {
			if delivery, exists := b.deliveries.Get(record.DeliveryID); exists && delivery.Status == BusinessDeliveryStatusSucceeded {
				record.Attempt = delivery.Attempt
				record.Status = BusinessRetryStatusSucceeded
				record.Error = ""
				record.NextRunAt = time.Time{}
				_, _ = b.retries.Save(record)
				processed++
			}
			continue
		}
		record.Attempt = nextAttempt
		record.Status = BusinessRetryStatusRunning
		record.Error = ""
		record.NextRunAt = time.Time{}
		record, _ = b.retries.Save(record)
		if err := subscription.Handler(ctx, cloneBusinessEvent(record.Event)); err != nil {
			failed++
			if record.MaxAttempts > 0 && record.Attempt >= record.MaxAttempts {
				b.deadLetter(record, err)
			} else {
				_ = b.deliveries.Mark(record.DeliveryID, BusinessDeliveryStatusFailed, err.Error(), b.now())
				record.Status = BusinessRetryStatusFailed
				record.Error = err.Error()
				record.NextRunAt = b.now().Add(record.RetryDelay)
				_, _ = b.retries.Save(record)
			}
			processed++
			continue
		}
		_ = b.deliveries.Mark(record.DeliveryID, BusinessDeliveryStatusSucceeded, "", b.now())
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

func (b *BusinessEventBus) DeliveryRecords() []BusinessDeliveryRecord {
	if b == nil || b.deliveries == nil {
		return nil
	}
	return b.deliveries.Snapshot()
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

func (b *BusinessEventBus) recordFailure(evt BusinessEvent, subscription BusinessSubscription, deliveryID string, attempt int, err error) {
	if b == nil || b.retries == nil {
		return
	}
	if attempt <= 0 {
		attempt = 1
	}
	record := BusinessRetryRecord{
		DeliveryID:  deliveryID,
		Event:       cloneBusinessEvent(evt),
		HandlerName: subscription.HandlerName,
		Attempt:     attempt,
		MaxAttempts: subscription.RetryPolicy.MaxAttempts,
		RetryDelay:  subscription.RetryPolicy.Delay,
		Status:      BusinessRetryStatusFailed,
		Error:       err.Error(),
		NextRunAt:   b.now().Add(subscription.RetryPolicy.Delay),
	}
	if record.MaxAttempts > 0 && attempt >= record.MaxAttempts {
		record.Status = BusinessRetryStatusDeadLetter
		record.NextRunAt = time.Time{}
		record.DeadLetteredAt = b.now()
		_ = b.deliveries.Mark(deliveryID, BusinessDeliveryStatusDeadLetter, err.Error(), b.now())
	} else {
		_ = b.deliveries.Mark(deliveryID, BusinessDeliveryStatusFailed, err.Error(), b.now())
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
	_ = b.deliveries.Mark(record.DeliveryID, BusinessDeliveryStatusDeadLetter, record.Error, b.now())
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
	evt.ID = strings.TrimSpace(evt.ID)
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

func BusinessDeliveryID(eventID, handlerName string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(eventID) + "\x00" + strings.TrimSpace(handlerName)))
	return fmt.Sprintf("business-delivery-%x", sum[:])
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
