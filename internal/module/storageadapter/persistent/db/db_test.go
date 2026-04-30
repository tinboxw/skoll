package db

import (
	"context"
	"errors"
	"testing"

	"gorm.io/gorm"
)

func TestOpenRejectsEmptyDSN(t *testing.T) {
	if _, err := Open(DialectMySQL, "", Options{}); err == nil {
		t.Fatalf("expected error for empty DSN")
	}
}

func TestOpenRejectsUnknownDialect(t *testing.T) {
	if _, err := Open("oracle", "x", Options{}); err == nil {
		t.Fatalf("expected error for unsupported dialect")
	}
}

func TestOpenSQLiteInMemory(t *testing.T) {
	gdb, err := OpenSQLite("", Options{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if gdb == nil {
		t.Fatalf("expected non-nil *gorm.DB")
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatalf("extract sql.DB: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
}

type smokeRow struct {
	ID    uint   `gorm:"primaryKey"`
	Value string `gorm:"size:64"`
}

func (smokeRow) TableName() string { return "smoke_rows" }

type smokeMigrator struct{}

func (smokeMigrator) Models() []any { return []any{&smokeRow{}} }

func TestMigrateAndTransaction(t *testing.T) {
	gdb, err := OpenSQLite("", Options{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := gdb.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	if err := MigrateAll(gdb, smokeMigrator{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	ctx := context.Background()
	if err := WithTx(ctx, gdb, func(tx *gorm.DB) error {
		return tx.Create(&smokeRow{Value: "hello"}).Error
	}); err != nil {
		t.Fatalf("commit tx: %v", err)
	}

	rollbackErr := errors.New("force rollback")
	if err := WithTx(ctx, gdb, func(tx *gorm.DB) error {
		if err := tx.Create(&smokeRow{Value: "world"}).Error; err != nil {
			return err
		}
		return rollbackErr
	}); !errors.Is(err, rollbackErr) {
		t.Fatalf("expected rollback error to propagate, got %v", err)
	}

	var rows []smokeRow
	if err := gdb.WithContext(ctx).Find(&rows).Error; err != nil {
		t.Fatalf("find: %v", err)
	}
	if len(rows) != 1 || rows[0].Value != "hello" {
		t.Fatalf("unexpected rows after rollback: %+v", rows)
	}
}
