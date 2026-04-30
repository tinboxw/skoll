// Package db provides shared infrastructure for SQL-backed storage adapter
// implementations: connection opening, migration execution and transaction
// helpers. Concrete repositories live in sibling sub-packages and depend on
// *gorm.DB obtained via Open.
package db

import (
	"fmt"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Supported dialect identifiers. Memory mode is intentionally excluded; the
// memory adapter does not depend on this package.
const (
	DialectMySQL    = "mysql"
	DialectPostgres = "postgres"
	DialectSQLite   = "sqlite"
)

// Options tunes connection-pool and logging behaviour. Zero values fall back
// to sensible defaults.
type Options struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	SlowThreshold   time.Duration
	LogLevel        logger.LogLevel
}

func (o Options) withDefaults() Options {
	out := o
	if out.MaxOpenConns <= 0 {
		out.MaxOpenConns = 32
	}
	if out.MaxIdleConns <= 0 {
		out.MaxIdleConns = 8
	}
	if out.ConnMaxLifetime <= 0 {
		out.ConnMaxLifetime = 30 * time.Minute
	}
	if out.SlowThreshold <= 0 {
		out.SlowThreshold = 200 * time.Millisecond
	}
	if out.LogLevel == 0 {
		out.LogLevel = logger.Warn
	}
	return out
}

// Open creates a *gorm.DB for the given dialect and DSN.
//
// mode selects the driver dialect (mysql, postgres, sqlite). DSN is passed
// through to the driver. Callers typically obtain mode/DSN via
// persistent.ResolveBootstrapConfig.
func Open(mode, dsn string, opts Options) (*gorm.DB, error) {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if strings.TrimSpace(dsn) == "" {
		return nil, fmt.Errorf("db.Open: empty DSN for mode %q", mode)
	}

	var dialector gorm.Dialector
	switch mode {
	case DialectMySQL:
		dialector = mysql.Open(dsn)
	case DialectPostgres:
		dialector = postgres.Open(dsn)
	case DialectSQLite:
		dialector = sqlite.Open(dsn)
	default:
		return nil, fmt.Errorf("db.Open: unsupported dialect %q", mode)
	}

	o := opts.withDefaults()

	gdb, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(o.LogLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("db.Open: %w", err)
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, fmt.Errorf("db.Open: extract sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(o.MaxOpenConns)
	sqlDB.SetMaxIdleConns(o.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(o.ConnMaxLifetime)

	return gdb, nil
}

// OpenSQLite is a convenience wrapper that opens an in-process pure-Go SQLite
// database. It is intended for tests and local harnesses; production
// deployments should use Open with mysql/postgres.
func OpenSQLite(dsn string, opts Options) (*gorm.DB, error) {
	if strings.TrimSpace(dsn) == "" {
		dsn = "file::memory:?cache=shared"
	}
	return Open(DialectSQLite, dsn, opts)
}
