package system

import (
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

func TestNewDictionaryType(t *testing.T) {
	now := time.Date(2026, 6, 22, 16, 0, 0, 0, time.UTC)
	got, err := NewDictionaryType(DictionaryTypeInput{
		ID:        "dict-type-1",
		Code:      " System.User_Status ",
		Name:      " User Status ",
		Sort:      20,
		Builtin:   true,
		CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("NewDictionaryType() error = %v", err)
	}
	if got.Code != "system.user_status" || got.Name != "User Status" || got.Status != DictionaryStatusEnabled || !got.Builtin {
		t.Fatalf("dictionary type = %+v", got)
	}
	if !got.Meta.CreatedAt.Equal(now) || !got.Meta.UpdatedAt.Equal(now) {
		t.Fatalf("meta = %+v", got.Meta)
	}
}

func TestNewDictionaryTypeRejectsInvalidInput(t *testing.T) {
	now := time.Date(2026, 6, 22, 16, 0, 0, 0, time.UTC)
	valid := DictionaryTypeInput{
		ID:        "dict-type-1",
		Code:      "system.user_status",
		Name:      "User Status",
		CreatedAt: now,
	}
	cases := []struct {
		name   string
		mutate func(*DictionaryTypeInput)
	}{
		{name: "missing id", mutate: func(in *DictionaryTypeInput) { in.ID = shared.ID("") }},
		{name: "bad code", mutate: func(in *DictionaryTypeInput) { in.Code = "!bad" }},
		{name: "missing name", mutate: func(in *DictionaryTypeInput) { in.Name = " " }},
		{name: "bad status", mutate: func(in *DictionaryTypeInput) { in.Status = DictionaryStatus("archived") }},
		{name: "bad sort", mutate: func(in *DictionaryTypeInput) { in.Sort = -1 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := valid
			tc.mutate(&in)
			if _, err := NewDictionaryType(in); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestNewDictionaryItem(t *testing.T) {
	now := time.Date(2026, 6, 22, 16, 0, 0, 0, time.UTC)
	got, err := NewDictionaryItem(DictionaryItemInput{
		ID:        "dict-item-1",
		TypeCode:  " System.User_Status ",
		Label:     " Enabled ",
		Value:     " enabled ",
		Sort:      10,
		Builtin:   true,
		CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("NewDictionaryItem() error = %v", err)
	}
	if got.TypeCode != "system.user_status" || got.Label != "Enabled" || got.Value != "enabled" || got.Status != DictionaryStatusEnabled || !got.Builtin {
		t.Fatalf("dictionary item = %+v", got)
	}
}

func TestNewDictionaryItemRejectsInvalidInput(t *testing.T) {
	now := time.Date(2026, 6, 22, 16, 0, 0, 0, time.UTC)
	valid := DictionaryItemInput{
		ID:        "dict-item-1",
		TypeCode:  "system.user_status",
		Label:     "Enabled",
		Value:     "enabled",
		CreatedAt: now,
	}
	cases := []struct {
		name   string
		mutate func(*DictionaryItemInput)
	}{
		{name: "missing id", mutate: func(in *DictionaryItemInput) { in.ID = shared.ID("") }},
		{name: "bad type code", mutate: func(in *DictionaryItemInput) { in.TypeCode = "bad key" }},
		{name: "missing label", mutate: func(in *DictionaryItemInput) { in.Label = "" }},
		{name: "missing value", mutate: func(in *DictionaryItemInput) { in.Value = " " }},
		{name: "bad status", mutate: func(in *DictionaryItemInput) { in.Status = DictionaryStatus("archived") }},
		{name: "bad sort", mutate: func(in *DictionaryItemInput) { in.Sort = -1 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := valid
			tc.mutate(&in)
			if _, err := NewDictionaryItem(in); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestDictionaryStatusSortAndBuiltinGuards(t *testing.T) {
	now := time.Date(2026, 6, 22, 16, 0, 0, 0, time.UTC)
	later := now.Add(time.Minute)
	customType, err := NewDictionaryType(DictionaryTypeInput{ID: "type-1", Code: "system.status", Name: "Status", CreatedAt: now})
	if err != nil {
		t.Fatalf("NewDictionaryType() error = %v", err)
	}
	customType.Disable(later)
	if customType.Status != DictionaryStatusDisabled || !customType.Meta.UpdatedAt.Equal(later) {
		t.Fatalf("disabled type = %+v", customType)
	}
	if err := customType.Reorder(30, later.Add(time.Minute)); err != nil {
		t.Fatalf("Reorder() error = %v", err)
	}
	if customType.Sort != 30 {
		t.Fatalf("sort = %d", customType.Sort)
	}
	if err := customType.RequireMutable(); err != nil {
		t.Fatalf("custom type should be mutable: %v", err)
	}
	customType.Builtin = true
	if err := customType.RequireMutable(); err == nil {
		t.Fatal("expected builtin type guard")
	}

	items := []DictionaryItem{
		mustDictionaryItem(t, "item-2", "beta", 20),
		mustDictionaryItem(t, "item-1", "alpha", 10),
		mustDictionaryItem(t, "item-3", "aaa", 20),
	}
	sorted := SortDictionaryItems(items)
	if sorted[0].Value != "alpha" || sorted[1].Value != "aaa" || sorted[2].Value != "beta" {
		t.Fatalf("sorted items = %+v", sorted)
	}
	if items[0].Value != "beta" {
		t.Fatalf("SortDictionaryItems should not mutate input: %+v", items)
	}

	item := mustDictionaryItem(t, "item-4", "delta", 40)
	item.Disable(later)
	if item.Status != DictionaryStatusDisabled {
		t.Fatalf("item status = %s", item.Status)
	}
	item.Builtin = true
	if err := item.RequireMutable(); err == nil {
		t.Fatal("expected builtin item guard")
	}
}

func mustDictionaryItem(t *testing.T, id, value string, sortValue int) DictionaryItem {
	t.Helper()
	item, err := NewDictionaryItem(DictionaryItemInput{
		ID:        shared.ID(id),
		TypeCode:  "system.status",
		Label:     value,
		Value:     value,
		Sort:      sortValue,
		CreatedAt: time.Date(2026, 6, 22, 16, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("NewDictionaryItem() error = %v", err)
	}
	return *item
}
