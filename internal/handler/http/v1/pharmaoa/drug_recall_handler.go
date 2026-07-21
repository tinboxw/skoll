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
	PermissionDrugRecallRead     = "pharma_oa.drug_recall.read"
	PermissionDrugRecallCreate   = "pharma_oa.drug_recall.create"
	PermissionDrugRecallComplete = "pharma_oa.drug_recall.complete"
)

type DrugRecallHandler struct{ service pharmaoasvc.DrugRecallService }

type drugRecallCreateRequest struct {
	Number            string `json:"number"`
	Title             string `json:"title"`
	Reason            string `json:"reason"`
	BatchID           string `json:"batchId"`
	SourceComplaintID string `json:"sourceComplaintId"`
	ActorID           string `json:"actorId"`
}

func RegisterDrugRecallRoutes(mux *http.ServeMux, service pharmaoasvc.DrugRecallService) {
	if mux == nil || service == nil {
		return
	}
	h := &DrugRecallHandler{service: service}
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/drug-recalls", h.list)
	mux.HandleFunc("POST /v1/plugins/pharma_oa/api/drug-recalls", h.create)
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/drug-recalls/batches", h.listBatches)
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/drug-recalls/scope", h.previewScope)
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/drug-recalls/{id}", h.get)
	mux.HandleFunc("POST /v1/plugins/pharma_oa/api/drug-recalls/{id}/tasks/{taskId}/complete", h.completeTask)
}

func RegisterDrugRecallPermissions(service permissionsvc.Service) error {
	if service == nil {
		return nil
	}
	items := []permissionsvc.RegisterResourceInput{
		{Key: PermissionDrugRecallRead, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Read drug recalls", Risk: domainpermission.RiskLevelLow},
		{Key: PermissionDrugRecallCreate, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Create drug recalls", Risk: domainpermission.RiskLevelHigh},
		{Key: PermissionDrugRecallComplete, Type: domainpermission.ResourceTypeButton, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Complete drug recall tasks", Risk: domainpermission.RiskLevelHigh},
	}
	for _, item := range items {
		if _, err := service.RegisterResource(context.Background(), item); err != nil {
			return err
		}
	}
	return nil
}

func (h *DrugRecallHandler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context(), pharmaoasvc.DrugRecallListInput{Keyword: r.URL.Query().Get("keyword"), Status: domainpharma.DrugRecallStatus(r.URL.Query().Get("status"))})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *DrugRecallHandler) get(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		apiv1.WriteError(w, http.StatusNotFound, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (h *DrugRecallHandler) listBatches(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListBatches(r.Context())
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *DrugRecallHandler) previewScope(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.PreviewScope(r.Context(), r.URL.Query().Get("batchId"))
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *DrugRecallHandler) create(w http.ResponseWriter, r *http.Request) {
	var req drugRecallCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	item, err := h.service.Create(r.Context(), pharmaoasvc.DrugRecallCreateInput{Number: req.Number, Title: req.Title, Reason: req.Reason, BatchID: req.BatchID, SourceComplaintID: req.SourceComplaintID, ActorID: contractActorIDFromRequest(r, req.ActorID)})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusCreated, map[string]any{"item": item})
}

func (h *DrugRecallHandler) completeTask(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ActorID string `json:"actorId"`
		Note    string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	item, err := h.service.CompleteTask(r.Context(), r.PathValue("id"), r.PathValue("taskId"), pharmaoasvc.DrugRecallTaskCompleteInput{ActorID: contractActorIDFromRequest(r, req.ActorID), Note: req.Note})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}
