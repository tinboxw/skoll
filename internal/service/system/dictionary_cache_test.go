package system

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/cache"
	domainsystem "github.com/tinboxw/skoll/internal/domain/system"
	"github.com/tinboxw/skoll/internal/store/memory"
)

type countingDictionaryRepo struct {
	*memory.SystemStore
	getTypeByCodeCalls int
	listItemCalls      int
	failGetType        bool
}

func (r *countingDictionaryRepo) GetDictionaryTypeByCode(ctx context.Context, code string) (*domainsystem.DictionaryType, error) {
	r.getTypeByCodeCalls++
	if r.failGetType {
		return nil, fmt.Errorf("dictionary repo unavailable")
	}
	return r.SystemStore.GetDictionaryTypeByCode(ctx, code)
}

func (r *countingDictionaryRepo) ListDictionaryItems(ctx context.Context, typeCode string, offset, limit int) ([]domainsystem.DictionaryItem, error) {
	r.listItemCalls++
	return r.SystemStore.ListDictionaryItems(ctx, typeCode, offset, limit)
}

func TestDictionaryCacheHitInvalidateAndTTL(t *testing.T) {
	ctx := context.Background()
	repo := &countingDictionaryRepo{SystemStore: memory.NewSystemStore()}
	bundle, err := cache.NewBundle(cache.Options{Mode: cache.ModeLocal, LocalSize: 32})
	if err != nil {
		t.Fatalf("NewBundle() error = %v", err)
	}
	svc := NewCachedService(repo, bundle.Dictionary, 2*time.Millisecond)

	if _, err := svc.SaveDictionaryType(ctx, DictionaryTypeInput{ID: "type-1", Code: "system.locale", Name: "Locale"}); err != nil {
		t.Fatalf("SaveDictionaryType() error = %v", err)
	}
	got, err := svc.GetDictionaryTypeByCode(ctx, "SYSTEM.LOCALE")
	if err != nil {
		t.Fatalf("GetDictionaryTypeByCode() miss error = %v", err)
	}
	if got == nil || got.Code != "system.locale" {
		t.Fatalf("got = %#v", got)
	}
	if _, err := svc.GetDictionaryTypeByCode(ctx, "system.locale"); err != nil {
		t.Fatalf("GetDictionaryTypeByCode() hit error = %v", err)
	}
	if repo.getTypeByCodeCalls != 1 {
		t.Fatalf("expected one repo lookup after cache hit, got %d", repo.getTypeByCodeCalls)
	}

	if _, err := svc.SaveDictionaryType(ctx, DictionaryTypeInput{ID: "type-1", Code: "system.locale", Name: "Locale Updated"}); err != nil {
		t.Fatalf("SaveDictionaryType(update) error = %v", err)
	}
	got, err = svc.GetDictionaryTypeByCode(ctx, "system.locale")
	if err != nil {
		t.Fatalf("GetDictionaryTypeByCode() after invalidation error = %v", err)
	}
	if got == nil || got.Name != "Locale Updated" {
		t.Fatalf("expected updated cache miss result, got %#v", got)
	}
	if repo.getTypeByCodeCalls != 2 {
		t.Fatalf("expected repo lookup after invalidation, got %d", repo.getTypeByCodeCalls)
	}

	time.Sleep(5 * time.Millisecond)
	if _, err := svc.GetDictionaryTypeByCode(ctx, "system.locale"); err != nil {
		t.Fatalf("GetDictionaryTypeByCode() after TTL error = %v", err)
	}
	if repo.getTypeByCodeCalls != 3 {
		t.Fatalf("expected repo lookup after TTL expiration, got %d", repo.getTypeByCodeCalls)
	}
}

func TestDictionaryCacheListInvalidationAndErrorFallback(t *testing.T) {
	ctx := context.Background()
	repo := &countingDictionaryRepo{SystemStore: memory.NewSystemStore()}
	bundle, err := cache.NewBundle(cache.Options{Mode: cache.ModeLocal, LocalSize: 32})
	if err != nil {
		t.Fatalf("NewBundle() error = %v", err)
	}
	svc := NewCachedService(repo, bundle.Dictionary, time.Minute)

	if _, err := svc.SaveDictionaryType(ctx, DictionaryTypeInput{ID: "type-1", Code: "system.locale", Name: "Locale"}); err != nil {
		t.Fatalf("SaveDictionaryType() error = %v", err)
	}
	if _, err := svc.SaveDictionaryItem(ctx, DictionaryItemInput{ID: "item-1", TypeCode: "system.locale", Label: "English", Value: "en"}); err != nil {
		t.Fatalf("SaveDictionaryItem() error = %v", err)
	}
	items, err := svc.ListDictionaryItems(ctx, DictionaryItemListInput{TypeCode: "system.locale"})
	if err != nil {
		t.Fatalf("ListDictionaryItems() miss error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one item, got %d", len(items))
	}
	if _, err := svc.ListDictionaryItems(ctx, DictionaryItemListInput{TypeCode: "system.locale"}); err != nil {
		t.Fatalf("ListDictionaryItems() hit error = %v", err)
	}
	if repo.listItemCalls != 1 {
		t.Fatalf("expected one repo list lookup after cache hit, got %d", repo.listItemCalls)
	}

	if _, err := svc.SaveDictionaryItem(ctx, DictionaryItemInput{ID: "item-2", TypeCode: "system.locale", Label: "Chinese", Value: "zh-CN"}); err != nil {
		t.Fatalf("SaveDictionaryItem(second) error = %v", err)
	}
	items, err = svc.ListDictionaryItems(ctx, DictionaryItemListInput{TypeCode: "system.locale"})
	if err != nil {
		t.Fatalf("ListDictionaryItems() after invalidation error = %v", err)
	}
	if len(items) != 2 || repo.listItemCalls != 2 {
		t.Fatalf("expected invalidated list to hit repo and return two items, len=%d calls=%d", len(items), repo.listItemCalls)
	}

	repo.failGetType = true
	if _, err := svc.GetDictionaryTypeByCode(ctx, "system.locale"); err == nil {
		t.Fatal("expected repository error")
	}
	repo.failGetType = false
	if _, err := svc.GetDictionaryTypeByCode(ctx, "system.locale"); err != nil {
		t.Fatalf("expected repo fallback after error, got %v", err)
	}
	if repo.getTypeByCodeCalls != 2 {
		t.Fatalf("expected failed read not to populate cache, got %d repo calls", repo.getTypeByCodeCalls)
	}
}

func TestDictionaryServiceValidation(t *testing.T) {
	svc := NewService(memory.NewSystemStore())
	if _, err := svc.GetDictionaryTypeByCode(context.Background(), " "); err == nil {
		t.Fatal("expected dictionary type code error")
	}
	if _, err := svc.ListDictionaryItems(context.Background(), DictionaryItemListInput{TypeCode: " ", Offset: 0, Limit: 1}); err == nil {
		t.Fatal("expected dictionary type code error")
	}
	if err := svc.DeleteDictionaryItem(context.Background(), " "); err == nil {
		t.Fatal("expected dictionary item id error")
	}
}
