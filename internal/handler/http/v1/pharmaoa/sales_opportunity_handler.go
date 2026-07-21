package pharmaoa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	domainpermission "github.com/tinboxw/skoll/internal/domain/permission"
	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	apiv1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	permissionsvc "github.com/tinboxw/skoll/internal/service/permission"
	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
	"github.com/tinboxw/skoll/pkg/security"
)

type SalesOpportunityHandler struct {
	service pharmaoasvc.SalesOpportunityService
}

type salesOpportunityRequest struct {
	Title               string   `json:"title"`
	CustomerID          string   `json:"customerId"`
	ProductIDs          []string `json:"productIds"`
	ExpectedAmountCents int64    `json:"expectedAmountCents"`
	EstimatedCloseDate  string   `json:"estimatedCloseDate"`
	ActorID             string   `json:"actorId"`
}

func RegisterSalesOpportunityRoutes(mux *http.ServeMux, service pharmaoasvc.SalesOpportunityService) {
	if mux == nil || service == nil {
		return
	}
	h := &SalesOpportunityHandler{service: service}
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/sales-opportunities", h.list)
	mux.HandleFunc("POST /v1/plugins/pharma_oa/api/sales-opportunities", h.create)
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/sales-opportunities/statistics", h.statistics)
	mux.HandleFunc("PUT /v1/plugins/pharma_oa/api/sales-opportunities/{id}", h.update)
	mux.HandleFunc("POST /v1/plugins/pharma_oa/api/sales-opportunities/{id}/advance", h.advance)
}

func RegisterSalesOpportunityPermissions(service permissionsvc.Service) error {
	if service == nil {
		return nil
	}
	for _, item := range []permissionsvc.RegisterResourceInput{
		{Key: "pharma_oa.sales_opportunity.read", Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Read sales opportunities and funnel statistics", Risk: domainpermission.RiskLevelLow},
		{Key: "pharma_oa.sales_opportunity.create", Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Create sales opportunities", Risk: domainpermission.RiskLevelMedium},
		{Key: "pharma_oa.sales_opportunity.update", Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Update sales opportunities", Risk: domainpermission.RiskLevelMedium},
		{Key: "pharma_oa.sales_opportunity.advance", Type: domainpermission.ResourceTypeButton, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Advance or close sales opportunities", Risk: domainpermission.RiskLevelHigh},
	} {
		if _, err := service.RegisterResource(context.Background(), item); err != nil {
			return err
		}
	}
	return nil
}

func (h *SalesOpportunityHandler) list(w http.ResponseWriter, r *http.Request) {
	actorID := salesOpportunityActorID(r)
	items, err := h.service.List(r.Context(), pharmaoasvc.SalesOpportunityListInput{Keyword: r.URL.Query().Get("keyword"), CustomerID: r.URL.Query().Get("customerId"), Stage: r.URL.Query().Get("stage"), ActorID: actorID, Scope: pharmaoasvc.SalesOpportunityAccessScope{OwnerID: actorID}})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *SalesOpportunityHandler) statistics(w http.ResponseWriter, r *http.Request) {
	actorID := salesOpportunityActorID(r)
	item, err := h.service.Statistics(r.Context(), pharmaoasvc.SalesOpportunityStatisticsInput{ActorID: actorID, Scope: pharmaoasvc.SalesOpportunityAccessScope{OwnerID: actorID}})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (h *SalesOpportunityHandler) create(w http.ResponseWriter, r *http.Request) {
	req, closeDate, ok := decodeSalesOpportunity(w, r)
	if !ok {
		return
	}
	actorID := salesOpportunityActorID(r)
	item, err := h.service.Create(r.Context(), pharmaoasvc.SalesOpportunityCreateInput{Title: req.Title, CustomerID: req.CustomerID, ProductIDs: req.ProductIDs, ExpectedAmountCents: req.ExpectedAmountCents, EstimatedCloseDate: closeDate, ActorID: actorID, Scope: pharmaoasvc.SalesOpportunityAccessScope{OwnerID: actorID}})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusCreated, map[string]any{"item": item})
}

func (h *SalesOpportunityHandler) update(w http.ResponseWriter, r *http.Request) {
	req, closeDate, ok := decodeSalesOpportunity(w, r)
	if !ok {
		return
	}
	actorID := salesOpportunityActorID(r)
	item, err := h.service.Update(r.Context(), r.PathValue("id"), pharmaoasvc.SalesOpportunityUpdateInput{Title: req.Title, ProductIDs: req.ProductIDs, ExpectedAmountCents: req.ExpectedAmountCents, EstimatedCloseDate: closeDate, ActorID: actorID, Scope: pharmaoasvc.SalesOpportunityAccessScope{OwnerID: actorID}})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (h *SalesOpportunityHandler) advance(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Stage   domainpharma.SalesOpportunityStage `json:"stage"`
		Note    string                             `json:"note"`
		ActorID string                             `json:"actorId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	actorID := salesOpportunityActorID(r)
	item, err := h.service.Advance(r.Context(), r.PathValue("id"), pharmaoasvc.SalesOpportunityAdvanceInput{Stage: req.Stage, Note: req.Note, ActorID: actorID, Scope: pharmaoasvc.SalesOpportunityAccessScope{OwnerID: actorID}})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

func decodeSalesOpportunity(w http.ResponseWriter, r *http.Request) (salesOpportunityRequest, time.Time, bool) {
	var req salesOpportunityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return req, time.Time{}, false
	}
	closeDate, err := time.Parse(time.RFC3339, strings.TrimSpace(req.EstimatedCloseDate))
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, fmt.Errorf("estimatedCloseDate must use RFC3339"))
		return req, time.Time{}, false
	}
	return req, closeDate.UTC(), true
}

func salesOpportunityActorID(r *http.Request) string {
	if r != nil {
		if claims, ok := security.JWTClaimsFromContext(r.Context()); ok && strings.TrimSpace(claims.Subject) != "" {
			return strings.TrimSpace(claims.Subject)
		}
	}
	return "system"
}
