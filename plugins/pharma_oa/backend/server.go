package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginclient"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

const (
	pluginID = "pharma_oa"
	apiBase  = "/v1/plugins/pharma_oa/api"
)

type foundationContract struct {
	PluginID        string   `json:"pluginId"`
	ContractVersion string   `json:"contractVersion"`
	Modules         []string `json:"modules"`
	DocumentTypes   []string `json:"documentTypes"`
	Events          []string `json:"events"`
}

type eventEnvelope struct {
	DeliveryID string `json:"deliveryId"`
	PluginID   string `json:"pluginId"`
	Handler    string `json:"handler"`
	EventName  string `json:"eventName"`
}

type server struct {
	host  pluginsdk.HostServices
	now   func() time.Time
	newID func() string
}

var foundationEvents = map[string]string{
	"onApprovalCompleted":     "approval-completed",
	"onQualificationExpiring": "qualification-expiring",
	"onInventoryChanged":      "inventory-changed",
	"onQualityLotReleased":    "quality-lot-released",
	"onQualityRecallStarted":  "quality-recall-started",
}

func newHandler(host pluginsdk.HostServices) (http.Handler, error) {
	if host.PluginID != pluginID || host.Transactions == nil || host.DataScopes == nil || host.DataStore == nil || host.DocumentNumbers == nil || host.Files == nil || host.Audit == nil || host.Workflows == nil || host.Jobs == nil {
		return nil, errors.New("complete Pharma OA host services are required")
	}
	s := &server{host: host, now: time.Now, newID: employeeID}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("POST /_skoll/events", s.handleEvent)
	mux.HandleFunc("GET "+apiBase+"/meta", s.meta)
	mux.HandleFunc("GET "+apiBase+"/employees", s.listEmployees)
	mux.HandleFunc("POST "+apiBase+"/employees", s.createEmployee)
	mux.HandleFunc("PUT "+apiBase+"/employees/{id}", s.updateEmployee)
	mux.HandleFunc("POST "+apiBase+"/employees/{id}/leave", s.leaveEmployee)
	mux.HandleFunc("POST "+apiBase+"/employees/{id}/attachments", s.attachEmployeeFile)
	mux.HandleFunc("GET "+apiBase+"/employees/qualification-reminders", s.employeeQualificationReminders)
	for _, kind := range []string{"customer", "supplier"} {
		plural := kind + "s"
		mux.HandleFunc("GET "+apiBase+"/"+plural, s.listParties(kind))
		mux.HandleFunc("POST "+apiBase+"/"+plural, s.createParty(kind))
		mux.HandleFunc("PUT "+apiBase+"/"+plural+"/{id}", s.updateParty(kind))
		mux.HandleFunc("POST "+apiBase+"/"+plural+"/{id}/disable", s.changePartyStatus(kind, false))
		mux.HandleFunc("POST "+apiBase+"/"+plural+"/{id}/enable", s.changePartyStatus(kind, true))
	}
	mux.HandleFunc("GET "+apiBase+"/customers/{id}/sales-eligibility", s.customerSalesEligibility)
	mux.HandleFunc("GET "+apiBase+"/suppliers/{id}/purchase-eligibility", s.supplierPurchaseEligibility)
	for kind, plural := range map[string]string{"category": "categories", "unit": "units", "manufacturer": "manufacturers"} {
		mux.HandleFunc("GET "+apiBase+"/"+plural, s.listCatalogs(kind))
		mux.HandleFunc("POST "+apiBase+"/"+plural, s.createCatalog(kind))
		mux.HandleFunc("PUT "+apiBase+"/"+plural+"/{id}", s.updateCatalog(kind))
		mux.HandleFunc("POST "+apiBase+"/"+plural+"/{id}/disable", s.changeCatalogStatus(kind, false))
		mux.HandleFunc("POST "+apiBase+"/"+plural+"/{id}/enable", s.changeCatalogStatus(kind, true))
	}
	mux.HandleFunc("GET "+apiBase+"/products", s.listProducts)
	mux.HandleFunc("POST "+apiBase+"/products", s.createProduct)
	mux.HandleFunc("PUT "+apiBase+"/products/{id}", s.updateProduct)
	mux.HandleFunc("POST "+apiBase+"/products/{id}/disable", s.changeProductStatus(false))
	mux.HandleFunc("POST "+apiBase+"/products/{id}/enable", s.changeProductStatus(true))
	mux.HandleFunc("GET "+apiBase+"/products/{id}/sale-eligibility", s.productSaleEligibility)
	mux.HandleFunc("GET "+apiBase+"/manufacturers/{id}/supply-eligibility", s.manufacturerSupplyEligibility)
	mux.HandleFunc("GET "+apiBase+"/qualification-types", s.listQualificationTypes)
	mux.HandleFunc("POST "+apiBase+"/qualification-types", s.createQualificationType)
	mux.HandleFunc("PUT "+apiBase+"/qualification-types/{id}", s.updateQualificationType)
	mux.HandleFunc("POST "+apiBase+"/qualification-types/{id}/disable", s.changeQualificationTypeStatus(false))
	mux.HandleFunc("POST "+apiBase+"/qualification-types/{id}/enable", s.changeQualificationTypeStatus(true))
	mux.HandleFunc("GET "+apiBase+"/qualifications", s.listQualifications)
	mux.HandleFunc("POST "+apiBase+"/qualifications", s.createQualification)
	mux.HandleFunc("PUT "+apiBase+"/qualifications/{id}", s.updateQualification)
	mux.HandleFunc("POST "+apiBase+"/qualifications/{id}/submit", s.submitQualification)
	mux.HandleFunc("POST "+apiBase+"/qualifications/{id}/approve", s.approveQualification)
	mux.HandleFunc("POST "+apiBase+"/qualifications/{id}/reject", s.rejectQualification)
	mux.HandleFunc("POST "+apiBase+"/qualifications/{id}/revoke", s.revokeQualification)
	mux.HandleFunc("POST "+apiBase+"/qualifications/expiry-scan", s.scanQualificationExpiry)
	mux.HandleFunc("GET "+apiBase+"/oa-requests", s.listOARequests)
	mux.HandleFunc("GET "+apiBase+"/oa-requests/{id}", s.getOARequestHandler)
	mux.HandleFunc("POST "+apiBase+"/oa-requests", s.createOARequest)
	mux.HandleFunc("PUT "+apiBase+"/oa-requests/{id}", s.updateOARequest)
	mux.HandleFunc("POST "+apiBase+"/oa-requests/{id}/submit", s.submitOARequest)
	mux.HandleFunc("POST "+apiBase+"/oa-requests/{id}/approve", s.approveOARequest)
	mux.HandleFunc("POST "+apiBase+"/oa-requests/{id}/reject", s.rejectOARequest)
	mux.HandleFunc("POST "+apiBase+"/oa-requests/{id}/withdraw", s.withdrawOARequest)
	mux.HandleFunc("POST "+apiBase+"/oa-requests/{id}/cancel", s.cancelOARequest)
	mux.HandleFunc("POST "+apiBase+"/oa-requests/{id}/delegate", s.delegateOARequest)
	mux.HandleFunc("POST "+apiBase+"/oa-requests/{id}/attachments", s.attachOARequestFile)
	mux.HandleFunc("POST "+apiBase+"/oa-requests/{id}/comments", s.commentOARequest)
	mux.HandleFunc("POST "+apiBase+"/oa-requests/{id}/reminders", s.remindOARequest)
	mux.HandleFunc("GET "+apiBase+"/purchase-requests", s.listPurchaseRequests)
	mux.HandleFunc("POST "+apiBase+"/purchase-requests", s.createPurchaseRequest)
	mux.HandleFunc("GET "+apiBase+"/purchase-requests/{id}", s.getPurchaseRequestHandler)
	mux.HandleFunc("POST "+apiBase+"/purchase-requests/{id}/approve", s.approvePurchaseRequest)
	mux.HandleFunc("POST "+apiBase+"/purchase-requests/{id}/reject", s.rejectPurchaseRequest)
	mux.HandleFunc("GET "+apiBase+"/purchase-orders", s.listPurchaseOrders)
	mux.HandleFunc("GET "+apiBase+"/purchase-orders/{id}", s.getPurchaseOrderHandler)
	mux.HandleFunc("GET "+apiBase+"/purchase-inbounds", s.listPurchaseInbounds)
	mux.HandleFunc("POST "+apiBase+"/purchase-inbounds", s.createPurchaseInbound)
	mux.HandleFunc("GET "+apiBase+"/purchase-inbounds/{id}", s.getPurchaseInboundHandler)
	return mux, nil
}

