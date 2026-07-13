package pharmaoa

import (
	"bytes"
	"context"
	"net/http"
	"testing"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
	"github.com/tinboxw/skoll/pkg/security"
)

type drugRecallHTTPSalesReader struct{ outbounds []*domainpharma.SalesOutbound }

func (f drugRecallHTTPSalesReader) ListOutbounds(context.Context) ([]*domainpharma.SalesOutbound, error) {
	return f.outbounds, nil
}

func TestDrugRecallHTTPCreateScopeAndCompleteTask(t *testing.T) {
	ctx := context.Background()
	customers := pharmaoasvc.NewCustomerService(nil)
	_, _ = customers.Create(ctx, pharmaoasvc.CustomerWriteInput{Code: "CUS-DRH", Name: "Recall Hospital", Region: "East", OrganizationID: "org-1", OwnerID: "owner-1", Contacts: []domainpharma.CustomerContact{{Name: "Alice"}}, Scope: pharmaoasvc.CustomerAccessScope{IncludeAll: true}})
	products := pharmaoasvc.NewProductService(nil)
	_, _ = products.Create(ctx, pharmaoasvc.ProductWriteInput{Code: "DRUG-DRH", Name: "Recall Drug", Spec: "10mg", DosageForm: "tablet", Manufacturer: "Skoll Pharma", ApprovalNumber: "APP-DRH"})
	inventory := pharmaoasvc.NewInventoryService(nil)
	_, _ = inventory.Inbound(ctx, pharmaoasvc.StockMovementInput{ReferenceID: "IN-DRH", ProductID: "pharma-product-1", WarehouseID: "warehouse-1", AreaID: "area-1", LocationID: "location-1", BatchNo: "BATCH-DRH", ProductionDate: time.Now().UTC().AddDate(0, -1, 0), ExpiresAt: time.Now().UTC().AddDate(1, 0, 0), Quantity: 20, ActorID: "warehouse-user"})
	sales := drugRecallHTTPSalesReader{outbounds: []*domainpharma.SalesOutbound{{ID: "sales-outbound-1", CustomerID: "pharma-customer-1", Lines: []domainpharma.SalesOutboundLine{{ProductID: "pharma-product-1", BatchID: "pharma-stock-batch-1", Quantity: 5}}}}}
	service := pharmaoasvc.NewDrugRecallService(sales, inventory, products, customers, nil, nil)
	mux := http.NewServeMux()
	RegisterDrugRecallRoutes(mux, service)

	batches := contractHTTPRequest(mux, http.MethodGet, "/v1/pharma-oa/drug-recalls/batches", "", security.JWTClaims{Subject: "quality-1", Role: "quality"})
	if batches.Code != http.StatusOK || !bytes.Contains(batches.Body.Bytes(), []byte(`"batchNo":"BATCH-DRH"`)) {
		t.Fatalf("batches status=%d body=%s", batches.Code, batches.Body.String())
	}
	scope := contractHTTPRequest(mux, http.MethodGet, "/v1/pharma-oa/drug-recalls/scope?batchId=pharma-stock-batch-1", "", security.JWTClaims{Subject: "quality-1", Role: "quality"})
	if scope.Code != http.StatusOK || !bytes.Contains(scope.Body.Bytes(), []byte(`"customerName":"Recall Hospital"`)) || !bytes.Contains(scope.Body.Bytes(), []byte(`"quantity":5`)) {
		t.Fatalf("scope status=%d body=%s", scope.Code, scope.Body.String())
	}
	create := contractHTTPRequest(mux, http.MethodPost, "/v1/pharma-oa/drug-recalls", `{"number":"DR-HTTP-1","title":"Recall batch","reason":"Quality deviation","batchId":"pharma-stock-batch-1","actorId":"spoofed"}`, security.JWTClaims{Subject: "quality-1", Role: "quality"})
	if create.Code != http.StatusCreated || !bytes.Contains(create.Body.Bytes(), []byte(`"initiatedBy":"quality-1"`)) || !bytes.Contains(create.Body.Bytes(), []byte(`"status":"active"`)) {
		t.Fatalf("create status=%d body=%s", create.Code, create.Body.String())
	}
	complete := contractHTTPRequest(mux, http.MethodPost, "/v1/pharma-oa/drug-recalls/drug-recall-1/tasks/drug-recall-1-task-1/complete", `{"actorId":"spoofed","note":"Customer returned all units"}`, security.JWTClaims{Subject: "handler-1", Role: "quality"})
	if complete.Code != http.StatusOK || !bytes.Contains(complete.Body.Bytes(), []byte(`"status":"completed"`)) || !bytes.Contains(complete.Body.Bytes(), []byte(`"completedBy":"handler-1"`)) {
		t.Fatalf("complete status=%d body=%s", complete.Code, complete.Body.String())
	}
	detail := contractHTTPRequest(mux, http.MethodGet, "/v1/pharma-oa/drug-recalls/drug-recall-1", "", security.JWTClaims{Subject: "quality-1", Role: "quality"})
	if detail.Code != http.StatusOK || !bytes.Contains(detail.Body.Bytes(), []byte(`"completionNote":"Customer returned all units"`)) {
		t.Fatalf("detail status=%d body=%s", detail.Code, detail.Body.String())
	}
	invalid := contractHTTPRequest(mux, http.MethodGet, "/v1/pharma-oa/drug-recalls?status=unknown", "", security.JWTClaims{Subject: "quality-1", Role: "quality"})
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid status=%d body=%s", invalid.Code, invalid.Body.String())
	}
}
