package pharmaoa

import (
	"context"
	"encoding/json"
	"net/http"

	domainpermission "github.com/tinboxw/skoll/internal/domain/permission"
	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	apiv1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	permissionsvc "github.com/tinboxw/skoll/internal/service/permission"
	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
)

const (
	PermissionInboundRead   = "pharma_oa.inbound.read"
	PermissionInboundCreate = "pharma_oa.inbound.create"
)

type PurchaseInboundHandler struct {
	service pharmaoasvc.PurchaseInboundService
}
type purchaseInboundCreateRequest struct {
	Number          string                             `json:"number"`
	PurchaseOrderID string                             `json:"purchaseOrderId"`
	WarehouseID     string                             `json:"warehouseId"`
	AreaID          string                             `json:"areaId"`
	LocationID      string                             `json:"locationId"`
	Lines           []domainpharma.PurchaseInboundLine `json:"lines"`
	Attachments     []domainpharma.InboundAttachment   `json:"attachments"`
	ActorID         string                             `json:"actorId"`
}

func RegisterPurchaseInboundRoutes(mux *http.ServeMux, service pharmaoasvc.PurchaseInboundService) {
	if mux == nil || service == nil {
		return
	}
	h := &PurchaseInboundHandler{service: service}
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/purchase-inbounds", h.list)
	mux.HandleFunc("POST /v1/plugins/pharma_oa/api/purchase-inbounds", h.create)
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/purchase-inbounds/{id}", h.get)
}
func RegisterPurchaseInboundPermissions(service permissionsvc.Service) error {
	if service == nil {
		return nil
	}
	for _, item := range []permissionsvc.RegisterResourceInput{{Key: PermissionInboundRead, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Read purchase inbounds", Risk: domainpermission.RiskLevelLow}, {Key: PermissionInboundCreate, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Create purchase inbounds", Risk: domainpermission.RiskLevelHigh}} {
		if _, err := service.RegisterResource(context.Background(), item); err != nil {
			return err
		}
	}
	return nil
}
func (h *PurchaseInboundHandler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context())
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}
func (h *PurchaseInboundHandler) get(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		apiv1.WriteError(w, http.StatusNotFound, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}
func (h *PurchaseInboundHandler) create(w http.ResponseWriter, r *http.Request) {
	var req purchaseInboundCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	item, err := h.service.Create(r.Context(), pharmaoasvc.PurchaseInboundCreateInput{Number: req.Number, PurchaseOrderID: req.PurchaseOrderID, WarehouseID: req.WarehouseID, AreaID: req.AreaID, LocationID: req.LocationID, Lines: req.Lines, Attachments: req.Attachments, ActorID: actorIDFromRequest(r, req.ActorID)})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusCreated, map[string]any{"item": item})
}
