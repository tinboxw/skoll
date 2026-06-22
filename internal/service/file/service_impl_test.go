package file

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
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

func TestUploadAppendsSuccessAuditEvent(t *testing.T) {
	repo := newFakeRepo()
	objects := &fakeObjectStore{}
	audit := &fakeAuditSink{}
	svc := testServiceWithOptions(repo, objects, nil, nil, audit)
	in := validUploadInput()
	in.Trace.RequestID = "req-upload"
	in.AuditMetadata = map[string]any{"tenant": "main"}

	if _, err := svc.Upload(context.Background(), in); err != nil {
		t.Fatalf("Upload() error = %v", err)
	}
	event := audit.last(t)
	if event.Action != domainaudit.AuditAction("file.object.upload") || event.Result != domainaudit.EventResultSuccess {
		t.Fatalf("unexpected audit event: action=%s result=%s", event.Action, event.Result)
	}
	if event.Actor.Type != "user" || event.Actor.ID != "u-1" || event.Resource.Type != "file_object" {
		t.Fatalf("unexpected audit actor/resource: actor=%+v resource=%+v", event.Actor, event.Resource)
	}
	if event.Trace.RequestID != "req-upload" || event.Metadata["tenant"] != "main" || event.SourceData["key"] != "uploads/a.txt" {
		t.Fatalf("unexpected audit trace/metadata/source: trace=%+v metadata=%+v source=%+v", event.Trace, event.Metadata, event.SourceData)
	}
}

func TestUploadFailureAppendsAuditEvent(t *testing.T) {
	repo := newFakeRepo()
	objects := &fakeObjectStore{putErr: fmt.Errorf("disk full")}
	audit := &fakeAuditSink{}
	svc := testServiceWithOptions(repo, objects, nil, nil, audit)

	if _, err := svc.Upload(context.Background(), validUploadInput()); err == nil {
		t.Fatal("expected upload error")
	}
	event := audit.last(t)
	if event.Action != domainaudit.AuditAction("file.object.upload") || event.Result != domainaudit.EventResultFailure {
		t.Fatalf("unexpected audit event: action=%s result=%s", event.Action, event.Result)
	}
	if event.Metadata["reason"] != "object_write_failed" {
		t.Fatalf("unexpected audit metadata: %+v", event.Metadata)
	}
}

