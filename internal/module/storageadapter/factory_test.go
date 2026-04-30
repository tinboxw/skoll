package storageadapter

import (
	"strings"
	"testing"
)

func TestNewByMode(t *testing.T) {
	adapter, err := NewByMode("memory")
	if err != nil {
		t.Fatalf("expected memory mode to succeed, got error: %v", err)
	}
	if adapter == nil {
		t.Fatalf("expected non-nil adapter")
	}

	if _, err := NewByMode("mysql"); err == nil {
		t.Fatalf("expected mysql mode to fail when dsn is missing")
	}

	if _, err := NewByMode("postgres"); err == nil {
		t.Fatalf("expected postgres mode to fail when dsn is missing")
	}

	t.Run("mysql mode succeeds when dsn is provided", func(t *testing.T) {
		t.Setenv(EnvStorageMySQLDSN, "mysql://root:secret@tcp(localhost:3306)/skoll")
		adapter, err := NewByMode("mysql")
		if err != nil {
			t.Fatalf("expected mysql mode to succeed with dsn, got %v", err)
		}
		if adapter == nil {
			t.Fatalf("expected non-nil adapter")
		}
	})

	t.Run("postgres mode succeeds when dsn is provided", func(t *testing.T) {
		t.Setenv(EnvStoragePostgresDSN, "postgres://postgres:secret@localhost:5432/skoll")
		adapter, err := NewByMode("postgres")
		if err != nil {
			t.Fatalf("expected postgres mode to succeed with dsn, got %v", err)
		}
		if adapter == nil {
			t.Fatalf("expected non-nil adapter")
		}
	})

	if _, err := NewByMode("unknown"); err == nil {
		t.Fatalf("expected unknown mode to fail")
	}
}

func TestResolvePersistentBootstrapConfig(t *testing.T) {
	t.Run("mysql mode reads dedicated env first", func(t *testing.T) {
		cfg, err := ResolvePersistentBootstrapConfig(ModeMySQL, func(key string) string {
			switch key {
			case EnvStorageMySQLDSN:
				return "mysql://root:secret@tcp(localhost:3306)/skoll"
			case EnvStorageDSN:
				return "generic://dsn"
			default:
				return ""
			}
		})
		if err != nil {
			t.Fatalf("expected mysql config resolve success, got %v", err)
		}
		if cfg.SourceEnv != EnvStorageMySQLDSN {
			t.Fatalf("expected source env %s, got %s", EnvStorageMySQLDSN, cfg.SourceEnv)
		}
	})

	t.Run("postgres mode falls back to generic env", func(t *testing.T) {
		cfg, err := ResolvePersistentBootstrapConfig(ModePostgres, func(key string) string {
			if key == EnvStorageDSN {
				return "postgres://postgres:secret@localhost:5432/skoll"
			}
			return ""
		})
		if err != nil {
			t.Fatalf("expected postgres config resolve success, got %v", err)
		}
		if cfg.SourceEnv != EnvStorageDSN {
			t.Fatalf("expected source env %s, got %s", EnvStorageDSN, cfg.SourceEnv)
		}
	})

	t.Run("missing dsn returns helpful error", func(t *testing.T) {
		_, err := ResolvePersistentBootstrapConfig(ModeMySQL, func(string) string { return "" })
		if err == nil {
			t.Fatalf("expected missing dsn error")
		}
		if !strings.Contains(err.Error(), EnvStorageMySQLDSN) || !strings.Contains(err.Error(), EnvStorageDSN) {
			t.Fatalf("expected error to list acceptable env keys, got %v", err)
		}
	})

	t.Run("unsupported mode is rejected", func(t *testing.T) {
		_, err := ResolvePersistentBootstrapConfig("sqlite", func(string) string { return "dsn" })
		if err == nil {
			t.Fatalf("expected unsupported mode error")
		}
	})
}
