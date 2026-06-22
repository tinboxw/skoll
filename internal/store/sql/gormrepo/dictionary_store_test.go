package gormrepo

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainsystem "github.com/tinboxw/skoll/internal/domain/system"
	systemrepo "github.com/tinboxw/skoll/internal/repository/system"
)

func TestSystemStoreImplementsRepository(t *testing.T) {
	var _ systemrepo.SystemRepository = (*SystemStore)(nil)
}

func TestSystemStoreDictionaryTypeAndItemContract(t *testing.T) {
	db := TestDB(t)
	store := NewSystemStore(db, nil)
	ctx := context.Background()

	dictType := mustGormDictionaryType(t, "type-1", " System.Locale ", 20)
	if err := store.SaveDictionaryType(ctx, dictType); err != nil {
		t.Fatalf("SaveDictionaryType() error = %v", err)
	}
	gotType, err := store.GetDictionaryTypeByCode(ctx, "SYSTEM.LOCALE")
	if err != nil {
		t.Fatalf("GetDictionaryTypeByCode() error = %v", err)
	}
	if gotType == nil || gotType.Code != "system.locale" {
		t.Fatalf("gotType = %#v", gotType)
	}

	secondType := mustGormDictionaryType(t, "type-2", "system.status", 10)
	if err := store.SaveDictionaryType(ctx, secondType); err != nil {
		t.Fatalf("SaveDictionaryType(second) error = %v", err)
	}
	types, err := store.ListDictionaryTypes(ctx, 0, 1)
	if err != nil {
		t.Fatalf("ListDictionaryTypes() error = %v", err)
	}
	if len(types) != 1 || types[0].Code != "system.status" {
		t.Fatalf("types = %#v", types)
	}

	firstItem := mustGormDictionaryItem(t, "item-1", "system.locale", "English", "en", 20)
	secondItem := mustGormDictionaryItem(t, "item-2", "system.locale", "Chinese", "zh-CN", 10)
	if err := store.SaveDictionaryItem(ctx, firstItem); err != nil {
		t.Fatalf("SaveDictionaryItem(first) error = %v", err)
	}
	if err := store.SaveDictionaryItem(ctx, secondItem); err != nil {
		t.Fatalf("SaveDictionaryItem(second) error = %v", err)
	}
	gotItem, err := store.GetDictionaryItemByTypeAndValue(ctx, "SYSTEM.LOCALE", "en")
	if err != nil {
		t.Fatalf("GetDictionaryItemByTypeAndValue() error = %v", err)
	}
	if gotItem == nil || gotItem.ID != "item-1" {
		t.Fatalf("gotItem = %#v", gotItem)
	}
	items, err := store.ListDictionaryItems(ctx, "system.locale", 0, 0)
	if err != nil {
		t.Fatalf("ListDictionaryItems() error = %v", err)
	}
	if len(items) != 2 || items[0].Value != "zh-CN" || items[1].Value != "en" {
		t.Fatalf("items = %#v", items)
	}

	if err := store.DeleteDictionaryType(ctx, "type-1"); err != nil {
		t.Fatalf("DeleteDictionaryType() error = %v", err)
	}
	items, err = store.ListDictionaryItems(ctx, "system.locale", 0, 0)
	if err != nil {
		t.Fatalf("ListDictionaryItems(after delete) error = %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected items to be cascade deleted, got %#v", items)
	}
}

func TestSystemStoreDictionaryRejectsDuplicatesAndMissingType(t *testing.T) {
	db := TestDB(t)
	store := NewSystemStore(db, nil)
	ctx := context.Background()

	dictType := mustGormDictionaryType(t, "type-1", "system.locale", 10)
	if err := store.SaveDictionaryType(ctx, dictType); err != nil {
		t.Fatalf("SaveDictionaryType() error = %v", err)
	}
	dupType := mustGormDictionaryType(t, "type-2", "SYSTEM.LOCALE", 20)
	if err := store.SaveDictionaryType(ctx, dupType); err == nil {
		t.Fatal("expected duplicate type error")
	}

	firstItem := mustGormDictionaryItem(t, "item-1", "system.locale", "English", "en", 10)
	if err := store.SaveDictionaryItem(ctx, firstItem); err != nil {
		t.Fatalf("SaveDictionaryItem() error = %v", err)
	}
	dupItem := mustGormDictionaryItem(t, "item-2", "system.locale", "English again", "en", 20)
	if err := store.SaveDictionaryItem(ctx, dupItem); err == nil {
		t.Fatal("expected duplicate item error")
	}
	missingTypeItem := mustGormDictionaryItem(t, "item-3", "system.missing", "Missing", "missing", 30)
	if err := store.SaveDictionaryItem(ctx, missingTypeItem); err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("expected missing type error, got %v", err)
	}
}

func mustGormDictionaryType(t *testing.T, id, code string, sort int) *domainsystem.DictionaryType {
	t.Helper()
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	item, err := domainsystem.NewDictionaryType(domainsystem.DictionaryTypeInput{
		ID:          shared.ID(id),
		Code:        code,
		Name:        "Locale",
		Description: "Locale options",
		Sort:        sort,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("NewDictionaryType() error = %v", err)
	}
	return item
}

func mustGormDictionaryItem(t *testing.T, id, typeCode, label, value string, sort int) *domainsystem.DictionaryItem {
	t.Helper()
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	item, err := domainsystem.NewDictionaryItem(domainsystem.DictionaryItemInput{
		ID:        shared.ID(id),
		TypeCode:  typeCode,
		Label:     label,
		Value:     value,
		Sort:      sort,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("NewDictionaryItem() error = %v", err)
	}
	return item
}
