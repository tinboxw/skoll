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

type EmployeeHandler struct {
	service pharmaoasvc.EmployeeService
}

type employeeRequest struct {
	Code         string               `json:"code"`
	Name         string               `json:"name"`
	DepartmentID string               `json:"departmentId"`
	PositionID   string               `json:"positionId"`
	Phone        string               `json:"phone"`
	Email        string               `json:"email"`
	Certificates []certificateRequest `json:"certificates"`
	ActorID      string               `json:"actorId"`
}

type certificateRequest struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Number    string `json:"number"`
	ExpiresAt string `json:"expiresAt"`
}

func RegisterEmployeeRoutes(mux *http.ServeMux, service pharmaoasvc.EmployeeService) {
	if mux == nil || service == nil {
		return
	}
	h := &EmployeeHandler{service: service}
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/employees", h.list)
	mux.HandleFunc("POST /v1/plugins/pharma_oa/api/employees", h.create)
	mux.HandleFunc("PUT /v1/plugins/pharma_oa/api/employees/{id}", h.update)
	mux.HandleFunc("POST /v1/plugins/pharma_oa/api/employees/{id}/leave", h.leave)
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/employees/qualification-reminders", h.qualificationReminders)
}

func RegisterEmployeePermissions(service permissionsvc.Service) error {
	if service == nil {
		return nil
	}
	for _, item := range []permissionsvc.RegisterResourceInput{
		{Key: "pharma_oa.employee.read", Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Read pharma employees", Risk: domainpermission.RiskLevelLow},
		{Key: "pharma_oa.employee.create", Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Create pharma employees", Risk: domainpermission.RiskLevelMedium},
		{Key: "pharma_oa.employee.update", Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Update pharma employees", Risk: domainpermission.RiskLevelMedium},
		{Key: "pharma_oa.employee.leave", Type: domainpermission.ResourceTypeButton, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Mark pharma employee left", Risk: domainpermission.RiskLevelHigh},
		{Key: "pharma_oa.employee.reminder", Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Read employee qualification reminders", Risk: domainpermission.RiskLevelLow},
	} {
		if _, err := service.RegisterResource(context.Background(), item); err != nil {
			return err
		}
	}
	return nil
}

func (h *EmployeeHandler) list(w http.ResponseWriter, r *http.Request) {
	pagination, err := parsePharmaListPagination(r)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	page, err := h.service.ListPage(r.Context(), pharmaoasvc.EmployeeListInput{
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

func normalizePagination(offset, limit int) (int, int) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	return offset, limit
}

func (h *EmployeeHandler) create(w http.ResponseWriter, r *http.Request) {
	input, ok := h.decodeWriteInput(w, r)
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

func (h *EmployeeHandler) update(w http.ResponseWriter, r *http.Request) {
	input, ok := h.decodeWriteInput(w, r)
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

func (h *EmployeeHandler) leave(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Reason  string `json:"reason"`
		ActorID string `json:"actorId"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	item, err := h.service.MarkLeft(r.Context(), r.PathValue("id"), pharmaoasvc.EmployeeLeaveInput{
		Reason:  req.Reason,
		ActorID: actorIDFromRequest(r, req.ActorID),
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (h *EmployeeHandler) qualificationReminders(w http.ResponseWriter, r *http.Request) {
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	items, err := h.service.QualificationReminders(r.Context(), days)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *EmployeeHandler) decodeWriteInput(w http.ResponseWriter, r *http.Request) (pharmaoasvc.EmployeeWriteInput, bool) {
	var req employeeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return pharmaoasvc.EmployeeWriteInput{}, false
	}
	certificates, err := parseCertificates(req.Certificates)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return pharmaoasvc.EmployeeWriteInput{}, false
	}
	return pharmaoasvc.EmployeeWriteInput{
		Code:         req.Code,
		Name:         req.Name,
		DepartmentID: req.DepartmentID,
		PositionID:   req.PositionID,
		Phone:        req.Phone,
		Email:        req.Email,
		Certificates: certificates,
		ActorID:      actorIDFromRequest(r, req.ActorID),
	}, true
}

func parseCertificates(items []certificateRequest) ([]domainpharma.EmployeeCertificate, error) {
	out := make([]domainpharma.EmployeeCertificate, 0, len(items))
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
		out = append(out, domainpharma.EmployeeCertificate{
			ID:        item.ID,
			Name:      item.Name,
			Number:    item.Number,
			ExpiresAt: expiresAt,
		})
	}
	return out, nil
}

func actorIDFromRequest(r *http.Request, raw string) string {
	if r != nil {
		if claims, ok := security.JWTClaimsFromContext(r.Context()); ok {
			if actorID := strings.TrimSpace(claims.Subject); actorID != "" {
				return actorID
			}
		}
	}
	if actorID := strings.TrimSpace(raw); actorID != "" {
		return actorID
	}
	return "system"
}
