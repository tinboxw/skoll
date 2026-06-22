package system

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainsystem "github.com/tinboxw/skoll/internal/domain/system"
	"github.com/tinboxw/skoll/internal/store/memory"
)

func TestSystemServiceUpsertAndList(t *testing.T) {
	svc := NewService(memory.NewSystemStore())

	first, err := svc.Upsert(context.Background(), UpsertInput{Key: "feature.alpha", Value: "on", Encrypted: false})
	if err != nil {
		t.Fatalf("Upsert create error: %v", err)
	}
	if first == nil || first.ID == "" {
		t.Fatalf("expected created setting")
	}

	second, err := svc.Upsert(context.Background(), UpsertInput{Key: "feature.alpha", Value: "off", Encrypted: true})
	if err != nil {
		t.Fatalf("Upsert update error: %v", err)
	}
	if second == nil || second.Value != "off" || !second.Encrypted {
		t.Fatalf("unexpected updated setting: %+v", second)
	}

	items, err := svc.List(context.Background(), ListInput{Offset: 0, Limit: 10})
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 setting, got %d", len(items))
	}
}

type failingSystemRepo struct {
	listItems []*domainsystem.Setting
	failOnID  shared.ID
}

func (f *failingSystemRepo) GetSettingByID(_ context.Context, _ shared.ID) (*domainsystem.Setting, error) {
	return nil, nil
}

func (f *failingSystemRepo) GetSettingByKey(_ context.Context, _ string) (*domainsystem.Setting, error) {
	return nil, nil
}

func (f *failingSystemRepo) ListSettings(_ context.Context, _ int, _ int) ([]*domainsystem.Setting, error) {
	return f.listItems, nil
}

func (f *failingSystemRepo) SaveSetting(_ context.Context, _ *domainsystem.Setting) error {
	return nil
}

func (f *failingSystemRepo) DeleteSetting(_ context.Context, id shared.ID) error {
	if id == f.failOnID {
		return fmt.Errorf("delete failed")
	}
	return nil
}

func (f *failingSystemRepo) GetDictionaryTypeByID(_ context.Context, _ shared.ID) (*domainsystem.DictionaryType, error) {
	return nil, nil
}

func (f *failingSystemRepo) GetDictionaryTypeByCode(_ context.Context, _ string) (*domainsystem.DictionaryType, error) {
	return nil, nil
}

func (f *failingSystemRepo) ListDictionaryTypes(_ context.Context, _ int, _ int) ([]domainsystem.DictionaryType, error) {
	return nil, nil
}

func (f *failingSystemRepo) SaveDictionaryType(_ context.Context, _ *domainsystem.DictionaryType) error {
	return nil
}

func (f *failingSystemRepo) DeleteDictionaryType(_ context.Context, _ shared.ID) error {
	return nil
}

func (f *failingSystemRepo) GetDictionaryItemByID(_ context.Context, _ shared.ID) (*domainsystem.DictionaryItem, error) {
	return nil, nil
}

func (f *failingSystemRepo) GetDictionaryItemByTypeAndValue(_ context.Context, _, _ string) (*domainsystem.DictionaryItem, error) {
	return nil, nil
}

func (f *failingSystemRepo) ListDictionaryItems(_ context.Context, _ string, _ int, _ int) ([]domainsystem.DictionaryItem, error) {
	return nil, nil
}

func (f *failingSystemRepo) SaveDictionaryItem(_ context.Context, _ *domainsystem.DictionaryItem) error {
	return nil
}

func (f *failingSystemRepo) DeleteDictionaryItem(_ context.Context, _ shared.ID) error {
	return nil
}

func TestSystemServiceValidationPaths(t *testing.T) {
	t.Run("nil repository", func(t *testing.T) {
		svc := NewService(nil)
		if _, err := svc.GetByKey(context.Background(), "x"); err == nil {
			t.Fatalf("expected repository not configured error")
		}
		if _, err := svc.List(context.Background(), ListInput{Offset: 0, Limit: 1}); err == nil {
			t.Fatalf("expected repository not configured error")
		}
		if _, err := svc.Upsert(context.Background(), UpsertInput{Key: "x", Value: "v"}); err == nil {
			t.Fatalf("expected repository not configured error")
		}
		if _, err := svc.Reset(context.Background()); err == nil {
			t.Fatalf("expected repository not configured error")
		}
	})

	t.Run("key and pagination validation", func(t *testing.T) {
		svc := NewService(memory.NewSystemStore())

		if _, err := svc.GetByKey(context.Background(), "   "); err == nil || !strings.Contains(strings.ToLower(err.Error()), "setting key") {
			t.Fatalf("expected key required error, got %v", err)
		}
		if _, err := svc.Upsert(context.Background(), UpsertInput{Key: "", Value: "v"}); err == nil || !strings.Contains(strings.ToLower(err.Error()), "setting key") {
			t.Fatalf("expected key required error, got %v", err)
		}
		if _, err := svc.List(context.Background(), ListInput{Offset: -1, Limit: 10}); err == nil || !strings.Contains(strings.ToLower(err.Error()), "invalid pagination") {
			t.Fatalf("expected invalid pagination error, got %v", err)
		}
	})
}

func TestSystemServiceResetCountsAndStopsOnError(t *testing.T) {
	now := time.Now().UTC()
	s1, err := domainsystem.NewSetting(shared.ID("s1"), "k1", "v1", false, now)
	if err != nil {
		t.Fatalf("new setting s1 error: %v", err)
	}
	s2, err := domainsystem.NewSetting(shared.ID("s2"), "k2", "v2", false, now)
	if err != nil {
		t.Fatalf("new setting s2 error: %v", err)
	}

	repo := &failingSystemRepo{
		listItems: []*domainsystem.Setting{s1, nil, s2},
		failOnID:  shared.ID("s2"),
	}
	svc := NewService(repo)

	deleted, err := svc.Reset(context.Background())
	if err == nil {
		t.Fatalf("expected delete error")
	}
	if deleted != 1 {
		t.Fatalf("expected deleted count 1 before failure, got %d", deleted)
	}
}
