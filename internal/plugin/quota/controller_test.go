package quota

import (
	"errors"
	"testing"
	"time"
)

func testPolicy() Policy {
	limit := Limit{RatePerSecond: 2, Burst: 2, MaxConcurrent: 1}
	return Policy{
		Request: limit, HostCall: limit, Query: limit, Mutation: limit, Event: limit,
		Job: limit, Export: limit, Storage: limit, Process: limit,
		MaxPendingEvents: 10, MaxPendingJobs: 10, MaxFileBytes: 1024, MaxStorageBytes: 4096,
		MaxRequestBytes: 1024, MaxResponseBytes: 2048, RequestTimeout: time.Second,
		ProcessMemoryBytes: 1 << 20, ProcessMaxProcs: 1,
	}
}

func TestControllerIsolatesConcurrencyByPluginAndRecoversOnRelease(t *testing.T) {
	now := time.Date(2026, 7, 28, 10, 0, 0, 0, time.UTC)
	controller, err := newController(testPolicy(), func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	first, err := controller.Acquire("plugin-a", ResourceRequest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = controller.Acquire("plugin-a", ResourceRequest); err == nil {
		t.Fatal("same plugin exceeded concurrent request quota")
	}
	other, err := controller.Acquire("plugin-b", ResourceRequest)
	if err != nil {
		t.Fatalf("another plugin must retain isolated capacity: %v", err)
	}
	other.Release()
	first.Release()
	recovered, err := controller.Acquire("plugin-a", ResourceRequest)
	if err != nil {
		t.Fatalf("released capacity did not recover: %v", err)
	}
	recovered.Release()
}

func TestControllerThrottlesBurstAndRefillsAutomatically(t *testing.T) {
	now := time.Date(2026, 7, 28, 10, 0, 0, 0, time.UTC)
	controller, err := newController(testPolicy(), func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		lease, acquireErr := controller.Acquire("plugin-a", ResourceEvent)
		if acquireErr != nil {
			t.Fatal(acquireErr)
		}
		lease.Release()
	}
	_, err = controller.Acquire("plugin-a", ResourceEvent)
	var quotaErr *Error
	if !errors.As(err, &quotaErr) || quotaErr.Resource != ResourceEvent || quotaErr.RetryAfter <= 0 {
		t.Fatalf("expected retryable event quota error, got %v", err)
	}
	snapshot := controller.Snapshot("plugin-a")
	if snapshot[4].Rejected != 1 {
		t.Fatalf("rejection evidence was not recorded: %+v", snapshot[4])
	}
	now = now.Add(time.Second)
	lease, err := controller.Acquire("plugin-a", ResourceEvent)
	if err != nil {
		t.Fatalf("rate quota did not recover: %v", err)
	}
	lease.Release()
}

func TestPolicyRejectsUnsafeOrAmbiguousLimits(t *testing.T) {
	policy := testPolicy()
	policy.MaxStorageBytes = policy.MaxFileBytes - 1
	if err := policy.Validate(); err == nil {
		t.Fatal("storage quota below file quota must fail")
	}
	policy = testPolicy()
	policy.Query.MaxConcurrent = 0
	if err := policy.Validate(); err == nil {
		t.Fatal("missing query concurrency quota must fail")
	}
}

func TestControllerCapacityReservationsPreventConcurrentOvercommit(t *testing.T) {
	controller, err := NewController(testPolicy())
	if err != nil {
		t.Fatal(err)
	}
	first, err := controller.Reserve("plugin-a", ResourceStorage, 3000, 1000, 4096)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = controller.Reserve("plugin-a", ResourceStorage, 3000, 1000, 4096); err == nil {
		t.Fatal("concurrent capacity reservation exceeded storage quota")
	}
	if _, err = controller.Reserve("plugin-b", ResourceStorage, 3000, 1000, 4096); err != nil {
		t.Fatalf("reservation leaked across plugin boundary: %v", err)
	}
	first.Release()
	recovered, err := controller.Reserve("plugin-a", ResourceStorage, 3000, 1000, 4096)
	if err != nil {
		t.Fatalf("released reservation did not recover: %v", err)
	}
	recovered.Release()
}
