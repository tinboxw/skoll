package pharmaoa

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	domainpermission "github.com/tinboxw/skoll/internal/domain/permission"
	apiv1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	permissionsvc "github.com/tinboxw/skoll/internal/service/permission"
	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
)

const PermissionBusinessMetricsRead = "pharma_oa.business_metrics.read"

type BusinessMetricsHandler struct {
	service pharmaoasvc.BusinessMetricsService
}

func RegisterBusinessMetricsRoutes(mux *http.ServeMux, service pharmaoasvc.BusinessMetricsService) {
	if mux == nil || service == nil {
		return
	}
	h := &BusinessMetricsHandler{service: service}
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/business-metrics", h.get)
}

func RegisterBusinessMetricsPermissions(service permissionsvc.Service) error {
	if service == nil {
		return nil
	}
	_, err := service.RegisterResource(context.Background(), permissionsvc.RegisterResourceInput{
		Key: PermissionBusinessMetricsRead, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Read aggregated business metrics", Risk: domainpermission.RiskLevelLow,
	})
	return err
}

func (h *BusinessMetricsHandler) get(w http.ResponseWriter, r *http.Request) {
	in, err := businessMetricsInput(r)
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

func businessMetricsInput(r *http.Request) (pharmaoasvc.BusinessMetricsInput, error) {
	query := r.URL.Query()
	from, err := parseBusinessMetricsTime(query.Get("from"), false)
	if err != nil {
		return pharmaoasvc.BusinessMetricsInput{}, fmt.Errorf("from: %w", err)
	}
	to, err := parseBusinessMetricsTime(query.Get("to"), true)
	if err != nil {
		return pharmaoasvc.BusinessMetricsInput{}, fmt.Errorf("to: %w", err)
	}
	qualificationDays := 0
	if raw := strings.TrimSpace(query.Get("qualificationDays")); raw != "" {
		qualificationDays, err = strconv.Atoi(raw)
		if err != nil {
			return pharmaoasvc.BusinessMetricsInput{}, fmt.Errorf("qualificationDays must be an integer")
		}
	}
	return pharmaoasvc.BusinessMetricsInput{
		From: from, To: to, Bucket: pharmaoasvc.BusinessMetricsBucket(strings.TrimSpace(query.Get("bucket"))), QualificationDays: qualificationDays, ActorID: contractActorIDFromRequest(r, ""),
	}, nil
}

func parseBusinessMetricsTime(raw string, endOfDay bool) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, nil
	}
	if value, err := time.Parse(time.RFC3339, raw); err == nil {
		return value.UTC(), nil
	}
	value, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("must use RFC3339 or YYYY-MM-DD")
	}
	if endOfDay {
		return value.AddDate(0, 0, 1).Add(-time.Nanosecond).UTC(), nil
	}
	return value.UTC(), nil
}
