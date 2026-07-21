package pharmaoa

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	domainpermission "github.com/tinboxw/skoll/internal/domain/permission"
	apiv1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	permissionsvc "github.com/tinboxw/skoll/internal/service/permission"
	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
)

const (
	PermissionQualificationRead      = "pharma_oa.qualification.read"
	PermissionQualificationExpiryRun = "pharma_oa.qualification.expiry.run"
)

type QualificationHandler struct {
	service pharmaoasvc.QualificationService
}

func RegisterQualificationRoutes(mux *http.ServeMux, service pharmaoasvc.QualificationService) {
	if mux == nil || service == nil {
		return
	}
	h := &QualificationHandler{service: service}
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/qualifications", h.list)
	mux.HandleFunc("POST /v1/plugins/pharma_oa/api/qualifications/expiry-scan", h.scanExpiry)
}

func RegisterQualificationPermissions(service permissionsvc.Service) error {
	if service == nil {
		return nil
	}
	items := []permissionsvc.RegisterResourceInput{
		{Key: PermissionQualificationRead, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Read qualification ledger", Risk: domainpermission.RiskLevelLow},
		{Key: PermissionQualificationExpiryRun, Type: domainpermission.ResourceTypeButton, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Run qualification expiry scan", Risk: domainpermission.RiskLevelHigh},
	}
	for _, item := range items {
		if _, err := service.RegisterResource(context.Background(), item); err != nil {
			return err
		}
	}
	return nil
}

func (h *QualificationHandler) list(w http.ResponseWriter, r *http.Request) {
	days, err := qualificationDays(r.URL.Query().Get("days"))
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	items, err := h.service.List(r.Context(), pharmaoasvc.QualificationListInput{Keyword: r.URL.Query().Get("keyword"), SubjectType: pharmaoasvc.QualificationSubjectType(r.URL.Query().Get("subjectType")), Status: pharmaoasvc.QualificationStatus(r.URL.Query().Get("status")), Days: days})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *QualificationHandler) scanExpiry(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Days    int    `json:"days"`
		ActorID string `json:"actorId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	days, err := qualificationDaysValue(req.Days)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	result, err := h.service.ScanExpiry(r.Context(), pharmaoasvc.QualificationScanInput{Days: days, ActorID: contractActorIDFromRequest(r, req.ActorID)})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": result})
}

func qualificationDays(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 30, nil
	}
	days, err := strconv.Atoi(raw)
	if err != nil {
		return 0, &qualificationInputError{message: "qualification days must be between 1 and 365"}
	}
	return qualificationDaysValue(days)
}

func qualificationDaysValue(days int) (int, error) {
	if days == 0 {
		return 30, nil
	}
	if days < 1 || days > 365 {
		return 0, &qualificationInputError{message: "qualification days must be between 1 and 365"}
	}
	return days, nil
}

type qualificationInputError struct{ message string }

func (e *qualificationInputError) Error() string { return e.message }
