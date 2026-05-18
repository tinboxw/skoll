package main

import "sort"

type QueueLoad struct {
	Name   string `json:"name"`
	Depth  int    `json:"depth"`
	Status string `json:"status"`
}

// MetricsSnapshot represents aggregated runtime metrics for the analytics demo plugin.
type MetricsSnapshot struct {
	AvailabilityScore int            `json:"availabilityScore"`
	TotalRequests     int            `json:"totalRequests"`
	ErrorRequests     int            `json:"errorRequests"`
	RouteHit          map[string]int `json:"routeHit"`
	Queues            []QueueLoad    `json:"queues"`
	Narrative         []string       `json:"narrative"`
}

// AuditDigest contains the top event categories from audit logs.
type AuditDigest struct {
	TopCategories []CategoryCount `json:"topCategories"`
	TotalEvents   int             `json:"totalEvents"`
	Window        string          `json:"window"`
	Actions       []string        `json:"actions"`
}

// CategoryCount binds one event category to its count.
type CategoryCount struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

// MetricsService provides analytics methods for backend-only plugins.
type MetricsService struct{}

// NewMetricsService creates the analytics service.
func NewMetricsService() *MetricsService {
	return &MetricsService{}
}

// Snapshot builds a metrics snapshot from route-hit data.
//
// Parameters:
// - routeHit: route name -> request count map.
// - errorRequests: number of failed requests in the same period.
func (s *MetricsService) Snapshot(routeHit map[string]int, errorRequests int) MetricsSnapshot {
	total := 0
	for _, count := range routeHit {
		total += count
	}
	if errorRequests < 0 {
		errorRequests = 0
	}
	score := 100
	if total > 0 {
		score = 100 - (errorRequests * 100 / total)
	}
	if score < 0 {
		score = 0
	}

	queues := []QueueLoad{
		{Name: "ingest", Depth: routeHit["api"] / 6, Status: "steady"},
		{Name: "jobs", Depth: routeHit["jobs"] / 4, Status: "watch"},
	}
	if cacheCount, ok := routeHit["cache"]; ok {
		queues = append(queues, QueueLoad{Name: "cache", Depth: cacheCount / 3, Status: "fast"})
	}

	return MetricsSnapshot{
		AvailabilityScore: score,
		TotalRequests:     total,
		ErrorRequests:     errorRequests,
		RouteHit:          routeHit,
		Queues:            queues,
		Narrative: []string{
			"Backend analytics is shaping a narrative homepage rather than a blank diagnostics screen.",
			"Route traffic remains balanced enough to demonstrate queue pressure and audit priority in one view.",
		},
	}
}

// BuildAuditDigest calculates top event categories from raw event names.
//
// Usage example:
//
//	digest := svc.BuildAuditDigest([]string{"user.create", "user.update", "user.create"}, 2)
func (s *MetricsService) BuildAuditDigest(events []string, limit int) AuditDigest {
	if limit <= 0 {
		limit = 3
	}

	freq := make(map[string]int, len(events))
	for _, event := range events {
		if event == "" {
			continue
		}
		freq[event]++
	}

	top := make([]CategoryCount, 0, len(freq))
	for category, count := range freq {
		top = append(top, CategoryCount{Category: category, Count: count})
	}
	sort.Slice(top, func(i, j int) bool {
		if top[i].Count == top[j].Count {
			return top[i].Category < top[j].Category
		}
		return top[i].Count > top[j].Count
	})
	if len(top) > limit {
		top = top[:limit]
	}

	actions := make([]string, 0, len(top))
	for _, item := range top {
		actions = append(actions, "Review "+item.Category+" because it dominates the current audit window")
	}

	return AuditDigest{TopCategories: top, TotalEvents: len(events), Window: "last 24 hours", Actions: actions}
}
