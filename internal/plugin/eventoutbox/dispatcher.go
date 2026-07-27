package eventoutbox

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

const (
	DefaultMaxAttempts   = 5
	DefaultLeaseDuration = 30 * time.Second
	DefaultPollInterval  = 500 * time.Millisecond
	DefaultRetryDelay    = time.Second
	MaxRetryDelay        = time.Hour
)

type DeliverySink interface {
	Deliver(context.Context, pluginsdk.EventEnvelope) error
}

type DeliverySinkFunc func(context.Context, pluginsdk.EventEnvelope) error

func (f DeliverySinkFunc) Deliver(ctx context.Context, envelope pluginsdk.EventEnvelope) error {
	return f(ctx, envelope)
}

type DispatcherOptions struct {
	LeaseDuration time.Duration
	RetryDelay    time.Duration
	Now           func() time.Time
}

type DispatcherMetrics struct {
	Leased      uint64
	Delivered   uint64
	Retried     uint64
	DeadLetters uint64
	LeaseLost   uint64
	Errors      uint64
}

type Dispatcher struct {
	store         Store
	sink          DeliverySink
	leaseDuration time.Duration
	retryDelay    time.Duration
	now           func() time.Time
	leased        atomic.Uint64
	delivered     atomic.Uint64
	retried       atomic.Uint64
	deadLetters   atomic.Uint64
	leaseLost     atomic.Uint64
	errors        atomic.Uint64
}

func NewDispatcher(store Store, sink DeliverySink, options DispatcherOptions) (*Dispatcher, error) {
	if store == nil || sink == nil {
		return nil, errors.New("plugin event dispatcher dependencies are required")
	}
	if options.LeaseDuration <= 0 {
		options.LeaseDuration = DefaultLeaseDuration
	}
	if options.RetryDelay <= 0 {
		options.RetryDelay = DefaultRetryDelay
	}
	if options.Now == nil {
		options.Now = func() time.Time { return time.Now().UTC() }
	}
	return &Dispatcher{
		store: store, sink: sink,
		leaseDuration: options.LeaseDuration, retryDelay: options.RetryDelay,
		now: options.Now,
	}, nil
}

func (d *Dispatcher) DispatchBatch(ctx context.Context, workerID string, limit int) (int, error) {
	if d == nil || d.store == nil || d.sink == nil {
		return 0, errors.New("plugin event dispatcher is not configured")
	}
	workerID = strings.TrimSpace(workerID)
	if workerID == "" {
		return 0, errors.New("plugin event worker identity is required")
	}
	if limit <= 0 || limit > 100 {
		return 0, errors.New("plugin event dispatch limit must be between 1 and 100")
	}
	now := d.now().UTC()
	records, err := d.store.LeaseDue(ctx, LeaseInput{
		WorkerID: workerID, Limit: limit, Now: now, LeaseDuration: d.leaseDuration,
	})
	if err != nil {
		return 0, err
	}
	d.leased.Add(uint64(len(records)))
	processed := 0
	var failures []error
	for _, record := range records {
		deliveryErr := d.sink.Deliver(ctx, record.Envelope)
		if deliveryErr == nil {
			if _, ackErr := d.store.Ack(ctx, AckInput{
				EventID: record.Envelope.ID, LeaseToken: record.LeaseToken, Now: d.now().UTC(),
			}); ackErr != nil {
				if errors.Is(ackErr, ErrLeaseLost) {
					d.leaseLost.Add(1)
				}
				failures = append(failures, fmt.Errorf("ack plugin event %s: %w", record.Envelope.ID, ackErr))
				continue
			}
			d.delivered.Add(1)
			processed++
			continue
		}
		failed, failErr := d.store.Fail(ctx, FailInput{
			EventID: record.Envelope.ID, LeaseToken: record.LeaseToken,
			Error: deliveryErr.Error(), RetryAt: d.nextRetry(record.AttemptCount, d.now().UTC()), Now: d.now().UTC(),
		})
		if failErr != nil {
			if errors.Is(failErr, ErrLeaseLost) {
				d.leaseLost.Add(1)
			}
			failures = append(failures, fmt.Errorf("fail plugin event %s: %w", record.Envelope.ID, failErr))
			continue
		}
		if failed.Status == StatusDeadLetter {
			d.deadLetters.Add(1)
		} else {
			d.retried.Add(1)
		}
		processed++
	}
	return processed, errors.Join(failures...)
}

func (d *Dispatcher) Run(ctx context.Context, workerID string, interval time.Duration, limit int) error {
	if interval <= 0 {
		interval = DefaultPollInterval
	}
	if _, err := d.DispatchBatch(ctx, workerID, limit); err != nil && ctx.Err() == nil {
		d.errors.Add(1)
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if _, err := d.DispatchBatch(ctx, workerID, limit); err != nil && ctx.Err() == nil {
				d.errors.Add(1)
			}
		}
	}
}

func (d *Dispatcher) Metrics() DispatcherMetrics {
	if d == nil {
		return DispatcherMetrics{}
	}
	return DispatcherMetrics{
		Leased: d.leased.Load(), Delivered: d.delivered.Load(), Retried: d.retried.Load(),
		DeadLetters: d.deadLetters.Load(), LeaseLost: d.leaseLost.Load(), Errors: d.errors.Load(),
	}
}

func (d *Dispatcher) nextRetry(attempt int, now time.Time) time.Time {
	delay := d.retryDelay
	for current := 1; current < attempt && delay < MaxRetryDelay; current++ {
		delay *= 2
		if delay > MaxRetryDelay {
			delay = MaxRetryDelay
		}
	}
	return now.Add(delay)
}
