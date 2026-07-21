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
	PermissionQualityComplaintRead    = "pharma_oa.quality_complaint.read"
	PermissionQualityComplaintCreate  = "pharma_oa.quality_complaint.create"
	PermissionQualityComplaintResolve = "pharma_oa.quality_complaint.resolve"
	PermissionQualityComplaintReject  = "pharma_oa.quality_complaint.reject"
)

type QualityComplaintHandler struct {
	service pharmaoasvc.QualityComplaintService
}

type qualityComplaintCreateRequest struct {
	Number        string   `json:"number"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	CustomerID    string   `json:"customerId"`
	ProductID     string   `json:"productId"`
	BatchID       string   `json:"batchId"`
	ReporterID    string   `json:"reporterId"`
	HandlerID     string   `json:"handlerId"`
	AttachmentIDs []string `json:"attachmentIds"`
}

func RegisterQualityComplaintRoutes(mux *http.ServeMux, service pharmaoasvc.QualityComplaintService) {
	if mux == nil || service == nil {
		return
	}
	h := &QualityComplaintHandler{service: service}
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/quality-complaints", h.list)
	mux.HandleFunc("POST /v1/plugins/pharma_oa/api/quality-complaints", h.create)
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/quality-complaints/batches", h.listBatches)
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/quality-complaints/{id}", h.get)
	mux.HandleFunc("POST /v1/plugins/pharma_oa/api/quality-complaints/{id}/resolve", h.resolve)
	mux.HandleFunc("POST /v1/plugins/pharma_oa/api/quality-complaints/{id}/reject", h.reject)
}

func RegisterQualityComplaintPermissions(service permissionsvc.Service) error {
	if service == nil {
		return nil
	}
	items := []permissionsvc.RegisterResourceInput{
		{Key: PermissionQualityComplaintRead, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Read quality complaints", Risk: domainpermission.RiskLevelLow},
		{Key: PermissionQualityComplaintCreate, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Register quality complaints", Risk: domainpermission.RiskLevelMedium},
		{Key: PermissionQualityComplaintResolve, Type: domainpermission.ResourceTypeButton, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Resolve quality complaints", Risk: domainpermission.RiskLevelHigh},
		{Key: PermissionQualityComplaintReject, Type: domainpermission.ResourceTypeButton, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Reject quality complaints", Risk: domainpermission.RiskLevelHigh},
	}
	for _, item := range items {
		if _, err := service.RegisterResource(context.Background(), item); err != nil {
			return err
		}
	}
	return nil
}

func (h *QualityComplaintHandler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context(), pharmaoasvc.QualityComplaintListInput{
		Keyword: r.URL.Query().Get("keyword"),
		Status:  domainpharma.QualityComplaintStatus(r.URL.Query().Get("status")),
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *QualityComplaintHandler) get(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		apiv1.WriteError(w, http.StatusNotFound, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (h *QualityComplaintHandler) listBatches(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListBatches(r.Context(), r.URL.Query().Get("productId"))
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *QualityComplaintHandler) create(w http.ResponseWriter, r *http.Request) {
	var req qualityComplaintCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	item, err := h.service.Create(r.Context(), pharmaoasvc.QualityComplaintCreateInput{
		Number:        req.Number,
		Title:         req.Title,
		Description:   req.Description,
		CustomerID:    req.CustomerID,
		ProductID:     req.ProductID,
		BatchID:       req.BatchID,
		ReporterID:    contractActorIDFromRequest(r, req.ReporterID),
		HandlerID:     req.HandlerID,
		AttachmentIDs: req.AttachmentIDs,
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusCreated, map[string]any{"item": item})
}

func (h *QualityComplaintHandler) resolve(w http.ResponseWriter, r *http.Request) {
	h.action(w, r, true)
}

func (h *QualityComplaintHandler) reject(w http.ResponseWriter, r *http.Request) {
	h.action(w, r, false)
}

func (h *QualityComplaintHandler) action(w http.ResponseWriter, r *http.Request, resolve bool) {
	var req struct {
		ActorID    string `json:"actorId"`
		Conclusion string `json:"conclusion"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	input := pharmaoasvc.QualityComplaintActionInput{
		ActorID:    contractActorIDFromRequest(r, req.ActorID),
		Conclusion: req.Conclusion,
	}
	var (
		item *domainpharma.QualityComplaint
		err  error
	)
	if resolve {
		item, err = h.service.Resolve(r.Context(), r.PathValue("id"), input)
	} else {
		item, err = h.service.Reject(r.Context(), r.PathValue("id"), input)
	}
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}