func (s *server) health(w http.ResponseWriter, _ *http.Request) {
	writeOK(w, map[string]string{"pluginId": pluginID, "status": "ready"})
}

func (s *server) meta(w http.ResponseWriter, _ *http.Request) {
	writeOK(w, foundationContract{
		PluginID: pluginID, ContractVersion: "0.10.0",
		Modules:       []string{"workforce", "parties", "catalog", "qualifications", "office", "crm", "purchasing", "sales", "inventory", "quality", "finance", "analytics"},
		DocumentTypes: []string{"leave_request", "expense_request", "purchase_request", "purchase_order", "purchase_inbound", "sales_order", "sales_outbound", "stocktake", "stock_transfer", "quality_inspection", "drug_recall", "business_contract"},
		Events:        []string{"approval-completed", "qualification-expiring", "inventory-changed", "quality-lot-released", "quality-recall-started"},
	})
}

func (s *server) handleEvent(w http.ResponseWriter, r *http.Request) {
	var event eventEnvelope
	if !decodeJSON(w, r, &event) {
		return
	}
	wantEvent, ok := foundationEvents[strings.TrimSpace(event.Handler)]
	if !ok || event.PluginID != pluginID || event.EventName != wantEvent || strings.TrimSpace(event.DeliveryID) == "" {
		writeError(w, http.StatusBadRequest, "invalid_event", "event contract does not match the manifest")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) requestContext(r *http.Request) (context.Context, error) {
	value := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(value) < 8 || !strings.EqualFold(value[:7], "Bearer ") {
		return nil, newHTTPError(http.StatusUnauthorized, "unauthorized", "bearer token is required")
	}
	token := strings.TrimSpace(value[7:])
	if token == "" {
		return nil, newHTTPError(http.StatusUnauthorized, "unauthorized", "bearer token is required")
	}
	return pluginclient.WithUserToken(r.Context(), token), nil
}

func (s *server) transaction(ctx context.Context, fn func(context.Context) error) error {
	return s.host.Transactions.Within(ctx, func(tx pluginsdk.Transaction) error { return fn(tx.Context()) })
}

func (s *server) audit(ctx context.Context, action, resourceID string, risk pluginsdk.AuditRisk, detail map[string]any) error {
	resource := action
	if index := strings.LastIndex(action, "."); index > 0 {
		resource = action[:index]
	}
	_, err := s.host.Audit.Record(ctx, pluginsdk.AuditEntry{
		Action: action, Resource: resource, ResourceID: resourceID,
		Result: pluginsdk.AuditResultSuccess, Risk: risk, Detail: detail,
	})
	return err
}

func employeeID() string {
	return newEntityID("employee")
}

func newEntityID(prefix string) string {
	var raw [12]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return prefix + "-" + strconv.FormatInt(time.Now().UTC().UnixNano(), 36)
	}
	return prefix + "-" + hex.EncodeToString(raw[:])
}

