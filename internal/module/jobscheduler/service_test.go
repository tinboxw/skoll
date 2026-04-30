package jobscheduler

import (
	"testing"
	"time"
)

func TestServiceCreateRunHistory(t *testing.T) {
	svc := NewService()
	job := svc.Create("daily-sync", "0 0 * * *")
	if job.ID <= 0 {
		t.Fatalf("expected generated job id")
	}

	exec, err := svc.Run(job.ID)
	if err != nil {
		t.Fatalf("run job failed: %v", err)
	}
	if exec.Status != "success" {
		t.Fatalf("unexpected execution status: %s", exec.Status)
	}

	history := svc.History(job.ID, 10)
	if len(history) != 1 {
		t.Fatalf("expected 1 history record, got %d", len(history))
	}
	if history[0].ID != exec.ID {
		t.Fatalf("unexpected execution id in history: %d", history[0].ID)
	}
}

func TestServiceClaimRunConsistency(t *testing.T) {
	svc := NewService()
	job := svc.Create("daily-sync", "0 0 * * *")
	now := time.Unix(1710000000, 0).UTC()

	claim1, err := svc.ClaimRun(job.ID, "job-1:20260426T100000Z", "node-a", now)
	if err != nil {
		t.Fatalf("claim run failed: %v", err)
	}
	if !claim1.Claimed || claim1.DuplicateBlocked {
		t.Fatalf("expected accepted claim, got %+v", claim1)
	}

	claim2, err := svc.ClaimRun(job.ID, "job-1:20260426T100000Z", "node-a", now.Add(2*time.Second))
	if err != nil {
		t.Fatalf("idempotent claim failed: %v", err)
	}
	if !claim2.Claimed || claim2.DuplicateBlocked {
		t.Fatalf("expected idempotent accepted claim, got %+v", claim2)
	}

	dup, err := svc.ClaimRun(job.ID, "job-1:20260426T100000Z", "node-b", now.Add(5*time.Second))
	if err != nil {
		t.Fatalf("duplicate claim call should not return error: %v", err)
	}
	if dup.Claimed || !dup.DuplicateBlocked {
		t.Fatalf("expected duplicate blocked claim, got %+v", dup)
	}

	status := svc.ClaimStatus("job-1:20260426T100000Z")
	if !status.Claimed || status.InstanceID != "node-a" {
		t.Fatalf("unexpected claim status: %+v", status)
	}
}

func TestServiceClaimLeaseRenewal(t *testing.T) {
	svc := NewService()
	job := svc.Create("daily-sync", "0 0 * * *")
	now := time.Unix(1710000000, 0).UTC()

	_, err := svc.ClaimRun(job.ID, "job-1:20260426T100000Z", "node-a", now)
	if err != nil {
		t.Fatalf("claim run failed: %v", err)
	}

	renewed, err := svc.RenewClaimLease("job-1:20260426T100000Z", "node-a", 45, now.Add(5*time.Second))
	if err != nil {
		t.Fatalf("renew claim lease failed: %v", err)
	}
	if renewed.LeaseRenewalCount != 1 || renewed.LeaseUntilUnixSec == 0 {
		t.Fatalf("expected lease renewal fields, got %+v", renewed)
	}

	_, err = svc.RenewClaimLease("job-1:20260426T100000Z", "node-b", 45, now.Add(10*time.Second))
	if err != ErrClaimLeaseOwnerMismatch {
		t.Fatalf("expected owner mismatch error, got %v", err)
	}
}

func BenchmarkServiceClaimRunConsistency(b *testing.B) {
	svc := NewService()
	job := svc.Create("daily-sync", "0 0 * * *")
	now := time.Unix(1710000000, 0).UTC()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "job-1:" + time.Unix(now.Unix()+int64(i), 0).UTC().Format("20060102T150405Z")
		_, _ = svc.ClaimRun(job.ID, key, "node-a", now)
	}
}
