package main

type MetricsService struct{}

func (s *MetricsService) Snapshot() map[string]int {
	return map[string]int{"ok": 1}
}
