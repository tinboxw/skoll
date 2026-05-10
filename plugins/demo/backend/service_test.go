package main

import (
	"testing"
	"time"
)

func TestBuildOverviewWithEmptyPluginID(t *testing.T) {
	svc := NewDemoService()
	now := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)

	overview := svc.BuildOverview("", now)
	if overview.PluginID != "unknown" {
		t.Fatalf("expected plugin id unknown, got %q", overview.PluginID)
	}
	if overview.Status != "degraded" {
		t.Fatalf("expected degraded status, got %q", overview.Status)
	}
	if len(overview.Highlights) < 4 {
		t.Fatalf("expected extended highlights, got %d", len(overview.Highlights))
	}
}

func TestRecommendActionsPriorityCoverage(t *testing.T) {
	svc := NewDemoService()
	recs := svc.RecommendActions(DashboardContext{ActiveUsers: 10, ErrorCount: 2, RequestsPerMinute: 220})
	if len(recs) != 3 {
		t.Fatalf("expected 3 recommendations, got %d", len(recs))
	}
	if recs[0].Priority != "high" || recs[1].Priority != "medium" || recs[2].Priority != "low" {
		t.Fatalf("unexpected priorities: %+v", recs)
	}
}
