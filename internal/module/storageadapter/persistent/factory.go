package persistent

import (
	"fmt"

	"github.com/tinboxw/skoll/internal/module/storageadapter/contracts"
)

// NewMySQLAdapter creates a MySQL-backed adapter.
// The persistent repositories are not implemented yet in this slice.
func NewMySQLAdapter(cfg BootstrapConfig) (contracts.Adapter, error) {
	if cfg.Mode != "mysql" {
		return nil, fmt.Errorf("mysql adapter requires mode=mysql, got %q", cfg.Mode)
	}
	return nil, fmt.Errorf("mysql storage adapter is not implemented yet")
}

// NewPostgresAdapter creates a Postgres-backed adapter.
// The persistent repositories are not implemented yet in this slice.
func NewPostgresAdapter(cfg BootstrapConfig) (contracts.Adapter, error) {
	if cfg.Mode != "postgres" {
		return nil, fmt.Errorf("postgres adapter requires mode=postgres, got %q", cfg.Mode)
	}
	return nil, fmt.Errorf("postgres storage adapter is not implemented yet")
}
