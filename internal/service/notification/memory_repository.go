package notification

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

type MemoryRepository struct {
	mu                  sync.RWMutex
	items               map[string]Item
	rules               map[string]ReminderRule
	deliveryAttempts    map[string][]DeliveryAttempt
	deliveryIdempotency map[string]DeliveryAttempt
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		items: map[string]Item{}, rules: map[string]ReminderRule{},
		deliveryAttempts: map[string][]DeliveryAttempt{}, deliveryIdempotency: map[string]DeliveryAttempt{},
	}
}

func (r *MemoryRepository) CreateItem(ctx context.Context, item Item) (Item, bool, error) {
	if err := ctx.Err(); err != nil {
		return Item{}, false, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if stored, ok := r.items[item.ID]; ok {
		return stored, false, nil
	}
	r.items[item.ID] = item
	return item, true, nil
}

func (r *MemoryRepository) CompleteItem(ctx context.Context, id, actorID string, now time.Time) (Item, error) {
	if err := ctx.Err(); err != nil {
		return Item{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok {
		return Item{}, errors.New("notification item not found")
	}
	if item.ActorID != actorID {
		return Item{}, errors.New("notification actor does not own item")
	}
	next := StatusDone
	if item.Category == CategoryMessage {
		next = StatusRead
	}
	if item.Status == next {
		return item, nil
	}
	item.Status = next
	item.UpdatedAt = now
	r.items[id] = item
	return item, nil
}

func (r *MemoryRepository) ListItems(ctx context.Context, filter Filter) ([]Item, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Item, 0, len(r.items))
	for _, item := range r.items {
		if filter.ActorID != "" && item.ActorID != filter.ActorID || filter.Category != "" && item.Category != filter.Category || filter.Status != "" && item.Status != filter.Status {
			continue
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].UpdatedAt.Equal(out[j].UpdatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out, nil
}

func (r *MemoryRepository) UpsertReminderRule(ctx context.Context, rule ReminderRule) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rules[rule.ID] = rule
	return nil
}

func (r *MemoryRepository) ListReminderRules(ctx context.Context) ([]ReminderRule, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ReminderRule, 0, len(r.rules))
	for _, rule := range r.rules {
		out = append(out, rule)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (r *MemoryRepository) RecordDeliveryAttempt(ctx context.Context, attempt DeliveryAttempt) (DeliveryAttempt, bool, error) {
	if err := ctx.Err(); err != nil {
		return DeliveryAttempt{}, false, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[attempt.NotificationID]; !ok {
		return DeliveryAttempt{}, false, errors.New("notification item not found")
	}
	key := deliveryIdempotencyKey(attempt.NotificationID, attempt.Channel, attempt.IdempotencyKey)
	if stored, ok := r.deliveryIdempotency[key]; ok {
		return stored, false, nil
	}
	streamKey := deliveryStreamKey(attempt.NotificationID, attempt.Channel)
	for _, stored := range r.deliveryAttempts[streamKey] {
		if stored.Status == DeliverySucceeded {
			return stored, false, nil
		}
	}
	attempt.Attempt = len(r.deliveryAttempts[streamKey]) + 1
	r.deliveryAttempts[streamKey] = append(r.deliveryAttempts[streamKey], attempt)
	r.deliveryIdempotency[key] = attempt
	return attempt, true, nil
}

func (r *MemoryRepository) ListDeliveryAttempts(ctx context.Context, notificationID, channel string) ([]DeliveryAttempt, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := append([]DeliveryAttempt(nil), r.deliveryAttempts[deliveryStreamKey(notificationID, channel)]...)
	return out, nil
}

func deliveryIdempotencyKey(notificationID, channel, key string) string {
	return strings.Join([]string{notificationID, channel, key}, "\x00")
}

func deliveryStreamKey(notificationID, channel string) string {
	return strings.Join([]string{notificationID, channel}, "\x00")
}

var _ Repository = (*MemoryRepository)(nil)
