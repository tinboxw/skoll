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

const (
	PermissionPaymentInvoiceRead  = "pharma_oa.payment_invoice.read"
	PermissionPaymentPlanCreate   = "pharma_oa.payment_plan.create"
	PermissionPaymentPlanReceive  = "pharma_oa.payment_plan.receive"
	PermissionInvoiceRecordCreate = "pharma_oa.invoice_record.create"
	PermissionInvoiceRecordVoid   = "pharma_oa.invoice_record.void"
	PermissionPaymentReminderRun  = "pharma_oa.payment_reminder.run"
)

type PaymentInvoiceHandler struct {
	service pharmaoasvc.PaymentInvoiceService
}

type paymentPlanCreateRequest struct {
	SalesOrderID string                             `json:"salesOrderId"`
	AmountCents  int64                              `json:"amountCents"`
	DueAt        string                             `json:"dueAt"`
	Note         string                             `json:"note"`
	Attachments  []domainpharma.FinancialAttachment `json:"attachments"`
	ActorID      string                             `json:"actorId"`
}

type paymentReceiveRequest struct {
	AmountCents int64                              `json:"amountCents"`
	PaidAt      string                             `json:"paidAt"`
	Reference   string                             `json:"reference"`
	Attachments []domainpharma.FinancialAttachment `json:"attachments"`
	ActorID     string                             `json:"actorId"`
}

type invoiceCreateRequest struct {
	Number       string                             `json:"number"`
	SalesOrderID string                             `json:"salesOrderId"`
	AmountCents  int64                              `json:"amountCents"`
	IssuedAt     string                             `json:"issuedAt"`
	Note         string                             `json:"note"`
	Attachments  []domainpharma.FinancialAttachment `json:"attachments"`
	ActorID      string                             `json:"actorId"`
}

func RegisterPaymentInvoiceRoutes(mux *http.ServeMux, service pharmaoasvc.PaymentInvoiceService) {
	if mux == nil || service == nil {
		return
	}
	h := &PaymentInvoiceHandler{service: service}
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/payment-plans", h.listPlans)
	mux.HandleFunc("POST /v1/plugins/pharma_oa/api/payment-plans", h.createPlan)
	mux.HandleFunc("POST /v1/plugins/pharma_oa/api/payment-plans/{id}/receive", h.receive)
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/invoice-records", h.listInvoices)
	mux.HandleFunc("POST /v1/plugins/pharma_oa/api/invoice-records", h.createInvoice)
	mux.HandleFunc("POST /v1/plugins/pharma_oa/api/invoice-records/{id}/void", h.voidInvoice)
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/payment-reminder-jobs", h.listJobs)
	mux.HandleFunc("POST /v1/plugins/pharma_oa/api/payment-reminder-jobs", h.runScan)
	mux.HandleFunc("POST /v1/plugins/pharma_oa/api/payment-reminder-jobs/{id}/retry", h.retryScan)
}

