package pharmaoa

import (
	"context"
	"encoding/json"
	"net/http"

	domainpermission "github.com/tinboxw/skoll/internal/domain/permission"
	apiv1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	permissionsvc "github.com/tinboxw/skoll/internal/service/permission"
	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
)

const (
	PermissionStocktakeRead    = "pharma_oa.stocktake.read"
	PermissionStocktakeCreate  = "pharma_oa.stocktake.create"
	PermissionStocktakeApprove = "pharma_oa.stocktake.approve"
	PermissionStocktakeReject  = "pharma_oa.stocktake.reject"
	PermissionTransferRead     = "pharma_oa.transfer.read"
	PermissionTransferCreate   = "pharma_oa.transfer.create"
)

type InventoryOperationHandler struct {
	service pharmaoasvc.InventoryOperationService
}
type stocktakeCreateRequest struct {
	Number         string `json:"number"`
	ProductID      string `json:"productId"`
	WarehouseID    string `json:"warehouseId"`
	AreaID         string `json:"areaId"`
	LocationID     string `json:"locationId"`
	BatchID        string `json:"batchId"`
	ActualQuantity int    `json:"actualQuantity"`
	Reason         string `json:"reason"`
	CreatorID      string `json:"creatorId"`
	ApproverID     string `json:"approverId"`
}
type inventoryApprovalRequest struct {
	ActorID string `json:"actorId"`
	Comment string `json:"comment"`
}
type transferCreateRequest struct {
	Number          string `json:"number"`
	ProductID       string `json:"productId"`
	BatchID         string `json:"batchId"`
	Quantity        int    `json:"quantity"`
	FromWarehouseID string `json:"fromWarehouseId"`
	FromAreaID      string `json:"fromAreaId"`
	FromLocationID  string `json:"fromLocationId"`
	ToWarehouseID   string `json:"toWarehouseId"`
	ToAreaID        string `json:"toAreaId"`
	ToLocationID    string `json:"toLocationId"`
	ActorID         string `json:"actorId"`
}

func RegisterInventoryOperationRoutes(mux *http.ServeMux, service pharmaoasvc.InventoryOperationService) {
	if mux == nil || service == nil {
		return
	}
	h := &InventoryOperationHandler{service: service}
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/stocktakes", h.listStocktakes)
	mux.HandleFunc("POST /v1/plugins/pharma_oa/api/stocktakes", h.createStocktake)
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/stocktakes/{id}", h.getStocktake)
	mux.HandleFunc("POST /v1/plugins/pharma_oa/api/stocktakes/{id}/approve", h.approveStocktake)
	mux.HandleFunc("POST /v1/plugins/pharma_oa/api/stocktakes/{id}/reject", h.rejectStocktake)
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/transfers", h.listTransfers)
	mux.HandleFunc("POST /v1/plugins/pharma_oa/api/transfers", h.createTransfer)
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/transfers/{id}", h.getTransfer)
}
func RegisterInventoryOperationPermissions(service permissionsvc.Service) error {
	if service == nil {
		return nil
	}
	items := []permissionsvc.RegisterResourceInput{{Key: PermissionStocktakeRead, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Read stocktakes", Risk: domainpermission.RiskLevelLow}, {Key: PermissionStocktakeCreate, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Create stocktakes", Risk: domainpermission.RiskLevelMedium}, {Key: PermissionStocktakeApprove, Type: domainpermission.ResourceTypeButton, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Approve stocktake differences", Risk: domainpermission.RiskLevelHigh}, {Key: PermissionStocktakeReject, Type: domainpermission.ResourceTypeButton, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Reject stocktake differences", Risk: domainpermission.RiskLevelHigh}, {Key: PermissionTransferRead, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Read transfers", Risk: domainpermission.RiskLevelLow}, {Key: PermissionTransferCreate, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Create transfers", Risk: domainpermission.RiskLevelHigh}}
	for _, item := range items {
		if _, err := service.RegisterResource(context.Background(), item); err != nil {
			return err
		}
	}
	return nil
}
func (h *InventoryOperationHandler) listStocktakes(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListStocktakes(r.Context())
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}
func (h *InventoryOperationHandler) getStocktake(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.GetStocktake(r.Context(), r.PathValue("id"))
	if err != nil {
		apiv1.WriteError(w, http.StatusNotFound, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}
func (h *InventoryOperationHandler) createStocktake(w http.ResponseWriter, r *http.Request) {
	var req stocktakeCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	item, err := h.service.CreateStocktake(r.Context(), pharmaoasvc.StocktakeOrderCreateInput{Number: req.Number, ProductID: req.ProductID, WarehouseID: req.WarehouseID, AreaID: req.AreaID, LocationID: req.LocationID, BatchID: req.BatchID, ActualQuantity: req.ActualQuantity, Reason: req.Reason, CreatorID: actorIDFromRequest(r, req.CreatorID), ApproverID: req.ApproverID})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusCreated, map[string]any{"item": item})
}
func (h *InventoryOperationHandler) approveStocktake(w http.ResponseWriter, r *http.Request) {
	h.stocktakeAction(w, r, true)
}
func (h *InventoryOperationHandler) rejectStocktake(w http.ResponseWriter, r *http.Request) {
	h.stocktakeAction(w, r, false)
}
func (h *InventoryOperationHandler) stocktakeAction(w http.ResponseWriter, r *http.Request, approve bool) {
	var req inventoryApprovalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	in := pharmaoasvc.InventoryApprovalInput{ActorID: actorIDFromRequest(r, req.ActorID), Comment: req.Comment}
	var item any
	var err error
	if approve {
		item, err = h.service.ApproveStocktake(r.Context(), r.PathValue("id"), in)
	} else {
		item, err = h.service.RejectStocktake(r.Context(), r.PathValue("id"), in)
	}
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}
func (h *InventoryOperationHandler) listTransfers(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListTransfers(r.Context())
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}
func (h *InventoryOperationHandler) getTransfer(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.GetTransfer(r.Context(), r.PathValue("id"))
	if err != nil {
		apiv1.WriteError(w, http.StatusNotFound, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}
func (h *InventoryOperationHandler) createTransfer(w http.ResponseWriter, r *http.Request) {
	var req transferCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	item, err := h.service.CreateTransfer(r.Context(), pharmaoasvc.TransferOrderCreateInput{Number: req.Number, ProductID: req.ProductID, BatchID: req.BatchID, Quantity: req.Quantity, FromWarehouseID: req.FromWarehouseID, FromAreaID: req.FromAreaID, FromLocationID: req.FromLocationID, ToWarehouseID: req.ToWarehouseID, ToAreaID: req.ToAreaID, ToLocationID: req.ToLocationID, ActorID: actorIDFromRequest(r, req.ActorID)})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusCreated, map[string]any{"item": item})
}
