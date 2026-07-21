package job

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sort"
	"strings"
	"sync"
	"time"
)

type MemoryRepository struct {
	mu          sync.RWMutex
	items       map[string]Job
	idempotency map[string]string
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{items: map[string]Job{}, idempotency: map[string]string{}}
}

func (r *MemoryRepository) Schedule(ctx context.Context, item Job) (Job, bool, error) {
	if err := ctx.Err(); err != nil {
		return Job{}, false, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if stored, ok := r.items[item.ID]; ok {
		return cloneJob(stored), false, nil
	}
	if item.IdempotencyKey != "" {
		key := jobIdempotencyKey(item.Namespace, item.IdempotencyKey)
		if id, ok := r.idempotency[key]; ok {
			return cloneJob(r.items[id]), false, nil
		}
		r.idempotency[key] = item.ID
	}
	r.items[item.ID] = cloneJob(item)
	return cloneJob(item), true, nil
}

func (r *MemoryRepository) LeaseDue(ctx context.Context, workerID string, now, leaseUntil time.Time, limit int) ([]Job, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.expireExhausted(now)
	candidates := make([]Job, 0, len(r.items))
	for _, item := range r.items {
		if leaseEligible(item, now) {
			candidates = append(candidates, item)
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].RunAt.Equal(candidates[j].RunAt) {
			return candidates[i].ID < candidates[j].ID
		}
		return candidates[i].RunAt.Before(candidates[j].RunAt)
	})
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	out := make([]Job, 0, len(candidates))
	for _, candidate := range candidates {
		token, err := newLeaseToken()
		if err != nil {
			return nil, err
		}
		candidate.Status = StatusRunning
		candidate.AttemptCount++
		candidate.LeaseOwner = workerID
		candidate.LeaseToken = token
		candidate.LeaseExpiresAt = timePointer(leaseUntil)
		candidate.UpdatedAt = now
		r.items[candidate.ID] = cloneJob(candidate)
		out = append(out, cloneJob(candidate))
	}
	return out, nil
}

func (r *MemoryRepository) Complete(ctx context.Context, id, leaseToken string, result []byte, now time.Time) (Job, error) {
	if err := ctx.Err(); err != nil {
		return Job{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok {
		return Job{}, ErrNotFound
	}
	if !ownsLease(item, leaseToken, now) {
		return Job{}, ErrLeaseLost
	}
	item.Status = StatusSucceeded
	item.Result = append([]byte(nil), result...)
	item.CompletedAt = timePointer(now)
	item.UpdatedAt = now
	clearLease(&item)
	r.items[id] = cloneJob(item)
	return cloneJob(item), nil
}

func (r *MemoryRepository) Fail(ctx context.Context, id, leaseToken, message string, retryAt, now time.Time) (Job, error) {
	if err := ctx.Err(); err != nil {
		return Job{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok {
		return Job{}, ErrNotFound
	}
	if !ownsLease(item, leaseToken, now) {
		return Job{}, ErrLeaseLost
	}
	item.LastError = message
	item.UpdatedAt = now
	clearLease(&item)
	if item.AttemptCount >= item.MaxAttempts {
		item.Status = StatusDeadLetter
		item.DeadLetteredAt = timePointer(now)
	} else {
		item.Status = StatusRetryWait
		item.RunAt = retryAt
	}
	r.items[id] = cloneJob(item)
	return cloneJob(item), nil
}

func (r *MemoryRepository) Get(ctx context.Context, id string) (Job, error) {
	if err := ctx.Err(); err != nil {
		return Job{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.items[id]
	if !ok {
		return Job{}, ErrNotFound
	}
	return cloneJob(item), nil
}

func (r *MemoryRepository) List(ctx context.Context, filter Filter) ([]Job, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Job, 0, len(r.items))
	for _, item := range r.items {
		if filter.Namespace != "" && item.Namespace != filter.Namespace || filter.Kind != "" && item.Kind != filter.Kind || filter.Status != "" && item.Status != filter.Status {
			continue
		}
		out = append(out, cloneJob(item))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].UpdatedAt.Equal(out[j].UpdatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	if filter.Limit > 0 && len(out) > filter.Limit {
		out = out[:filter.Limit]
	}
	return out, nil
}

func (r *MemoryRepository) expireExhausted(now time.Time) {
	for id, item := range r.items {
		if item.Status != StatusRunning || item.LeaseExpiresAt == nil || item.LeaseExpiresAt.After(now) || item.AttemptCount < item.MaxAttempts {
			continue
		}
		item.Status = StatusDeadLetter
		if item.LastError == "" {
			item.LastError = "job lease expired after final attempt"
		}
		item.DeadLetteredAt = timePointer(now)
		item.UpdatedAt = now
		clearLease(&item)
		r.items[id] = cloneJob(item)
	}
}

func leaseEligible(item Job, now time.Time) bool {
	if item.AttemptCount >= item.MaxAttempts {
		return false
	}
	if (item.Status == StatusScheduled || item.Status == StatusRetryWait) && !item.RunAt.After(now) {
		return true
	}
	return item.Status == StatusRunning && item.LeaseExpiresAt != nil && !item.LeaseExpiresAt.After(now)
}

func ownsLease(item Job, token string, now time.Time) bool {
	return item.Status == StatusRunning && item.LeaseToken == token && item.LeaseExpiresAt != nil && item.LeaseExpiresAt.After(now)
}

func clearLease(item *Job) {
	item.LeaseOwner = ""
	item.LeaseToken = ""
	item.LeaseExpiresAt = nil
}

func newLeaseToken() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

func jobIdempotencyKey(namespace, key string) string {
	return strings.TrimSpace(namespace) + "\x00" + strings.TrimSpace(key)
}

func timePointer(value time.Time) *time.Time {
	value = value.UTC()
	return &value
}

var _ Repository = (*MemoryRepository)(nil)
