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
	"github.com/tinboxw/skoll/pkg/security"
)

type CustomerHandler struct {
	service pharmaoasvc.CustomerService
}

type customerRequest struct {
	Code           string                         `json:"code"`
	Name           string                         `json:"name"`
	Region         string                         `json:"region"`
	OrganizationID string                         `json:"organizationId"`
	OwnerID        string                         `json:"ownerId"`
	Rating         int                            `json:"rating"`
	Contacts       []domainpharma.CustomerContact `json:"contacts"`
	Qualifications []customerQualificationRequest `json:"qualifications"`
	ActorID        string                         `json:"actorId"`
	Scope          customerScopeRequest           `json:"scope"`
}

type customerScopeRequest struct {
	OwnerID        string `json:"ownerId"`
	OrganizationID string `json:"organizationId"`
	IncludeAll     bool   `json:"includeAll"`
}

type customerQualificationRequest struct {
	ID          string                            `json:"id"`
	Name        string                            `json:"name"`
	Number      string                            `json:"number"`
	ExpiresAt   string                            `json:"expiresAt"`
	Attachments []domainpharma.CustomerAttachment `json:"attachments"`
}

func RegisterCustomerRoutes(mux *http.ServeMux, service pharmaoasvc.CustomerService) {
	if mux == nil || service == nil {
		return
	}
	h := &CustomerHandler{service: service}
	mux.HandleFunc("GET /v1/pharma-oa/customers", h.list)
	mux.HandleFunc("POST /v1/pharma-oa/customers", h.create)
	mux.HandleFunc("PUT /v1/pharma-oa/customers/{id}", h.update)
	mux.HandleFunc("POST /v1/pharma-oa/customers/{id}/disable", h.disable)
	mux.HandleFunc("GET /v1/pharma-oa/customers/{id}/sales-eligibility", h.salesEligibility)
	mux.HandleFunc("GET /v1/pharma-oa/customers/qualification-reminders", h.qualificationReminders)
}

func RegisterCustomerPermissions(service permissionsvc.Service) error {
	if service == nil {
		return nil
	}
	for _, item := range []permissionsvc.RegisterResourceInput{
		{Key: "pharma_oa.customer.read", Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Read pharma customers", Risk: domainpermission.RiskLevelLow},
		{Key: "pharma_oa.customer.create", Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Create pharma customers", Risk: domainpermission.RiskLevelMedium},
		{Key: "pharma_oa.customer.update", Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Update pharma customers", Risk: domainpermission.RiskLevelMedium},
		{Key: "pharma_oa.customer.disable", Type: domainpermission.ResourceTypeButton, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Disable pharma customers", Risk: domainpermission.RiskLevelHigh},
		{Key: "pharma_oa.customer.reminder", Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Read customer qualification reminders", Risk: domainpermission.RiskLevelLow},
		{Key: "pharma_oa.customer.sales", Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Validate sales customer", Risk: domainpermission.RiskLevelHigh},
	} {
		if _, err := service.RegisterResource(context.Background(), item); err != nil {
			return err
		}
	}
	return nil
}

func (h *CustomerHandler) list(w http.ResponseWriter, r *http.Request) {
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, limit = normalizePagination(offset, limit)
	items, err := h.service.List(r.Context(), pharmaoasvc.CustomerListInput{
		Keyword: r.URL.Query().Get("keyword"),
		Status:  r.URL.Query().Get("status"),
		Region:  r.URL.Query().Get("region"),
		Offset:  offset,
		Limit:   limit,
		Scope:   customerScopeFromRequest(r, customerScopeRequest{}),
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "offset": offset, "limit": limit})
}

func (h *CustomerHandler) create(w http.ResponseWriter, r *http.Request) {
	input, ok := h.decodeCustomerWriteInput(w, r)
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

func (h *CustomerHandler) update(w http.ResponseWriter, r *http.Request) {
	input, ok := h.decodeCustomerWriteInput(w, r)
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

func (h *CustomerHandler) disable(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Reason  string               `json:"reason"`
		ActorID string               `json:"actorId"`
		Scope   customerScopeRequest `json:"scope"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	item, err := h.service.Disable(r.Context(), r.PathValue("id"), pharmaoasvc.CustomerDisableInput{
		Reason:  req.Reason,
		ActorID: actorIDFromRequest(r, req.ActorID),
		Scope:   customerScopeFromRequest(r, req.Scope),
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (h *CustomerHandler) qualificationReminders(w http.ResponseWriter, r *http.Request) {
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	items, err := h.service.QualificationReminders(r.Context(), pharmaoasvc.CustomerReminderInput{
		Days:  days,
		Scope: customerScopeFromRequest(r, customerScopeRequest{}),
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *CustomerHandler) salesEligibility(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.ValidateSalesCustomer(r.Context(), r.PathValue("id"), customerScopeFromRequest(r, customerScopeRequest{}))
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": result})
}

func (h *CustomerHandler) decodeCustomerWriteInput(w http.ResponseWriter, r *http.Request) (pharmaoasvc.CustomerWriteInput, bool) {
	var req customerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return pharmaoasvc.CustomerWriteInput{}, false
	}
	qualifications, err := parseCustomerQualifications(req.Qualifications)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return pharmaoasvc.CustomerWriteInput{}, false
	}
	actorID := actorIDFromRequest(r, req.ActorID)
	return pharmaoasvc.CustomerWriteInput{
		Code:           req.Code,
		Name:           req.Name,
		Region:         req.Region,
		OrganizationID: req.OrganizationID,
		OwnerID:        req.OwnerID,
		Rating:         req.Rating,
		Contacts:       req.Contacts,
		Qualifications: qualifications,
		ActorID:        actorID,
		Scope:          customerScopeFromRequest(r, req.Scope),
	}, true
}

func parseCustomerQualifications(items []customerQualificationRequest) ([]domainpharma.CustomerQualification, error) {
	out := make([]domainpharma.CustomerQualification, 0, len(items))
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
		out = append(out, domainpharma.CustomerQualification{
			ID:          item.ID,
			Name:        item.Name,
			Number:      item.Number,
			ExpiresAt:   expiresAt,
			Attachments: item.Attachments,
		})
	}
	return out, nil
}

func customerScopeFromRequest(r *http.Request, body customerScopeRequest) pharmaoasvc.CustomerAccessScope {
	if r != nil {
		if claims, ok := security.JWTClaimsFromContext(r.Context()); ok {
			if claims.HasRole("super_admin") {
				return pharmaoasvc.CustomerAccessScope{IncludeAll: true}
			}
			return pharmaoasvc.CustomerAccessScope{OwnerID: strings.TrimSpace(claims.Subject)}
		}
	}

	scope := pharmaoasvc.CustomerAccessScope{
		OwnerID:        body.OwnerID,
		OrganizationID: body.OrganizationID,
		IncludeAll:     body.IncludeAll,
	}
	if r != nil {
		query := r.URL.Query()
		if ownerID := strings.TrimSpace(query.Get("ownerId")); ownerID != "" {
			scope.OwnerID = ownerID
		}
		if organizationID := strings.TrimSpace(query.Get("organizationId")); organizationID != "" {
			scope.OrganizationID = organizationID
		}
		if includeAll, err := strconv.ParseBool(strings.TrimSpace(query.Get("includeAll"))); err == nil && includeAll {
			scope.IncludeAll = true
		}
		if scope.OwnerID == "" {
			scope.OwnerID = actorIDFromRequest(r, "")
		}
	}
	return scope
}
