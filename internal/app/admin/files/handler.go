package files

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tinboxw/skoll/internal/module/audit"
	"github.com/tinboxw/skoll/internal/module/fileservice"
)

type FileService interface {
	Upload(name string, content []byte) (fileservice.File, error)
	Get(id int64) (fileservice.File, error)
	List() []fileservice.File
	Download(id int64) (fileservice.File, []byte, error)
}

// AuditService is declared to maintain interface parity with other subpackages; not used directly.
type AuditService interface {
	Append(actor, action, target string) audit.Record
}

type APIRegistry interface {
	RegisterMany(entries []string)
}

type Handler struct {
	svc FileService
}

func NewHandler(svc FileService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(mux *http.ServeMux, wrapper func(http.Handler) http.Handler, apis APIRegistry) {
	if mux == nil || h.svc == nil {
		return
	}

	handle := func(pattern string, next http.HandlerFunc) {
		if apis != nil {
			apis.RegisterMany([]string{pattern})
		}
		hd := http.Handler(next)
		if wrapper != nil {
			hd = wrapper(hd)
		}
		mux.Handle(pattern, hd)
	}

	handle("POST /admin/v1/files", h.uploadFile)
	handle("GET /admin/v1/files", h.listFiles)
	handle("GET /admin/v1/files/{id}", h.getFile)
	handle("GET /admin/v1/files/{id}/download", h.downloadFile)
}

func (h *Handler) uploadFile(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(16 << 20); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid multipart form"})
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "file is required"})
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to read file content"})
		return
	}

	uploaded, err := h.svc.Upload(header.Filename, content)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusCreated, uploaded)
}

func (h *Handler) listFiles(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, h.svc.List())
}

func (h *Handler) getFile(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathInt64(r, "id")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	item, err := h.svc.Get(id)
	if err != nil {
		if err == fileservice.ErrFileNotFound {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	respondJSON(w, http.StatusOK, item)
}

func (h *Handler) downloadFile(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathInt64(r, "id")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	item, content, err := h.svc.Download(id)
	if err != nil {
		if err == fileservice.ErrFileNotFound {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filepath.Base(item.Name)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func parsePathInt64(r *http.Request, key string) (int64, error) {
	raw := strings.TrimSpace(r.PathValue(key))
	if raw == "" {
		return 0, fmt.Errorf("%s is required", key)
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}
	return id, nil
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
