package clickhouse

import "sync"

type MetricsStore struct {
	mu     sync.RWMutex
	gauges map[string]float64
}

func NewMetricsStore() *MetricsStore {
	return &MetricsStore{gauges: map[string]float64{}}
}

func (s *MetricsStore) SetGauge(name string, value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gauges[name] = value
}

func (s *MetricsStore) Snapshot() map[string]float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]float64, len(s.gauges))
	for k, v := range s.gauges {
		out[k] = v
	}
	return out
}
