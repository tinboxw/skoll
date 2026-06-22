package file

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
		apiv1.WriteMessage(w, http.StatusBadRequest, "invalid_file_upload", err.Error())
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
		apiv1.WriteMessage(w, http.StatusBadRequest, "invalid_file_upload", err.Error())
		return
	}
	apiv1.WriteJSON(w, http.StatusCreated, map[string]any{"item": fileRecordFromDomain(*object)})
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
