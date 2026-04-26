package service

import (
	"sync"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain"
)

type fakeClock struct {
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	return c.now
}

func TestSystemServiceHealth(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := &fakeClock{now: base}

	svc := NewSystemService("v1.0.0-test", clock)
	clock.now = base.Add(5 * time.Second)

	health := svc.Health()
	if health.Status != "ok" {
		t.Fatalf("expected status ok, got %q", health.Status)
	}
	if health.Version != "v1.0.0-test" {
		t.Fatalf("expected version v1.0.0-test, got %q", health.Version)
	}
	if health.Uptime != 5*time.Second {
		t.Fatalf("expected uptime 5s, got %s", health.Uptime)
	}
}

func TestSystemServiceReadiness(t *testing.T) {
	svc := NewSystemService("v1.0.0-test", &fakeClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})

	if got := svc.Readiness(); got != domain.ReadinessNotReady {
		t.Fatalf("expected not_ready initially, got %q", got)
	}

	svc.SetReady(true)
	if got := svc.Readiness(); got != domain.ReadinessReady {
		t.Fatalf("expected ready after SetReady(true), got %q", got)
	}
}

func TestSystemServiceHealthCacheTTL(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := &fakeClock{now: base}
	svc := NewSystemService("v1.0.0-test", clock)

	first := svc.Health()

	clock.now = base.Add(100 * time.Millisecond)
	second := svc.Health()
	if second.Uptime != first.Uptime {
		t.Fatalf("expected cached health within TTL, got different uptime: %s vs %s", second.Uptime, first.Uptime)
	}

	clock.now = base.Add(250 * time.Millisecond)
	third := svc.Health()
	if third.Uptime == second.Uptime {
		t.Fatalf("expected refreshed health after TTL, got same uptime %s", third.Uptime)
	}
}

func TestSystemServiceHealthConcurrent(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	svc := NewSystemService("v1.0.0-test", clock)

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				health := svc.Health()
				if health.Status != "ok" {
					t.Errorf("unexpected status %q", health.Status)
					return
				}
			}
		}()
	}
	wg.Wait()
}

func BenchmarkSystemServiceHealthParallel(b *testing.B) {
	svc := NewSystemService("bench", nil)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = svc.Health()
		}
	})
}
