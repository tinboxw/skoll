package pharmaoa

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
)

func TestProductHandlerCreateImportDisable(t *testing.T) {
	service := pharmaoasvc.NewProductService(nil)
	mux := http.NewServeMux()
	RegisterProductRoutes(mux, service)

	createBody := []byte(`{
		"code":"DRUG001",
		"name":"Amoxicillin",
		"spec":"0.25g*12",
		"dosageForm":"capsule",
		"manufacturer":"Skoll Pharma",
		"approvalNumber":"NMPA-H20260001",
		"temperature":{"required":true,"minCelsius":2,"maxCelsius":8},
		"actorId":"admin"
	}`)
	createResp := performRequest(mux, http.MethodPost, "/v1/plugins/pharma_oa/api/products", createBody)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", createResp.Code, createResp.Body.String())
	}
	var createPayload struct {
		Data struct {
			Item struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"item"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createResp.Body.Bytes(), &createPayload); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if createPayload.Data.Item.ID == "" || createPayload.Data.Item.Status != "active" {
		t.Fatalf("unexpected create response: %+v", createPayload.Data.Item)
	}

	listResp := performRequest(mux, http.MethodGet, "/v1/plugins/pharma_oa/api/products?keyword=amoxicillin", nil)
	if listResp.Code != http.StatusOK || !bytes.Contains(listResp.Body.Bytes(), []byte("DRUG001")) {
		t.Fatalf("list response invalid: status=%d body=%s", listResp.Code, listResp.Body.String())
	}

	importResp := performRequest(mux, http.MethodPost, "/v1/plugins/pharma_oa/api/products/import", []byte(`{"rows":[
		{"code":"DRUG002","name":"Ibuprofen","spec":"0.2g*24","dosageForm":"tablet","manufacturer":"Skoll Pharma","approvalNumber":"NMPA-H20260002"},
		{"code":"DRUG002","name":"Duplicate","spec":"0.2g*24","dosageForm":"tablet","manufacturer":"Skoll Pharma","approvalNumber":"NMPA-H20260002"}
	]}`))
	if importResp.Code != http.StatusOK || !bytes.Contains(importResp.Body.Bytes(), []byte(`"created":1`)) {
		t.Fatalf("import response invalid: status=%d body=%s", importResp.Code, importResp.Body.String())
	}

	disableResp := performRequest(mux, http.MethodPost, "/v1/plugins/pharma_oa/api/products/"+createPayload.Data.Item.ID+"/disable", []byte(`{"reason":"stopped","actorId":"admin"}`))
	if disableResp.Code != http.StatusOK || !bytes.Contains(disableResp.Body.Bytes(), []byte("disabled")) {
		t.Fatalf("disable response invalid: status=%d body=%s", disableResp.Code, disableResp.Body.String())
	}
}
