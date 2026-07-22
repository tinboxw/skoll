package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

const pluginID = "pharma_oa"

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

var foundationEvents = map[string]string{
	"onApprovalCompleted":     "approval-completed",
	"onQualificationExpiring": "qualification-expiring",
	"onInventoryChanged":      "inventory-changed",
	"onQualityLotReleased":    "quality-lot-released",
	"onQualityRecallStarted":  "quality-recall-started",
}

func newHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"pluginId": pluginID, "status": "ready"})
	})
	mux.HandleFunc("POST /_skoll/events", handleEvent)
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/meta", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, foundationContract{
			PluginID: pluginID, ContractVersion: "0.2.0",
			Modules:       []string{"workforce", "parties", "catalog", "qualifications", "office", "crm", "purchasing", "sales", "inventory", "quality", "finance", "analytics"},
			DocumentTypes: []string{"leave_request", "expense_request", "purchase_request", "purchase_order", "purchase_inbound", "sales_order", "sales_outbound", "stocktake", "stock_transfer", "quality_inspection", "drug_recall", "business_contract"},
			Events:        []string{"approval-completed", "qualification-expiring", "inventory-changed", "quality-lot-released", "quality-recall-started"},
		})
	})
	return mux
}

func handleEvent(w http.ResponseWriter, r *http.Request) {
	var event eventEnvelope
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"code": "invalid_event"})
		return
	}
	wantEvent, ok := foundationEvents[strings.TrimSpace(event.Handler)]
	if !ok || event.PluginID != pluginID || event.EventName != wantEvent || strings.TrimSpace(event.DeliveryID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"code": "invalid_event"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
