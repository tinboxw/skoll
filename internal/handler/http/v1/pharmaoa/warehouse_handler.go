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

type WarehouseHandler struct {
	service pharmaoasvc.WarehouseService
}

type warehouseRequest struct {
	Code        string                            `json:"code"`
	Name        string                            `json:"name"`
	Region      string                            `json:"region"`
	Temperature domainpharma.WarehouseTemperature `json:"temperature"`
	Areas       []domainpharma.WarehouseArea      `json:"areas"`
	ActorID     string                            `json:"actorId"`
}

func RegisterWarehouseRoutes(mux *http.ServeMux, service pharmaoasvc.WarehouseService) {
	if mux == nil || service == nil {
		return
	}
	h := &WarehouseHandler{service: service}
	mux.HandleFunc("GET /v1/pharma-oa/warehouses", h.list)
	mux.HandleFunc("POST /v1/pharma-oa/warehouses", h.create)
	mux.HandleFunc("PUT /v1/pharma-oa/warehouses/{id}", h.update)
	mux.HandleFunc("POST /v1/pharma-oa/warehouses/{id}/disable", h.disable)
	mux.HandleFunc("GET /v1/pharma-oa/warehouses/{id}/movement-eligibility", h.movementEligibility)
}

func RegisterWarehousePermissions(service permissionsvc.Service) error {
	if service == nil {
		return nil
	}
	for _, item := range []permissionsvc.RegisterResourceInput{
		{Key: "pharma_oa.warehouse.read", Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Read pharma warehouses", Risk: domainpermission.RiskLevelLow},
		{Key: "pharma_oa.warehouse.create", Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Create pharma warehouses", Risk: domainpermission.RiskLevelMedium},
		{Key: "pharma_oa.warehouse.update", Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Update pharma warehouses", Risk: domainpermission.RiskLevelMedium},
		{Key: "pharma_oa.warehouse.disable", Type: domainpermission.ResourceTypeButton, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Disable pharma warehouses", Risk: domainpermission.RiskLevelHigh},
		{Key: "pharma_oa.warehouse.movement", Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Validate warehouse movement location", Risk: domainpermission.RiskLevelHigh},
	} {
		if _, err := service.RegisterResource(context.Background(), item); err != nil {
			return err
		}
	}
	return nil
}

func (h *WarehouseHandler) list(w http.ResponseWriter, r *http.Request) {
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.service.List(r.Context(), pharmaoasvc.WarehouseListInput{
		Keyword: r.URL.Query().Get("keyword"),
		Status:  r.URL.Query().Get("status"),
		Region:  r.URL.Query().Get("region"),
		Offset:  offset,
		Limit:   limit,
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "offset": offset, "limit": limit})
}

func (h *WarehouseHandler) create(w http.ResponseWriter, r *http.Request) {
	input, ok := h.decodeWarehouseWriteInput(w, r)
	if !ok {
		return
	}
	item, err := h.service.Create(r.Context(), input)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusCreated, map[string]any{"item": item})
}

func (h *WarehouseHandler) update(w http.ResponseWriter, r *http.Request) {
	input, ok := h.decodeWarehouseWriteInput(w, r)
	if !ok {
		return
	}
	item, err := h.service.Update(r.Context(), r.PathValue("id"), input)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (h *WarehouseHandler) disable(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Reason  string `json:"reason"`
		ActorID string `json:"actorId"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	item, err := h.service.Disable(r.Context(), r.PathValue("id"), pharmaoasvc.WarehouseDisableInput{
		Reason:  req.Reason,
		ActorID: actorIDFromRequest(r, req.ActorID),
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (h *WarehouseHandler) movementEligibility(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.ValidateMovementLocation(r.Context(), pharmaoasvc.WarehouseMovementLocationInput{
		WarehouseID: r.PathValue("id"),
		AreaID:      r.URL.Query().Get("areaId"),
		LocationID:  r.URL.Query().Get("locationId"),
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": result})
}

func (h *WarehouseHandler) decodeWarehouseWriteInput(w http.ResponseWriter, r *http.Request) (pharmaoasvc.WarehouseWriteInput, bool) {
	var req warehouseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return pharmaoasvc.WarehouseWriteInput{}, false
	}
	return pharmaoasvc.WarehouseWriteInput{
		Code:        req.Code,
		Name:        req.Name,
		Region:      req.Region,
		Temperature: req.Temperature,
		Areas:       req.Areas,
		ActorID:     actorIDFromRequest(r, req.ActorID),
	}, true
}