type httpError struct {
	status  int
	code    string
	message string
}

func (e *httpError) Error() string { return e.message }

func newHTTPError(status int, code, message string) error {
	return &httpError{status: status, code: code, message: message}
}

func writeServiceError(w http.ResponseWriter, err error) {
	var transport *httpError
	if errors.As(err, &transport) {
		writeError(w, transport.status, transport.code, transport.message)
		return
	}
	var datastore *pluginsdk.DataStoreError
	if errors.As(err, &datastore) {
		status := http.StatusInternalServerError
		switch datastore.Code {
		case pluginsdk.DataStoreErrorInvalidRequest:
			status = http.StatusBadRequest
		case pluginsdk.DataStoreErrorForbidden:
			status = http.StatusForbidden
		case pluginsdk.DataStoreErrorNotFound:
			status = http.StatusNotFound
		case pluginsdk.DataStoreErrorConflict:
			status = http.StatusConflict
		case pluginsdk.DataStoreErrorLimitExceeded:
			status = http.StatusRequestEntityTooLarge
		case pluginsdk.DataStoreErrorUnsupported:
			status = http.StatusUnprocessableEntity
		case pluginsdk.DataStoreErrorUnavailable:
			status = http.StatusServiceUnavailable
		}
		writeError(w, status, string(datastore.Code), datastore.Message)
		return
	}
	var hostError *pluginclient.Error
	if errors.As(err, &hostError) {
		writeError(w, hostError.StatusCode, hostError.Code, hostError.Message)
		return
	}
	writeError(w, http.StatusInternalServerError, "internal_error", "request could not be completed")
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return false
	}
	return true
}

func writeOK(w http.ResponseWriter, value any) {
	writeJSON(w, http.StatusOK, map[string]any{"code": "ok", "message": "", "data": value})
}

func writeCreated(w http.ResponseWriter, value any) {
	writeJSON(w, http.StatusCreated, map[string]any{"code": "ok", "message": "", "data": value})
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"code": code, "message": message, "data": nil})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
