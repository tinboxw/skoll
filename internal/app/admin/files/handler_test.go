package files

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tinboxw/skoll/internal/module/fileservice"
)

type testAPIRegistry struct {
	entries []string
}

func (r *testAPIRegistry) RegisterMany(entries []string) {
	r.entries = append(r.entries, entries...)
}

type memoryFileBackend struct {
	items map[string][]byte
}

func (b *memoryFileBackend) Save(_ string, content []byte) (string, error) {
	if b.items == nil {
		b.items = make(map[string][]byte)
	}
	key := fmt.Sprintf("f-%d", len(b.items)+1)
	b.items[key] = append([]byte(nil), content...)
	return key, nil
}

func (b *memoryFileBackend) Open(storageKey string) ([]byte, error) {
	out, ok := b.items[storageKey]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return append([]byte(nil), out...), nil
}

func newFilesMux() *http.ServeMux {
	mux := http.NewServeMux()
	svc := fileservice.NewService(&memoryFileBackend{})
	NewHandler(svc).Register(mux, nil, &testAPIRegistry{})
	return mux
}

func TestFileRoutes_UploadListGetDownload(t *testing.T) {
	mux := newFilesMux()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "hello.txt")
	if err != nil {
		t.Fatalf("create form file failed: %v", err)
	}
	if _, err := part.Write([]byte("hello skoll")); err != nil {
		t.Fatalf("write multipart content failed: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer failed: %v", err)
	}

	uploadReq := httptest.NewRequest(http.MethodPost, "/admin/v1/files", &body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	uploadRR := httptest.NewRecorder()
	mux.ServeHTTP(uploadRR, uploadReq)
	if uploadRR.Code != http.StatusCreated {
		t.Fatalf("expected upload status 201, got %d", uploadRR.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/admin/v1/files", nil)
	listRR := httptest.NewRecorder()
	mux.ServeHTTP(listRR, listReq)
	if listRR.Code != http.StatusOK {
		t.Fatalf("expected file list status 200, got %d", listRR.Code)
	}
	if !strings.Contains(listRR.Body.String(), `"Name":"hello.txt"`) {
		t.Fatalf("expected uploaded file in list, got %s", listRR.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/admin/v1/files/1", nil)
	getRR := httptest.NewRecorder()
	mux.ServeHTTP(getRR, getReq)
	if getRR.Code != http.StatusOK {
		t.Fatalf("expected file get status 200, got %d", getRR.Code)
	}

	downloadReq := httptest.NewRequest(http.MethodGet, "/admin/v1/files/1/download", nil)
	downloadRR := httptest.NewRecorder()
	mux.ServeHTTP(downloadRR, downloadReq)
	if downloadRR.Code != http.StatusOK {
		t.Fatalf("expected file download status 200, got %d", downloadRR.Code)
	}
	if downloadRR.Body.String() != "hello skoll" {
		t.Fatalf("unexpected downloaded body: %s", downloadRR.Body.String())
	}
}
