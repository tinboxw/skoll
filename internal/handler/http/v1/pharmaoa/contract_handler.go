package pharmaoa

import (
	"context"
	"encoding/json"
	"fmt"
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

const (
	PermissionContractRead      = "pharma_oa.contract.read"
	PermissionContractCreate    = "pharma_oa.contract.create"
	PermissionContractApprove   = "pharma_oa.contract.approve"
	PermissionContractReject    = "pharma_oa.contract.reject"
	PermissionContractExpiryRun = "pharma_oa.contract.expiry.run"
)

type ContractHandler struct {
	service pharmaoasvc.ContractService
}

type contractCreateRequest struct {
	Number        string                         `json:"number"`
	Title         string                         `json:"title"`
	PartyType     domainpharma.ContractPartyType `json:"partyType"`
	PartyID       string                         `json:"partyId"`
	OwnerID       string                         `json:"ownerId"`
	ApproverID    string                         `json:"approverId"`
	Amount        float64                        `json:"amount"`
	Currency      string                         `json:"currency"`
	EffectiveAt   string                         `json:"effectiveAt"`
	ExpiresAt     string                         `json:"expiresAt"`
	AttachmentIDs []string                       `json:"attachmentIds"`
}

func RegisterContractRoutes(mux *http.ServeMux, service pharmaoasvc.ContractService) {
	if mux == nil || service == nil {
		return
	}
	h := &ContractHandler{service: service}
	mux.HandleFunc("GET /v1/pharma-oa/contracts", h.list)
	mux.HandleFunc("POST /v1/pharma-oa/contracts", h.create)
	mux.HandleFunc("GET /v1/pharma-oa/contracts/{id}", h.get)
	mux.HandleFunc("POST /v1/pharma-oa/contracts/{id}/approve", h.approve)
	mux.HandleFunc("POST /v1/pharma-oa/contracts/{id}/reject", h.reject)
	mux.HandleFunc("POST /v1/pharma-oa/contracts/expiry-scan", h.scanExpiry)
}

func RegisterContractPermissions(service permissionsvc.Service) error {
	if service == nil {
		return nil
	}
	items := []permissionsvc.RegisterResourceInput{
		{Key: PermissionContractRead, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Read contract archive", Risk: domainpermission.RiskLevelLow},
		{Key: PermissionContractCreate, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Create contract archive entries", Risk: domainpermission.RiskLevelMedium},
		{Key: PermissionContractApprove, Type: domainpermission.ResourceTypeButton, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Approve contracts", Risk: domainpermission.RiskLevelHigh},
		{Key: PermissionContractReject, Type: domainpermission.ResourceTypeButton, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Reject contracts", Risk: domainpermission.RiskLevelHigh},
		{Key: PermissionContractExpiryRun, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Run contract expiry scan", Risk: domainpermission.RiskLevelHigh},
	}
	for _, item := range items {
		if _, err := service.RegisterResource(context.Background(), item); err != nil {
			return err
		}
	}
	return nil
}

func (h *ContractHandler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context(), pharmaoasvc.ContractListInput{Keyword: r.URL.Query().Get("keyword"), PartyType: domainpharma.ContractPartyType(r.URL.Query().Get("partyType")), Status: domainpharma.ContractStatus(r.URL.Query().Get("status"))})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *ContractHandler) get(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		apiv1.WriteError(w, http.StatusNotFound, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (h *ContractHandler) create(w http.ResponseWriter, r *http.Request) {
	var req contractCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	effectiveAt, err := parseContractDate(req.EffectiveAt)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	expiresAt, err := parseContractDate(req.ExpiresAt)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	item, err := h.service.Create(r.Context(), pharmaoasvc.ContractCreateInput{Number: req.Number, Title: req.Title, PartyType: req.PartyType, PartyID: req.PartyID, OwnerID: contractActorIDFromRequest(r, req.OwnerID), ApproverID: req.ApproverID, Amount: req.Amount, Currency: req.Currency, EffectiveAt: effectiveAt, ExpiresAt: expiresAt, AttachmentIDs: req.AttachmentIDs})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusCreated, map[string]any{"item": item})
}

func (h *ContractHandler) approve(w http.ResponseWriter, r *http.Request) {
	h.action(w, r, true)
}

func (h *ContractHandler) reject(w http.ResponseWriter, r *http.Request) {
	h.action(w, r, false)
}

func (h *ContractHandler) action(w http.ResponseWriter, r *http.Request, approve bool) {
	var req struct {
		ActorID string `json:"actorId"`
		Comment string `json:"comment"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	input := pharmaoasvc.ContractActionInput{ActorID: contractActorIDFromRequest(r, req.ActorID), Comment: req.Comment}
	var (
		item *domainpharma.Contract
		err  error
	)
	if approve {
		item, err = h.service.Approve(r.Context(), r.PathValue("id"), input)
	} else {
		item, err = h.service.Reject(r.Context(), r.PathValue("id"), input)
	}
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (h *ContractHandler) scanExpiry(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Days    int    `json:"days"`
		ActorID string `json:"actorId"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if raw := strings.TrimSpace(r.URL.Query().Get("days")); raw != "" && req.Days == 0 {
		req.Days, _ = strconv.Atoi(raw)
	}
	result, err := h.service.ScanExpiry(r.Context(), pharmaoasvc.ContractExpiryScanInput{Days: req.Days, ActorID: contractActorIDFromRequest(r, req.ActorID)})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": result})
}

func parseContractDate(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if value, err := time.Parse(layout, raw); err == nil {
			return value.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("contract date must use RFC3339 or YYYY-MM-DD")
}

func contractActorIDFromRequest(r *http.Request, fallback string) string {
	if r != nil {
		if claims, ok := security.JWTClaimsFromContext(r.Context()); ok {
			if actorID := strings.TrimSpace(claims.Subject); actorID != "" {
				return actorID
			}
		}
	}
	return actorIDFromRequest(r, fallback)
}
