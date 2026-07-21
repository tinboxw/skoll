package pharmaoa

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	domainfile "github.com/tinboxw/skoll/internal/domain/file"
	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	filesvc "github.com/tinboxw/skoll/internal/service/file"
	notificationsvc "github.com/tinboxw/skoll/internal/service/notification"
	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
	"github.com/tinboxw/skoll/pkg/security"
)

type contractHTTPFileReader struct{}

func (contractHTTPFileReader) Get(_ context.Context, in filesvc.GetInput) (*domainfile.FileObject, filesvc.AccessDecision, error) {
	if in.FileID != "file-1" {
		return nil, filesvc.AccessDecision{}, fmt.Errorf("file not found")
	}
	return &domainfile.FileObject{ID: "file-1", Name: "agreement.pdf", MIME: "application/pdf", Size: 512, Status: domainfile.StatusAvailable}, filesvc.AccessDecision{Allowed: true}, nil
}

func TestContractHTTPCreateApproveAndExpiryScan(t *testing.T) {
	ctx := context.Background()
	suppliers := pharmaoasvc.NewSupplierService(nil)
	_, _ = suppliers.Create(ctx, pharmaoasvc.SupplierWriteInput{Code: "SUP-HTTP", Name: "HTTP Supplier", Contacts: []domainpharma.SupplierContact{{Name: "Alice"}}})
	customers := pharmaoasvc.NewCustomerService(nil)
	_, _ = customers.Create(ctx, pharmaoasvc.CustomerWriteInput{Code: "CUS-HTTP", Name: "HTTP Customer", Region: "East", OrganizationID: "org-1", OwnerID: "owner-1", Contacts: []domainpharma.CustomerContact{{Name: "Bob"}}, Scope: pharmaoasvc.CustomerAccessScope{IncludeAll: true}})
	service := pharmaoasvc.NewContractService(suppliers, customers, workflowsvc.NewService(workflowsvc.NewMemoryRepository()), contractHTTPFileReader{}, notificationsvc.NewService(nil, nil), nil)
	mux := http.NewServeMux()
	RegisterContractRoutes(mux, service)

	effectiveAt := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	expiresAt := time.Now().UTC().AddDate(0, 0, 10).Format("2006-01-02")
	body := fmt.Sprintf(`{"number":"CTR-HTTP-1","title":"Supply agreement","partyType":"supplier","partyId":"pharma-supplier-1","ownerId":"owner-1","approverId":"approver-1","amount":8000,"currency":"CNY","effectiveAt":%q,"expiresAt":%q,"attachmentIds":["file-1"]}`, effectiveAt, expiresAt)
	create := contractHTTPRequest(mux, http.MethodPost, "/v1/plugins/pharma_oa/api/contracts", body, security.JWTClaims{Subject: "owner-1", Role: "legal"})
	if create.Code != http.StatusCreated || !bytes.Contains(create.Body.Bytes(), []byte("contract-workflow-1")) {
		t.Fatalf("create status=%d body=%s", create.Code, create.Body.String())
	}
	spoofedApprove := contractHTTPRequest(mux, http.MethodPost, "/v1/plugins/pharma_oa/api/contracts/contract-1/approve", `{"actorId":"approver-1","comment":"spoofed"}`, security.JWTClaims{Subject: "intruder-1", Role: "manager"})
	if spoofedApprove.Code != http.StatusBadRequest {
		t.Fatalf("spoofed approve status=%d body=%s", spoofedApprove.Code, spoofedApprove.Body.String())
	}
	approve := contractHTTPRequest(mux, http.MethodPost, "/v1/plugins/pharma_oa/api/contracts/contract-1/approve", `{"comment":"ok"}`, security.JWTClaims{Subject: "approver-1", Role: "manager"})
	if approve.Code != http.StatusOK || !bytes.Contains(approve.Body.Bytes(), []byte(`"status":"active"`)) {
		t.Fatalf("approve status=%d body=%s", approve.Code, approve.Body.String())
	}
	scan := contractHTTPRequest(mux, http.MethodPost, "/v1/plugins/pharma_oa/api/contracts/expiry-scan", `{"days":30}`, security.JWTClaims{Subject: "scheduler", Role: "admin"})
	if scan.Code != http.StatusOK || !bytes.Contains(scan.Body.Bytes(), []byte(`"createdCount":1`)) {
		t.Fatalf("scan status=%d body=%s", scan.Code, scan.Body.String())
	}
	detail := contractHTTPRequest(mux, http.MethodGet, "/v1/plugins/pharma_oa/api/contracts/contract-1", "", security.JWTClaims{Subject: "owner-1", Role: "legal"})
	if detail.Code != http.StatusOK || !bytes.Contains(detail.Body.Bytes(), []byte("contract-expiry-contract-1")) {
		t.Fatalf("detail status=%d body=%s", detail.Code, detail.Body.String())
	}
}

func contractHTTPRequest(handler http.Handler, method, target, body string, claims security.JWTClaims) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(security.WithJWTClaimsContext(req.Context(), &claims))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder
}
