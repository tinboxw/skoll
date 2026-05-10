package mysql

import "strings"

func normalizeRoleKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}

func normalizeSettingKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}
