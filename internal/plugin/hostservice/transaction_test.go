package hostservice

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type pluginOperationHeader struct {
	ID string `gorm:"primaryKey"`
}

type pluginOperationLine struct {
	ID       string `gorm:"primaryKey"`
	HeaderID string
}

func TestTransactionServiceRollsBackPluginOperationAtomically(t *testing.T) {
	db := openHostServiceTestDB(t)
	service, err := NewTransactionService(storesql.NewUnitOfWorkWithDB(db))
	if err != nil {
		t.Fatalf("NewTransactionService error: %v", err)
	}
	want := errors.New("line validation failed")
	err = service.Within(context.Background(), func(tx pluginsdk.Transaction) error {
		scoped := storesql.ResolveDB(tx.Context(), db)
		if err := scoped.Create(&pluginOperationHeader{ID: "header-1"}).Error; err != nil {
			return err
		}
		if err := scoped.Create(&pluginOperationLine{ID: "line-1", HeaderID: "header-1"}).Error; err != nil {
			return err
		}
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("unexpected transaction error: %v", err)
	}
	assertHostServiceRowCount(t, db, &pluginOperationHeader{}, 0)
	assertHostServiceRowCount(t, db, &pluginOperationLine{}, 0)
}

func openHostServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "host-service.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "cgo") {
			t.Skipf("sqlite test requires cgo: %v", err)
		}
		t.Fatalf("open database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("access database handle: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&pluginOperationHeader{}, &pluginOperationLine{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	return db
}

func assertHostServiceRowCount(t *testing.T, db *gorm.DB, model any, want int64) {
	t.Helper()
	var count int64
	if err := db.Model(model).Count(&count).Error; err != nil || count != want {
		t.Fatalf("row count=%d want=%d err=%v", count, want, err)
	}
}
