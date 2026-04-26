package fileservice

import (
	"path/filepath"
	"testing"
)

func TestServiceUploadListGetDownload(t *testing.T) {
	backend, err := NewLocalBackend(filepath.Join(t.TempDir(), "uploads"))
	if err != nil {
		t.Fatalf("create backend failed: %v", err)
	}
	svc := NewService(backend)

	item, err := svc.Upload("hello.txt", []byte("hello world"))
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}
	if item.ID <= 0 {
		t.Fatalf("expected generated file id")
	}

	list := svc.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 file in list, got %d", len(list))
	}

	getItem, err := svc.Get(item.ID)
	if err != nil {
		t.Fatalf("get file failed: %v", err)
	}
	if getItem.Name != "hello.txt" {
		t.Fatalf("unexpected file name: %s", getItem.Name)
	}

	downloadedItem, content, err := svc.Download(item.ID)
	if err != nil {
		t.Fatalf("download failed: %v", err)
	}
	if downloadedItem.ID != item.ID {
		t.Fatalf("unexpected downloaded file id: %d", downloadedItem.ID)
	}
	if string(content) != "hello world" {
		t.Fatalf("unexpected downloaded content: %s", string(content))
	}
}
