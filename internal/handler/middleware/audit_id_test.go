package middleware

import (
	"sync"
	"testing"
	"time"
)

func TestNewAuditIDIsUniqueAtFixedTimestamp(t *testing.T) {
	const workers = 32
	const perWorker = 100
	fixed := time.Date(2026, 7, 22, 22, 0, 0, 0, time.UTC)
	ids := make(chan string, workers*perWorker)
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				ids <- NewAuditID("audit-event", fixed).String()
			}
		}()
	}
	wg.Wait()
	close(ids)
	seen := make(map[string]struct{}, workers*perWorker)
	for id := range ids {
		if _, exists := seen[id]; exists {
			t.Fatalf("duplicate audit id %s", id)
		}
		seen[id] = struct{}{}
	}
}
