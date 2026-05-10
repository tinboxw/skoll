package metrics

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type Kind string

const (
	KindCounter   Kind = "counter"
	KindGauge     Kind = "gauge"
	KindHistogram Kind = "histogram"
)

type Snapshot struct {
	Name      string
	Kind      Kind
	Labels    map[string]string
	Value     float64
	Count     uint64
	Sum       float64
	Min       float64
	Max       float64
	UpdatedAt time.Time
}

type Collector struct {
	mu         sync.RWMutex
	counters   map[string]Snapshot
	gauges     map[string]Snapshot
	histograms map[string]Snapshot
}

func NewCollector() *Collector {
	return &Collector{
		counters:   make(map[string]Snapshot),
		gauges:     make(map[string]Snapshot),
		histograms: make(map[string]Snapshot),
	}
}

func (c *Collector) IncCounter(name string, labels map[string]string) {
	c.AddCounter(name, 1, labels)
}

func (c *Collector) AddCounter(name string, delta float64, labels map[string]string) {
	if delta < 0 {
		return
	}
	key := metricKey(name, labels)
	now := time.Now().UTC()

	c.mu.Lock()
	defer c.mu.Unlock()

	s, exists := c.counters[key]
	if !exists {
		s = Snapshot{Name: name, Kind: KindCounter, Labels: cloneLabels(labels)}
	}
	s.Value += delta
	s.UpdatedAt = now
	c.counters[key] = s
}

func (c *Collector) SetGauge(name string, value float64, labels map[string]string) {
	key := metricKey(name, labels)
	now := time.Now().UTC()

	c.mu.Lock()
	defer c.mu.Unlock()

	c.gauges[key] = Snapshot{
		Name:      name,
		Kind:      KindGauge,
		Labels:    cloneLabels(labels),
		Value:     value,
		UpdatedAt: now,
	}
}

func (c *Collector) Observe(name string, value float64, labels map[string]string) {
	key := metricKey(name, labels)
	now := time.Now().UTC()

	c.mu.Lock()
	defer c.mu.Unlock()

	s, exists := c.histograms[key]
	if !exists {
		s = Snapshot{
			Name:   name,
			Kind:   KindHistogram,
			Labels: cloneLabels(labels),
			Min:    value,
			Max:    value,
		}
	}
	s.Count++
	s.Sum += value
	if value < s.Min {
		s.Min = value
	}
	if value > s.Max {
		s.Max = value
	}
	s.Value = s.Sum / float64(s.Count)
	s.UpdatedAt = now
	c.histograms[key] = s
}

func (c *Collector) Snapshot() []Snapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()

	all := make([]Snapshot, 0, len(c.counters)+len(c.gauges)+len(c.histograms))
	for _, s := range c.counters {
		all = append(all, cloneSnapshot(s))
	}
	for _, s := range c.gauges {
		all = append(all, cloneSnapshot(s))
	}
	for _, s := range c.histograms {
		all = append(all, cloneSnapshot(s))
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Name == all[j].Name {
			return all[i].Kind < all[j].Kind
		}
		return all[i].Name < all[j].Name
	})
	return all
}

func metricKey(name string, labels map[string]string) string {
	parts := []string{name}
	if len(labels) == 0 {
		return name
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, labels[k]))
	}
	return strings.Join(parts, "|")
}

func cloneLabels(labels map[string]string) map[string]string {
	if len(labels) == 0 {
		return nil
	}
	out := make(map[string]string, len(labels))
	for k, v := range labels {
		out[k] = v
	}
	return out
}

func cloneSnapshot(s Snapshot) Snapshot {
	s.Labels = cloneLabels(s.Labels)
	return s
}
