package metrics

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var promInvalidChar = regexp.MustCompile(`[^a-zA-Z0-9_:]`)

type PrometheusExporter struct{}

func (PrometheusExporter) Export(metrics []Snapshot) ([]byte, error) {
	lines := make([]string, 0, len(metrics)*3)
	for i := range metrics {
		m := metrics[i]
		name := sanitizeName(m.Name)
		labels := prometheusLabels(m.Labels)

		switch m.Kind {
		case KindCounter:
			lines = append(lines, fmt.Sprintf("%s%s %g", name+"_total", labels, m.Value))
		case KindGauge:
			lines = append(lines, fmt.Sprintf("%s%s %g", name, labels, m.Value))
		case KindHistogram:
			lines = append(lines, fmt.Sprintf("%s_count%s %d", name, labels, m.Count))
			lines = append(lines, fmt.Sprintf("%s_sum%s %g", name, labels, m.Sum))
			lines = append(lines, fmt.Sprintf("%s_avg%s %g", name, labels, m.Value))
		}
	}
	return []byte(strings.Join(lines, "\n")), nil
}

func sanitizeName(name string) string {
	if name == "" {
		return "metric"
	}
	return promInvalidChar.ReplaceAllString(name, "_")
}

func prometheusLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		safeKey := sanitizeName(k)
		safeVal := strings.ReplaceAll(labels[k], "\"", "\\\"")
		parts = append(parts, fmt.Sprintf("%s=\"%s\"", safeKey, safeVal))
	}
	return "{" + strings.Join(parts, ",") + "}"
}
