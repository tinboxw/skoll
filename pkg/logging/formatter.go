package logging

import (
	"strings"

	"go.uber.org/zap"
)

// KV converts key-value tuples into zap fields.
func KV(keyvals ...any) []zap.Field {
	attrs := make([]zap.Field, 0, len(keyvals)/2+1)
	for i := 0; i+1 < len(keyvals); i += 2 {
		k, ok := keyvals[i].(string)
		if !ok || strings.TrimSpace(k) == "" {
			continue
		}
		attrs = append(attrs, zap.Any(k, keyvals[i+1]))
	}
	return attrs
}
