package storageadapter

import (
	"fmt"
	"strings"
)

const (
	ModeMemory   = "memory"
	ModeMySQL    = "mysql"
	ModePostgres = "postgres"
)

func NewByMode(mode string) (Adapter, error) {
	resolved := strings.ToLower(strings.TrimSpace(mode))
	if resolved == "" {
		resolved = ModeMemory
	}

	switch resolved {
	case ModeMemory:
		return NewInMemoryAdapter(), nil
	case ModeMySQL, ModePostgres:
		return nil, fmt.Errorf("storage adapter mode %q is planned but not implemented yet", resolved)
	default:
		return nil, fmt.Errorf("unsupported storage adapter mode %q, valid: memory|mysql|postgres", resolved)
	}
}
