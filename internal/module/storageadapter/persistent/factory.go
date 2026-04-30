package persistent

import (
	"fmt"

	"github.com/tinboxw/skoll/internal/module/storageadapter/contracts"
	"github.com/tinboxw/skoll/internal/module/storageadapter/memory"
)

// NewMySQLAdapter returns a transitional adapter for mode rollout.
// A real MySQL-backed repository set will replace this in follow-up slices.
func NewMySQLAdapter(cfg BootstrapConfig) (contracts.Adapter, error) {
	if cfg.Mode != "mysql" {
		return nil, fmt.Errorf("mysql adapter requires mode=mysql, got %q", cfg.Mode)
	}
	return memory.NewAdapter()
}

// NewPostgresAdapter returns a transitional adapter for mode rollout.
// A real Postgres-backed repository set will replace this in follow-up slices.
func NewPostgresAdapter(cfg BootstrapConfig) (contracts.Adapter, error) {
	if cfg.Mode != "postgres" {
		return nil, fmt.Errorf("postgres adapter requires mode=postgres, got %q", cfg.Mode)
	}
	return memory.NewAdapter()
}