func TestUploadAuditFailureDoesNotBlockUpload(t *testing.T) {
	repo := newFakeRepo()
	objects := &fakeObjectStore{}
	audit := &fakeAuditSink{err: fmt.Errorf("audit down")}
	svc := testServiceWithOptions(repo, objects, nil, nil, audit)

	if _, err := svc.Upload(context.Background(), validUploadInput()); err != nil {
		t.Fatalf("Upload() should ignore audit error, got %v", err)
	}
	if len(audit.events) != 1 {
		t.Fatalf("expected one attempted audit event, got %d", len(audit.events))
	}
	if len(repo.items) != 1 {
		t.Fatalf("metadata should still be written, got %+v", repo.items)
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

func TestAuthorizeAccessAppendsDownloadAuditEvent(t *testing.T) {
	repo := newFakeRepo()
	object := validFileObject()
	object.Visibility = domainfile.VisibilityPublic
	repo.items[object.ID] = object
	audit := &fakeAuditSink{}
	svc := testServiceWithOptions(repo, &fakeObjectStore{}, nil, nil, audit)

	decision, err := svc.AuthorizeAccess(context.Background(), AccessInput{
		FileID: object.ID,
		Trace:  domainaudit.TraceContext{RequestID: "req-download"},
	})
	if err != nil {
		t.Fatalf("AuthorizeAccess() error = %v", err)
	}
	if !decision.Allowed {
		t.Fatalf("expected access allowed, got %+v", decision)
	}
	event := audit.last(t)
	if event.Action != domainaudit.AuditAction("file.object.download") || event.Result != domainaudit.EventResultSuccess {
		t.Fatalf("unexpected audit event: action=%s result=%s", event.Action, event.Result)
	}
	if event.Actor.Type != "anonymous" || event.Trace.RequestID != "req-download" {
		t.Fatalf("unexpected audit actor/trace: actor=%+v trace=%+v", event.Actor, event.Trace)
	}
}

func TestAuthorizeAccessAppendsDeleteAuditEvent(t *testing.T) {
	repo := newFakeRepo()
	object := validFileObject()
	repo.items[object.ID] = object
	audit := &fakeAuditSink{}
	svc := testServiceWithOptions(repo, &fakeObjectStore{}, nil, nil, audit)

	decision, err := svc.AuthorizeAccess(context.Background(), AccessInput{
		FileID:      object.ID,
		Action:      AccessActionDelete,
		SubjectType: domainrbac.SubjectUser,
		SubjectID:   "u-1",
	})
	if err != nil {
		t.Fatalf("AuthorizeAccess() error = %v", err)
	}
	if !decision.Allowed {
		t.Fatalf("expected delete allowed for owner, got %+v", decision)
	}
	event := audit.last(t)
	if event.Action != domainaudit.AuditAction("file.object.delete") || event.Result != domainaudit.EventResultSuccess {
		t.Fatalf("unexpected audit event: action=%s result=%s", event.Action, event.Result)
	}
}

func TestAuthorizeAccessAppendsForbiddenAuditEvent(t *testing.T) {
	repo := newFakeRepo()
	object := validFileObject()
	repo.items[object.ID] = object
	audit := &fakeAuditSink{}
	svc := testServiceWithOptions(repo, &fakeObjectStore{}, nil, nil, audit)

	decision, err := svc.AuthorizeAccess(context.Background(), AccessInput{
		FileID:      object.ID,
		SubjectType: domainrbac.SubjectUser,
		SubjectID:   "u-2",
	})
	if err != nil {
		t.Fatalf("AuthorizeAccess() error = %v", err)
	}
	if decision.Allowed {
		t.Fatalf("expected access denied, got %+v", decision)
	}
	event := audit.last(t)
	if event.Action != domainaudit.AuditAction("file.object.forbidden") || event.Result != domainaudit.EventResultDenied || event.Risk != domainaudit.EventRiskHigh {
		t.Fatalf("unexpected audit event: action=%s result=%s risk=%s", event.Action, event.Result, event.Risk)
	}
	if event.Metadata["decisionReason"] != "permission_required" || event.Metadata["requestedAction"] != AccessActionDownload {
		t.Fatalf("unexpected forbidden metadata: %+v", event.Metadata)
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

func TestDownloadPresignsAllowedFile(t *testing.T) {
	repo := newFakeRepo()
	object := validFileObject()
	object.Visibility = domainfile.VisibilityPublic
	repo.items[object.ID] = object
	objects := &fakeObjectStore{}
	svc := testService(repo, objects)

	result, err := svc.Download(context.Background(), DownloadInput{FileID: object.ID})
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	if result == nil || result.Object == nil || !result.Decision.Allowed {
		t.Fatalf("unexpected download result: %+v", result)
	}
	if objects.presignInput.Key != object.Key || objects.presignInput.Operation != domainfile.PresignOperationGet {
		t.Fatalf("presign input = %+v", objects.presignInput)
	}
	if result.Presign.URL == "" {
		t.Fatalf("presign url should be returned")
	}
}

func TestDeleteRemovesObjectAndMarksMetadataDeleted(t *testing.T) {
	repo := newFakeRepo()
	object := validFileObject()
	repo.items[object.ID] = object
	objects := &fakeObjectStore{}
	svc := testService(repo, objects)

	decision, err := svc.Delete(context.Background(), DeleteInput{
		FileID:      object.ID,
		SubjectType: domainrbac.SubjectUser,
		SubjectID:   object.Owner.ID,
	})
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !decision.Allowed {
		t.Fatalf("expected delete allowed, got %+v", decision)
	}
	if !objects.deleteCalled || objects.deletedKey != object.Key {
		t.Fatalf("delete called=%v key=%q", objects.deleteCalled, objects.deletedKey)
	}
	if repo.items[object.ID].Status != domainfile.StatusDeleted {
		t.Fatalf("metadata status = %q", repo.items[object.ID].Status)
	}
}

func TestInitMultipartCreatesPendingMetadata(t *testing.T) {
	repo := newFakeRepo()
	multipart := &fakeMultipartStore{}
	svc := testServiceWithOptions(repo, &fakeObjectStore{}, multipart, nil, nil)

	result, err := svc.InitMultipart(context.Background(), validMultipartInitInput())
	if err != nil {
		t.Fatalf("InitMultipart() error = %v", err)
	}
	if result == nil || result.Object == nil || result.Upload.UploadID == "" {
		t.Fatalf("unexpected init result: %+v", result)
	}
	if !multipart.initCalled || multipart.initInput.Key != "uploads/large.bin" {
		t.Fatalf("multipart init input = %+v", multipart.initInput)
	}
	stored := repo.items[result.Object.ID]
	if stored.Status != domainfile.StatusPending || stored.Hash != "abcdef123456" {
		t.Fatalf("stored metadata = %+v", stored)
	}
}

func TestUploadMultipartPartDelegatesToStore(t *testing.T) {
	multipart := &fakeMultipartStore{}
	svc := testServiceWithOptions(newFakeRepo(), &fakeObjectStore{}, multipart, nil, nil)

	part, err := svc.UploadMultipartPart(context.Background(), MultipartUploadPartInput{
		UploadID:   "upload-1",
		Key:        "uploads/large.bin",
		PartNumber: 1,
		Size:       5,
		Hash:       "11111111",
		Body:       strings.NewReader("hello"),
	})
	if err != nil {
		t.Fatalf("UploadMultipartPart() error = %v", err)
	}
	if !multipart.partCalled || multipart.partInput.PartNumber != 1 || part.Hash != "11111111" {
		t.Fatalf("part=%+v input=%+v", part, multipart.partInput)
	}
}

func TestCompleteMultipartMarksMetadataAvailable(t *testing.T) {
	repo := newFakeRepo()
	object := validFileObject()
	object.Key = "uploads/large.bin"
	object.Name = "large.bin"
	object.Size = 10
	object.Hash = "abcdef123456"
	object.Status = domainfile.StatusPending
	repo.items[object.ID] = object
	multipart := &fakeMultipartStore{completeHash: "abcdef123456"}
	svc := testServiceWithOptions(repo, &fakeObjectStore{}, multipart, nil, nil)

	got, err := svc.CompleteMultipart(context.Background(), MultipartCompleteInput{
		FileID:       object.ID,
		UploadID:     "upload-1",
		ExpectedSize: object.Size,
		ExpectedHash: object.Hash,
		Parts: []domainfile.MultipartPart{
			{PartNumber: 1, Size: 5, Hash: "11111111"},
			{PartNumber: 2, Size: 5, Hash: "22222222"},
		},
	})
	if err != nil {
		t.Fatalf("CompleteMultipart() error = %v", err)
	}
	if got.Status != domainfile.StatusAvailable || repo.items[object.ID].Status != domainfile.StatusAvailable {
		t.Fatalf("metadata status got=%q stored=%q", got.Status, repo.items[object.ID].Status)
	}
	if !multipart.completeCalled || multipart.completeInput.ExpectedHash != object.Hash {
		t.Fatalf("complete input = %+v", multipart.completeInput)
	}
}

func TestCompleteMultipartHashMismatchMarksFailed(t *testing.T) {
	repo := newFakeRepo()
	object := validFileObject()
	object.Key = "uploads/large.bin"
	object.Size = 10
	object.Hash = "abcdef123456"
	object.Status = domainfile.StatusPending
	repo.items[object.ID] = object
	objects := &fakeObjectStore{}
	multipart := &fakeMultipartStore{completeHash: "99999999"}
	svc := testServiceWithOptions(repo, objects, multipart, nil, nil)

	_, err := svc.CompleteMultipart(context.Background(), MultipartCompleteInput{
		FileID:       object.ID,
		UploadID:     "upload-1",
		ExpectedSize: object.Size,
		ExpectedHash: object.Hash,
		Parts: []domainfile.MultipartPart{
			{PartNumber: 1, Size: 5, Hash: "11111111"},
			{PartNumber: 2, Size: 5, Hash: "22222222"},
		},
	})
	if err == nil {
		t.Fatal("expected hash mismatch error")
	}
	if repo.items[object.ID].Status != domainfile.StatusFailed {
		t.Fatalf("metadata status = %q", repo.items[object.ID].Status)
	}
	if !objects.deleteCalled || objects.deletedKey != object.Key {
		t.Fatalf("completed object should be cleaned up: called=%v key=%q", objects.deleteCalled, objects.deletedKey)
	}
}

func TestAbortMultipartMarksMetadataFailed(t *testing.T) {
	repo := newFakeRepo()
	object := validFileObject()
	object.Key = "uploads/large.bin"
	object.Status = domainfile.StatusPending
	repo.items[object.ID] = object
	multipart := &fakeMultipartStore{}
	svc := testServiceWithOptions(repo, &fakeObjectStore{}, multipart, nil, nil)

	if err := svc.AbortMultipart(context.Background(), MultipartAbortInput{FileID: object.ID, UploadID: "upload-1"}); err != nil {
		t.Fatalf("AbortMultipart() error = %v", err)
	}
	if !multipart.abortCalled || multipart.abortInput.Key != object.Key {
		t.Fatalf("abort input = %+v", multipart.abortInput)
	}
	if repo.items[object.ID].Status != domainfile.StatusFailed {
		t.Fatalf("metadata status = %q", repo.items[object.ID].Status)
	}
}

func testService(repo filerepo.FileRepository, objects domainfile.ObjectStore) Service {
	return testServiceWithOptions(repo, objects, nil, nil, nil)
}

func testServiceWithPermission(repo filerepo.FileRepository, objects domainfile.ObjectStore, permission PermissionChecker) Service {
	return testServiceWithOptions(repo, objects, nil, permission, nil)
}

func testServiceWithOptions(repo filerepo.FileRepository, objects domainfile.ObjectStore, multipart domainfile.MultipartStore, permission PermissionChecker, audit AuditEventSink) Service {
	now := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)
	return NewService(repo, objects, Options{
		Now:        func() time.Time { return now },
		NewID:      func() shared.ID { return "file-1" },
		NewAuditID: func() shared.ID { return "audit-1" },
		Permission: permission,
		Audit:      audit,
		Multipart:  multipart,
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

func validMultipartInitInput() MultipartInitInput {
	return MultipartInitInput{
		Key:           " Uploads/Large.BIN ",
		Name:          "large.bin",
		Size:          10,
		MIME:          " application/octet-stream ",
		Hash:          " abcdef123456 ",
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
	deletedKey   string
	putErr       error
	presignInput domainfile.PresignInput
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

func (s *fakeObjectStore) Delete(_ context.Context, key string) error {
	s.deleteCalled = true
	s.deletedKey = domainfile.NormalizeKey(key)
	return nil
}

func (s *fakeObjectStore) Stat(context.Context, string) (domainfile.ObjectInfo, error) {
	return domainfile.ObjectInfo{}, nil
}

func (s *fakeObjectStore) Presign(_ context.Context, in domainfile.PresignInput) (domainfile.PresignedObject, error) {
	s.presignInput = domainfile.NormalizePresignInput(in)
	return domainfile.PresignedObject{
		Key:       s.presignInput.Key,
		Operation: s.presignInput.Operation,
		URL:       "local://object/" + s.presignInput.Key,
		ExpiresAt: time.Now().UTC().Add(s.presignInput.ExpiresIn),
	}, nil
}

type fakeMultipartStore struct {
	initCalled     bool
	partCalled     bool
	completeCalled bool
	abortCalled    bool
	initInput      domainfile.MultipartInitInput
	partInput      domainfile.MultipartUploadPartInput
	completeInput  domainfile.MultipartCompleteInput
	abortInput     domainfile.MultipartAbortInput
	completeHash   string
}

func (s *fakeMultipartStore) Init(_ context.Context, in domainfile.MultipartInitInput) (domainfile.MultipartUpload, error) {
	s.initCalled = true
	s.initInput = domainfile.NormalizeMultipartInitInput(in)
	return domainfile.MultipartUpload{
		UploadID:  "upload-1",
		Key:       s.initInput.Key,
		ExpiresAt: time.Now().UTC().Add(time.Hour),
		Metadata:  s.initInput.Metadata,
	}, nil
}

func (s *fakeMultipartStore) UploadPart(_ context.Context, in domainfile.MultipartUploadPartInput) (domainfile.MultipartPart, error) {
	s.partCalled = true
	s.partInput = domainfile.NormalizeMultipartUploadPartInput(in)
	return domainfile.MultipartPart{
		PartNumber: s.partInput.PartNumber,
		Size:       s.partInput.Size,
		Hash:       s.partInput.Hash,
		ETag:       s.partInput.Hash,
	}, nil
}

func (s *fakeMultipartStore) Complete(_ context.Context, in domainfile.MultipartCompleteInput) (domainfile.ObjectInfo, error) {
	s.completeCalled = true
	s.completeInput = domainfile.NormalizeMultipartCompleteInput(in)
	hash := s.completeHash
	if hash == "" {
		hash = s.completeInput.ExpectedHash
	}
	return domainfile.ObjectInfo{
		Key:          s.completeInput.Key,
		Size:         s.completeInput.ExpectedSize,
		MIME:         "application/octet-stream",
		Hash:         hash,
		ETag:         hash,
		LastModified: time.Now().UTC(),
	}, nil
}

func (s *fakeMultipartStore) Abort(_ context.Context, in domainfile.MultipartAbortInput) error {
	s.abortCalled = true
	s.abortInput = domainfile.NormalizeMultipartAbortInput(in)
	return nil
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

type fakeAuditSink struct {
	events []*domainaudit.Event
	err    error
}

func (s *fakeAuditSink) AppendEvent(_ context.Context, event *domainaudit.Event) error {
	s.events = append(s.events, event)
	return s.err
}

func (s *fakeAuditSink) last(t *testing.T) *domainaudit.Event {
	t.Helper()
	if len(s.events) == 0 {
		t.Fatal("expected audit event")
	}
	return s.events[len(s.events)-1]
}

var _ io.Reader = strings.NewReader("")
