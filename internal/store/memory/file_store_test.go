package memory

import (
	"context"
	"testing"
	"time"

	domainfile "github.com/tinboxw/skoll/internal/domain/file"
	"github.com/tinboxw/skoll/internal/domain/shared"
	filerepo "github.com/tinboxw/skoll/internal/repository/file"
)

func TestFileStoreUpsertGetListStatusDelete(t *testing.T) {
	ctx := context.Background()
	store := NewFileStore()
	item := mustFileObject(t, "file-1", "uploads/a.txt", domainfile.StatusPending)

	if err := store.Upsert(ctx, item); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	got, err := store.Get(ctx, "file-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got == nil || got.Key != "uploads/a.txt" || got.Metadata["trace-id"] != "req-1" {
		t.Fatalf("got = %#v", got)
	}

	byKey, err := store.GetByKey(ctx, " Uploads/A.TXT ")
	if err != nil {
		t.Fatalf("GetByKey() error = %v", err)
	}
	if byKey == nil || byKey.ID != "file-1" {
		t.Fatalf("byKey = %#v", byKey)
	}

	items, err := store.List(ctx, filerepo.ListFilter{OwnerType: "USER", OwnerID: "u-1", Status: domainfile.StatusPending}, 0, 0)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len = %d", len(items))
	}

	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	if err := store.SetStatus(ctx, "file-1", domainfile.StatusAvailable, now); err != nil {
		t.Fatalf("SetStatus() error = %v", err)
	}
	got, _ = store.Get(ctx, "file-1")
	if got.Status != domainfile.StatusAvailable || !got.Meta.UpdatedAt.Equal(now) {
		t.Fatalf("status/meta = %s/%s", got.Status, got.Meta.UpdatedAt)
	}

	if err := store.Delete(ctx, "file-1"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	got, _ = store.Get(ctx, "file-1")
	if got != nil {
		t.Fatalf("got after delete = %#v", got)
	}
}

func TestFileStoreRejectsDuplicateKeyForDifferentID(t *testing.T) {
	store := NewFileStore()
	ctx := context.Background()
	if err := store.Upsert(ctx, mustFileObject(t, "file-1", "uploads/a.txt", domainfile.StatusPending)); err != nil {
		t.Fatalf("Upsert(file-1) error = %v", err)
	}
	if err := store.Upsert(ctx, mustFileObject(t, "file-2", "uploads/a.txt", domainfile.StatusPending)); err == nil {
		t.Fatal("expected duplicate key error")
	}
}

func mustFileObject(t *testing.T, id, key string, status domainfile.Status) *domainfile.FileObject {
	t.Helper()
	now := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)
	item, err := domainfile.NewFileObject(domainfile.FileObjectInput{
		ID:            shared.ID(id),
		Key:           key,
		Name:          "a.txt",
		Size:          1,
		MIME:          "text/plain",
		Hash:          "abcdef123456",
		Owner:         domainfile.OwnerRef{Type: "user", ID: "u-1"},
		Visibility:    domainfile.VisibilityPrivate,
		StorageDriver: "local",
		Status:        status,
		Source:        domainfile.SourceRef{Module: "system"},
		Metadata:      map[string]string{"trace-id": "req-1"},
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	if err != nil {
		t.Fatalf("NewFileObject() error = %v", err)
	}
	return item
}
