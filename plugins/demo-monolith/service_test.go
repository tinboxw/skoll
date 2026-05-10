package main

import (
	"strings"
	"testing"
	"time"
)

func TestPingIncludesTimestamp(t *testing.T) {
	svc := NewService()
	v := svc.Ping(time.Date(2026, 5, 10, 1, 2, 3, 0, time.UTC))
	if !strings.HasPrefix(v, "pong@2026-05-10T01:02:03Z") {
		t.Fatalf("unexpected ping value: %q", v)
	}
}

func TestBuildManifestAndWidgets(t *testing.T) {
	svc := NewService()
	manifest := svc.BuildManifest()
	if manifest.EntryPath != "/plugins/demo-monolith" {
		t.Fatalf("unexpected entry path: %q", manifest.EntryPath)
	}

	widgets := svc.BuildWidgets(48, 2.1)
	if len(widgets) != 3 {
		t.Fatalf("expected 3 widgets, got %d", len(widgets))
	}
	if widgets[1].Status != "warn" {
		t.Fatalf("expected warn status for error_rate widget, got %q", widgets[1].Status)
	}
}
