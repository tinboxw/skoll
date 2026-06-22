package file

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	domainfile "github.com/tinboxw/skoll/internal/domain/file"
	domainrbac "github.com/tinboxw/skoll/internal/domain/rbac"
	"github.com/tinboxw/skoll/internal/domain/shared"
	filerepo "github.com/tinboxw/skoll/internal/repository/file"
	rbacsvc "github.com/tinboxw/skoll/internal/service/rbac"
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

func TestAuthorizeAccessAllowsPublicAvailableFile(t *testing.T) {
	repo := newFakeRepo()
	object := validFileObject()
	object.Visibility = domainfile.VisibilityPublic
	repo.items[object.ID] = object
	svc := testService(repo, &fakeObjectStore{})

	decision, err := svc.AuthorizeAccess(context.Background(), AccessInput{FileID: object.ID})
	if err != nil {
		t.Fatalf("AuthorizeAccess() error = %v", err)
	}
	if !decision.Allowed || decision.Reason != "public" || decision.Resource != "file:public" {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestAuthorizeAccessAllowsPrivateOwner(t *testing.T) {
	repo := newFakeRepo()
	object := validFileObject()
	repo.items[object.ID] = object
	svc := testService(repo, &fakeObjectStore{})

	decision, err := svc.AuthorizeAccess(context.Background(), AccessInput{
		FileID:      object.ID,
		SubjectType: domainrbac.SubjectUser,
		SubjectID:   "u-1",
	})
	if err != nil {
		t.Fatalf("AuthorizeAccess() error = %v", err)
	}
	if !decision.Allowed || decision.Reason != "owner" {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestAuthorizeAccessDeniesPrivateWithoutPermission(t *testing.T) {
	repo := newFakeRepo()
	object := validFileObject()
	repo.items[object.ID] = object
	svc := testService(repo, &fakeObjectStore{})

	decision, err := svc.AuthorizeAccess(context.Background(), AccessInput{
		FileID:      object.ID,
		SubjectType: domainrbac.SubjectUser,
		SubjectID:   "u-2",
	})
	if err != nil {
		t.Fatalf("AuthorizeAccess() error = %v", err)
	}
	if decision.Allowed || decision.Reason != "permission_required" {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestAuthorizeAccessAllowsPrivateWithRBACPermission(t *testing.T) {
	repo := newFakeRepo()
	object := validFileObject()
	repo.items[object.ID] = object
	checker := &fakePermissionChecker{allowed: map[string]bool{"user:u-2|file:private|download": true}}
	svc := testServiceWithPermission(repo, &fakeObjectStore{}, checker)

	decision, err := svc.AuthorizeAccess(context.Background(), AccessInput{
		FileID:      object.ID,
		SubjectType: domainrbac.SubjectUser,
		SubjectID:   "u-2",
	})
	if err != nil {
		t.Fatalf("AuthorizeAccess() error = %v", err)
	}
	if !decision.Allowed || decision.Reason != "permission" {
		t.Fatalf("unexpected decision: %+v", decision)
	}
	if checker.last.Resource != "file:private" || checker.last.Action != "download" {
		t.Fatalf("unexpected permission input: %+v", checker.last)
	}
}

func TestAuthorizeAccessPluginAssetRequiresPermission(t *testing.T) {
	repo := newFakeRepo()
	object := validFileObject()
	object.Visibility = domainfile.VisibilityPluginAsset
	object.Source.PluginID = "demo"
	repo.items[object.ID] = object

	deniedSvc := testService(repo, &fakeObjectStore{})
	denied, err := deniedSvc.AuthorizeAccess(context.Background(), AccessInput{
		FileID:      object.ID,
		SubjectType: domainrbac.SubjectUser,
		SubjectID:   "u-2",
	})
	if err != nil {
		t.Fatalf("AuthorizeAccess() denied path error = %v", err)
	}
	if denied.Allowed || denied.Resource != "plugin:demo:asset" || denied.Reason != "permission_required" {
		t.Fatalf("unexpected denied decision: %+v", denied)
	}

	checker := &fakePermissionChecker{allowed: map[string]bool{"user:u-2|plugin:demo:asset|download": true}}
	allowedSvc := testServiceWithPermission(repo, &fakeObjectStore{}, checker)
	allowed, err := allowedSvc.AuthorizeAccess(context.Background(), AccessInput{
		FileID:      object.ID,
		SubjectType: domainrbac.SubjectUser,
		SubjectID:   "u-2",
	})
	if err != nil {
		t.Fatalf("AuthorizeAccess() allowed path error = %v", err)
	}
	if !allowed.Allowed || allowed.Reason != "permission" {
		t.Fatalf("unexpected allowed decision: %+v", allowed)
	}
}

func TestAuthorizeAccessDeniesUnavailableFile(t *testing.T) {
	repo := newFakeRepo()
	object := validFileObject()
	object.Status = domainfile.StatusPending
	repo.items[object.ID] = object
	svc := testService(repo, &fakeObjectStore{})

	decision, err := svc.AuthorizeAccess(context.Background(), AccessInput{
		FileID:      object.ID,
		SubjectType: domainrbac.SubjectUser,
		SubjectID:   "u-1",
	})
	if err != nil {
		t.Fatalf("AuthorizeAccess() error = %v", err)
	}
	if decision.Allowed || decision.Reason != "not_available" {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func testService(repo filerepo.FileRepository, objects domainfile.ObjectStore) Service {
	return testServiceWithPermission(repo, objects, nil)
}

func testServiceWithPermission(repo filerepo.FileRepository, objects domainfile.ObjectStore, permission PermissionChecker) Service {
	now := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)
	return NewService(repo, objects, Options{
		Now:        func() time.Time { return now },
		NewID:      func() shared.ID { return "file-1" },
		Permission: permission,
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

func validFileObject() domainfile.FileObject {
	now := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)
	object, err := domainfile.NewFileObject(domainfile.FileObjectInput{
		ID:            "file-1",
		Key:           "uploads/a.txt",
		Name:          "a.txt",
		Size:          5,
		MIME:          "text/plain",
		Hash:          "abcdef123456",
		Owner:         domainfile.OwnerRef{Type: "user", ID: "u-1"},
		Visibility:    domainfile.VisibilityPrivate,
		StorageDriver: "local",
		Status:        domainfile.StatusAvailable,
		Source:        domainfile.SourceRef{Module: "system"},
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	if err != nil {
		panic(err)
	}
	return *object
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

type fakePermissionChecker struct {
	allowed map[string]bool
	last    rbacsvc.CheckPermissionInput
	err     error
}

func (c *fakePermissionChecker) CheckPermission(_ context.Context, in rbacsvc.CheckPermissionInput) (bool, error) {
	c.last = in
	if c.err != nil {
		return false, c.err
	}
	key := string(in.SubjectType) + ":" + in.SubjectID + "|" + in.Resource + "|" + in.Action
	return c.allowed[key], nil
}

var _ io.Reader = strings.NewReader("")
