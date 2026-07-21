package sql

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tinboxw/skoll/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestUnitOfWorkDoWithoutDB(t *testing.T) {
	uow := NewUnitOfWork()
	ctx := context.WithValue(context.Background(), "k", "v")
	called := false

	err := uow.Do(ctx, func(txCtx repository.Tx) error {
		called = true
		if txCtx.Context().Value("k") != "v" {
			t.Fatalf("expected context value to be preserved")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Do unexpected error: %v", err)
	}
	if !called {
		t.Fatalf("expected callback called")
	}
}

func TestUnitOfWorkDoPropagatesCallbackError(t *testing.T) {
	uow := NewUnitOfWork()
	expectedErr := errors.New("callback failed")

	err := uow.Do(context.Background(), func(_ repository.Tx) error {
		return expectedErr
	})
	if err == nil {
		t.Fatalf("expected callback error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "callback failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnitOfWorkDoWithNilReceiver(t *testing.T) {
	var uow *UnitOfWork
	called := false

	err := uow.Do(context.Background(), func(_ repository.Tx) error {
		called = true
		return nil
	})
	if err == nil || called {
		t.Fatalf("nil unit of work must fail closed: called=%v err=%v", called, err)
	}
}

func TestNewUnitOfWorkWithNilDBPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("nil SQL database must not create a fallback unit of work")
		}
	}()
	_ = NewUnitOfWorkWithDB(nil)
}

type unitOfWorkProbe struct {
	ID   string `gorm:"primaryKey"`
	Name string
}

func TestUnitOfWorkPublishesTransactionContextAndRollsBack(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "unit-of-work.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
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
	if err := db.AutoMigrate(&unitOfWorkProbe{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	uow := NewUnitOfWorkWithDB(db)
	want := errors.New("rollback plugin operation")
	err = uow.Do(context.Background(), func(tx repository.Tx) error {
		scoped := ResolveDB(tx.Context(), db)
		if scoped == nil || DBFromContext(tx.Context()) == nil {
			t.Fatal("transaction database is missing from callback context")
		}
		if err := scoped.Create(&unitOfWorkProbe{ID: "rolled-back", Name: "first"}).Error; err != nil {
			return err
		}
		return uow.Do(tx.Context(), func(nested repository.Tx) error {
			if err := ResolveDB(nested.Context(), db).Create(&unitOfWorkProbe{ID: "nested", Name: "second"}).Error; err != nil {
				return err
			}
			return want
		})
	})
	if !errors.Is(err, want) {
		t.Fatalf("unexpected rollback error: %v", err)
	}
	var count int64
	if err := db.Model(&unitOfWorkProbe{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("transaction writes survived rollback: count=%d err=%v", count, err)
	}
}
