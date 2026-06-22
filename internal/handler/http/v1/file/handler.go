package file

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	domainfile "github.com/tinboxw/skoll/internal/domain/file"
	domainrbac "github.com/tinboxw/skoll/internal/domain/rbac"
	"github.com/tinboxw/skoll/internal/domain/shared"
	apiv1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	filerepo "github.com/tinboxw/skoll/internal/repository/file"
	filesvc "github.com/tinboxw/skoll/internal/service/file"
	"github.com/tinboxw/skoll/pkg/security"
)

type Handler struct {
	service filesvc.Service
}

func RegisterFileRoutes(mux *http.ServeMux, service filesvc.Service) {
	if service == nil {
		return
	}
	h := &Handler{service: service}
	mux.HandleFunc("GET /v1/files", h.list)
	mux.HandleFunc("POST /v1/files", h.upload)
	mux.HandleFunc("POST /v1/files/multipart/init", h.multipartInit)
	mux.HandleFunc("PUT /v1/files/multipart/{uploadId}/parts/{partNumber}", h.multipartUploadPart)
	mux.HandleFunc("POST /v1/files/multipart/{uploadId}/complete", h.multipartComplete)
	mux.HandleFunc("POST /v1/files/multipart/{uploadId}/abort", h.multipartAbort)
	mux.HandleFunc("GET /v1/files/{id}", h.get)
	mux.HandleFunc("DELETE /v1/files/{id}", h.delete)
	mux.HandleFunc("GET /v1/files/{id}/download", h.download)
}

