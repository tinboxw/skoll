package pharmaoa

import (
	"context"
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"strconv"
	"strings"

	domainpermission "github.com/tinboxw/skoll/internal/domain/permission"
	apiv1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	permissionsvc "github.com/tinboxw/skoll/internal/service/permission"
	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
)

const (
	PermissionReportExportRead     = "pharma_oa.report_export.read"
	PermissionReportExportCreate   = "pharma_oa.report_export.create"
	PermissionReportExportRetry    = "pharma_oa.report_export.retry"
	PermissionReportExportDownload = "pharma_oa.report_export.download"
)

type ReportExportHandler struct {
	service pharmaoasvc.ReportExportService
}

type reportExportCreateRequest struct {
	ReportType        pharmaoasvc.ReportExportType      `json:"reportType"`
	From              string                            `json:"from"`
	To                string                            `json:"to"`
	Bucket            pharmaoasvc.BusinessMetricsBucket `json:"bucket"`
	QualificationDays int                               `json:"qualificationDays"`
	ActorID           string                            `json:"actorId"`
}

func RegisterReportExportRoutes(mux *http.ServeMux, service pharmaoasvc.ReportExportService) {
	if mux == nil || service == nil {
		return
	}
	h := &ReportExportHandler{service: service}
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/report-export-jobs", h.list)
	mux.HandleFunc("POST /v1/plugins/pharma_oa/api/report-export-jobs", h.queue)
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/report-export-jobs/{id}", h.get)
	mux.HandleFunc("POST /v1/plugins/pharma_oa/api/report-export-jobs/{id}/retry", h.retry)
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/report-export-jobs/{id}/download", h.download)
}

func RegisterReportExportPermissions(service permissionsvc.Service) error {
	if service == nil {
		return nil
	}
	for _, item := range []permissionsvc.RegisterResourceInput{
		{Key: PermissionReportExportRead, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Read report export jobs", Risk: domainpermission.RiskLevelLow},
		{Key: PermissionReportExportCreate, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Queue report exports", Risk: domainpermission.RiskLevelMedium},
		{Key: PermissionReportExportRetry, Type: domainpermission.ResourceTypeButton, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Retry failed report exports", Risk: domainpermission.RiskLevelHigh},
		{Key: PermissionReportExportDownload, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Download completed report exports", Risk: domainpermission.RiskLevelMedium},
	} {
		if _, err := service.RegisterResource(context.Background(), item); err != nil {
			return err
		}
	}
	return nil
}

func (h *ReportExportHandler) queue(w http.ResponseWriter, r *http.Request) {
	var req reportExportCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	from, err := parseBusinessMetricsTime(req.From, false)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	to, err := parseBusinessMetricsTime(req.To, true)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	item, err := h.service.Queue(r.Context(), pharmaoasvc.ReportExportCreateInput{
		ReportType: req.ReportType, From: from, To: to, Bucket: req.Bucket, QualificationDays: req.QualificationDays,
		ActorID: contractActorIDFromRequest(r, ""),
	})
	if err != nil {
		writeReportExportError(w, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusAccepted, map[string]any{"item": item})
}

func (h *ReportExportHandler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context(), contractActorIDFromRequest(r, ""))
	if err != nil {
		writeReportExportError(w, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *ReportExportHandler) get(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.Get(r.Context(), r.PathValue("id"), contractActorIDFromRequest(r, ""))
	if err != nil {
		writeReportExportError(w, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (h *ReportExportHandler) retry(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.Retry(r.Context(), r.PathValue("id"), contractActorIDFromRequest(r, ""))
	if err != nil {
		writeReportExportError(w, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusAccepted, map[string]any{"item": item})
}

func (h *ReportExportHandler) download(w http.ResponseWriter, r *http.Request) {
	file, err := h.service.Download(r.Context(), r.PathValue("id"), contractActorIDFromRequest(r, ""))
	if err != nil {
		writeReportExportError(w, err)
		return
	}
	contentType := strings.TrimSpace(file.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": file.Filename}))
	w.Header().Set("Content-Length", strconv.Itoa(len(file.Body)))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(file.Body)
}

func writeReportExportError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, pharmaoasvc.ErrReportExportNotFound):
		apiv1.WriteError(w, http.StatusNotFound, err)
	case errors.Is(err, pharmaoasvc.ErrReportExportAccessDenied):
		apiv1.WriteError(w, http.StatusForbidden, err)
	case errors.Is(err, pharmaoasvc.ErrReportExportNotReady), errors.Is(err, pharmaoasvc.ErrReportExportRetryState):
		apiv1.WriteError(w, http.StatusConflict, err)
	default:
		apiv1.WriteError(w, http.StatusBadRequest, err)
	}
}
