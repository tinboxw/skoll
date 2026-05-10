package main

import "testing"

func TestSnapshotComputesScoreAndTotals(t *testing.T) {
	svc := NewMetricsService()
	snapshot := svc.Snapshot(map[string]int{"api": 120, "jobs": 30}, 3)

	if snapshot.TotalRequests != 150 {
		t.Fatalf("expected total requests 150, got %d", snapshot.TotalRequests)
	}
	if snapshot.AvailabilityScore != 98 {
		t.Fatalf("expected availability score 98, got %d", snapshot.AvailabilityScore)
	}
}

func TestBuildAuditDigestTopLimit(t *testing.T) {
	svc := NewMetricsService()
	digest := svc.BuildAuditDigest([]string{"user.create", "role.bind", "user.create", "user.update"}, 2)

	if digest.TotalEvents != 4 {
		t.Fatalf("expected total events 4, got %d", digest.TotalEvents)
	}
	if len(digest.TopCategories) != 2 {
		t.Fatalf("expected 2 top categories, got %d", len(digest.TopCategories))
	}
	if digest.TopCategories[0].Category != "user.create" || digest.TopCategories[0].Count != 2 {
		t.Fatalf("unexpected top category: %+v", digest.TopCategories[0])
	}
}
