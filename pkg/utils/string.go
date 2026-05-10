package utils

import (
	"strings"
	"unicode/utf8"
)

// IsBlank reports whether the input contains only whitespace.
func IsBlank(v string) bool {
	return strings.TrimSpace(v) == ""
}

// NormalizeSpace trims and collapses consecutive whitespace into single spaces.
func NormalizeSpace(v string) string {
	parts := strings.Fields(v)
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " ")
}

// Coalesce returns the first non-blank string.
func Coalesce(values ...string) string {
	for i := range values {
		if !IsBlank(values[i]) {
			return values[i]
		}
	}
	return ""
}

// TruncateByRune truncates string by rune length.
func TruncateByRune(v string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	if utf8.RuneCountInString(v) <= maxRunes {
		return v
	}
	runes := []rune(v)
	return string(runes[:maxRunes])
}

// ContainsFold reports whether substr is within s using case-insensitive matching.
func ContainsFold(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
