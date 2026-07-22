package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFoundationEndpoints(t *testing.T) {
	handler := newHandler()
	health := httptest.NewRecorder()
	handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/health", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("health status=%d body=%s", health.Code, health.Body.String())
	}
	meta := httptest.NewRecorder()
	handler.ServeHTTP(meta, httptest.NewRequest(http.MethodGet, "/v1/plugins/pharma_oa/api/meta", nil))
	if meta.Code != http.StatusOK {
		t.Fatalf("meta status=%d body=%s", meta.Code, meta.Body.String())
	}
	var contract foundationContract
	if err := json.Unmarshal(meta.Body.Bytes(), &contract); err != nil {
		t.Fatal(err)
	}
	if contract.PluginID != pluginID || contract.ContractVersion != "0.2.0" || len(contract.Modules) != 12 || len(contract.DocumentTypes) != 12 || len(contract.Events) != 5 {
		t.Fatalf("unexpected foundation contract: %+v", contract)
	}
}

func TestFoundationEventContract(t *testing.T) {
	for handler, eventName := range foundationEvents {
		body, err := json.Marshal(eventEnvelope{DeliveryID: "delivery-1", PluginID: pluginID, Handler: handler, EventName: eventName})
		if err != nil {
			t.Fatal(err)
		}
		recorder := httptest.NewRecorder()
		newHandler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/_skoll/events", bytes.NewReader(body)))
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("%s status=%d body=%s", handler, recorder.Code, recorder.Body.String())
		}
	}

	recorder := httptest.NewRecorder()
	newHandler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/_skoll/events", bytes.NewBufferString(`{"deliveryId":"delivery-2","pluginId":"pharma_oa","handler":"privateHandler","eventName":"approval-completed"}`)))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unknown handler status=%d want=%d", recorder.Code, http.StatusBadRequest)
	}
}

func TestFoundationRejectsHostAndUnknownRoutes(t *testing.T) {
	for _, path := range []string{"/v1/pharma-oa/api/meta", "/v1/plugins/other/api/meta", "/v1/plugins/pharma_oa/api/employees"} {
		recorder := httptest.NewRecorder()
		newHandler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("%s status=%d want=%d", path, recorder.Code, http.StatusNotFound)
		}
	}
}
