package event

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/testing/jobsoak"
)

func TestBusinessEventBusPublishesKnownBusinessEvents(t *testing.T) {
	bus := NewBusinessEventBus(nil)
	called := 0
	if _, err := bus.Subscribe(BusinessEventApprovalCompleted, "approval.handler", func(_ context.Context, evt BusinessEvent) error {
		called++
		if evt.EventName != BusinessEventApprovalCompleted || evt.Payload["approvalId"] != "ap-1" {
			t.Fatalf("unexpected event: %+v", evt)
		}
		return nil
	}); err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	if err := bus.Publish(context.Background(), BusinessEvent{
		EventName: BusinessEventApprovalCompleted,
		Source:    "workflow",
		Payload:   map[string]any{"approvalId": "ap-1"},
	}); err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	if called != 1 {
		t.Fatalf("expected handler called once, got %d", called)
	}
}

func TestBusinessEventBusRecordsAndRetriesFailedHandlers(t *testing.T) {
	store := NewMemoryBusinessRetryStore()
	bus := NewBusinessEventBus(store)
	now := time.Date(2026, 7, 4, 8, 0, 0, 0, time.UTC)
	bus.now = func() time.Time { return now }
	attempts := 0
	if _, err := bus.Subscribe(BusinessEventInboundCompleted, "inventory.sync", func(_ context.Context, evt BusinessEvent) error {
		attempts++
		if evt.SubjectID != "in-1" {
			t.Fatalf("unexpected subject id: %s", evt.SubjectID)
		}
		if attempts == 1 {
			return errors.New("temporary inventory sync failure")
		}
		return nil
	}); err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	if err := bus.Publish(context.Background(), BusinessEvent{
		EventName: BusinessEventInboundCompleted,
		Source:    "purchase",
		SubjectID: "in-1",
	}); err == nil {
		t.Fatal("expected publish to report failed handler")
	}
	records := store.Snapshot()
	if len(records) != 1 || records[0].Status != BusinessRetryStatusFailed || records[0].Attempt != 1 {
		t.Fatalf("expected failed retry record, got %+v", records)
	}

	processed, err := bus.RetryDue(context.Background(), now.Add(2*time.Second))
	if err != nil {
		t.Fatalf("retry due failed: %v", err)
	}
	if processed != 1 {
		t.Fatalf("expected one retry processed, got %d", processed)
	}
	records = store.Snapshot()
	if len(records) != 1 || records[0].Status != BusinessRetryStatusSucceeded || records[0].Attempt != 2 {
		t.Fatalf("expected succeeded retry record, got %+v", records)
	}
}

