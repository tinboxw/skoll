package pharmaoa

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	domainpermission "github.com/tinboxw/skoll/internal/domain/permission"
	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	apiv1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	permissionsvc "github.com/tinboxw/skoll/internal/service/permission"
	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
	"github.com/tinboxw/skoll/pkg/security"
)

const (
	PermissionAnnouncementRead        = "pharma_oa.announcement.read"
	PermissionAnnouncementCreate      = "pharma_oa.announcement.create"
	PermissionAnnouncementPublish     = "pharma_oa.announcement.publish"
	PermissionAnnouncementConfirm     = "pharma_oa.announcement.confirm"
	PermissionAnnouncementReceiptRead = "pharma_oa.announcement.receipt.read"
)

type AnnouncementHandler struct {
	service pharmaoasvc.AnnouncementService
}

type announcementCreateRequest struct {
	Kind      domainpharma.AnnouncementKind       `json:"kind"`
	Title     string                              `json:"title"`
	Content   string                              `json:"content"`
	Audience  domainpharma.AnnouncementAudience   `json:"audience"`
	Documents []domainpharma.AnnouncementDocument `json:"documents"`
	ActorID   string                              `json:"actorId"`
}

func RegisterAnnouncementRoutes(mux *http.ServeMux, service pharmaoasvc.AnnouncementService) {
	if mux == nil || service == nil {
		return
	}
	h := &AnnouncementHandler{service: service}
	mux.HandleFunc("GET /v1/pharma-oa/announcements", h.list)
	mux.HandleFunc("POST /v1/pharma-oa/announcements", h.create)
	mux.HandleFunc("POST /v1/pharma-oa/announcements/{id}/publish", h.publish)
	mux.HandleFunc("POST /v1/pharma-oa/announcements/{id}/read", h.confirmRead)
	mux.HandleFunc("GET /v1/pharma-oa/announcements/{id}/read-confirmations", h.listReadConfirmations)
}

func RegisterAnnouncementPermissions(service permissionsvc.Service) error {
	if service == nil {
		return nil
	}
	items := []permissionsvc.RegisterResourceInput{
		{Key: PermissionAnnouncementRead, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Read announcements", Risk: domainpermission.RiskLevelLow},
		{Key: PermissionAnnouncementCreate, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Create announcement drafts", Risk: domainpermission.RiskLevelMedium},
		{Key: PermissionAnnouncementPublish, Type: domainpermission.ResourceTypeButton, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Publish announcements", Risk: domainpermission.RiskLevelHigh},
		{Key: PermissionAnnouncementConfirm, Type: domainpermission.ResourceTypeButton, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Confirm announcement read", Risk: domainpermission.RiskLevelLow},
		{Key: PermissionAnnouncementReceiptRead, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Read announcement confirmations", Risk: domainpermission.RiskLevelMedium},
	}
	for _, item := range items {
		if _, err := service.RegisterResource(context.Background(), item); err != nil {
			return err
		}
	}
	return nil
}

func (h *AnnouncementHandler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context(), announcementListInputFromRequest(r))
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *AnnouncementHandler) create(w http.ResponseWriter, r *http.Request) {
	var req announcementCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	item, err := h.service.Create(r.Context(), pharmaoasvc.AnnouncementCreateInput{Kind: req.Kind, Title: req.Title, Content: req.Content, Audience: req.Audience, Documents: req.Documents, ActorID: actorIDFromRequest(r, req.ActorID)})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusCreated, map[string]any{"item": item})
}

func (h *AnnouncementHandler) publish(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ActorID string `json:"actorId"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	item, err := h.service.Publish(r.Context(), r.PathValue("id"), actorIDFromRequest(r, req.ActorID))
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (h *AnnouncementHandler) confirmRead(w http.ResponseWriter, r *http.Request) {
	query := announcementListInputFromRequest(r)
	receipt, err := h.service.ConfirmRead(r.Context(), r.PathValue("id"), query.ActorID, query.OrganizationIDs, query.RoleIDs)
	if err != nil {
		apiv1.WriteError(w, http.StatusForbidden, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": receipt})
}

func (h *AnnouncementHandler) listReadConfirmations(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListReadConfirmations(r.Context(), r.PathValue("id"))
	if err != nil {
		apiv1.WriteError(w, http.StatusNotFound, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func announcementListInputFromRequest(r *http.Request) pharmaoasvc.AnnouncementListInput {
	input := pharmaoasvc.AnnouncementListInput{ActorID: actorIDFromRequest(r, "")}
	if r == nil {
		return input
	}
	input.OrganizationIDs = cleanAnnouncementQueryValues(r.URL.Query()["organizationId"])
	input.RoleIDs = cleanAnnouncementQueryValues(r.URL.Query()["roleId"])
	if claims, ok := security.JWTClaimsFromContext(r.Context()); ok && strings.TrimSpace(claims.Role) != "" {
		input.RoleIDs = append(input.RoleIDs, strings.TrimSpace(claims.Role))
	}
	input.IncludeDraft, _ = strconv.ParseBool(r.URL.Query().Get("includeDraft"))
	return input
}

func cleanAnnouncementQueryValues(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			if part = strings.TrimSpace(part); part != "" {
				out = append(out, part)
			}
		}
	}
	return out
}
