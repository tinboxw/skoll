package pharmaoa

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	domainpermission "github.com/tinboxw/skoll/internal/domain/permission"
	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	apiv1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	permissionsvc "github.com/tinboxw/skoll/internal/service/permission"
	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
)

const (
	PermissionInventoryAlertRead = "pharma_oa.inventory_alert.read"
	PermissionInventoryAlertRun  = "pharma_oa.inventory_alert.run"
)

type InventoryAlertHandler struct {
	service pharmaoasvc.InventoryAlertService
}

type inventoryAlertRunRequest struct {
	NearExpiryDays     int    `json:"nearExpiryDays"`
	LowStockThreshold  int    `json:"lowStockThreshold"`
	OverStockThreshold int    `json:"overStockThreshold"`
	RecipientID        string `json:"recipientId"`
}

func RegisterInventoryAlertRoutes(mux *http.ServeMux, service pharmaoasvc.InventoryAlertService) {
	if mux == nil || service == nil {
		return
	}
	h := &InventoryAlertHandler{service: service}
	mux.HandleFunc("GET /v1/pharma-oa/inventory-alerts", h.listAlerts)
	mux.HandleFunc("GET /v1/pharma-oa/inventory-alert-jobs", h.listJobs)
	mux.HandleFunc("POST /v1/pharma-oa/inventory-alert-jobs", h.run)
	mux.HandleFunc("POST /v1/pharma-oa/inventory-alert-jobs/{id}/retry", h.retry)
}

func RegisterInventoryAlertPermissions(service permissionsvc.Service) error {
	if service == nil {
		return nil
	}
	items := []permissionsvc.RegisterResourceInput{
		{Key: PermissionInventoryAlertRead, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Read inventory alerts and jobs", Risk: domainpermission.RiskLevelLow},
		{Key: PermissionInventoryAlertRun, Type: domainpermission.ResourceTypeButton, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Run or retry inventory alert jobs", Risk: domainpermission.RiskLevelHigh},
	}
	for _, item := range items {
		if _, err := service.RegisterResource(context.Background(), item); err != nil {
			return err
		}
	}
	return nil
}

func (h *InventoryAlertHandler) listAlerts(w http.ResponseWriter, r *http.Request) {
	activeOnly, err := strconv.ParseBool(r.URL.Query().Get("activeOnly"))
	if err != nil && r.URL.Query().Get("activeOnly") != "" {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	items, err := h.service.ListAlerts(r.Context(), activeOnly)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *InventoryAlertHandler) listJobs(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListJobs(r.Context())
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *InventoryAlertHandler) run(w http.ResponseWriter, r *http.Request) {
	var req inventoryAlertRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	job, err := h.service.Run(r.Context(), domainpharma.InventoryAlertPolicy{NearExpiryDays: req.NearExpiryDays, LowStockThreshold: req.LowStockThreshold, OverStockThreshold: req.OverStockThreshold, RecipientID: req.RecipientID})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusCreated, map[string]any{"item": job})
}

func (h *InventoryAlertHandler) retry(w http.ResponseWriter, r *http.Request) {
	job, err := h.service.Retry(r.Context(), r.PathValue("id"))
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": job})
}
