package pharmaoa

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	domainpermission "github.com/tinboxw/skoll/internal/domain/permission"
	apiv1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	permissionsvc "github.com/tinboxw/skoll/internal/service/permission"
	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
)

const (
	PermissionComplianceDashboardRead   = "pharma_oa.compliance_dashboard.read"
	PermissionComplianceDashboardExport = "pharma_oa.compliance_dashboard.export"
)

type ComplianceDashboardHandler struct {
	service pharmaoasvc.ComplianceDashboardService
}

func RegisterComplianceDashboardRoutes(mux *http.ServeMux, service pharmaoasvc.ComplianceDashboardService) {
	if mux == nil || service == nil {
		return
	}
	h := &ComplianceDashboardHandler{service: service}
	mux.HandleFunc("GET /v1/pharma-oa/compliance-dashboard", h.get)
	mux.HandleFunc("GET /v1/pharma-oa/compliance-dashboard/export", h.export)
}

func RegisterComplianceDashboardPermissions(service permissionsvc.Service) error {
	if service == nil {
		return nil
	}
	items := []permissionsvc.RegisterResourceInput{
		{Key: PermissionComplianceDashboardRead, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Read the compliance audit dashboard", Risk: domainpermission.RiskLevelLow},
		{Key: PermissionComplianceDashboardExport, Type: domainpermission.ResourceTypeButton, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Export compliance audit evidence", Risk: domainpermission.RiskLevelMedium},
	}
	for _, item := range items {
		if _, err := service.RegisterResource(context.Background(), item); err != nil {
			return err
		}
	}
	return nil
}

func (h *ComplianceDashboardHandler) get(w http.ResponseWriter, r *http.Request) {
	in, err := complianceDashboardInput(r)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	item, err := h.service.Get(r.Context(), in)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (h *ComplianceDashboardHandler) export(w http.ResponseWriter, r *http.Request) {
	in, err := complianceDashboardInput(r)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	content, err := h.service.Export(r.Context(), in)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="pharma-oa-compliance-risks.csv"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func complianceDashboardInput(r *http.Request) (pharmaoasvc.ComplianceDashboardInput, error) {
	limit := 200
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 500 {
			return pharmaoasvc.ComplianceDashboardInput{}, fmt.Errorf("limit must be between 1 and 500")
		}
		limit = value
	}
	return pharmaoasvc.ComplianceDashboardInput{
		Keyword: r.URL.Query().Get("keyword"),
		Risk:    pharmaoasvc.ComplianceRisk(r.URL.Query().Get("risk")),
		Source:  pharmaoasvc.ComplianceSource(r.URL.Query().Get("source")),
		Limit:   limit,
		ActorID: contractActorIDFromRequest(r, ""),
	}, nil
}
