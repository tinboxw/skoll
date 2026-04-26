package app

import (
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
)

const (
	maxStatusCode  = 599
	pathLabelOther = "__other__"
)

var metricsPathWhitelist = map[string]struct{}{
	"/health":     {},
	"/ready":      {},
	"/metrics":    {},
	"/admin/ping": {},
}

type metricsPathTemplate struct {
	prefix string
	label  string
}

var metricsPathTemplates = []metricsPathTemplate{
	{prefix: "/admin/", label: "/admin/:path"},
}

type Metrics struct {
	requestsTotal atomic.Uint64

	pathHealth  atomic.Uint64
	pathReady   atomic.Uint64
	pathMetrics atomic.Uint64
	pathAdmin   atomic.Uint64
	pathAdminT  atomic.Uint64
	pathOther   atomic.Uint64

	statusTotal [maxStatusCode + 1]atomic.Uint64
}

func NewMetrics() *Metrics {
	return &Metrics{}
}

func (m *Metrics) ObserveRequest(path string, statusCode int) {
	m.requestsTotal.Add(1)

	switch normalizePathLabel(path) {
	case "/health":
		m.pathHealth.Add(1)
	case "/ready":
		m.pathReady.Add(1)
	case "/metrics":
		m.pathMetrics.Add(1)
	case "/admin/ping":
		m.pathAdmin.Add(1)
	case "/admin/:path":
		m.pathAdminT.Add(1)
	default:
		m.pathOther.Add(1)
	}

	if statusCode >= 0 && statusCode <= maxStatusCode {
		m.statusTotal[statusCode].Add(1)
	}
}

func (m *Metrics) Prometheus() string {
	var b strings.Builder
	b.WriteString("# HELP skoll_http_requests_total Total number of HTTP requests.\n")
	b.WriteString("# TYPE skoll_http_requests_total counter\n")
	b.WriteString(fmt.Sprintf("skoll_http_requests_total %d\n", m.requestsTotal.Load()))

	b.WriteString("# HELP skoll_http_requests_path_total Total number of HTTP requests by path.\n")
	b.WriteString("# TYPE skoll_http_requests_path_total counter\n")
	paths := map[string]uint64{
		"/health":      m.pathHealth.Load(),
		"/ready":       m.pathReady.Load(),
		"/metrics":     m.pathMetrics.Load(),
		"/admin/ping":  m.pathAdmin.Load(),
		"/admin/:path": m.pathAdminT.Load(),
		pathLabelOther: m.pathOther.Load(),
	}
	for path, total := range paths {
		if total == 0 {
			delete(paths, path)
		}
	}
	for _, path := range sortedPathKeys(paths) {
		b.WriteString(fmt.Sprintf("skoll_http_requests_path_total{path=%q} %d\n", path, paths[path]))
	}

	b.WriteString("# HELP skoll_http_responses_status_total Total number of HTTP responses by status code.\n")
	b.WriteString("# TYPE skoll_http_responses_status_total counter\n")
	statusMap := make(map[int]uint64)
	for code := 0; code <= maxStatusCode; code++ {
		total := m.statusTotal[code].Load()
		if total > 0 {
			statusMap[code] = total
		}
	}
	for _, code := range sortedStatusKeys(statusMap) {
		b.WriteString(fmt.Sprintf("skoll_http_responses_status_total{code=%q} %d\n", fmt.Sprintf("%d", code), statusMap[code]))
	}

	provenance := strings.TrimSpace(dashboardJWTProvenanceMetricsPrometheus())
	if provenance != "" {
		b.WriteString("\n")
		b.WriteString(provenance)
		b.WriteString("\n")
	}

	return b.String()
}

func sortedPathKeys(m map[string]uint64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedStatusKeys(m map[int]uint64) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

func normalizePathLabel(path string) string {
	if _, ok := metricsPathWhitelist[path]; ok {
		return path
	}
	for _, template := range metricsPathTemplates {
		if strings.HasPrefix(path, template.prefix) {
			return template.label
		}
	}
	return pathLabelOther
}