type fileRecord struct {
	ID            string            `json:"id"`
	Key           string            `json:"key"`
	Name          string            `json:"name"`
	Size          int64             `json:"size"`
	MIME          string            `json:"mime"`
	Hash          string            `json:"hash"`
	Owner         ownerRecord       `json:"owner"`
	Visibility    string            `json:"visibility"`
	StorageDriver string            `json:"storageDriver"`
	Status        string            `json:"status"`
	Source        sourceRecord      `json:"source"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	CreatedAt     string            `json:"createdAt"`
	UpdatedAt     string            `json:"updatedAt"`
}

type ownerRecord struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type sourceRecord struct {
	Module   string `json:"module"`
	PluginID string `json:"pluginId,omitempty"`
}

type multipartInitRequest struct {
	Key            string            `json:"key"`
	Name           string            `json:"name"`
	Size           int64             `json:"size"`
	MIME           string            `json:"mime"`
	Hash           string            `json:"hash"`
	OwnerType      string            `json:"ownerType"`
	OwnerID        string            `json:"ownerId"`
	Visibility     string            `json:"visibility"`
	StorageDriver  string            `json:"storageDriver"`
	SourceModule   string            `json:"sourceModule"`
	SourcePluginID string            `json:"sourcePluginId"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

type multipartUploadRecord struct {
	UploadID  string            `json:"uploadId"`
	Key       string            `json:"key"`
	ExpiresAt string            `json:"expiresAt"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type multipartPartRecord struct {
	PartNumber int    `json:"partNumber"`
	Size       int64  `json:"size"`
	Hash       string `json:"hash"`
	ETag       string `json:"etag,omitempty"`
}

type multipartCompleteRequest struct {
	FileID       string                `json:"fileId"`
	Key          string                `json:"key"`
	ExpectedSize int64                 `json:"expectedSize"`
	ExpectedHash string                `json:"expectedHash"`
	Parts        []multipartPartRecord `json:"parts"`
}

type multipartAbortRequest struct {
	FileID string `json:"fileId"`
	Key    string `json:"key"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.service.List(r.Context(), filesvc.ListInput{
		Filter: filerepo.ListFilter{
			OwnerType:      strings.TrimSpace(r.URL.Query().Get("ownerType")),
			OwnerID:        shared.ID(strings.TrimSpace(r.URL.Query().Get("ownerId"))),
			Visibility:     domainfile.Visibility(strings.TrimSpace(r.URL.Query().Get("visibility"))),
			Status:         domainfile.Status(strings.TrimSpace(r.URL.Query().Get("status"))),
			SourceModule:   strings.TrimSpace(r.URL.Query().Get("sourceModule")),
			SourcePluginID: strings.TrimSpace(r.URL.Query().Get("sourcePluginId")),
		},
		Offset: offset,
		Limit:  limit,
	})
	if err != nil {
		apiv1.WriteMessage(w, http.StatusBadRequest, "invalid_file_request", err.Error())
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{
		"items":  fileRecords(items),
		"offset": maxInt(offset, 0),
		"limit":  limit,
	})
}

func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeUploadFailure(w, err)
		return
	}
	part, header, err := r.FormFile("file")
	if err != nil {
		apiv1.WriteMessage(w, http.StatusBadRequest, "invalid_file_upload", "file is required")
		return
	}
	defer part.Close()

	raw, err := io.ReadAll(part)
	if err != nil {
		apiv1.WriteMessage(w, http.StatusBadRequest, "invalid_file_upload", err.Error())
		return
	}
	mime := strings.TrimSpace(header.Header.Get("Content-Type"))
	if mime == "" {
		mime = http.DetectContentType(raw)
	}
	hashBytes := sha256.Sum256(raw)
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		name = header.Filename
	}
	actorID, actorName := actorFromRequest(r)
	ownerID := strings.TrimSpace(r.FormValue("ownerId"))
	if ownerID == "" {
		ownerID = actorID
	}
	if ownerID == "" {
		apiv1.WriteMessage(w, http.StatusBadRequest, "invalid_file_request", "ownerId is required")
		return
	}
	ownerType := strings.TrimSpace(r.FormValue("ownerType"))
	if ownerType == "" {
		ownerType = "user"
	}
	sourceModule := strings.TrimSpace(r.FormValue("sourceModule"))
	if sourceModule == "" {
		sourceModule = "system"
	}
	metadata, err := parseMetadata(r.FormValue("metadata"))
	if err != nil {
		apiv1.WriteMessage(w, http.StatusBadRequest, "invalid_file_request", err.Error())
		return
	}

	object, err := h.service.Upload(r.Context(), filesvc.UploadInput{
		Key:           r.FormValue("key"),
		Name:          name,
		Size:          int64(len(raw)),
		MIME:          mime,
		Hash:          hex.EncodeToString(hashBytes[:]),
		Body:          bytes.NewReader(raw),
		Owner:         domainfile.OwnerRef{Type: ownerType, ID: shared.ID(ownerID)},
		Visibility:    domainfile.Visibility(strings.TrimSpace(r.FormValue("visibility"))),
		StorageDriver: "local",
		Source:        domainfile.SourceRef{Module: sourceModule, PluginID: strings.TrimSpace(r.FormValue("sourcePluginId"))},
		Metadata:      metadata,
		Actor:         domainaudit.ActorRef{Type: "user", ID: shared.ID(actorID), Name: actorName},
		Trace:         traceFromRequest(r),
		AuditMetadata: map[string]any{"path": r.URL.Path},
	})
	if err != nil {
		writeUploadFailure(w, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusCreated, map[string]any{"item": fileRecordFromDomain(*object)})
}

