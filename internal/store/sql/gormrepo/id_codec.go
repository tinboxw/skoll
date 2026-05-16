package gormrepo

import (
	"strconv"
	"strings"
)

func parseUintID(raw string) uint64 {
	v, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0
	}
	return v
}

func formatUintID(v uint64) string {
	return strconv.FormatUint(v, 10)
}
