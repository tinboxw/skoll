package file

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	domainfile "github.com/tinboxw/skoll/internal/domain/file"
	domainrbac "github.com/tinboxw/skoll/internal/domain/rbac"
	filerepo "github.com/tinboxw/skoll/internal/repository/file"
	filesvc "github.com/tinboxw/skoll/internal/service/file"
	"github.com/tinboxw/skoll/pkg/security"
)

func TestFileHandlerListAndDetail(t *testing.T) {
	svc := &fakeFileService{items: []domainfile.FileObject{mustFileObject(t)}}
	mux := http.NewServeMux()
	RegisterFileRoutes(mux, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/files?visibility=private&status=available&offset=0&limit=20", nil)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", resp.Code, resp.Body.String())
	}
	if svc.listInput.Filter.Visibility != domainfile.VisibilityPrivate || svc.listInput.Filter.Status != domainfile.StatusAvailable {
		t.Fatalf("list input = %+v", svc.listInput)
	}
	body := decodeBody(t, resp)
	items := body["data"].(map[string]any)["items"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["id"] != "file-1" {
		t.Fatalf("items = %+v", items)
	}

	detailReq := withClaims(httptest.NewRequest(http.MethodGet, "/v1/files/file-1", nil), "u-1")
	detailResp := httptest.NewRecorder()
	mux.ServeHTTP(detailResp, detailReq)
	if detailResp.Code != http.StatusOK {
		t.Fatalf("detail status=%d body=%s", detailResp.Code, detailResp.Body.String())
	}
	if svc.getInput.FileID != "file-1" || svc.getInput.SubjectID != "u-1" {
		t.Fatalf("get input = %+v", svc.getInput)
	}
}

func TestFileHandlerUpload(t *testing.T) {
	svc := &fakeFileService{uploadObject: mustFileObject(t)}
	mux := http.NewServeMux()
	RegisterFileRoutes(mux, svc)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("key", "uploads/a.txt")
	_ = writer.WriteField("visibility", "private")
	_ = writer.WriteField("ownerId", "u-1")
	_ = writer.WriteField("metadata", `{"trace-id":"req-1"}`)
	part, err := writer.CreateFormFile("file", "a.txt")
	if err != nil {
		t.Fatalf("CreateFormFile error: %v", err)
	}
	_, _ = part.Write([]byte("hello"))
	_ = writer.Close()

	req := withClaims(httptest.NewRequest(http.MethodPost, "/v1/files", &body), "u-1")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("upload status=%d body=%s", resp.Code, resp.Body.String())
	}
	if svc.uploadInput.Key != "uploads/a.txt" || svc.uploadInput.Size != 5 || svc.uploadInput.Owner.ID != "u-1" {
		t.Fatalf("upload input = %+v", svc.uploadInput)
	}
	if svc.uploadInput.Metadata["trace-id"] != "req-1" {
		t.Fatalf("upload metadata = %+v", svc.uploadInput.Metadata)
	}
}

func TestFileHandlerUploadSecurityFailures(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "path traversal", err: fmt.Errorf("file key must not contain relative path segments"), wantStatus: http.StatusBadRequest, wantCode: "invalid_file_upload"},
		{name: "invalid mime", err: fmt.Errorf("file mime type is invalid"), wantStatus: http.StatusBadRequest, wantCode: "invalid_file_upload"},
		{name: "too large", err: domainfile.ErrFileTooLarge, wantStatus: http.StatusRequestEntityTooLarge, wantCode: "file_too_large"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &fakeFileService{uploadErr: tc.err}
			mux := http.NewServeMux()
			RegisterFileRoutes(mux, svc)

			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			_ = writer.WriteField("key", "uploads/a.txt")
			_ = writer.WriteField("visibility", "private")
			_ = writer.WriteField("ownerId", "u-1")
			part, err := writer.CreateFormFile("file", "a.txt")
			if err != nil {
				t.Fatalf("CreateFormFile error: %v", err)
			}
			_, _ = part.Write([]byte("hello"))
			_ = writer.Close()

			req := withClaims(httptest.NewRequest(http.MethodPost, "/v1/files", &body), "u-1")
			req.Header.Set("Content-Type", writer.FormDataContentType())
			resp := httptest.NewRecorder()
			mux.ServeHTTP(resp, req)
			if resp.Code != tc.wantStatus || decodeBody(t, resp)["code"] != tc.wantCode {
				t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
			}
		})
	}
}

