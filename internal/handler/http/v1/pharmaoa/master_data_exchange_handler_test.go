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

func TestMasterDataExchangeHandlerTemplateImportExport(t *testing.T) {
	employees := pharmaoasvc.NewEmployeeService(nil)
	products := pharmaoasvc.NewProductService(nil)
	suppliers := pharmaoasvc.NewSupplierService(nil)
	customers := pharmaoasvc.NewCustomerService(nil)
	service := pharmaoasvc.NewMasterDataExchangeService(employees, products, suppliers, customers)
	mux := http.NewServeMux()
	RegisterMasterDataExchangeRoutes(mux, service)

	template := performMasterDataRequest(mux, http.MethodGet, "/v1/plugins/pharma_oa/api/master-data/template?resource=products", nil)
	if template.Code != http.StatusOK || !strings.Contains(template.Body.String(), "approvalNumber") {
		t.Fatalf("expected product template, status=%d body=%s", template.Code, template.Body.String())
	}

	importResp := performMasterDataRequest(mux, http.MethodPost, "/v1/plugins/pharma_oa/api/master-data/import", map[string]any{
		"resource": "products",
		"rows": []map[string]string{{
			"code":                  "DRUG-API-001",
			"name":                  "API Imported Drug",
			"spec":                  "20ml",
			"dosageForm":            "Injection",
			"manufacturer":          "Acme",
			"approvalNumber":        "NMPA-API-001",
			"temperatureRequired":   "true",
			"temperatureMinCelsius": "2",
			"temperatureMaxCelsius": "8",
		}},
		"actorId": "importer",
	})
	if importResp.Code != http.StatusOK || !strings.Contains(importResp.Body.String(), `"created":1`) {
		t.Fatalf("expected import success, status=%d body=%s", importResp.Code, importResp.Body.String())
	}

	exportResp := performMasterDataRequest(mux, http.MethodGet, "/v1/plugins/pharma_oa/api/master-data/export?resource=products", nil)
	if exportResp.Code != http.StatusOK || !strings.Contains(exportResp.Body.String(), "DRUG-API-001") || !strings.Contains(exportResp.Body.String(), `"status":"completed"`) {
		t.Fatalf("expected export job with imported product, status=%d body=%s", exportResp.Code, exportResp.Body.String())
	}
}

func TestMasterDataExchangeHandlerReportsInvalidRows(t *testing.T) {
	service := pharmaoasvc.NewMasterDataExchangeService(pharmaoasvc.NewEmployeeService(nil), pharmaoasvc.NewProductService(nil), pharmaoasvc.NewSupplierService(nil), pharmaoasvc.NewCustomerService(nil))
	mux := http.NewServeMux()
	RegisterMasterDataExchangeRoutes(mux, service)

	resp := performMasterDataRequest(mux, http.MethodPost, "/v1/plugins/pharma_oa/api/master-data/import", map[string]any{
		"resource": "employees",
		"rows": []map[string]string{{
			"code":         "EMP-API-BAD",
			"name":         "",
			"departmentId": "quality",
			"positionId":   "qa",
		}},
	})
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), `"row":1`) || !strings.Contains(resp.Body.String(), "name is required") {
		t.Fatalf("expected row-level error report, status=%d body=%s", resp.Code, resp.Body.String())
	}
}

func performMasterDataRequest(mux http.Handler, method string, path string, body any) *httptest.ResponseRecorder {
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
