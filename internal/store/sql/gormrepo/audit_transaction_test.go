package gormrepo

import (
	"context"
	"errors"
	"testing"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	"github.com/tinboxw/skoll/internal/domain/shared"
	"github.com/tinboxw/skoll/internal/repository"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
)

func TestAuditStoreAppendHonorsUnitOfWorkRollback(t *testing.T) {
	db := TestDB(t)
	store := NewAuditStore(db)
	uow := storesql.NewUnitOfWorkWithDB(db)
	sentinel := errors.New("rollback")
	err := uow.Do(context.Background(), func(tx repository.Tx) error {
		record, recordErr := domainaudit.NewRecord(shared.ID("audit-tx-1"), shared.ID("actor-1"), "plugin.write", "plugin:data", "resource-1", map[string]any{"result": "success"}, time.Now())
		if recordErr != nil {
			return recordErr
		}
		if appendErr := store.Append(tx.Context(), record); appendErr != nil {
			return appendErr
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("transaction error=%v", err)
	}
	var count int64
	if err = db.Model(&AuditRecordModel{}).Where("id = ?", "audit-tx-1").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("rolled-back audit count=%d", count)
	}
}
