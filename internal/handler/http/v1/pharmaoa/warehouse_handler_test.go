package pharmaoa

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
)

func TestWarehouseHandlerMovementEligibilityAndDisable(t *testing.T) {
	service := pharmaoasvc.NewWarehouseService(nil)
	mux := http.NewServeMux()
	RegisterWarehouseRoutes(mux, service)

	createResp := performWarehouseRequest(mux, http.MethodPost, "/v1/plugins/pharma_oa/api/warehouses", map[string]any{
		"code":   "WH-API-001",
		"name":   "East Cold Warehouse",
		"region": "East",
		"temperature": map[string]any{
			"controlled": true,
			"minCelsius": 2,
			"maxCelsius": 8,
		},
		"areas": []map[string]any{{
			"id":     "area-cold",
			"code":   "A-COLD",
			"name":   "Cold Area",
			"status": "enabled",
			"locations": []map[string]any{{
				"id":     "loc-001",
				"code":   "L-001",
				"name":   "Shelf 001",
				"status": "enabled",
			}},
		}},
		"actorId": "qa-admin",
	})
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", createResp.Code, createResp.Body.String())
	}
	var createPayload struct {
		Data struct {
			Item struct {
				ID string `json:"id"`
			} `json:"item"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createResp.Body.Bytes(), &createPayload); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	id := createPayload.Data.Item.ID
	if id == "" {
		t.Fatalf("expected warehouse id in response: %s", createResp.Body.String())
	}

	listResp := performWarehouseRequest(mux, http.MethodGet, "/v1/plugins/pharma_oa/api/warehouses?region=East", nil)
	if listResp.Code != http.StatusOK || !strings.Contains(listResp.Body.String(), "WH-API-001") {
		t.Fatalf("expected list to include warehouse, status=%d body=%s", listResp.Code, listResp.Body.String())
	}

	eligible := performWarehouseRequest(mux, http.MethodGet, "/v1/plugins/pharma_oa/api/warehouses/"+id+"/movement-eligibility?areaId=area-cold&locationId=loc-001", nil)
	if eligible.Code != http.StatusOK || !strings.Contains(eligible.Body.String(), `"allowed":true`) {
		t.Fatalf("expected enabled location eligible, status=%d body=%s", eligible.Code, eligible.Body.String())
	}

	disable := performWarehouseRequest(mux, http.MethodPost, "/v1/plugins/pharma_oa/api/warehouses/"+id+"/disable", map[string]any{
		"reason":  "maintenance",
		"actorId": "qa-admin",
	})
	if disable.Code != http.StatusOK {
		t.Fatalf("disable status=%d body=%s", disable.Code, disable.Body.String())
	}
	blocked := performWarehouseRequest(mux, http.MethodGet, "/v1/plugins/pharma_oa/api/warehouses/"+id+"/movement-eligibility?areaId=area-cold&locationId=loc-001", nil)
	if blocked.Code != http.StatusOK || !strings.Contains(blocked.Body.String(), `"allowed":false`) || !strings.Contains(blocked.Body.String(), "warehouse is disabled") {
		t.Fatalf("expected disabled warehouse blocked, status=%d body=%s", blocked.Code, blocked.Body.String())
	}
}

func TestWarehouseHandlerRejectsInvalidTemperature(t *testing.T) {
	service := pharmaoasvc.NewWarehouseService(nil)
	mux := http.NewServeMux()
	RegisterWarehouseRoutes(mux, service)

	resp := performWarehouseRequest(mux, http.MethodPost, "/v1/plugins/pharma_oa/api/warehouses", map[string]any{
		"code":   "WH-API-002",
		"name":   "Invalid Temp Warehouse",
		"region": "East",
		"temperature": map[string]any{
			"controlled": true,
			"minCelsius": 10,
			"maxCelsius": 2,
		},
		"areas": []map[string]any{{
			"code": "A1",
			"name": "Area 1",
			"locations": []map[string]any{{
				"code": "L1",
				"name": "Loc 1",
			}},
		}},
	})
	if resp.Code != http.StatusBadRequest || !strings.Contains(resp.Body.String(), "minCelsius") {
		t.Fatalf("expected invalid temperature rejection, status=%d body=%s", resp.Code, resp.Body.String())
	}
}

func performWarehouseRequest(mux http.Handler, method string, path string, body any) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}
