package logging

import "log/slog"

// KV converts key-value tuples into slog attributes.
func KV(keyvals ...any) []any {
	attrs := make([]any, 0, len(keyvals))
	for i := 0; i+1 < len(keyvals); i += 2 {
		k, ok := keyvals[i].(string)
		if !ok {
			continue
		}
		attrs = append(attrs, slog.Any(k, keyvals[i+1]))
	}
	return attrs
}
