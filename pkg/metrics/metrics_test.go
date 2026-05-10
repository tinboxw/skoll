package metrics

import (
	"strings"
	"testing"
)

func TestCollectorSnapshot(t *testing.T) {
	c := NewCollector()
	c.IncCounter("http_requests", map[string]string{"method": "GET"})
	c.AddCounter("http_requests", 2, map[string]string{"method": "GET"})
	c.SetGauge("inflight", 3, nil)
	c.Observe("latency_ms", 10, nil)
	c.Observe("latency_ms", 20, nil)

	snap := c.Snapshot()
	if len(snap) != 3 {
		t.Fatalf("expected 3 snapshots, got %d", len(snap))
	}
}

func TestExporters(t *testing.T) {
	c := NewCollector()
	c.IncCounter("http_requests", map[string]string{"method": "GET"})
	snap := c.Snapshot()

	j, err := (JSONExporter{}).Export(snap)
	if err != nil || len(j) == 0 {
		t.Fatalf("json export failed: len=%d err=%v", len(j), err)
	}

	text, err := (TextExporter{}).Export(snap)
	if err != nil || len(text) == 0 {
		t.Fatalf("text export failed: len=%d err=%v", len(text), err)
	}

	prom, err := (PrometheusExporter{}).Export(snap)
	if err != nil {
		t.Fatalf("prometheus export failed: %v", err)
	}
	if !strings.Contains(string(prom), "http_requests_total") {
		t.Fatalf("expected prometheus counter suffix in output: %s", string(prom))
	}
}
