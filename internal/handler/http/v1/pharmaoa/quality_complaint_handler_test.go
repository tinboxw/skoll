package pharmaoa

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	domainfile "github.com/tinboxw/skoll/internal/domain/file"
	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	filesvc "github.com/tinboxw/skoll/internal/service/file"
	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
	"github.com/tinboxw/skoll/pkg/security"
)

type qualityComplaintHTTPFileReader struct{}

func (qualityComplaintHTTPFileReader) Get(_ context.Context, in filesvc.GetInput) (*domainfile.FileObject, filesvc.AccessDecision, error) {
	if in.FileID != "quality-file-1" {
		return nil, filesvc.AccessDecision{}, fmt.Errorf("file not found")
	}
	return &domainfile.FileObject{ID: "quality-file-1", Name: "damage.jpg", MIME: "image/jpeg", Size: 1024, Status: domainfile.StatusAvailable}, filesvc.AccessDecision{Allowed: true}, nil
}

func TestQualityComplaintHTTPCreateResolveAndReject(t *testing.T) {
	ctx := context.Background()
	customers := pharmaoasvc.NewCustomerService(nil)
	_, _ = customers.Create(ctx, pharmaoasvc.CustomerWriteInput{Code: "CUS-QCH", Name: "Quality Hospital", Region: "East", OrganizationID: "org-1", OwnerID: "owner-1", Contacts: []domainpharma.CustomerContact{{Name: "Alice"}}, Scope: pharmaoasvc.CustomerAccessScope{IncludeAll: true}})
	products := pharmaoasvc.NewProductService(nil)
	_, _ = products.Create(ctx, pharmaoasvc.ProductWriteInput{Code: "DRUG-QCH", Name: "Quality Drug", Spec: "10mg", DosageForm: "tablet", Manufacturer: "Skoll Pharma", ApprovalNumber: "APP-QCH"})
	inventory := pharmaoasvc.NewInventoryService(nil)
	_, _ = inventory.Inbound(ctx, pharmaoasvc.StockMovementInput{ReferenceID: "IN-QCH", ProductID: "pharma-product-1", WarehouseID: "warehouse-1", AreaID: "area-1", LocationID: "location-1", BatchNo: "BATCH-QCH", ProductionDate: time.Now().UTC().AddDate(0, -1, 0), ExpiresAt: time.Now().UTC().AddDate(1, 0, 0), Quantity: 20, ActorID: "warehouse-user"})
	service := pharmaoasvc.NewQualityComplaintService(customers, products, inventory, workflowsvc.NewService(workflowsvc.NewMemoryRepository()), qualityComplaintHTTPFileReader{}, nil)
	mux := http.NewServeMux()
	RegisterQualityComplaintRoutes(mux, service)

	body := `{"number":"QC-HTTP-1","title":"Damaged package","description":"Package was damaged","customerId":"pharma-customer-1","productId":"pharma-product-1","batchId":"pharma-stock-batch-1","reporterId":"spoofed-reporter","handlerId":"handler-1","attachmentIds":["quality-file-1"]}`
	create := contractHTTPRequest(mux, http.MethodPost, "/v1/plugins/pharma_oa/api/quality-complaints", body, security.JWTClaims{Subject: "reporter-1", Role: "quality"})
	if create.Code != http.StatusCreated || !bytes.Contains(create.Body.Bytes(), []byte(`"reporterId":"reporter-1"`)) || !bytes.Contains(create.Body.Bytes(), []byte(`"workflowInstanceId":"quality-complaint-workflow-1"`)) {
		t.Fatalf("create status=%d body=%s", create.Code, create.Body.String())
	}
	batches := contractHTTPRequest(mux, http.MethodGet, "/v1/plugins/pharma_oa/api/quality-complaints/batches?productId=pharma-product-1", "", security.JWTClaims{Subject: "reporter-1", Role: "quality"})
	if batches.Code != http.StatusOK || !bytes.Contains(batches.Body.Bytes(), []byte(`"batchNo":"BATCH-QCH"`)) {
		t.Fatalf("batches status=%d body=%s", batches.Code, batches.Body.String())
	}
	spoofed := contractHTTPRequest(mux, http.MethodPost, "/v1/plugins/pharma_oa/api/quality-complaints/quality-complaint-1/resolve", `{"actorId":"handler-1","conclusion":"spoofed"}`, security.JWTClaims{Subject: "intruder-1", Role: "quality"})
	if spoofed.Code != http.StatusBadRequest {
		t.Fatalf("spoofed resolve status=%d body=%s", spoofed.Code, spoofed.Body.String())
	}
	resolve := contractHTTPRequest(mux, http.MethodPost, "/v1/plugins/pharma_oa/api/quality-complaints/quality-complaint-1/resolve", `{"conclusion":"Replacement shipped"}`, security.JWTClaims{Subject: "handler-1", Role: "quality"})
	if resolve.Code != http.StatusOK || !bytes.Contains(resolve.Body.Bytes(), []byte(`"status":"resolved"`)) || !bytes.Contains(resolve.Body.Bytes(), []byte(`"conclusion":"Replacement shipped"`)) {
		t.Fatalf("resolve status=%d body=%s", resolve.Code, resolve.Body.String())
	}

	body = `{"number":"QC-HTTP-2","title":"Unfounded complaint","description":"No quality deviation","customerId":"pharma-customer-1","productId":"pharma-product-1","batchId":"pharma-stock-batch-1","handlerId":"handler-1","attachmentIds":["quality-file-1"]}`
	create = contractHTTPRequest(mux, http.MethodPost, "/v1/plugins/pharma_oa/api/quality-complaints", body, security.JWTClaims{Subject: "reporter-1", Role: "quality"})
	if create.Code != http.StatusCreated {
		t.Fatalf("create rejectable status=%d body=%s", create.Code, create.Body.String())
	}
	reject := contractHTTPRequest(mux, http.MethodPost, "/v1/plugins/pharma_oa/api/quality-complaints/quality-complaint-2/reject", `{"conclusion":"No quality deviation found"}`, security.JWTClaims{Subject: "handler-1", Role: "quality"})
	if reject.Code != http.StatusOK || !bytes.Contains(reject.Body.Bytes(), []byte(`"status":"rejected"`)) {
		t.Fatalf("reject status=%d body=%s", reject.Code, reject.Body.String())
	}
	detail := contractHTTPRequest(mux, http.MethodGet, "/v1/plugins/pharma_oa/api/quality-complaints/quality-complaint-2", "", security.JWTClaims{Subject: "reporter-1", Role: "quality"})
	if detail.Code != http.StatusOK || !bytes.Contains(detail.Body.Bytes(), []byte(`"batchNo":"BATCH-QCH"`)) {
		t.Fatalf("detail status=%d body=%s", detail.Code, detail.Body.String())
	}
}
