package hostservice

import (
	"context"
	"strings"
	"testing"

	filesvc "github.com/tinboxw/skoll/internal/service/file"
	"github.com/tinboxw/skoll/internal/store/memory"
	objectstore "github.com/tinboxw/skoll/internal/store/object"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"github.com/tinboxw/skoll/pkg/security"
)

func TestFileServiceEnforcesPluginOwnershipAndContentPolicy(t *testing.T) {
	objects, err := objectstore.NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalStore error: %v", err)
	}
	backend := filesvc.NewService(memory.NewFileStore(), objects, filesvc.Options{})
	service, err := NewFileService("pharma_oa", backend)
	if err != nil {
		t.Fatalf("NewFileService error: %v", err)
	}
	ctx := security.WithJWTClaimsContext(context.Background(), &security.JWTClaims{Subject: "employee-1"})
	item, err := service.Store(ctx, pluginsdk.FileWrite{
		Key: "documents/license.txt", Name: "license.txt", Content: []byte("licensed"),
	})
	if err != nil {
		t.Fatalf("Store error: %v", err)
	}
	if !strings.HasPrefix(item.Key, "plugins/pharma_oa/") || item.MIME == "" || item.Hash == "" {
		t.Fatalf("file boundary was not applied: %+v", item)
	}
	got, err := service.Get(ctx, item.ID)
	if err != nil || got.ID != item.ID {
		t.Fatalf("Get item=%+v err=%v", got, err)
	}
	items, err := service.List(ctx, pluginsdk.FileQuery{})
	if err != nil || len(items) != 1 || items[0].ID != item.ID {
		t.Fatalf("List items=%+v err=%v", items, err)
	}
	other, err := NewFileService("other_plugin", backend)
	if err != nil {
		t.Fatalf("NewFileService other error: %v", err)
	}
	if _, err := other.Download(ctx, item.ID); err == nil {
		t.Fatal("another plugin downloaded a foreign file")
	}
	if _, err := other.Get(ctx, item.ID); err == nil {
		t.Fatal("another plugin read a foreign file")
	}
	if err := other.Delete(ctx, item.ID); err == nil {
		t.Fatal("another plugin deleted a foreign file")
	}
	if _, err := service.Store(ctx, pluginsdk.FileWrite{Key: "bin/tool.exe", Name: "tool.exe", Content: []byte("MZ")}); err == nil {
		t.Fatal("executable upload was accepted")
	}
	if _, err := service.Store(ctx, pluginsdk.FileWrite{Key: "../escape.txt", Name: "escape.txt", Content: []byte("escape")}); err == nil {
		t.Fatal("path traversal key was accepted")
	}
}