func TestFileHandlerDownloadAndDelete(t *testing.T) {
	object := mustFileObject(t)
	svc := &fakeFileService{
		downloadResult: &filesvc.DownloadResult{
			Object:   &object,
			Decision: filesvc.AccessDecision{Allowed: true},
			Presign: domainfile.PresignedObject{
				Key:       object.Key,
				Operation: domainfile.PresignOperationGet,
				URL:       "local://object/uploads/a.txt",
				ExpiresAt: time.Date(2026, 6, 22, 10, 5, 0, 0, time.UTC),
			},
		},
		deleteDecision: filesvc.AccessDecision{Allowed: true},
	}
	mux := http.NewServeMux()
	RegisterFileRoutes(mux, svc)

	downloadReq := withClaims(httptest.NewRequest(http.MethodGet, "/v1/files/file-1/download", nil), "u-1")
	downloadResp := httptest.NewRecorder()
	mux.ServeHTTP(downloadResp, downloadReq)
	if downloadResp.Code != http.StatusOK {
		t.Fatalf("download status=%d body=%s", downloadResp.Code, downloadResp.Body.String())
	}
	downloadBody := decodeBody(t, downloadResp)
	if downloadBody["data"].(map[string]any)["url"] != "local://object/uploads/a.txt" {
		t.Fatalf("download body = %+v", downloadBody)
	}
	if svc.downloadInput.FileID != "file-1" || svc.downloadInput.SubjectID != "u-1" {
		t.Fatalf("download input = %+v", svc.downloadInput)
	}

	deleteReq := withClaims(httptest.NewRequest(http.MethodDelete, "/v1/files/file-1", nil), "u-1")
	deleteResp := httptest.NewRecorder()
	mux.ServeHTTP(deleteResp, deleteReq)
	if deleteResp.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", deleteResp.Code, deleteResp.Body.String())
	}
	if svc.deleteInput.FileID != "file-1" || svc.deleteInput.SubjectID != "u-1" {
		t.Fatalf("delete input = %+v", svc.deleteInput)
	}
}

func TestFileHandlerMultipartFlow(t *testing.T) {
	object := mustPendingFileObject(t)
	svc := &fakeFileService{
		multipartInitResult: &filesvc.MultipartInitResult{
			Object: &object,
			Upload: domainfile.MultipartUpload{
				UploadID:  "upload-1",
				Key:       object.Key,
				ExpiresAt: time.Date(2026, 6, 22, 11, 0, 0, 0, time.UTC),
			},
		},
		multipartPart:  domainfile.MultipartPart{PartNumber: 1, Size: 5, Hash: strings.Repeat("1", 8), ETag: strings.Repeat("1", 8)},
		completeObject: mustFileObject(t),
	}
	mux := http.NewServeMux()
	RegisterFileRoutes(mux, svc)

	initBody := strings.NewReader(`{"key":"uploads/large.bin","name":"large.bin","size":10,"mime":"application/octet-stream","hash":"abcdef123456","ownerId":"u-1","visibility":"private","metadata":{"trace-id":"req-1"}}`)
	initReq := withClaims(httptest.NewRequest(http.MethodPost, "/v1/files/multipart/init", initBody), "u-1")
	initResp := httptest.NewRecorder()
	mux.ServeHTTP(initResp, initReq)
	if initResp.Code != http.StatusCreated {
		t.Fatalf("multipart init status=%d body=%s", initResp.Code, initResp.Body.String())
	}
	if svc.multipartInitInput.Key != "uploads/large.bin" || svc.multipartInitInput.Owner.ID != "u-1" {
		t.Fatalf("multipart init input = %+v", svc.multipartInitInput)
	}

	partReq := withClaims(httptest.NewRequest(http.MethodPut, "/v1/files/multipart/upload-1/parts/1?key=uploads/large.bin&size=5&hash=11111111", strings.NewReader("hello")), "u-1")
	partResp := httptest.NewRecorder()
	mux.ServeHTTP(partResp, partReq)
	if partResp.Code != http.StatusOK {
		t.Fatalf("multipart part status=%d body=%s", partResp.Code, partResp.Body.String())
	}
	if svc.multipartPartInput.UploadID != "upload-1" || svc.multipartPartInput.PartNumber != 1 || svc.multipartPartInput.Size != 5 {
		t.Fatalf("multipart part input = %+v", svc.multipartPartInput)
	}

	completeBody := strings.NewReader(`{"fileId":"file-1","expectedSize":10,"expectedHash":"abcdef123456","parts":[{"partNumber":1,"size":5,"hash":"11111111"},{"partNumber":2,"size":5,"hash":"22222222"}]}`)
	completeReq := withClaims(httptest.NewRequest(http.MethodPost, "/v1/files/multipart/upload-1/complete", completeBody), "u-1")
	completeResp := httptest.NewRecorder()
	mux.ServeHTTP(completeResp, completeReq)
	if completeResp.Code != http.StatusOK {
		t.Fatalf("multipart complete status=%d body=%s", completeResp.Code, completeResp.Body.String())
	}
	if svc.multipartCompleteInput.UploadID != "upload-1" || len(svc.multipartCompleteInput.Parts) != 2 {
		t.Fatalf("multipart complete input = %+v", svc.multipartCompleteInput)
	}

	abortReq := withClaims(httptest.NewRequest(http.MethodPost, "/v1/files/multipart/upload-1/abort", strings.NewReader(`{"fileId":"file-1"}`)), "u-1")
	abortResp := httptest.NewRecorder()
	mux.ServeHTTP(abortResp, abortReq)
	if abortResp.Code != http.StatusOK {
		t.Fatalf("multipart abort status=%d body=%s", abortResp.Code, abortResp.Body.String())
	}
	if svc.multipartAbortInput.UploadID != "upload-1" || svc.multipartAbortInput.FileID != "file-1" {
		t.Fatalf("multipart abort input = %+v", svc.multipartAbortInput)
	}
}

