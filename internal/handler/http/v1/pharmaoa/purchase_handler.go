package pharmaoa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	domainpermission "github.com/tinboxw/skoll/internal/domain/permission"
	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	apiv1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	permissionsvc "github.com/tinboxw/skoll/internal/service/permission"
	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
)

const (
	PermissionPurchaseRead      = "pharma_oa.purchase.read"
	PermissionPurchaseCreate    = "pharma_oa.purchase.create"
	PermissionPurchaseApprove   = "pharma_oa.purchase.approve"
	PermissionPurchaseReject    = "pharma_oa.purchase.reject"
	PermissionPurchaseOrderRead = "pharma_oa.order.read"
)

type PurchaseHandler struct{ service pharmaoasvc.PurchaseService }

type purchaseRequestCreateRequest struct {
	Number      string                      `json:"number"`
	SupplierID  string                      `json:"supplierId"`
	RequesterID string                      `json:"requesterId"`
	ApproverID  string                      `json:"approverId"`
	Reason      string                      `json:"reason"`
	Lines       []domainpharma.PurchaseLine `json:"lines"`
}

type purchaseApprovalRequest struct {
	ActorID string `json:"actorId"`
	Comment string `json:"comment"`
}

func RegisterPurchaseRoutes(mux *http.ServeMux, service pharmaoasvc.PurchaseService) {
	if mux == nil || service == nil {
		return
	}
	h := &PurchaseHandler{service: service}
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/purchase-requests", h.listRequests)
	mux.HandleFunc("POST /v1/plugins/pharma_oa/api/purchase-requests", h.createRequest)
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/purchase-requests/{id}", h.getRequest)
	mux.HandleFunc("POST /v1/plugins/pharma_oa/api/purchase-requests/{id}/approve", h.approveRequest)
	mux.HandleFunc("POST /v1/plugins/pharma_oa/api/purchase-requests/{id}/reject", h.rejectRequest)
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/purchase-orders", h.listOrders)
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/purchase-orders/{id}", h.getOrder)
}

func RegisterPurchasePermissions(service permissionsvc.Service) error {
	if service == nil {
		return nil
	}
	for _, item := range []permissionsvc.RegisterResourceInput{
		{Key: PermissionPurchaseRead, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Read purchase requests", Risk: domainpermission.RiskLevelLow},
		{Key: PermissionPurchaseCreate, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Create purchase requests", Risk: domainpermission.RiskLevelMedium},
		{Key: PermissionPurchaseApprove, Type: domainpermission.ResourceTypeButton, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Approve purchase requests", Risk: domainpermission.RiskLevelHigh},
		{Key: PermissionPurchaseReject, Type: domainpermission.ResourceTypeButton, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Reject purchase requests", Risk: domainpermission.RiskLevelHigh},
		{Key: PermissionPurchaseOrderRead, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Read purchase orders", Risk: domainpermission.RiskLevelLow},
	} {
		if _, err := service.RegisterResource(context.Background(), item); err != nil {
			return err
		}
	}
	return nil
}

func (h *PurchaseHandler) listRequests(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListRequests(r.Context())
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *PurchaseHandler) createRequest(w http.ResponseWriter, r *http.Request) {
	var req purchaseRequestCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	requesterID := actorIDFromRequest(r, req.RequesterID)
	item, err := h.service.CreateRequest(r.Context(), pharmaoasvc.PurchaseRequestCreateInput{
		Number: req.Number, SupplierID: req.SupplierID, RequesterID: requesterID, ApproverID: req.ApproverID, Reason: req.Reason, Lines: req.Lines,
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusCreated, map[string]any{"item": item})
}

func (h *PurchaseHandler) getRequest(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.GetRequest(r.Context(), r.PathValue("id"))
	if err != nil {
		apiv1.WriteError(w, http.StatusNotFound, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (h *PurchaseHandler) approveRequest(w http.ResponseWriter, r *http.Request) {
	req, ok := decodePurchaseApproval(w, r)
	if !ok {
		return
	}
	item, err := h.service.ApproveRequest(r.Context(), r.PathValue("id"), req)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (h *PurchaseHandler) rejectRequest(w http.ResponseWriter, r *http.Request) {
	req, ok := decodePurchaseApproval(w, r)
	if !ok {
		return
	}
	item, err := h.service.RejectRequest(r.Context(), r.PathValue("id"), req)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (h *PurchaseHandler) listOrders(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListOrders(r.Context())
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *PurchaseHandler) getOrder(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.GetOrder(r.Context(), r.PathValue("id"))
	if err != nil {
		apiv1.WriteError(w, http.StatusNotFound, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

func decodePurchaseApproval(w http.ResponseWriter, r *http.Request) (pharmaoasvc.PurchaseApprovalInput, bool) {
	var req purchaseApprovalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return pharmaoasvc.PurchaseApprovalInput{}, false
	}
	actorID := actorIDFromRequest(r, req.ActorID)
	if actorID == "" {
		apiv1.WriteError(w, http.StatusBadRequest, fmt.Errorf("actor id is required"))
		return pharmaoasvc.PurchaseApprovalInput{}, false
	}
	return pharmaoasvc.PurchaseApprovalInput{ActorID: actorID, Comment: req.Comment}, true
}
