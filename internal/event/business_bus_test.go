package event

import (
	"context"
	"errors"
	"testing"
	"time"
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
