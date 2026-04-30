package persistent_test

import (
	"os"
	"testing"

	"github.com/tinboxw/skoll/internal/module/storageadapter/persistent"
	"github.com/tinboxw/skoll/internal/module/storageadapter/persistent/db"
)

// TestAdapter_Postgres exercises the full SQL adapter against a real
// PostgreSQL database. Skips when SKOLL_TEST_PG_DSN is unset so default
// CI does not require the dependency.
func TestAdapter_Postgres(t *testing.T) {
	dsn := os.Getenv("SKOLL_TEST_PG_DSN")
	if dsn == "" {
		t.Skip("SKOLL_TEST_PG_DSN unset; skipping postgres parity")
	}
	gdb, err := db.Open(db.DialectPostgres, dsn, db.Options{})
	if err != nil {
		t.Fatalf("open pg: %v", err)
	}
	a, err := persistent.NewAdapter(gdb, t.TempDir())
	if err != nil {
		t.Fatalf("adapter: %v", err)
	}
	if u := a.Users().Create("pg-alice", "pg-alice@example.com"); u.ID == 0 {
		t.Fatalf("create user")
	}
	if p := a.Plugins().Install("auth", "1.0.0", []string{"login"}); p.Name != "auth" {
		t.Fatalf("plugin install: %+v", p)
	}
}

// TestAdapter_MySQL exercises the full SQL adapter against a real MySQL
// database. Skips when SKOLL_TEST_MYSQL_DSN is unset.
func TestAdapter_MySQL(t *testing.T) {
	dsn := os.Getenv("SKOLL_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("SKOLL_TEST_MYSQL_DSN unset; skipping mysql parity")
	}
	gdb, err := db.Open(db.DialectMySQL, dsn, db.Options{})
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	a, err := persistent.NewAdapter(gdb, t.TempDir())
	if err != nil {
		t.Fatalf("adapter: %v", err)
	}
	if u := a.Users().Create("my-alice", "my-alice@example.com"); u.ID == 0 {
		t.Fatalf("create user")
	}
	if p := a.Plugins().Install("auth", "1.0.0", []string{"login"}); p.Name != "auth" {
		t.Fatalf("plugin install: %+v", p)
	}
}
