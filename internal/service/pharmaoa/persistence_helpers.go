package pharmaoa

import (
	"strconv"
	"strings"
)

func sequenceFromID(id, prefix string) int64 {
	value := strings.TrimPrefix(strings.TrimSpace(id), prefix)
	if value == id || value == "" {
		return 0
	}
	sequence, _ := strconv.ParseInt(value, 10, 64)
	return sequence
}
