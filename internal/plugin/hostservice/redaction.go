package hostservice

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

const maxHostMetadataBytes = 64 * 1024

var sensitiveKeyPattern = regexp.MustCompile(`(?i)(^|[_.-])(secret|password|passwd|token|credential|authorization|cookie|session|api[_-]?key|private[_-]?key)($|[_.-])`)

func containsSensitiveKey(values map[string]any) bool {
	for key, value := range values {
		if sensitiveKeyPattern.MatchString(strings.TrimSpace(key)) {
			return true
		}
		switch nested := value.(type) {
		case map[string]any:
			if containsSensitiveKey(nested) {
				return true
			}
		case map[string]string:
			for nestedKey := range nested {
				if sensitiveKeyPattern.MatchString(strings.TrimSpace(nestedKey)) {
					return true
				}
			}
		case []any:
			for _, item := range nested {
				if object, ok := item.(map[string]any); ok && containsSensitiveKey(object) {
					return true
				}
				if object, ok := item.(map[string]string); ok {
					for nestedKey := range object {
						if sensitiveKeyPattern.MatchString(strings.TrimSpace(nestedKey)) {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

func redactMap(values map[string]any) map[string]any {
	out := make(map[string]any, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if sensitiveKeyPattern.MatchString(key) {
			out[key] = "[REDACTED]"
			continue
		}
		out[key] = redactValue(value)
	}
	return out
}

func redactValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return redactMap(typed)
	case map[string]string:
		out := make(map[string]string, len(typed))
		for key, item := range typed {
			if sensitiveKeyPattern.MatchString(strings.TrimSpace(key)) {
				out[key] = "[REDACTED]"
			} else {
				out[key] = item
			}
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = redactValue(item)
		}
		return out
	default:
		return typed
	}
}

func validateMetadataSize(values map[string]any) error {
	raw, err := json.Marshal(values)
	if err != nil {
		return err
	}
	if len(raw) > maxHostMetadataBytes {
		return fmt.Errorf("plugin host metadata exceeds %d bytes", maxHostMetadataBytes)
	}
	return nil
}
