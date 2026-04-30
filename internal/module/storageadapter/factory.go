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
	case ModeMySQL:
		cfg, err := ResolvePersistentBootstrapConfig(resolved, nil)
		if err != nil {
			return nil, err
		}
		return newMySQLAdapter(cfg)
	case ModePostgres:
		cfg, err := ResolvePersistentBootstrapConfig(resolved, nil)
		if err != nil {
			return nil, err
		}
		return newPostgresAdapter(cfg)
	default:
		return nil, fmt.Errorf("unsupported storage adapter mode %q, valid: memory|mysql|postgres", resolved)
	}
}