func (h *Handler) multipartInit(w http.ResponseWriter, r *http.Request) {
	var req multipartInitRequest
	if err := decodeJSON(r, &req); err != nil {
		apiv1.WriteMessage(w, http.StatusBadRequest, "invalid_multipart_request", err.Error())
		return
	}
	actorID, actorName := actorFromRequest(r)
	ownerID := strings.TrimSpace(req.OwnerID)
	if ownerID == "" {
		ownerID = actorID
	}
	if ownerID == "" {
		apiv1.WriteMessage(w, http.StatusBadRequest, "invalid_multipart_request", "ownerId is required")
		return
	}
	ownerType := strings.TrimSpace(req.OwnerType)
	if ownerType == "" {
		ownerType = "user"
	}
	sourceModule := strings.TrimSpace(req.SourceModule)
	if sourceModule == "" {
		sourceModule = "system"
	}
	storageDriver := strings.TrimSpace(req.StorageDriver)
	if storageDriver == "" {
		storageDriver = "local"
	}
	result, err := h.service.InitMultipart(r.Context(), filesvc.MultipartInitInput{
		Key:           req.Key,
		Name:          req.Name,
		Size:          req.Size,
		MIME:          req.MIME,
		Hash:          req.Hash,
		Owner:         domainfile.OwnerRef{Type: ownerType, ID: shared.ID(ownerID)},
		Visibility:    domainfile.Visibility(strings.TrimSpace(req.Visibility)),
		StorageDriver: storageDriver,
		Source:        domainfile.SourceRef{Module: sourceModule, PluginID: strings.TrimSpace(req.SourcePluginID)},
		Metadata:      domainfile.NormalizeObjectMetadata(req.Metadata),
		Actor:         domainaudit.ActorRef{Type: "user", ID: shared.ID(actorID), Name: actorName},
		Trace:         traceFromRequest(r),
		AuditMetadata: map[string]any{"path": r.URL.Path},
	})
	if err != nil {
		apiv1.WriteMessage(w, http.StatusBadRequest, "invalid_multipart_request", err.Error())
		return
	}
	apiv1.WriteJSON(w, http.StatusCreated, map[string]any{
		"item":   fileRecordFromDomain(*result.Object),
		"upload": multipartUploadFromDomain(result.Upload),
	})
}

func (h *Handler) multipartUploadPart(w http.ResponseWriter, r *http.Request) {
	uploadID := strings.TrimSpace(r.PathValue("uploadId"))
	partNumber, err := strconv.Atoi(strings.TrimSpace(r.PathValue("partNumber")))
	if err != nil {
		apiv1.WriteMessage(w, http.StatusBadRequest, "invalid_multipart_request", "partNumber must be an integer")
		return
	}
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		apiv1.WriteMessage(w, http.StatusBadRequest, "invalid_multipart_request", err.Error())
		return
	}
	size, err := parseInt64Default(r.URL.Query().Get("size"), int64(len(raw)))
	if err != nil {
		apiv1.WriteMessage(w, http.StatusBadRequest, "invalid_multipart_request", "size must be an integer")
		return
	}
	part, err := h.service.UploadMultipartPart(r.Context(), filesvc.MultipartUploadPartInput{
		UploadID:      uploadID,
		Key:           r.URL.Query().Get("key"),
		PartNumber:    partNumber,
		Size:          size,
		Hash:          r.URL.Query().Get("hash"),
		Body:          bytes.NewReader(raw),
		Trace:         traceFromRequest(r),
		AuditMetadata: map[string]any{"path": r.URL.Path},
	})
	if err != nil {
		apiv1.WriteMessage(w, http.StatusBadRequest, "invalid_multipart_request", err.Error())
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"part": multipartPartFromDomain(part)})
}

