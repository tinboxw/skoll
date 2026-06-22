package object

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	domainfile "github.com/tinboxw/skoll/internal/domain/file"
)

func TestLocalStorePutGetStatDelete(t *testing.T) {
	store, err := NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalStore() error = %v", err)
	}

	info, err := store.Put(context.Background(), domainfile.PutObjectInput{
		Key:      " Uploads/A.TXT ",
		Size:     5,
		MIME:     " Text/Plain ",
		Hash:     " abcdef123456 ",
		Body:     strings.NewReader("hello"),
		Metadata: map[string]string{" Trace-ID ": " req-1 "},
	})
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	if info.Key != "uploads/a.txt" || info.Size != 5 || info.MIME != "text/plain" || info.Hash != "abcdef123456" {
		t.Fatalf("info = %#v", info)
	}

	stat, err := store.Stat(context.Background(), "uploads/a.txt")
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if stat.Metadata["trace-id"] != "req-1" {
		t.Fatalf("metadata = %#v", stat.Metadata)
	}

	stream, err := store.Get(context.Background(), "uploads/a.txt")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	body, err := io.ReadAll(stream.Body)
	closeErr := stream.Body.Close()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if closeErr != nil {
		t.Fatalf("Close() error = %v", closeErr)
	}
	if string(body) != "hello" {
		t.Fatalf("body = %q", body)
	}

	if err := store.Delete(context.Background(), "uploads/a.txt"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := store.Stat(context.Background(), "uploads/a.txt"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Stat() after delete error = %v, want not exist", err)
	}
}

func TestLocalStoreRejectsInvalidRootAndKey(t *testing.T) {
	if _, err := NewLocalStore(" "); err == nil {
		t.Fatal("expected empty root error")
	}

	store, err := NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalStore() error = %v", err)
	}

	keys := []string{
		"../secret.txt",
		"uploads/../secret.txt",
		"/absolute.txt",
		`\absolute.txt`,
		`uploads\..\secret.txt`,
		"bad key.txt",
	}
	for _, key := range keys {
		t.Run(key, func(t *testing.T) {
			_, err = store.Put(context.Background(), domainfile.PutObjectInput{
				Key:  key,
				Size: 1,
				MIME: "text/plain",
				Hash: "abcdef123456",
				Body: strings.NewReader("x"),
			})
			if err == nil {
				t.Fatal("expected invalid key error")
			}
		})
	}
}

func TestLocalStoreRejectsOverwrite(t *testing.T) {
	store, err := NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalStore() error = %v", err)
	}
	input := domainfile.PutObjectInput{
		Key:  "uploads/a.txt",
		Size: 5,
		MIME: "text/plain",
		Hash: "abcdef123456",
		Body: strings.NewReader("hello"),
	}
	if _, err := store.Put(context.Background(), input); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	input.Body = strings.NewReader("other")
	input.Hash = "bbbbbbbb"
	err = putOnlyError(store, input)
	if !errors.Is(err, ErrObjectExists) {
		t.Fatalf("overwrite error = %v, want ErrObjectExists", err)
	}

	stream, err := store.Get(context.Background(), "uploads/a.txt")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	body, readErr := io.ReadAll(stream.Body)
	closeErr := stream.Body.Close()
	if readErr != nil {
		t.Fatalf("ReadAll() error = %v", readErr)
	}
	if closeErr != nil {
		t.Fatalf("Close() error = %v", closeErr)
	}
	if string(body) != "hello" {
		t.Fatalf("body after rejected overwrite = %q", body)
	}
}

func TestLocalStoreRejectsSizeMismatch(t *testing.T) {
	store, err := NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalStore() error = %v", err)
	}
	_, err = store.Put(context.Background(), domainfile.PutObjectInput{
		Key:  "uploads/a.txt",
		Size: 10,
		MIME: "text/plain",
		Hash: "abcdef123456",
		Body: strings.NewReader("short"),
	})
	if err == nil {
		t.Fatal("expected size mismatch")
	}
}

func TestLocalStorePresign(t *testing.T) {
	store, err := NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalStore() error = %v", err)
	}
	got, err := store.Presign(context.Background(), domainfile.PresignInput{
		Key:       "uploads/a.txt",
		Operation: domainfile.PresignOperationGet,
		ExpiresIn: time.Minute,
	})
	if err != nil {
		t.Fatalf("Presign() error = %v", err)
	}
	if got.Operation != domainfile.PresignOperationGet || !strings.HasPrefix(got.URL, "local://object/uploads/a.txt") {
		t.Fatalf("presigned object = %#v", got)
	}
}

func TestLocalStoreResolveStaysUnderRoot(t *testing.T) {
	root := t.TempDir()
	store, err := NewLocalStore(root)
	if err != nil {
		t.Fatalf("NewLocalStore() error = %v", err)
	}
	target, err := store.resolve("uploads/a.txt")
	if err != nil {
		t.Fatalf("resolve() error = %v", err)
	}
	rel, err := filepath.Rel(root, target)
	if err != nil {
		t.Fatalf("Rel() error = %v", err)
	}
	if strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		t.Fatalf("target escaped root: %s", target)
	}

	if _, err := os.Stat(filepath.Dir(target)); err == nil {
		t.Fatal("resolve should not create directories")
	}
}

func putOnlyError(store *LocalStore, input domainfile.PutObjectInput) error {
	_, err := store.Put(context.Background(), input)
	return err
}
