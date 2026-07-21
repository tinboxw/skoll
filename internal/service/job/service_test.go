package job

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestServiceRetriesAndDeadLettersJob(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 22, 18, 0, 0, 0, time.UTC)
	service := NewService(NewMemoryRepository(), func() time.Time { return now })
	input := ScheduleInput{
		ID: "job-1", Namespace: "plugin.pharma_oa", Kind: "report-export", IdempotencyKey: "report-2026-07",
		Payload: []byte(`{"report":"inventory"}`), RunAt: now.Add(time.Minute), MaxAttempts: 2,
	}
	created, err := service.Schedule(ctx, input)
	if err != nil || created.Status != StatusScheduled {
		t.Fatalf("Schedule error=%v item=%+v", err, created)
	}
	duplicate, err := service.Schedule(ctx, input)
	if err != nil || duplicate.ID != created.ID || !duplicate.CreatedAt.Equal(created.CreatedAt) {
		t.Fatalf("duplicate schedule was not idempotent: item=%+v err=%v", duplicate, err)
	}
	retriedInput := input
	retriedInput.ID = "job-retried-request"
	retried, err := service.Schedule(ctx, retriedInput)
	if err != nil || retried.ID != created.ID {
		t.Fatalf("idempotency key did not return persisted job: item=%+v err=%v", retried, err)
	}
	conflict := input
	conflict.Kind = "different-kind"
	if _, err := service.Schedule(ctx, conflict); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected schedule conflict, got %v", err)
	}
	if leased, err := service.LeaseDue(ctx, LeaseInput{WorkerID: "worker-1", Limit: 1, LeaseDuration: time.Minute}); err != nil || len(leased) != 0 {
		t.Fatalf("future job was leased: items=%+v err=%v", leased, err)
	}

	now = now.Add(time.Minute)
	leased, err := service.LeaseDue(ctx, LeaseInput{WorkerID: "worker-1", Limit: 1, LeaseDuration: time.Minute})
	if err != nil || len(leased) != 1 || leased[0].AttemptCount != 1 || leased[0].LeaseToken == "" {
		t.Fatalf("first lease failed: items=%+v err=%v", leased, err)
	}
	firstToken := leased[0].LeaseToken
	if duplicateLease, err := service.LeaseDue(ctx, LeaseInput{WorkerID: "worker-2", Limit: 1, LeaseDuration: time.Minute}); err != nil || len(duplicateLease) != 0 {
		t.Fatalf("active lease allowed duplicate execution: items=%+v err=%v", duplicateLease, err)
	}
	failed, err := service.Fail(ctx, FailInput{JobID: created.ID, LeaseToken: firstToken, Error: "temporary outage", RetryAfter: time.Minute})
	if err != nil || failed.Status != StatusRetryWait || failed.AttemptCount != 1 {
		t.Fatalf("first failure did not schedule retry: item=%+v err=%v", failed, err)
	}

	now = now.Add(time.Minute)
	retry, err := service.LeaseDue(ctx, LeaseInput{WorkerID: "worker-2", Limit: 1, LeaseDuration: time.Minute})
	if err != nil || len(retry) != 1 || retry[0].AttemptCount != 2 || retry[0].LeaseToken == firstToken {
		t.Fatalf("retry lease failed: items=%+v err=%v", retry, err)
	}
	dead, err := service.Fail(ctx, FailInput{JobID: created.ID, LeaseToken: retry[0].LeaseToken, Error: "permanent outage", RetryAfter: time.Minute})
	if err != nil || dead.Status != StatusDeadLetter || dead.DeadLetteredAt == nil {
		t.Fatalf("retry limit did not create dead letter: item=%+v err=%v", dead, err)
	}
	if _, err := service.Complete(ctx, CompleteInput{JobID: created.ID, LeaseToken: firstToken}); !errors.Is(err, ErrLeaseLost) {
		t.Fatalf("stale lease completed dead letter: %v", err)
	}
	items, err := service.List(ctx, Filter{Namespace: input.Namespace, Status: StatusDeadLetter})
	if err != nil || len(items) != 1 || items[0].LastError != "permanent outage" {
		t.Fatalf("dead letter was not observable: items=%+v err=%v", items, err)
	}
}

func TestMemoryRepositoryLeasesJobOnceConcurrently(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 22, 19, 0, 0, 0, time.UTC)
	service := NewService(NewMemoryRepository(), func() time.Time { return now })
	if _, err := service.Schedule(ctx, ScheduleInput{ID: "job-concurrent", Namespace: "system", Kind: "scan", MaxAttempts: 1}); err != nil {
		t.Fatalf("Schedule error: %v", err)
	}

	const workers = 24
	start := make(chan struct{})
	counts := make(chan int, workers)
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			items, err := service.LeaseDue(ctx, LeaseInput{WorkerID: string(rune('a' + i)), Limit: 1, LeaseDuration: time.Minute})
			counts <- len(items)
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(counts)
	close(errs)
	total := 0
	for count := range counts {
		total += count
	}
	for err := range errs {
		if err != nil {
			t.Fatalf("LeaseDue error: %v", err)
		}
	}
	if total != 1 {
		t.Fatalf("expected one lease, got %d", total)
	}
}

func TestServiceCompletesCurrentLease(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 22, 19, 30, 0, 0, time.UTC)
	service := NewService(NewMemoryRepository(), func() time.Time { return now })
	if _, err := service.Schedule(ctx, ScheduleInput{ID: "job-success", Namespace: "system", Kind: "snapshot", MaxAttempts: 1}); err != nil {
		t.Fatalf("Schedule error: %v", err)
	}
	leased, err := service.LeaseDue(ctx, LeaseInput{WorkerID: "worker-success", Limit: 1, LeaseDuration: time.Minute})
	if err != nil || len(leased) != 1 {
		t.Fatalf("LeaseDue error=%v items=%+v", err, leased)
	}
	completed, err := service.Complete(ctx, CompleteInput{JobID: leased[0].ID, LeaseToken: leased[0].LeaseToken, Result: []byte(`{"rows":42}`)})
	if err != nil || completed.Status != StatusSucceeded || completed.CompletedAt == nil || string(completed.Result) != `{"rows":42}` {
		t.Fatalf("Complete error=%v item=%+v", err, completed)
	}
	if _, err := service.Complete(ctx, CompleteInput{JobID: leased[0].ID, LeaseToken: leased[0].LeaseToken}); !errors.Is(err, ErrLeaseLost) {
		t.Fatalf("completed lease accepted twice: %v", err)
	}
}
