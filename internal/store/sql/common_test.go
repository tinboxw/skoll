package sql

import (
	"strings"
	"testing"
)

func TestParseDialect(t *testing.T) {
	t.Run("mysql", func(t *testing.T) {
		d, err := ParseDialect(" MySQL ")
		if err != nil {
			t.Fatalf("ParseDialect mysql error: %v", err)
		}
		if d != DialectMySQL {
			t.Fatalf("expected mysql dialect, got %q", d)
		}
	})

	t.Run("postgres", func(t *testing.T) {
		d, err := ParseDialect("POSTGRES")
		if err != nil {
			t.Fatalf("ParseDialect postgres error: %v", err)
		}
		if d != DialectPostgres {
			t.Fatalf("expected postgres dialect, got %q", d)
		}
	})

	t.Run("unsupported", func(t *testing.T) {
		_, err := ParseDialect("sqlite")
		if err == nil {
			t.Fatalf("expected unsupported dialect error")
		}
		if !strings.Contains(strings.ToLower(err.Error()), "unsupported sql dialect") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestNormalizeDSN(t *testing.T) {
	got := NormalizeDSN("  user:pass@tcp(127.0.0.1:3306)/db  ")
	if got != "user:pass@tcp(127.0.0.1:3306)/db" {
		t.Fatalf("unexpected normalized dsn: %q", got)
	}
}
