package metrics

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type Exporter interface {
	Export(metrics []Snapshot) ([]byte, error)
}

type JSONExporter struct {
	Pretty bool
}

func (e JSONExporter) Export(metrics []Snapshot) ([]byte, error) {
	if e.Pretty {
		return json.MarshalIndent(metrics, "", "  ")
	}
	return json.Marshal(metrics)
}

type TextExporter struct{}

func (TextExporter) Export(metrics []Snapshot) ([]byte, error) {
	lines := make([]string, 0, len(metrics))
	for i := range metrics {
		m := metrics[i]
		line := fmt.Sprintf("name=%s kind=%s", m.Name, m.Kind)
		if len(m.Labels) > 0 {
			line += " labels={" + formatLabels(m.Labels) + "}"
		}
		switch m.Kind {
		case KindHistogram:
			line += fmt.Sprintf(" count=%d sum=%f min=%f max=%f avg=%f", m.Count, m.Sum, m.Min, m.Max, m.Value)
		default:
			line += fmt.Sprintf(" value=%f", m.Value)
		}
		lines = append(lines, line)
	}
	return []byte(strings.Join(lines, "\n")), nil
}

func formatLabels(labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=\"%s\"", k, labels[k]))
	}
	return strings.Join(parts, ",")
}
