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

func TestCustomerFollowUpHTTPRejectsScopeAndActorSpoofing(t *testing.T) {
	ctx := context.Background()
	customers := pharmaoasvc.NewCustomerService(nil)
	first, _ := customers.Create(ctx, pharmaoasvc.CustomerWriteInput{Code: "CUS-1", Name: "First", Region: "East", OrganizationID: "org-a", OwnerID: "sales-1", Contacts: []domainpharma.CustomerContact{{Name: "Amy"}}})
	second, _ := customers.Create(ctx, pharmaoasvc.CustomerWriteInput{Code: "CUS-2", Name: "Second", Region: "East", OrganizationID: "org-a", OwnerID: "sales-2", Contacts: []domainpharma.CustomerContact{{Name: "Bob"}}})
	service := pharmaoasvc.NewCustomerFollowUpService(customers, nil)
	when := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	_, _ = service.Create(ctx, pharmaoasvc.CustomerFollowUpCreateInput{CustomerID: first.ID.String(), Channel: domainpharma.CustomerFollowUpOnsite, ScheduledAt: time.Now().UTC(), ActorID: "sales-1"})
	_, _ = service.Create(ctx, pharmaoasvc.CustomerFollowUpCreateInput{CustomerID: second.ID.String(), Channel: domainpharma.CustomerFollowUpPhone, ScheduledAt: time.Now().UTC(), ActorID: "sales-2"})
	mux := http.NewServeMux()
	RegisterCustomerFollowUpRoutes(mux, service)
	list := followUpHTTPRequest(mux, http.MethodGet, "/v1/plugins/pharma_oa/api/customer-follow-ups?includeAll=true&ownerId=sales-2", "", "sales-1")
	if list.Code != http.StatusOK || !bytes.Contains(list.Body.Bytes(), []byte(`"ownerId":"sales-1"`)) || bytes.Contains(list.Body.Bytes(), []byte(`"ownerId":"sales-2"`)) {
		t.Fatalf("spoofed list scope status=%d body=%s", list.Code, list.Body.String())
	}
	body := `{"customerId":"` + second.ID.String() + `","channel":"phone","scheduledAt":"` + when + `","actorId":"sales-2"}`
	create := followUpHTTPRequest(mux, http.MethodPost, "/v1/plugins/pharma_oa/api/customer-follow-ups", body, "sales-1")
	if create.Code != http.StatusBadRequest {
		t.Fatalf("spoofed create status=%d body=%s", create.Code, create.Body.String())
	}
}

func followUpHTTPRequest(handler http.Handler, method, target, body, actorID string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(security.WithJWTClaimsContext(req.Context(), &security.JWTClaims{Subject: actorID, Role: "sales"}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder
}