func TestBusinessEventBusMovesExhaustedRetriesToDeadLetter(t *testing.T) {
	store := NewMemoryBusinessRetryStore()
	bus := NewBusinessEventBus(store)
	bus.retryDelay = 0
	bus.maxAttempts = 3
	_, err := bus.Subscribe(BusinessEventInboundCompleted, "always-fails", func(context.Context, BusinessEvent) error {
		return errors.New("plugin unavailable")
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = bus.Publish(context.Background(), BusinessEvent{EventName: BusinessEventInboundCompleted}); err == nil {
		t.Fatal("expected initial handler failure")
	}
	for attempt := 2; attempt <= 3; attempt++ {
		processed, retryErr := bus.RetryDue(context.Background(), time.Now().Add(time.Hour))
		if processed != 1 || retryErr == nil {
			t.Fatalf("attempt %d: processed=%d err=%v", attempt, processed, retryErr)
		}
	}
	records := bus.RetryRecords()
	if len(records) != 1 || records[0].Status != BusinessRetryStatusDeadLetter || records[0].Attempt != 3 || records[0].DeadLetteredAt.IsZero() {
		t.Fatalf("retry was not terminal and observable: %+v", records)
	}
	processed, err := bus.RetryDue(context.Background(), time.Now().Add(time.Hour))
	if processed != 0 || err != nil {
		t.Fatalf("dead letter retried: processed=%d err=%v", processed, err)
	}
}

func TestBusinessEventBusDeadLettersUnavailableRetryHandler(t *testing.T) {
	store := NewMemoryBusinessRetryStore()
	bus := NewBusinessEventBus(store)
	bus.retryDelay = 0
	unsubscribe, err := bus.Subscribe(BusinessEventInboundCompleted, "removed-handler", func(context.Context, BusinessEvent) error {
		return errors.New("temporary failure")
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = bus.Publish(context.Background(), BusinessEvent{EventName: BusinessEventInboundCompleted}); err == nil {
		t.Fatal("expected initial handler failure")
	}
	unsubscribe()
	processed, err := bus.RetryDue(context.Background(), time.Now().Add(time.Hour))
	if processed != 1 || err == nil {
		t.Fatalf("missing handler failure was silent: processed=%d err=%v", processed, err)
	}
	records := bus.RetryRecords()
	if len(records) != 1 || records[0].Status != BusinessRetryStatusDeadLetter || !strings.Contains(records[0].Error, "unavailable") {
		t.Fatalf("missing handler was not dead-lettered: %+v", records)
	}
}

func TestH5BusinessEventSoak(t *testing.T) {
	iterations := jobsoak.IterationsFromEnv("SKOLL_H5_SOAK_ITERATIONS", 25)
	thresholds := jobsoak.Thresholds{
		MaxP95Milliseconds: 25, MaxP99Milliseconds: 75,
		MaxHeapGrowthBytes: 16 << 20, MaxGoroutineGrowth: 2,
		MaxFailures: 0, MaxDuplicateSideEffects: 0,
	}
	scenario := jobsoak.Measure("插件业务事件重试与副作用幂等", iterations, thresholds, func(index int) (jobsoak.Observation, error) {
		store := NewMemoryBusinessRetryStore()
		bus := NewBusinessEventBus(store)
		bus.retryDelay = 0
		calls, sideEffects := 0, 0
		_, err := bus.Subscribe(BusinessEventApprovalCompleted, fmt.Sprintf("soak-handler-%d", index), func(context.Context, BusinessEvent) error {
			calls++
			if calls == 1 {
				return errors.New("transient plugin failure")
			}
			sideEffects++
			return nil
		})
		if err != nil {
			return jobsoak.Observation{}, err
		}
		if err = bus.Publish(context.Background(), BusinessEvent{EventName: BusinessEventApprovalCompleted, SubjectID: fmt.Sprintf("approval-%d", index)}); err == nil {
			return jobsoak.Observation{}, errors.New("initial transient failure was not exposed")
		}
		processed, err := bus.RetryDue(context.Background(), time.Now().Add(time.Hour))
		duplicates := 0
		if sideEffects > 1 {
			duplicates = sideEffects - 1
		}
		observation := jobsoak.Observation{Retries: processed, DuplicateSideEffects: duplicates}
		if err != nil || processed != 1 || sideEffects != 1 {
			return observation, fmt.Errorf("retry mismatch: processed=%d effects=%d err=%v", processed, sideEffects, err)
		}
		records := bus.RetryRecords()
		if len(records) != 1 || records[0].Status != BusinessRetryStatusSucceeded || records[0].Attempt != 2 {
			return observation, fmt.Errorf("retry record mismatch: %+v", records)
		}
		return observation, nil
	})

	// A permanently failing subscriber proves that exhausted work reaches a visible terminal state.
	deadStore := NewMemoryBusinessRetryStore()
	deadBus := NewBusinessEventBus(deadStore)
	deadBus.retryDelay = 0
	_, _ = deadBus.Subscribe(BusinessEventQualificationExpiring, "dead-letter-proof", func(context.Context, BusinessEvent) error {
		return errors.New("permanent plugin failure")
	})
	_ = deadBus.Publish(context.Background(), BusinessEvent{EventName: BusinessEventQualificationExpiring})
	_, _ = deadBus.RetryDue(context.Background(), time.Now().Add(time.Hour))
	_, _ = deadBus.RetryDue(context.Background(), time.Now().Add(time.Hour))
	deadLetters := 0
	for _, record := range deadBus.RetryRecords() {
		if record.Status == BusinessRetryStatusDeadLetter {
			deadLetters++
		}
	}
	scenario.DeadLetters = deadLetters
	if deadLetters != 1 {
		scenario.Alerts = append(scenario.Alerts, fmt.Sprintf("dead letters %d, expected 1", deadLetters))
		scenario.Passed = false
	}
	report := jobsoak.NewReport(jobsoak.BoolFromEnv("SKOLL_H5_RACE_ENABLED"), scenario)
	report.Metadata = map[string]string{"scope": "business/plugin events", "defaultLanguage": "zh-CN"}
	if err := jobsoak.WriteReport(os.Getenv("SKOLL_H5_EVENT_SOAK_OUTPUT"), report); err != nil {
		t.Fatal(err)
	}
	if !report.Passed {
		t.Fatalf("business event soak thresholds failed: %+v", scenario.Alerts)
	}
}

func TestAfterCommitQueuePublishesOnlyOnCommit(t *testing.T) {
	bus := NewBusinessEventBus(nil)
	called := 0
	if _, err := bus.Subscribe(BusinessEventQualificationExpiring, "qualification.notice", func(context.Context, BusinessEvent) error {
		called++
		return nil
	}); err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}
	rolledBack := NewAfterCommitQueue(bus)
	if err := rolledBack.Enqueue(BusinessEvent{EventName: BusinessEventQualificationExpiring}); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	rolledBack.Rollback()
	if called != 0 {
		t.Fatalf("rollback should not publish events, got %d calls", called)
	}

	committed := NewAfterCommitQueue(bus)
	if err := committed.Enqueue(BusinessEvent{EventName: BusinessEventQualificationExpiring}); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	if err := committed.Commit(context.Background()); err != nil {
		t.Fatalf("commit failed: %v", err)
	}
	if called != 1 {
		t.Fatalf("commit should publish one event, got %d calls", called)
	}
}

func TestBusinessEventBusRejectsInvalidSubscriptionsAndEvents(t *testing.T) {
	bus := NewBusinessEventBus(nil)
	if _, err := bus.Subscribe("bad event", "handler", func(context.Context, BusinessEvent) error { return nil }); err == nil {
		t.Fatal("expected invalid event subscription error")
	}
	if err := bus.Publish(context.Background(), BusinessEvent{EventName: "bad event"}); err == nil {
		t.Fatal("expected invalid event publish error")
	}
}