func TestFileHandlerMultipartHashMismatch(t *testing.T) {
	svc := &fakeFileService{completeErr: filesvc.ErrMultipartHashMismatch}
	mux := http.NewServeMux()
	RegisterFileRoutes(mux, svc)

	req := withClaims(httptest.NewRequest(http.MethodPost, "/v1/files/multipart/upload-1/complete", strings.NewReader(`{"fileId":"file-1","parts":[{"partNumber":1,"size":5,"hash":"11111111"}]}`)), "u-1")
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest || decodeBody(t, resp)["code"] != "multipart_hash_mismatch" {
		t.Fatalf("hash mismatch status=%d body=%s", resp.Code, resp.Body.String())
	}
}

func TestFileHandlerForbiddenAndNotFound(t *testing.T) {
	svc := &fakeFileService{
		getDecision:     filesvc.AccessDecision{Reason: "permission_required"},
		downloadResult:  &filesvc.DownloadResult{Decision: filesvc.AccessDecision{Reason: "not_found"}},
		deleteDecision:  filesvc.AccessDecision{Reason: "permission_denied"},
		downloadNilBody: true,
	}
	mux := http.NewServeMux()
	RegisterFileRoutes(mux, svc)

	getReq := withClaims(httptest.NewRequest(http.MethodGet, "/v1/files/file-1", nil), "u-2")
	getResp := httptest.NewRecorder()
	mux.ServeHTTP(getResp, getReq)
	if getResp.Code != http.StatusForbidden || decodeBody(t, getResp)["code"] != "file_forbidden" {
		t.Fatalf("forbidden status=%d body=%s", getResp.Code, getResp.Body.String())
	}

	downloadReq := withClaims(httptest.NewRequest(http.MethodGet, "/v1/files/file-missing/download", nil), "u-2")
	downloadResp := httptest.NewRecorder()
	mux.ServeHTTP(downloadResp, downloadReq)
	if downloadResp.Code != http.StatusNotFound || decodeBody(t, downloadResp)["code"] != "file_not_found" {
		t.Fatalf("missing status=%d body=%s", downloadResp.Code, downloadResp.Body.String())
	}

	deleteReq := withClaims(httptest.NewRequest(http.MethodDelete, "/v1/files/file-1", nil), "u-2")
	deleteResp := httptest.NewRecorder()
	mux.ServeHTTP(deleteResp, deleteReq)
	if deleteResp.Code != http.StatusForbidden || decodeBody(t, deleteResp)["code"] != "file_forbidden" {
		t.Fatalf("delete forbidden status=%d body=%s", deleteResp.Code, deleteResp.Body.String())
	}
}

type fakeFileService struct {
	uploadInput            filesvc.UploadInput
	listInput              filesvc.ListInput
	getInput               filesvc.GetInput
	downloadInput          filesvc.DownloadInput
	deleteInput            filesvc.DeleteInput
	multipartInitInput     filesvc.MultipartInitInput
	multipartPartInput     filesvc.MultipartUploadPartInput
	multipartCompleteInput filesvc.MultipartCompleteInput
	multipartAbortInput    filesvc.MultipartAbortInput
	items                  []domainfile.FileObject
	uploadObject           domainfile.FileObject
	uploadErr              error
	multipartInitResult    *filesvc.MultipartInitResult
	multipartPart          domainfile.MultipartPart
	completeObject         domainfile.FileObject
	completeErr            error
	getDecision            filesvc.AccessDecision
	downloadResult         *filesvc.DownloadResult
	deleteDecision         filesvc.AccessDecision
	downloadNilBody        bool
}