func RegisterPaymentInvoicePermissions(service permissionsvc.Service) error {
	if service == nil {
		return nil
	}
	for _, item := range []permissionsvc.RegisterResourceInput{
		{Key: PermissionPaymentInvoiceRead, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Read payment plans, invoices, and reminder jobs", Risk: domainpermission.RiskLevelLow},
		{Key: PermissionPaymentPlanCreate, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Create payment plans", Risk: domainpermission.RiskLevelMedium},
		{Key: PermissionPaymentPlanReceive, Type: domainpermission.ResourceTypeButton, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Record received payments", Risk: domainpermission.RiskLevelHigh},
		{Key: PermissionInvoiceRecordCreate, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Create invoice records", Risk: domainpermission.RiskLevelMedium},
		{Key: PermissionInvoiceRecordVoid, Type: domainpermission.ResourceTypeButton, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Void invoice records", Risk: domainpermission.RiskLevelHigh},
		{Key: PermissionPaymentReminderRun, Type: domainpermission.ResourceTypeButton, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Run or retry overdue payment reminders", Risk: domainpermission.RiskLevelHigh},
	} {
		if _, err := service.RegisterResource(context.Background(), item); err != nil {
			return err
		}
	}
	return nil
}

func (h *PaymentInvoiceHandler) listPlans(w http.ResponseWriter, r *http.Request) {
	actorID := paymentInvoiceActorID(r)
	items, err := h.service.ListPaymentPlans(r.Context(), pharmaoasvc.PaymentPlanListInput{Keyword: r.URL.Query().Get("keyword"), SalesOrderID: r.URL.Query().Get("salesOrderId"), Status: domainpharma.PaymentPlanStatus(r.URL.Query().Get("status")), ActorID: actorID, Scope: pharmaoasvc.PaymentFinanceAccessScope{OwnerID: actorID}})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *PaymentInvoiceHandler) createPlan(w http.ResponseWriter, r *http.Request) {
	var req paymentPlanCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	dueAt, err := parsePaymentDate(req.DueAt)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	actorID := paymentInvoiceActorID(r)
	item, err := h.service.CreatePaymentPlan(r.Context(), pharmaoasvc.PaymentPlanCreateInput{SalesOrderID: req.SalesOrderID, AmountCents: req.AmountCents, DueAt: dueAt, Note: req.Note, Attachments: req.Attachments, ActorID: actorID, Scope: pharmaoasvc.PaymentFinanceAccessScope{OwnerID: actorID}})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusCreated, map[string]any{"item": item})
}

func (h *PaymentInvoiceHandler) receive(w http.ResponseWriter, r *http.Request) {
	var req paymentReceiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	paidAt, err := parsePaymentDate(req.PaidAt)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	actorID := paymentInvoiceActorID(r)
	item, err := h.service.RecordPayment(r.Context(), r.PathValue("id"), pharmaoasvc.PaymentRecordInput{AmountCents: req.AmountCents, PaidAt: paidAt, Reference: req.Reference, Attachments: req.Attachments, ActorID: actorID, Scope: pharmaoasvc.PaymentFinanceAccessScope{OwnerID: actorID}})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (h *PaymentInvoiceHandler) listInvoices(w http.ResponseWriter, r *http.Request) {
	actorID := paymentInvoiceActorID(r)
	items, err := h.service.ListInvoices(r.Context(), pharmaoasvc.InvoiceListInput{Keyword: r.URL.Query().Get("keyword"), SalesOrderID: r.URL.Query().Get("salesOrderId"), Status: domainpharma.InvoiceRecordStatus(r.URL.Query().Get("status")), ActorID: actorID, Scope: pharmaoasvc.PaymentFinanceAccessScope{OwnerID: actorID}})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *PaymentInvoiceHandler) createInvoice(w http.ResponseWriter, r *http.Request) {
	var req invoiceCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	issuedAt, err := parsePaymentDate(req.IssuedAt)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	actorID := paymentInvoiceActorID(r)
	item, err := h.service.CreateInvoice(r.Context(), pharmaoasvc.InvoiceCreateInput{Number: req.Number, SalesOrderID: req.SalesOrderID, AmountCents: req.AmountCents, IssuedAt: issuedAt, Note: req.Note, Attachments: req.Attachments, ActorID: actorID, Scope: pharmaoasvc.PaymentFinanceAccessScope{OwnerID: actorID}})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusCreated, map[string]any{"item": item})
}

func (h *PaymentInvoiceHandler) voidInvoice(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Reason  string `json:"reason"`
		ActorID string `json:"actorId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	actorID := paymentInvoiceActorID(r)
	item, err := h.service.VoidInvoice(r.Context(), r.PathValue("id"), pharmaoasvc.InvoiceVoidInput{Reason: req.Reason, ActorID: actorID, Scope: pharmaoasvc.PaymentFinanceAccessScope{OwnerID: actorID}})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (h *PaymentInvoiceHandler) listJobs(w http.ResponseWriter, r *http.Request) {
	actorID := paymentInvoiceActorID(r)
	items, err := h.service.ListReminderJobs(r.Context(), pharmaoasvc.PaymentFinanceAccessScope{OwnerID: actorID}, actorID)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *PaymentInvoiceHandler) runScan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RecipientID string `json:"recipientId"`
		ActorID     string `json:"actorId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	actorID := paymentInvoiceActorID(r)
	item, err := h.service.RunOverdueScan(r.Context(), pharmaoasvc.PaymentOverdueScanInput{RecipientID: req.RecipientID, ActorID: actorID, Scope: pharmaoasvc.PaymentFinanceAccessScope{OwnerID: actorID}})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusCreated, map[string]any{"item": item})
}

func (h *PaymentInvoiceHandler) retryScan(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.RetryOverdueScan(r.Context(), r.PathValue("id"), paymentInvoiceActorID(r))
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

func parsePaymentDate(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if value, err := time.Parse(layout, raw); err == nil {
			return value.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("date must use RFC3339 or YYYY-MM-DD")
}

func paymentInvoiceActorID(r *http.Request) string {
	if r != nil {
		if claims, ok := security.JWTClaimsFromContext(r.Context()); ok && strings.TrimSpace(claims.Subject) != "" {
			return strings.TrimSpace(claims.Subject)
		}
	}
	return "system"
}