func (h *Handler) multipartComplete(w http.ResponseWriter, r *http.Request) {
	var req multipartCompleteRequest
	if err := decodeJSON(r, &req); err != nil {
		apiv1.WriteMessage(w, http.StatusBadRequest, "invalid_multipart_request", err.Error())
		return
	}
	actorID, actorName := actorFromRequest(r)
	object, err := h.service.CompleteMultipart(r.Context(), filesvc.MultipartCompleteInput{
		FileID:        shared.ID(strings.TrimSpace(req.FileID)),
		UploadID:      strings.TrimSpace(r.PathValue("uploadId")),
		Key:           req.Key,
		ExpectedSize:  req.ExpectedSize,
		ExpectedHash:  req.ExpectedHash,
		Parts:         multipartPartsToDomain(req.Parts),
		Actor:         domainaudit.ActorRef{Type: "user", ID: shared.ID(actorID), Name: actorName},
		Trace:         traceFromRequest(r),
		AuditMetadata: map[string]any{"path": r.URL.Path},
	})
	if err != nil {
		code := "invalid_multipart_request"
		if errors.Is(err, filesvc.ErrMultipartHashMismatch) {
			code = "multipart_hash_mismatch"
		}
		apiv1.WriteMessage(w, http.StatusBadRequest, code, err.Error())
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": fileRecordFromDomain(*object)})
}

func (h *Handler) multipartAbort(w http.ResponseWriter, r *http.Request) {
	var req multipartAbortRequest
	if err := decodeJSON(r, &req); err != nil {
		apiv1.WriteMessage(w, http.StatusBadRequest, "invalid_multipart_request", err.Error())
		return
	}
	actorID, actorName := actorFromRequest(r)
	if err := h.service.AbortMultipart(r.Context(), filesvc.MultipartAbortInput{
		FileID:        shared.ID(strings.TrimSpace(req.FileID)),
		UploadID:      strings.TrimSpace(r.PathValue("uploadId")),
		Key:           req.Key,
		Actor:         domainaudit.ActorRef{Type: "user", ID: shared.ID(actorID), Name: actorName},
		Trace:         traceFromRequest(r),
		AuditMetadata: map[string]any{"path": r.URL.Path},
	}); err != nil {
		apiv1.WriteMessage(w, http.StatusBadRequest, "invalid_multipart_request", err.Error())
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"uploadId": strings.TrimSpace(r.PathValue("uploadId")), "aborted": true})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	object, decision, err := h.service.Get(r.Context(), filesvc.GetInput{
		FileID:      shared.ID(strings.TrimSpace(r.PathValue("id"))),
		SubjectType: domainrbac.SubjectUser,
		SubjectID:   shared.ID(actorIDOnly(r)),
		ActorName:   actorNameOnly(r),
		Trace:       traceFromRequest(r),
	})
	if err != nil {
		apiv1.WriteMessage(w, http.StatusBadRequest, "invalid_file_request", err.Error())
		return
	}
	if object == nil {
		writeAccessFailure(w, decision)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": fileRecordFromDomain(*object)})
}

func (h *Handler) download(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.Download(r.Context(), filesvc.DownloadInput{
		FileID:      shared.ID(strings.TrimSpace(r.PathValue("id"))),
		ExpiresIn:   5 * time.Minute,
		SubjectType: domainrbac.SubjectUser,
		SubjectID:   shared.ID(actorIDOnly(r)),
		ActorName:   actorNameOnly(r),
		Trace:       traceFromRequest(r),
	})
	if err != nil {
		apiv1.WriteMessage(w, http.StatusInternalServerError, "file_storage_error", err.Error())
		return
	}
	if result == nil || result.Object == nil {
		decision := filesvc.AccessDecision{Reason: "not_found"}
		if result != nil {
			decision = result.Decision
		}
		writeAccessFailure(w, decision)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{
		"id":        result.Object.ID.String(),
		"url":       result.Presign.URL,
		"method":    strings.ToUpper(string(result.Presign.Operation)),
		"expiresAt": result.Presign.ExpiresAt.Format(time.RFC3339Nano),
		"headers":   result.Presign.Headers,
	})
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	decision, err := h.service.Delete(r.Context(), filesvc.DeleteInput{
		FileID:      shared.ID(id),
		SubjectType: domainrbac.SubjectUser,
		SubjectID:   shared.ID(actorIDOnly(r)),
		ActorName:   actorNameOnly(r),
		Trace:       traceFromRequest(r),
	})
	if err != nil {
		apiv1.WriteMessage(w, http.StatusInternalServerError, "file_storage_error", err.Error())
		return
	}
	if !decision.Allowed {
		writeAccessFailure(w, decision)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"id": id, "deleted": true})
}

func writeAccessFailure(w http.ResponseWriter, decision filesvc.AccessDecision) {
	switch decision.Reason {
	case "not_found":
		apiv1.WriteMessage(w, http.StatusNotFound, "file_not_found", "file not found")
	default:
		apiv1.WriteMessage(w, http.StatusForbidden, "file_forbidden", "file access denied")
	}
}

