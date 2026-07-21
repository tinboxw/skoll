package pharmaoa

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	domainpermission "github.com/tinboxw/skoll/internal/domain/permission"
	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	apiv1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	permissionsvc "github.com/tinboxw/skoll/internal/service/permission"
	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
)

type SupplierHandler struct {
	service pharmaoasvc.SupplierService
}

type supplierRequest struct {
	Code           string                         `json:"code"`
	Name           string                         `json:"name"`
	Rating         int                            `json:"rating"`
	Contacts       []domainpharma.SupplierContact `json:"contacts"`
	Qualifications []supplierQualificationRequest `json:"qualifications"`
	ActorID        string                         `json:"actorId"`
}

type supplierQualificationRequest struct {
	ID          string                            `json:"id"`
	Name        string                            `json:"name"`
	Number      string                            `json:"number"`
	ExpiresAt   string                            `json:"expiresAt"`
	Attachments []domainpharma.SupplierAttachment `json:"attachments"`
}

func RegisterSupplierRoutes(mux *http.ServeMux, service pharmaoasvc.SupplierService) {
	if mux == nil || service == nil {
		return
	}
	h := &SupplierHandler{service: service}
	mux.HandleFunc("GET /v1/pharma-oa/suppliers", h.list)
	mux.HandleFunc("POST /v1/pharma-oa/suppliers", h.create)
	mux.HandleFunc("PUT /v1/pharma-oa/suppliers/{id}", h.update)
	mux.HandleFunc("POST /v1/pharma-oa/suppliers/{id}/disable", h.disable)
	mux.HandleFunc("GET /v1/pharma-oa/suppliers/{id}/purchase-eligibility", h.purchaseEligibility)
	mux.HandleFunc("GET /v1/pharma-oa/suppliers/qualification-reminders", h.qualificationReminders)
}

func RegisterSupplierPermissions(service permissionsvc.Service) error {
	if service == nil {
		return nil
	}
	for _, item := range []permissionsvc.RegisterResourceInput{
		{Key: "pharma_oa.supplier.read", Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Read pharma suppliers", Risk: domainpermission.RiskLevelLow},
		{Key: "pharma_oa.supplier.create", Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Create pharma suppliers", Risk: domainpermission.RiskLevelMedium},
		{Key: "pharma_oa.supplier.update", Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Update pharma suppliers", Risk: domainpermission.RiskLevelMedium},
		{Key: "pharma_oa.supplier.disable", Type: domainpermission.ResourceTypeButton, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Disable pharma suppliers", Risk: domainpermission.RiskLevelHigh},
		{Key: "pharma_oa.supplier.reminder", Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Read supplier qualification reminders", Risk: domainpermission.RiskLevelLow},
		{Key: "pharma_oa.supplier.purchase", Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Validate purchase supplier", Risk: domainpermission.RiskLevelHigh},
	} {
		if _, err := service.RegisterResource(context.Background(), item); err != nil {
			return err
		}
	}
	return nil
}

func (h *SupplierHandler) list(w http.ResponseWriter, r *http.Request) {
	pagination, err := parsePharmaListPagination(r)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	page, err := h.service.ListPage(r.Context(), pharmaoasvc.SupplierListInput{
		Keyword: r.URL.Query().Get("keyword"),
		Status:  r.URL.Query().Get("status"),
		Offset:  pagination.Offset,
		Limit:   pagination.Limit,
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	writePharmaListPage(w, page, pagination)
}

func (h *SupplierHandler) create(w http.ResponseWriter, r *http.Request) {
	input, ok := h.decodeSupplierWriteInput(w, r)
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

func (h *SupplierHandler) update(w http.ResponseWriter, r *http.Request) {
	input, ok := h.decodeSupplierWriteInput(w, r)
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

func (h *SupplierHandler) disable(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Reason  string `json:"reason"`
		ActorID string `json:"actorId"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	item, err := h.service.Disable(r.Context(), r.PathValue("id"), pharmaoasvc.SupplierDisableInput{
		Reason:  req.Reason,
		ActorID: actorIDFromRequest(r, req.ActorID),
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (h *SupplierHandler) qualificationReminders(w http.ResponseWriter, r *http.Request) {
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	items, err := h.service.QualificationReminders(r.Context(), days)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *SupplierHandler) purchaseEligibility(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.ValidatePurchaseSupplier(r.Context(), r.PathValue("id"))
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": result})
}

func (h *SupplierHandler) decodeSupplierWriteInput(w http.ResponseWriter, r *http.Request) (pharmaoasvc.SupplierWriteInput, bool) {
	var req supplierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return pharmaoasvc.SupplierWriteInput{}, false
	}
	qualifications, err := parseSupplierQualifications(req.Qualifications)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return pharmaoasvc.SupplierWriteInput{}, false
	}
	return pharmaoasvc.SupplierWriteInput{
		Code:           req.Code,
		Name:           req.Name,
		Rating:         req.Rating,
		Contacts:       req.Contacts,
		Qualifications: qualifications,
		ActorID:        actorIDFromRequest(r, req.ActorID),
	}, true
}

func parseSupplierQualifications(items []supplierQualificationRequest) ([]domainpharma.SupplierQualification, error) {
	out := make([]domainpharma.SupplierQualification, 0, len(items))
	for _, item := range items {
		var expiresAt time.Time
		if raw := strings.TrimSpace(item.ExpiresAt); raw != "" {
			parsed, err := time.Parse(time.RFC3339, raw)
			if err != nil {
				parsed, err = time.Parse("2006-01-02", raw)
				if err != nil {
					return nil, err
				}
			}
			expiresAt = parsed.UTC()
		}
		out = append(out, domainpharma.SupplierQualification{
			ID:          item.ID,
			Name:        item.Name,
			Number:      item.Number,
			ExpiresAt:   expiresAt,
			Attachments: item.Attachments,
		})
	}
	return out, nil
}
