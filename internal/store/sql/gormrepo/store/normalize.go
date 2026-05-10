package store

import "strings"

func defaultNormalize(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}
