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

type MasterDataExchangeHandler struct {
	service pharmaoasvc.MasterDataExchangeService
}

type masterDataImportRequest struct {
	Resource string              `json:"resource"`
	Rows     []map[string]string `json:"rows"`
	ActorID  string              `json:"actorId"`
}

func RegisterMasterDataExchangeRoutes(mux *http.ServeMux, service pharmaoasvc.MasterDataExchangeService) {
	if mux == nil || service == nil {
		return
	}
	h := &MasterDataExchangeHandler{service: service}
	mux.HandleFunc("GET /v1/pharma-oa/master-data/template", h.template)
	mux.HandleFunc("POST /v1/pharma-oa/master-data/import", h.importRows)
	mux.HandleFunc("GET /v1/pharma-oa/master-data/export", h.export)
}

func RegisterMasterDataExchangePermissions(service permissionsvc.Service) error {
	if service == nil {
		return nil
	}
	for _, item := range []permissionsvc.RegisterResourceInput{
		{Key: "pharma_oa.master.template", Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Read pharma master data templates", Risk: domainpermission.RiskLevelLow},
		{Key: "pharma_oa.master.import", Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Import pharma master data", Risk: domainpermission.RiskLevelMedium},
		{Key: "pharma_oa.master.export", Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Export pharma master data", Risk: domainpermission.RiskLevelMedium},
	} {
		if _, err := service.RegisterResource(context.Background(), item); err != nil {
			return err
		}
	}
	return nil
}

func (h *MasterDataExchangeHandler) template(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.Template(r.URL.Query().Get("resource"))
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (h *MasterDataExchangeHandler) importRows(w http.ResponseWriter, r *http.Request) {
	var req masterDataImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	result, err := h.service.Import(r.Context(), pharmaoasvc.MasterDataImportInput{
		Resource: req.Resource,
		Rows:     req.Rows,
		ActorID:  actorIDFromRequest(r, req.ActorID),
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"result": result})
}

func (h *MasterDataExchangeHandler) export(w http.ResponseWriter, r *http.Request) {
	job, err := h.service.Export(r.Context(), pharmaoasvc.MasterDataExportInput{
		Resource: r.URL.Query().Get("resource"),
		ActorID:  actorIDFromRequest(r, ""),
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"job": job})
}
