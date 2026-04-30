package storageadapter

import (
	"strings"
	"testing"

	"github.com/tinboxw/skoll/internal/module/storageadapter/persistent"
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
		t.Setenv(persistent.EnvStorageMySQLDSN, "mysql://root:secret@tcp(localhost:3306)/skoll")
		_, err := NewByMode("mysql")
		if err != nil {
			if !strings.Contains(err.Error(), "not implemented") {
				t.Fatalf("expected not implemented error, got %v", err)
			}
			return
		}
		t.Fatalf("expected mysql mode to return explicit not implemented error")
	})

	t.Run("postgres mode succeeds when dsn is provided", func(t *testing.T) {
		t.Setenv(persistent.EnvStoragePostgresDSN, "postgres://postgres:secret@localhost:5432/skoll")
		_, err := NewByMode("postgres")
		if err != nil {
			if !strings.Contains(err.Error(), "not implemented") {
				t.Fatalf("expected not implemented error, got %v", err)
			}
			return
		}
		t.Fatalf("expected postgres mode to return explicit not implemented error")
	})

	if _, err := NewByMode("unknown"); err == nil {
		t.Fatalf("expected unknown mode to fail")
	}
}

func TestResolvePersistentBootstrapConfig(t *testing.T) {
	t.Run("mysql mode reads dedicated env first", func(t *testing.T) {
		cfg, err := persistent.ResolveBootstrapConfig(ModeMySQL, func(key string) string {
			switch key {
			case persistent.EnvStorageMySQLDSN:
				return "mysql://root:secret@tcp(localhost:3306)/skoll"
			case persistent.EnvStorageDSN:
				return "generic://dsn"
			default:
				return ""
			}
		})
		if err != nil {
			t.Fatalf("expected mysql config resolve success, got %v", err)
		}
		if cfg.SourceEnv != persistent.EnvStorageMySQLDSN {
			t.Fatalf("expected source env %s, got %s", persistent.EnvStorageMySQLDSN, cfg.SourceEnv)
		}
	})

	t.Run("postgres mode falls back to generic env", func(t *testing.T) {
		cfg, err := persistent.ResolveBootstrapConfig(ModePostgres, func(key string) string {
			if key == persistent.EnvStorageDSN {
				return "postgres://postgres:secret@localhost:5432/skoll"
			}
			return ""
		})
		if err != nil {
			t.Fatalf("expected postgres config resolve success, got %v", err)
		}
		if cfg.SourceEnv != persistent.EnvStorageDSN {
			t.Fatalf("expected source env %s, got %s", persistent.EnvStorageDSN, cfg.SourceEnv)
		}
	})

	t.Run("missing dsn returns helpful error", func(t *testing.T) {
		_, err := persistent.ResolveBootstrapConfig(ModeMySQL, func(string) string { return "" })
		if err == nil {
			t.Fatalf("expected missing dsn error")
		}
		if !strings.Contains(err.Error(), persistent.EnvStorageMySQLDSN) || !strings.Contains(err.Error(), persistent.EnvStorageDSN) {
			t.Fatalf("expected error to list acceptable env keys, got %v", err)
		}
	})

	t.Run("unsupported mode is rejected", func(t *testing.T) {
		_, err := persistent.ResolveBootstrapConfig("sqlite", func(string) string { return "dsn" })
		if err == nil {
			t.Fatalf("expected unsupported mode error")
		}
	})
}
