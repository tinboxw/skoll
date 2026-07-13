package pharmaoa

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
)

func TestPurchaseHandlerApprovalCreatesOrder(t *testing.T) {
	suppliers := pharmaoasvc.NewSupplierService(nil)
	supplier, err := suppliers.Create(context.Background(), pharmaoasvc.SupplierWriteInput{
		Code: "SUP-API-P", Name: "Purchase Supplier",
		Qualifications: []domainpharma.SupplierQualification{{ID: "q-1", Name: "License", Number: "L-1", ExpiresAt: time.Now().UTC().AddDate(1, 0, 0)}},
	})
	if err != nil {
		t.Fatalf("create supplier: %v", err)
	}
	service := pharmaoasvc.NewPurchaseService(suppliers, workflowsvc.NewService(workflowsvc.NewMemoryRepository()), nil)
	mux := http.NewServeMux()
	RegisterPurchaseRoutes(mux, service)
	created := performPurchaseRequest(mux, http.MethodPost, "/v1/pharma-oa/purchase-requests", map[string]any{
		"number": "PR-API-001", "supplierId": supplier.ID.String(), "requesterId": "buyer-1", "approverId": "manager-1",
		"lines": []map[string]any{{"productId": "product-1", "quantity": 3, "unitPrice": 8}},
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	var payload struct {
		Data struct {
			Item struct {
				ID string `json:"id"`
			} `json:"item"`
		} `json:"data"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	approved := performPurchaseRequest(mux, http.MethodPost, "/v1/pharma-oa/purchase-requests/"+payload.Data.Item.ID+"/approve", map[string]any{"actorId": "manager-1", "comment": "ok"})
	if approved.Code != http.StatusOK || !strings.Contains(approved.Body.String(), `"purchaseRequestId":"`+payload.Data.Item.ID+`"`) {
		t.Fatalf("approve status=%d body=%s", approved.Code, approved.Body.String())
	}
	orders := performPurchaseRequest(mux, http.MethodGet, "/v1/pharma-oa/purchase-orders", nil)
	if orders.Code != http.StatusOK || !strings.Contains(orders.Body.String(), "PO-API-001") {
		t.Fatalf("orders status=%d body=%s", orders.Code, orders.Body.String())
	}
}

func performPurchaseRequest(mux http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	var raw []byte
	if body != nil {
		raw, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}
