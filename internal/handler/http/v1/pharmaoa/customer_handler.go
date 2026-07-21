package pharmaoa

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	domainpermission "github.com/tinboxw/skoll/internal/domain/permission"
	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	apiv1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	permissionsvc "github.com/tinboxw/skoll/internal/service/permission"
	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

type CustomerHandler struct {
	service       pharmaoasvc.CustomerService
	scopeResolver pluginsdk.DataScopeService
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
}

type customerQualificationRequest struct {
	ID          string                            `json:"id"`
	Name        string                            `json:"name"`
	Number      string                            `json:"number"`
	ExpiresAt   string                            `json:"expiresAt"`
	Attachments []domainpharma.CustomerAttachment `json:"attachments"`
}

func RegisterCustomerRoutes(mux *http.ServeMux, service pharmaoasvc.CustomerService, scopeResolver pluginsdk.DataScopeService) {
	if mux == nil || service == nil || scopeResolver == nil {
		return
	}
	h := &CustomerHandler{service: service, scopeResolver: scopeResolver}
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/customers", h.list)
	mux.HandleFunc("POST /v1/plugins/pharma_oa/api/customers", h.create)
	mux.HandleFunc("PUT /v1/plugins/pharma_oa/api/customers/{id}", h.update)
	mux.HandleFunc("POST /v1/plugins/pharma_oa/api/customers/{id}/disable", h.disable)
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/customers/{id}/sales-eligibility", h.salesEligibility)
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/customers/qualification-reminders", h.qualificationReminders)
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
	scope, ok := h.resolveScope(w, r, "read")
	if !ok {
		return
	}
	pagination, err := parsePharmaListPagination(r)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	page, err := h.service.ListPage(r.Context(), pharmaoasvc.CustomerListInput{
		Keyword: r.URL.Query().Get("keyword"),
		Status:  r.URL.Query().Get("status"),
		Region:  r.URL.Query().Get("region"),
		Offset:  pagination.Offset,
		Limit:   pagination.Limit,
		Scope:   scope,
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	writePharmaListPage(w, page, pagination)
}

func (h *CustomerHandler) create(w http.ResponseWriter, r *http.Request) {
	scope, ok := h.resolveScope(w, r, "create")
	if !ok {
		return
	}
	input, ok := h.decodeCustomerWriteInput(w, r)
	if !ok {
		return
	}
	input.Scope = scope
	item, err := h.service.Create(r.Context(), input)
	if err != nil {
		writeCustomerServiceError(w, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusCreated, map[string]any{"item": item})
}

func (h *CustomerHandler) update(w http.ResponseWriter, r *http.Request) {
	scope, ok := h.resolveScope(w, r, "update")
	if !ok {
		return
	}
	input, ok := h.decodeCustomerWriteInput(w, r)
	if !ok {
		return
	}
	input.Scope = scope
	item, err := h.service.Update(r.Context(), r.PathValue("id"), input)
	if err != nil {
		writeCustomerServiceError(w, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (h *CustomerHandler) disable(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Reason  string `json:"reason"`
		ActorID string `json:"actorId"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	scope, ok := h.resolveScope(w, r, "disable")
	if !ok {
		return
	}
	item, err := h.service.Disable(r.Context(), r.PathValue("id"), pharmaoasvc.CustomerDisableInput{
		Reason:  req.Reason,
		ActorID: actorIDFromRequest(r, req.ActorID),
		Scope:   scope,
	})
	if err != nil {
		writeCustomerServiceError(w, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (h *CustomerHandler) qualificationReminders(w http.ResponseWriter, r *http.Request) {
	scope, ok := h.resolveScope(w, r, "reminder")
	if !ok {
		return
	}
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	items, err := h.service.QualificationReminders(r.Context(), pharmaoasvc.CustomerReminderInput{
		Days:  days,
		Scope: scope,
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *CustomerHandler) salesEligibility(w http.ResponseWriter, r *http.Request) {
	scope, ok := h.resolveScope(w, r, "sales")
	if !ok {
		return
	}
	result, err := h.service.ValidateSalesCustomer(r.Context(), r.PathValue("id"), scope)
	if err != nil {
		writeCustomerServiceError(w, err)
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

func (h *CustomerHandler) resolveScope(w http.ResponseWriter, r *http.Request, action string) (pharmaoasvc.CustomerAccessScope, bool) {
	predicate, err := h.scopeResolver.Resolve(r.Context(), pluginsdk.Permission{Resource: "pharma_oa.customer", Action: action})
	if err != nil {
		apiv1.WriteError(w, http.StatusForbidden, err)
		return pharmaoasvc.CustomerAccessScope{}, false
	}
	scope := pharmaoasvc.CustomerAccessScope{
		ActorID:         predicate.SubjectID(),
		IncludeAll:      predicate.AllTenants() && predicate.AllOwners() && predicate.AllOrganizations(),
		OrganizationIDs: predicate.OrganizationIDs(),
	}
	if ownerIDs := predicate.OwnerIDs(); len(ownerIDs) > 0 {
		scope.OwnerID = strings.TrimSpace(ownerIDs[0])
	}
	return scope, true
}

func writeCustomerServiceError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	if errors.Is(err, pharmaoasvc.ErrCustomerAccessDenied) {
		status = http.StatusForbidden
	}
	apiv1.WriteError(w, status, err)
}
