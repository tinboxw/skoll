package storageadapter

import (
	"fmt"
	"strings"

	"github.com/tinboxw/skoll/internal/module/storageadapter/memory"
	"github.com/tinboxw/skoll/internal/module/storageadapter/persistent"
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
		return memory.NewAdapter()
	case ModeMySQL:
		cfg, err := persistent.ResolveBootstrapConfig(resolved, nil)
		if err != nil {
			return nil, err
		}
		return persistent.NewMySQLAdapter(cfg)
	case ModePostgres:
		cfg, err := persistent.ResolveBootstrapConfig(resolved, nil)
		if err != nil {
			return nil, err
		}
		return persistent.NewPostgresAdapter(cfg)
	default:
		return nil, fmt.Errorf("unsupported storage adapter mode %q, valid: memory|mysql|postgres", resolved)
	}
}
