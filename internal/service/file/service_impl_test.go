package file

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	domainfile "github.com/tinboxw/skoll/internal/domain/file"
	"github.com/tinboxw/skoll/internal/domain/shared"
	filerepo "github.com/tinboxw/skoll/internal/repository/file"
)

func TestUploadWritesObjectThenAvailableMetadata(t *testing.T) {
	repo := newFakeRepo()
	objects := &fakeObjectStore{}
	svc := testService(repo, objects)

	got, err := svc.Upload(context.Background(), validUploadInput())
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}
	if got.Status != domainfile.StatusAvailable {
		t.Fatalf("status = %q", got.Status)
	}
	if !objects.putCalled {
		t.Fatal("object store Put should be called")
	}
	stored := repo.items[got.ID]
	if stored.Status != domainfile.StatusAvailable || stored.Key != "uploads/a.txt" {
		t.Fatalf("stored = %#v", stored)
	}
}

func TestUploadObjectFailureDoesNotWriteMetadata(t *testing.T) {
	repo := newFakeRepo()
	objects := &fakeObjectStore{putErr: fmt.Errorf("disk full")}
	svc := testService(repo, objects)

	if _, err := svc.Upload(context.Background(), validUploadInput()); err == nil {
		t.Fatal("expected upload error")
	}
	if len(repo.items) != 0 {
		t.Fatalf("metadata should not be written: %#v", repo.items)
	}
	if objects.deleteCalled {
		t.Fatal("delete should not be called when put failed")
	}
}

func TestUploadMetadataFailureDeletesObject(t *testing.T) {
	repo := newFakeRepo()
	repo.upsertErr = fmt.Errorf("db down")
	objects := &fakeObjectStore{}
	svc := testService(repo, objects)

	if _, err := svc.Upload(context.Background(), validUploadInput()); err == nil {
		t.Fatal("expected metadata error")
	}
	if !objects.deleteCalled {
		t.Fatal("object should be cleaned up after metadata failure")
	}
	if len(repo.items) != 0 {
		t.Fatalf("metadata should not be retained: %#v", repo.items)
	}
}

func testService(repo filerepo.FileRepository, objects domainfile.ObjectStore) Service {
	now := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)
	return NewService(repo, objects, Options{
		Now:   func() time.Time { return now },
		NewID: func() shared.ID { return "file-1" },
	})
}

func validUploadInput() UploadInput {
	return UploadInput{
		Key:           " Uploads/A.TXT ",
		Name:          "a.txt",
		Size:          5,
		MIME:          " Text/Plain ",
		Hash:          " abcdef123456 ",
		Body:          strings.NewReader("hello"),
		Owner:         domainfile.OwnerRef{Type: "user", ID: "u-1"},
		Visibility:    domainfile.VisibilityPrivate,
		StorageDriver: "local",
		Source:        domainfile.SourceRef{Module: "system"},
		Metadata:      map[string]string{"trace-id": "req-1"},
	}
}

type fakeRepo struct {
	items     map[shared.ID]domainfile.FileObject
	upsertErr error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{items: make(map[shared.ID]domainfile.FileObject)}
}

func (r *fakeRepo) Upsert(_ context.Context, object *domainfile.FileObject) error {
	if r.upsertErr != nil {
		return r.upsertErr
	}
	r.items[object.ID] = *object
	return nil
}

func (r *fakeRepo) Get(_ context.Context, id shared.ID) (*domainfile.FileObject, error) {
	item, ok := r.items[id]
	if !ok {
		return nil, nil
	}
	return &item, nil
}

func (r *fakeRepo) GetByKey(_ context.Context, key string) (*domainfile.FileObject, error) {
	for _, item := range r.items {
		if item.Key == domainfile.NormalizeKey(key) {
			copy := item
			return &copy, nil
		}
	}
	return nil, nil
}

func (r *fakeRepo) List(context.Context, filerepo.ListFilter, int, int) ([]domainfile.FileObject, error) {
	out := make([]domainfile.FileObject, 0, len(r.items))
	for _, item := range r.items {
		out = append(out, item)
	}
	return out, nil
}

func (r *fakeRepo) SetStatus(_ context.Context, id shared.ID, status domainfile.Status, now time.Time) error {
	item, ok := r.items[id]
	if !ok {
		return nil
	}
	item.Status = status
	item.Meta.Touch(now)
	r.items[id] = item
	return nil
}

func (r *fakeRepo) Delete(_ context.Context, id shared.ID) error {
	delete(r.items, id)
	return nil
}

type fakeObjectStore struct {
	putCalled    bool
	deleteCalled bool
	putErr       error
}

func (s *fakeObjectStore) Put(_ context.Context, in domainfile.PutObjectInput) (domainfile.ObjectInfo, error) {
	s.putCalled = true
	if s.putErr != nil {
		return domainfile.ObjectInfo{}, s.putErr
	}
	return domainfile.ObjectInfo{
		Key:          domainfile.NormalizeKey(in.Key),
		Size:         in.Size,
		MIME:         domainfile.NormalizeMIME(in.MIME),
		Hash:         domainfile.NormalizeHash(in.Hash),
		ETag:         domainfile.NormalizeHash(in.Hash),
		LastModified: time.Now().UTC(),
		Metadata:     domainfile.NormalizeObjectMetadata(in.Metadata),
	}, nil
}

func (s *fakeObjectStore) Get(context.Context, string) (domainfile.ObjectStream, error) {
	return domainfile.ObjectStream{}, nil
}

func (s *fakeObjectStore) Delete(context.Context, string) error {
	s.deleteCalled = true
	return nil
}

func (s *fakeObjectStore) Stat(context.Context, string) (domainfile.ObjectInfo, error) {
	return domainfile.ObjectInfo{}, nil
}

func (s *fakeObjectStore) Presign(context.Context, domainfile.PresignInput) (domainfile.PresignedObject, error) {
	return domainfile.PresignedObject{}, nil
}

var _ io.Reader = strings.NewReader("")
