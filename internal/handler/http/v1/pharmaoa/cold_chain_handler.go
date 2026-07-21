package pharmaoa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	domainpermission "github.com/tinboxw/skoll/internal/domain/permission"
	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	apiv1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	permissionsvc "github.com/tinboxw/skoll/internal/service/permission"
	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
)

const (
	PermissionColdChainRead   = "pharma_oa.cold_chain.read"
	PermissionColdChainCreate = "pharma_oa.cold_chain.create"
	PermissionColdChainRun    = "pharma_oa.cold_chain.run"
)

type ColdChainHandler struct{ service pharmaoasvc.ColdChainService }

type coldChainRecordCreateRequest struct {
	BalanceID          string    `json:"balanceId"`
	TemperatureCelsius float64   `json:"temperatureCelsius"`
	HumidityPercent    float64   `json:"humidityPercent"`
	Source             string    `json:"source"`
	RecordedAt         time.Time `json:"recordedAt"`
	ActorID            string    `json:"actorId"`
}

type coldChainRunRequest struct {
	MinHumidityPercent float64 `json:"minHumidityPercent"`
	MaxHumidityPercent float64 `json:"maxHumidityPercent"`
	RecipientID        string  `json:"recipientId"`
	ActorID            string  `json:"actorId"`
}

func RegisterColdChainRoutes(mux *http.ServeMux, service pharmaoasvc.ColdChainService) {
	if mux == nil || service == nil {
		return
	}
	h := &ColdChainHandler{service: service}
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/cold-chain-contexts", h.listContexts)
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/cold-chain-records", h.listRecords)
	mux.HandleFunc("POST /v1/plugins/pharma_oa/api/cold-chain-records", h.createRecord)
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/cold-chain-anomalies", h.listAnomalies)
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/cold-chain-jobs", h.listJobs)
	mux.HandleFunc("POST /v1/plugins/pharma_oa/api/cold-chain-jobs", h.run)
	mux.HandleFunc("POST /v1/plugins/pharma_oa/api/cold-chain-jobs/{id}/retry", h.retry)
}

func RegisterColdChainPermissions(service permissionsvc.Service) error {
	if service == nil {
		return nil
	}
	items := []permissionsvc.RegisterResourceInput{
		{Key: PermissionColdChainRead, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Read cold-chain records, anomalies, and jobs", Risk: domainpermission.RiskLevelLow},
		{Key: PermissionColdChainCreate, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Create cold-chain records", Risk: domainpermission.RiskLevelMedium},
		{Key: PermissionColdChainRun, Type: domainpermission.ResourceTypeButton, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Run or retry cold-chain anomaly scans", Risk: domainpermission.RiskLevelHigh},
	}
	for _, item := range items {
		if _, err := service.RegisterResource(context.Background(), item); err != nil {
			return err
		}
	}
	return nil
}

func (h *ColdChainHandler) listContexts(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListContexts(r.Context())
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *ColdChainHandler) listRecords(w http.ResponseWriter, r *http.Request) {
	limit, err := parseColdChainLimit(r.URL.Query().Get("limit"))
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	items, err := h.service.ListRecords(r.Context(), pharmaoasvc.ColdChainRecordListInput{BatchID: r.URL.Query().Get("batchId"), WarehouseID: r.URL.Query().Get("warehouseId"), Limit: limit})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *ColdChainHandler) createRecord(w http.ResponseWriter, r *http.Request) {
	var req coldChainRecordCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	item, err := h.service.CreateRecord(r.Context(), pharmaoasvc.ColdChainRecordCreateInput{BalanceID: req.BalanceID, TemperatureCelsius: req.TemperatureCelsius, HumidityPercent: req.HumidityPercent, Source: req.Source, RecordedAt: req.RecordedAt, ActorID: contractActorIDFromRequest(r, req.ActorID)})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusCreated, map[string]any{"item": item})
}

func (h *ColdChainHandler) listAnomalies(w http.ResponseWriter, r *http.Request) {
	activeOnly, err := strconv.ParseBool(r.URL.Query().Get("activeOnly"))
	if err != nil && r.URL.Query().Get("activeOnly") != "" {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	items, err := h.service.ListAnomalies(r.Context(), activeOnly)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *ColdChainHandler) listJobs(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListJobs(r.Context())
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *ColdChainHandler) run(w http.ResponseWriter, r *http.Request) {
	var req coldChainRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	actorID := contractActorIDFromRequest(r, req.ActorID)
	item, err := h.service.Run(r.Context(), domainpharma.ColdChainScanPolicy{MinHumidityPercent: req.MinHumidityPercent, MaxHumidityPercent: req.MaxHumidityPercent, RecipientID: req.RecipientID}, actorID)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusCreated, map[string]any{"item": item})
}

func (h *ColdChainHandler) retry(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ActorID string `json:"actorId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	item, err := h.service.Retry(r.Context(), r.PathValue("id"), contractActorIDFromRequest(r, req.ActorID))
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

func parseColdChainLimit(raw string) (int, error) {
	if raw == "" {
		return 100, nil
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit <= 0 || limit > 500 {
		return 0, fmt.Errorf("limit must be between 1 and 500")
	}
	return limit, nil
}
