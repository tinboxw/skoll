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

func TestCustomerHandlerScopeSalesAndDisable(t *testing.T) {
	service := pharmaoasvc.NewCustomerService(nil)
	mux := http.NewServeMux()
	RegisterCustomerRoutes(mux, service)

	createBody := map[string]any{
		"code":           "CUST-API-001",
		"name":           "East Hospital",
		"region":         "East",
		"organizationId": "org-a",
		"ownerId":        "sales-a",
		"contacts": []map[string]any{{
			"name":  "Alice",
			"phone": "10086",
		}},
		"qualifications": []map[string]any{{
			"name":      "Medical Institution License",
			"number":    "LIC-API-001",
			"expiresAt": "2026-08-01",
			"attachments": []map[string]any{{
				"fileId":   "file-1",
				"fileName": "license.pdf",
				"size":     128,
			}},
		}},
		"actorId": "sales-a",
	}
	createResp := performCustomerRequest(mux, http.MethodPost, "/v1/pharma-oa/customers", createBody)
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
		t.Fatalf("expected customer id in response: %s", createResp.Body.String())
	}

	listOwned := performCustomerRequest(mux, http.MethodGet, "/v1/pharma-oa/customers?ownerId=sales-a", nil)
	if listOwned.Code != http.StatusOK || !strings.Contains(listOwned.Body.String(), "CUST-API-001") {
		t.Fatalf("expected owner list to include customer, status=%d body=%s", listOwned.Code, listOwned.Body.String())
	}
	listOther := performCustomerRequest(mux, http.MethodGet, "/v1/pharma-oa/customers?ownerId=sales-b&organizationId=org-b", nil)
	if listOther.Code != http.StatusOK || strings.Contains(listOther.Body.String(), "CUST-API-001") {
		t.Fatalf("expected other owner/org isolation, status=%d body=%s", listOther.Code, listOther.Body.String())
	}

	updateBody := createBody
	updateBody["name"] = "East Hospital Updated"
	updateBody["scope"] = map[string]any{"ownerId": "sales-b", "organizationId": "org-b"}
	denied := performCustomerRequest(mux, http.MethodPut, "/v1/pharma-oa/customers/"+id, updateBody)
	if denied.Code != http.StatusBadRequest || !strings.Contains(denied.Body.String(), "access denied") {
		t.Fatalf("expected cross scope update denial, status=%d body=%s", denied.Code, denied.Body.String())
	}

	eligible := performCustomerRequest(mux, http.MethodGet, "/v1/pharma-oa/customers/"+id+"/sales-eligibility?ownerId=sales-a", nil)
	if eligible.Code != http.StatusOK || !strings.Contains(eligible.Body.String(), `"allowed":true`) {
		t.Fatalf("expected eligible customer, status=%d body=%s", eligible.Code, eligible.Body.String())
	}

	disable := performCustomerRequest(mux, http.MethodPost, "/v1/pharma-oa/customers/"+id+"/disable", map[string]any{
		"reason":  "blacklisted",
		"actorId": "sales-a",
		"scope":   map[string]any{"ownerId": "sales-a"},
	})
	if disable.Code != http.StatusOK {
		t.Fatalf("disable status=%d body=%s", disable.Code, disable.Body.String())
	}
	blocked := performCustomerRequest(mux, http.MethodGet, "/v1/pharma-oa/customers/"+id+"/sales-eligibility?ownerId=sales-a", nil)
	if blocked.Code != http.StatusOK || !strings.Contains(blocked.Body.String(), `"allowed":false`) || !strings.Contains(blocked.Body.String(), "disabled") {
		t.Fatalf("expected disabled customer blocked, status=%d body=%s", blocked.Code, blocked.Body.String())
	}
}

func TestCustomerHandlerRejectsUnsafeAttachment(t *testing.T) {
	service := pharmaoasvc.NewCustomerService(nil)
	mux := http.NewServeMux()
	RegisterCustomerRoutes(mux, service)

	resp := performCustomerRequest(mux, http.MethodPost, "/v1/pharma-oa/customers", map[string]any{
		"code":           "CUST-API-002",
		"name":           "Unsafe Hospital",
		"region":         "North",
		"organizationId": "org-a",
		"ownerId":        "sales-a",
		"contacts":       []map[string]any{{"name": "Alice"}},
		"qualifications": []map[string]any{{
			"name": "License",
			"attachments": []map[string]any{{
				"fileId":   "file-unsafe",
				"fileName": "..\\license.exe",
				"size":     1,
			}},
		}},
	})
	if resp.Code != http.StatusBadRequest || !strings.Contains(resp.Body.String(), "unsafe") {
		t.Fatalf("expected unsafe attachment rejection, status=%d body=%s", resp.Code, resp.Body.String())
	}
}

func performCustomerRequest(mux http.Handler, method string, path string, body any) *httptest.ResponseRecorder {
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
