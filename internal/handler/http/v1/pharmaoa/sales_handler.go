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
	PermissionSalesOrderRead      = "pharma_oa.sales.order.read"
	PermissionSalesOrderCreate    = "pharma_oa.sales.order.create"
	PermissionSalesOutboundRead   = "pharma_oa.sales.outbound.read"
	PermissionSalesOutboundCreate = "pharma_oa.sales.outbound.create"
)

type SalesHandler struct{ service pharmaoasvc.SalesService }
type salesOrderCreateRequest struct {
	Number     string                   `json:"number"`
	CustomerID string                   `json:"customerId"`
	Lines      []domainpharma.SalesLine `json:"lines"`
	ActorID    string                   `json:"actorId"`
}
type salesOutboundCreateRequest struct {
	Number       string                           `json:"number"`
	SalesOrderID string                           `json:"salesOrderId"`
	WarehouseID  string                           `json:"warehouseId"`
	AreaID       string                           `json:"areaId"`
	LocationID   string                           `json:"locationId"`
	Lines        []domainpharma.SalesOutboundLine `json:"lines"`
	ActorID      string                           `json:"actorId"`
}

func RegisterSalesRoutes(mux *http.ServeMux, service pharmaoasvc.SalesService) {
	if mux == nil || service == nil {
		return
	}
	h := &SalesHandler{service: service}
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/sales-orders", h.listOrders)
	mux.HandleFunc("POST /v1/plugins/pharma_oa/api/sales-orders", h.createOrder)
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/sales-orders/{id}", h.getOrder)
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/sales-outbounds", h.listOutbounds)
	mux.HandleFunc("POST /v1/plugins/pharma_oa/api/sales-outbounds", h.createOutbound)
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/sales-outbounds/{id}", h.getOutbound)
}

func RegisterSalesPermissions(service permissionsvc.Service) error {
	if service == nil {
		return nil
	}
	items := []permissionsvc.RegisterResourceInput{
		{Key: PermissionSalesOrderRead, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Read sales orders", Risk: domainpermission.RiskLevelLow},
		{Key: PermissionSalesOrderCreate, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Create sales orders", Risk: domainpermission.RiskLevelMedium},
		{Key: PermissionSalesOutboundRead, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Read sales outbounds", Risk: domainpermission.RiskLevelLow},
		{Key: PermissionSalesOutboundCreate, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Create sales outbounds", Risk: domainpermission.RiskLevelHigh},
	}
	for _, item := range items {
		if _, err := service.RegisterResource(context.Background(), item); err != nil {
			return err
		}
	}
	return nil
}

func (h *SalesHandler) listOrders(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListOrders(r.Context())
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}
func (h *SalesHandler) getOrder(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.GetOrder(r.Context(), r.PathValue("id"))
	if err != nil {
		apiv1.WriteError(w, http.StatusNotFound, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}
func (h *SalesHandler) createOrder(w http.ResponseWriter, r *http.Request) {
	var req salesOrderCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	item, err := h.service.CreateOrder(r.Context(), pharmaoasvc.SalesOrderCreateInput{Number: req.Number, CustomerID: req.CustomerID, Lines: req.Lines, ActorID: actorIDFromRequest(r, req.ActorID)})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusCreated, map[string]any{"item": item})
}
func (h *SalesHandler) listOutbounds(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListOutbounds(r.Context())
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}
func (h *SalesHandler) getOutbound(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.GetOutbound(r.Context(), r.PathValue("id"))
	if err != nil {
		apiv1.WriteError(w, http.StatusNotFound, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}
func (h *SalesHandler) createOutbound(w http.ResponseWriter, r *http.Request) {
	var req salesOutboundCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	item, err := h.service.CreateOutbound(r.Context(), pharmaoasvc.SalesOutboundCreateInput{Number: req.Number, SalesOrderID: req.SalesOrderID, WarehouseID: req.WarehouseID, AreaID: req.AreaID, LocationID: req.LocationID, Lines: req.Lines, ActorID: actorIDFromRequest(r, req.ActorID)})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusCreated, map[string]any{"item": item})
}