func (s *fakeFileService) Upload(_ context.Context, in filesvc.UploadInput) (*domainfile.FileObject, error) {
	s.uploadInput = in
	if s.uploadErr != nil {
		return nil, s.uploadErr
	}
	object := s.uploadObject
	if object.ID.IsZero() {
		object = mustFileObject(nil)
	}
	return &object, nil
}

func (s *fakeFileService) InitMultipart(_ context.Context, in filesvc.MultipartInitInput) (*filesvc.MultipartInitResult, error) {
	s.multipartInitInput = in
	if s.multipartInitResult != nil {
		return s.multipartInitResult, nil
	}
	object := mustPendingFileObject(nil)
	return &filesvc.MultipartInitResult{Object: &object, Upload: domainfile.MultipartUpload{UploadID: "upload-1", Key: object.Key, ExpiresAt: time.Now().UTC()}}, nil
}

func (s *fakeFileService) UploadMultipartPart(_ context.Context, in filesvc.MultipartUploadPartInput) (domainfile.MultipartPart, error) {
	s.multipartPartInput = in
	if s.multipartPart.PartNumber != 0 {
		return s.multipartPart, nil
	}
	return domainfile.MultipartPart{PartNumber: in.PartNumber, Size: in.Size, Hash: in.Hash, ETag: in.Hash}, nil
}

func (s *fakeFileService) CompleteMultipart(_ context.Context, in filesvc.MultipartCompleteInput) (*domainfile.FileObject, error) {
	s.multipartCompleteInput = in
	if s.completeErr != nil {
		return nil, s.completeErr
	}
	object := s.completeObject
	if object.ID.IsZero() {
		object = mustFileObject(nil)
	}
	return &object, nil
}

func (s *fakeFileService) AbortMultipart(_ context.Context, in filesvc.MultipartAbortInput) error {
	s.multipartAbortInput = in
	return nil
}

func (s *fakeFileService) List(_ context.Context, in filesvc.ListInput) ([]domainfile.FileObject, error) {
	s.listInput = in
	return s.items, nil
}

func (s *fakeFileService) Get(_ context.Context, in filesvc.GetInput) (*domainfile.FileObject, filesvc.AccessDecision, error) {
	s.getInput = in
	if s.getDecision.Reason != "" && !s.getDecision.Allowed {
		return nil, s.getDecision, nil
	}
	object := mustFileObject(nil)
	return &object, filesvc.AccessDecision{Allowed: true}, nil
}

func (s *fakeFileService) Download(_ context.Context, in filesvc.DownloadInput) (*filesvc.DownloadResult, error) {
	s.downloadInput = in
	if s.downloadNilBody {
		return s.downloadResult, nil
	}
	return s.downloadResult, nil
}

func (s *fakeFileService) Delete(_ context.Context, in filesvc.DeleteInput) (filesvc.AccessDecision, error) {
	s.deleteInput = in
	return s.deleteDecision, nil
}

func (s *fakeFileService) AuthorizeAccess(context.Context, filesvc.AccessInput) (filesvc.AccessDecision, error) {
	return filesvc.AccessDecision{}, nil
}

func mustFileObject(t *testing.T) domainfile.FileObject {
	if t != nil {
		t.Helper()
	}
	now := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)
	object, err := domainfile.NewFileObject(domainfile.FileObjectInput{
		ID:            "file-1",
		Key:           "uploads/a.txt",
		Name:          "a.txt",
		Size:          5,
		MIME:          "text/plain",
		Hash:          strings.Repeat("a", 64),
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

func mustPendingFileObject(t *testing.T) domainfile.FileObject {
	object := mustFileObject(t)
	object.Key = "uploads/large.bin"
	object.Name = "large.bin"
	object.Size = 10
	object.MIME = "application/octet-stream"
	object.Hash = "abcdef123456"
	object.Status = domainfile.StatusPending
	return object
}

func withClaims(req *http.Request, subject string) *http.Request {
	return req.WithContext(security.WithJWTClaimsContext(req.Context(), &security.JWTClaims{Subject: subject, Role: "admin"}))
}

func decodeBody(t *testing.T, resp *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v; %s", err, resp.Body.String())
	}
	return body
}

var _ = filerepo.ListFilter{}
var _ = domainrbac.SubjectUser
