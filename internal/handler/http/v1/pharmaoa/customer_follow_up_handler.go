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

type CustomerFollowUpHandler struct {
	service pharmaoasvc.CustomerFollowUpService
}

type customerFollowUpRequest struct {
	CustomerID  string                                    `json:"customerId"`
	ContactName string                                    `json:"contactName"`
	Channel     domainpharma.CustomerFollowUpChannel      `json:"channel"`
	ScheduledAt string                                    `json:"scheduledAt"`
	NextAction  string                                    `json:"nextAction"`
	Attachments []domainpharma.CustomerFollowUpAttachment `json:"attachments"`
	ActorID     string                                    `json:"actorId"`
}

func RegisterCustomerFollowUpRoutes(mux *http.ServeMux, service pharmaoasvc.CustomerFollowUpService) {
	if mux == nil || service == nil {
		return
	}
	h := &CustomerFollowUpHandler{service: service}
	mux.HandleFunc("GET /v1/pharma-oa/customer-follow-ups", h.list)
	mux.HandleFunc("POST /v1/pharma-oa/customer-follow-ups", h.create)
	mux.HandleFunc("PUT /v1/pharma-oa/customer-follow-ups/{id}", h.update)
	mux.HandleFunc("POST /v1/pharma-oa/customer-follow-ups/{id}/complete", h.complete)
	mux.HandleFunc("POST /v1/pharma-oa/customer-follow-ups/{id}/cancel", h.cancel)
}

func RegisterCustomerFollowUpPermissions(service permissionsvc.Service) error {
	if service == nil {
		return nil
	}
	for _, item := range []permissionsvc.RegisterResourceInput{
		{Key: "pharma_oa.customer_follow_up.read", Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Read customer follow-ups", Risk: domainpermission.RiskLevelLow},
		{Key: "pharma_oa.customer_follow_up.create", Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Plan customer follow-ups", Risk: domainpermission.RiskLevelMedium},
		{Key: "pharma_oa.customer_follow_up.update", Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Update customer follow-up plans", Risk: domainpermission.RiskLevelMedium},
		{Key: "pharma_oa.customer_follow_up.complete", Type: domainpermission.ResourceTypeButton, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Complete customer follow-ups", Risk: domainpermission.RiskLevelMedium},
		{Key: "pharma_oa.customer_follow_up.cancel", Type: domainpermission.ResourceTypeButton, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Cancel customer follow-ups", Risk: domainpermission.RiskLevelHigh},
	} {
		if _, err := service.RegisterResource(context.Background(), item); err != nil {
			return err
		}
	}
	return nil
}

func (h *CustomerFollowUpHandler) list(w http.ResponseWriter, r *http.Request) {
	actorID := customerFollowUpActorID(r)
	from, err := parseOptionalFollowUpTime(r.URL.Query().Get("from"))
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	to, err := parseOptionalFollowUpTime(r.URL.Query().Get("to"))
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	items, err := h.service.List(r.Context(), pharmaoasvc.CustomerFollowUpListInput{
		Keyword: r.URL.Query().Get("keyword"), CustomerID: r.URL.Query().Get("customerId"), Status: r.URL.Query().Get("status"), From: from, To: to,
		ActorID: actorID, Scope: pharmaoasvc.CustomerFollowUpAccessScope{OwnerID: actorID},
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *CustomerFollowUpHandler) create(w http.ResponseWriter, r *http.Request) {
	req, scheduledAt, ok := decodeCustomerFollowUpPlan(w, r)
	if !ok {
		return
	}
	actorID := customerFollowUpActorID(r)
	item, err := h.service.Create(r.Context(), pharmaoasvc.CustomerFollowUpCreateInput{CustomerID: req.CustomerID, ContactName: req.ContactName, Channel: req.Channel, ScheduledAt: scheduledAt, NextAction: req.NextAction, Attachments: req.Attachments, ActorID: actorID, Scope: pharmaoasvc.CustomerFollowUpAccessScope{OwnerID: actorID}})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusCreated, map[string]any{"item": item})
}

func (h *CustomerFollowUpHandler) update(w http.ResponseWriter, r *http.Request) {
	req, scheduledAt, ok := decodeCustomerFollowUpPlan(w, r)
	if !ok {
		return
	}
	actorID := customerFollowUpActorID(r)
	item, err := h.service.Update(r.Context(), r.PathValue("id"), pharmaoasvc.CustomerFollowUpPlanInput{ContactName: req.ContactName, Channel: req.Channel, ScheduledAt: scheduledAt, NextAction: req.NextAction, Attachments: req.Attachments, ActorID: actorID, Scope: pharmaoasvc.CustomerFollowUpAccessScope{OwnerID: actorID}})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (h *CustomerFollowUpHandler) complete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Summary     string                                    `json:"summary"`
		NextAction  string                                    `json:"nextAction"`
		Attachments []domainpharma.CustomerFollowUpAttachment `json:"attachments"`
		ActorID     string                                    `json:"actorId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	actorID := customerFollowUpActorID(r)
	item, err := h.service.Complete(r.Context(), r.PathValue("id"), pharmaoasvc.CustomerFollowUpCompleteInput{Summary: req.Summary, NextAction: req.NextAction, Attachments: req.Attachments, ActorID: actorID, Scope: pharmaoasvc.CustomerFollowUpAccessScope{OwnerID: actorID}})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (h *CustomerFollowUpHandler) cancel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Reason  string `json:"reason"`
		ActorID string `json:"actorId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	actorID := customerFollowUpActorID(r)
	item, err := h.service.Cancel(r.Context(), r.PathValue("id"), pharmaoasvc.CustomerFollowUpCancelInput{Reason: req.Reason, ActorID: actorID, Scope: pharmaoasvc.CustomerFollowUpAccessScope{OwnerID: actorID}})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

func decodeCustomerFollowUpPlan(w http.ResponseWriter, r *http.Request) (customerFollowUpRequest, time.Time, bool) {
	var req customerFollowUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return req, time.Time{}, false
	}
	scheduledAt, err := parseRequiredFollowUpTime(req.ScheduledAt)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return req, time.Time{}, false
	}
	return req, scheduledAt, true
}

func parseRequiredFollowUpTime(raw string) (time.Time, error) {
	value, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}, fmt.Errorf("scheduledAt must use RFC3339")
	}
	return value.UTC(), nil
}

func parseOptionalFollowUpTime(raw string) (time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return time.Time{}, nil
	}
	return parseRequiredFollowUpTime(raw)
}

func customerFollowUpActorID(r *http.Request) string {
	if r != nil {
		if claims, ok := security.JWTClaimsFromContext(r.Context()); ok && strings.TrimSpace(claims.Subject) != "" {
			return strings.TrimSpace(claims.Subject)
		}
	}
	return "system"
}
