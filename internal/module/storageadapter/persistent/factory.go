package persistent

import (
	"fmt"

	"github.com/tinboxw/skoll/internal/module/storageadapter/contracts"
	"github.com/tinboxw/skoll/internal/module/storageadapter/persistent/db"
)

// NewMySQLAdapter creates a MySQL-backed adapter.
func NewMySQLAdapter(cfg BootstrapConfig) (contracts.Adapter, error) {
	if cfg.Mode != "mysql" {
		return nil, fmt.Errorf("mysql adapter requires mode=mysql, got %q", cfg.Mode)
	}
	gdb, err := db.Open(cfg.Mode, cfg.DSN, db.Options{})
	if err != nil {
		return nil, err
	}
	return NewAdapter(gdb, "")
}

// NewPostgresAdapter creates a Postgres-backed adapter.
func NewPostgresAdapter(cfg BootstrapConfig) (contracts.Adapter, error) {
	if cfg.Mode != "postgres" {
		return nil, fmt.Errorf("postgres adapter requires mode=postgres, got %q", cfg.Mode)
	}
	gdb, err := db.Open(cfg.Mode, cfg.DSN, db.Options{})
	if err != nil {
		return nil, err
	}
	return NewAdapter(gdb, "")
}