func writeUploadFailure(w http.ResponseWriter, err error) {
	if errors.Is(err, domainfile.ErrFileTooLarge) || strings.Contains(strings.ToLower(err.Error()), "request body too large") {
		apiv1.WriteMessage(w, http.StatusRequestEntityTooLarge, "file_too_large", "file is too large")
		return
	}
	apiv1.WriteMessage(w, http.StatusBadRequest, "invalid_file_upload", err.Error())
}

func fileRecords(items []domainfile.FileObject) []fileRecord {
	out := make([]fileRecord, 0, len(items))
	for _, item := range items {
		out = append(out, fileRecordFromDomain(item))
	}
	return out
}

func fileRecordFromDomain(item domainfile.FileObject) fileRecord {
	return fileRecord{
		ID:            item.ID.String(),
		Key:           item.Key,
		Name:          item.Name,
		Size:          item.Size,
		MIME:          item.MIME,
		Hash:          item.Hash,
		Owner:         ownerRecord{Type: item.Owner.Type, ID: item.Owner.ID.String()},
		Visibility:    string(item.Visibility),
		StorageDriver: item.StorageDriver,
		Status:        string(item.Status),
		Source:        sourceRecord{Module: item.Source.Module, PluginID: item.Source.PluginID},
		Metadata:      domainfile.NormalizeObjectMetadata(item.Metadata),
		CreatedAt:     item.Meta.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:     item.Meta.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func parseMetadata(raw string) (map[string]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var metadata map[string]string
	if err := json.Unmarshal([]byte(raw), &metadata); err != nil {
		return nil, fmt.Errorf("metadata must be a JSON object string: %w", err)
	}
	return domainfile.NormalizeObjectMetadata(metadata), nil
}

func decodeJSON(r *http.Request, out any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	return decoder.Decode(out)
}

func parseInt64Default(raw string, fallback int64) (int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback, nil
	}
	return strconv.ParseInt(raw, 10, 64)
}

func multipartUploadFromDomain(upload domainfile.MultipartUpload) multipartUploadRecord {
	return multipartUploadRecord{
		UploadID:  upload.UploadID,
		Key:       upload.Key,
		ExpiresAt: upload.ExpiresAt.Format(time.RFC3339Nano),
		Metadata:  domainfile.NormalizeObjectMetadata(upload.Metadata),
	}
}

func multipartPartFromDomain(part domainfile.MultipartPart) multipartPartRecord {
	return multipartPartRecord{
		PartNumber: part.PartNumber,
		Size:       part.Size,
		Hash:       part.Hash,
		ETag:       part.ETag,
	}
}

func multipartPartsToDomain(parts []multipartPartRecord) []domainfile.MultipartPart {
	out := make([]domainfile.MultipartPart, 0, len(parts))
	for _, part := range parts {
		out = append(out, domainfile.MultipartPart{
			PartNumber: part.PartNumber,
			Size:       part.Size,
			Hash:       part.Hash,
			ETag:       part.ETag,
		})
	}
	return out
}

func actorFromRequest(r *http.Request) (string, string) {
	if r == nil {
		return "", ""
	}
	if claims, ok := security.JWTClaimsFromContext(r.Context()); ok {
		return strings.TrimSpace(claims.Subject), strings.TrimSpace(claims.Role)
	}
	return "", ""
}

func actorIDOnly(r *http.Request) string {
	id, _ := actorFromRequest(r)
	return id
}

func actorNameOnly(r *http.Request) string {
	_, name := actorFromRequest(r)
	return name
}

func traceFromRequest(r *http.Request) domainaudit.TraceContext {
	if r == nil {
		return domainaudit.TraceContext{}
	}
	return domainaudit.TraceContext{
		TraceID:   strings.TrimSpace(r.Header.Get("X-Trace-Id")),
		RequestID: strings.TrimSpace(r.Header.Get("X-Request-Id")),
		Method:    r.Method,
		Path:      r.URL.Path,
		IP:        strings.TrimSpace(r.RemoteAddr),
		UserAgent: r.UserAgent(),
	}
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
