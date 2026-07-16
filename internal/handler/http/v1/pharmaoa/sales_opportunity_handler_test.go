package pharmaoa

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
	"github.com/tinboxw/skoll/pkg/security"
)

func TestSalesOpportunityHTTPRejectsScopeAndActorSpoofing(t *testing.T) {
	ctx := context.Background()
	customers := pharmaoasvc.NewCustomerService(nil)
	products := pharmaoasvc.NewProductService(nil)
	first, _ := customers.Create(ctx, pharmaoasvc.CustomerWriteInput{Code: "CUS-1", Name: "First", Region: "East", OrganizationID: "org-a", OwnerID: "sales-1", Contacts: []domainpharma.CustomerContact{{Name: "Amy"}}})
	second, _ := customers.Create(ctx, pharmaoasvc.CustomerWriteInput{Code: "CUS-2", Name: "Second", Region: "East", OrganizationID: "org-a", OwnerID: "sales-2", Contacts: []domainpharma.CustomerContact{{Name: "Bob"}}})
	product, _ := products.Create(ctx, pharmaoasvc.ProductWriteInput{Code: "DRUG-1", Name: "Drug", Spec: "10mg", DosageForm: "tablet", Manufacturer: "Skoll", ApprovalNumber: "A-1"})
	service := pharmaoasvc.NewSalesOpportunityService(customers, products, nil)
	closeDate := time.Now().UTC().AddDate(0, 1, 0)
	_, _ = service.Create(ctx, pharmaoasvc.SalesOpportunityCreateInput{Title: "First deal", CustomerID: first.ID.String(), ProductIDs: []string{product.ID.String()}, ExpectedAmountCents: 1000, EstimatedCloseDate: closeDate, ActorID: "sales-1"})
	_, _ = service.Create(ctx, pharmaoasvc.SalesOpportunityCreateInput{Title: "Second deal", CustomerID: second.ID.String(), ProductIDs: []string{product.ID.String()}, ExpectedAmountCents: 2000, EstimatedCloseDate: closeDate, ActorID: "sales-2"})
	mux := http.NewServeMux()
	RegisterSalesOpportunityRoutes(mux, service)
	list := salesOpportunityHTTPRequest(mux, http.MethodGet, "/v1/pharma-oa/sales-opportunities?includeAll=true&ownerId=sales-2", "", "sales-1")
	if list.Code != http.StatusOK || !bytes.Contains(list.Body.Bytes(), []byte(`"ownerId":"sales-1"`)) || bytes.Contains(list.Body.Bytes(), []byte(`"ownerId":"sales-2"`)) {
		t.Fatalf("spoofed list scope status=%d body=%s", list.Code, list.Body.String())
	}
	body := `{"title":"Spoofed","customerId":"` + second.ID.String() + `","productIds":["` + product.ID.String() + `"],"expectedAmountCents":5000,"estimatedCloseDate":"` + closeDate.Format(time.RFC3339) + `","actorId":"sales-2"}`
	create := salesOpportunityHTTPRequest(mux, http.MethodPost, "/v1/pharma-oa/sales-opportunities", body, "sales-1")
	if create.Code != http.StatusBadRequest {
		t.Fatalf("spoofed create status=%d body=%s", create.Code, create.Body.String())
	}
	statistics := salesOpportunityHTTPRequest(mux, http.MethodGet, "/v1/pharma-oa/sales-opportunities/statistics?includeAll=true", "", "sales-1")
	if statistics.Code != http.StatusOK || !bytes.Contains(statistics.Body.Bytes(), []byte(`"totalCount":1`)) {
		t.Fatalf("spoofed statistics status=%d body=%s", statistics.Code, statistics.Body.String())
	}
}

func salesOpportunityHTTPRequest(handler http.Handler, method, target, body, actorID string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(security.WithJWTClaimsContext(req.Context(), &security.JWTClaims{Subject: actorID, Role: "sales"}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder
}
