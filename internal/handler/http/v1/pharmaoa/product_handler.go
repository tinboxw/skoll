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

type ProductHandler struct {
	service pharmaoasvc.ProductService
}

type productRequest struct {
	Code           string                          `json:"code"`
	Name           string                          `json:"name"`
	Spec           string                          `json:"spec"`
	DosageForm     string                          `json:"dosageForm"`
	Manufacturer   string                          `json:"manufacturer"`
	ApprovalNumber string                          `json:"approvalNumber"`
	Temperature    domainpharma.ProductTemperature `json:"temperature"`
	ActorID        string                          `json:"actorId"`
}

func RegisterProductRoutes(mux *http.ServeMux, service pharmaoasvc.ProductService) {
	if mux == nil || service == nil {
		return
	}
	h := &ProductHandler{service: service}
	mux.HandleFunc("GET /v1/pharma-oa/products", h.list)
	mux.HandleFunc("POST /v1/pharma-oa/products", h.create)
	mux.HandleFunc("PUT /v1/pharma-oa/products/{id}", h.update)
	mux.HandleFunc("POST /v1/pharma-oa/products/{id}/disable", h.disable)
	mux.HandleFunc("POST /v1/pharma-oa/products/import", h.importRows)
}

func RegisterProductPermissions(service permissionsvc.Service) error {
	if service == nil {
		return nil
	}
	for _, item := range []permissionsvc.RegisterResourceInput{
		{Key: "pharma_oa.product.read", Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Read pharma products", Risk: domainpermission.RiskLevelLow},
		{Key: "pharma_oa.product.create", Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Create pharma products", Risk: domainpermission.RiskLevelMedium},
		{Key: "pharma_oa.product.update", Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Update pharma products", Risk: domainpermission.RiskLevelMedium},
		{Key: "pharma_oa.product.disable", Type: domainpermission.ResourceTypeButton, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Disable pharma products", Risk: domainpermission.RiskLevelHigh},
		{Key: "pharma_oa.product.import", Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Import pharma products", Risk: domainpermission.RiskLevelMedium},
	} {
		if _, err := service.RegisterResource(context.Background(), item); err != nil {
			return err
		}
	}
	return nil
}

func (h *ProductHandler) list(w http.ResponseWriter, r *http.Request) {
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.service.List(r.Context(), pharmaoasvc.ProductListInput{
		Keyword: r.URL.Query().Get("keyword"),
		Status:  r.URL.Query().Get("status"),
		Offset:  offset,
		Limit:   limit,
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "offset": offset, "limit": limit})
}

func (h *ProductHandler) create(w http.ResponseWriter, r *http.Request) {
	input, ok := h.decodeProductWriteInput(w, r)
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

func (h *ProductHandler) update(w http.ResponseWriter, r *http.Request) {
	input, ok := h.decodeProductWriteInput(w, r)
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

func (h *ProductHandler) disable(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Reason  string `json:"reason"`
		ActorID string `json:"actorId"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	item, err := h.service.Disable(r.Context(), r.PathValue("id"), pharmaoasvc.ProductDisableInput{
		Reason:  req.Reason,
		ActorID: actorIDFromRequest(r, req.ActorID),
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (h *ProductHandler) importRows(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Rows []productRequest `json:"rows"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	rows := make([]pharmaoasvc.ProductWriteInput, 0, len(req.Rows))
	for _, row := range req.Rows {
		rows = append(rows, productWriteInputFromRequest(row, actorIDFromRequest(r, row.ActorID)))
	}
	result, err := h.service.Import(r.Context(), rows)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"result": result})
}

func (h *ProductHandler) decodeProductWriteInput(w http.ResponseWriter, r *http.Request) (pharmaoasvc.ProductWriteInput, bool) {
	var req productRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return pharmaoasvc.ProductWriteInput{}, false
	}
	return productWriteInputFromRequest(req, actorIDFromRequest(r, req.ActorID)), true
}

func productWriteInputFromRequest(req productRequest, actorID string) pharmaoasvc.ProductWriteInput {
	return pharmaoasvc.ProductWriteInput{
		Code:           req.Code,
		Name:           req.Name,
		Spec:           req.Spec,
		DosageForm:     req.DosageForm,
		Manufacturer:   req.Manufacturer,
		ApprovalNumber: req.ApprovalNumber,
		Temperature:    req.Temperature,
		ActorID:        actorID,
	}
}
