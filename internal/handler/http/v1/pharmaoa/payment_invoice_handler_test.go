package pharmaoa

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	"github.com/tinboxw/skoll/internal/domain/shared"
	notificationsvc "github.com/tinboxw/skoll/internal/service/notification"
	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
	"github.com/tinboxw/skoll/pkg/security"
)

func TestPaymentInvoiceHTTPRejectsScopeAndActorSpoofing(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	reader := handlerPaymentOrderReader{items: map[string]*domainpharma.SalesOrder{
		"order-1": {ID: shared.ID("order-1"), Number: "SO-001", CustomerID: "customer-1", TotalAmount: 100, CreatedBy: "sales-1", Status: domainpharma.SalesOrderOpen},
		"order-2": {ID: shared.ID("order-2"), Number: "SO-002", CustomerID: "customer-2", TotalAmount: 100, CreatedBy: "sales-2", Status: domainpharma.SalesOrderOpen},
	}}
	service := pharmaoasvc.NewPaymentInvoiceService(reader, notificationsvc.NewService(notificationsvc.NewMemoryRepository(), nil, nil), nil)
	_, _ = service.CreatePaymentPlan(ctx, pharmaoasvc.PaymentPlanCreateInput{SalesOrderID: "order-1", AmountCents: 10000, DueAt: now, ActorID: "sales-1"})
	_, _ = service.CreatePaymentPlan(ctx, pharmaoasvc.PaymentPlanCreateInput{SalesOrderID: "order-2", AmountCents: 10000, DueAt: now, ActorID: "sales-2"})
	mux := http.NewServeMux()
	RegisterPaymentInvoiceRoutes(mux, service)

	list := paymentInvoiceHTTPRequest(mux, http.MethodGet, "/v1/plugins/pharma_oa/api/payment-plans?includeAll=true&ownerId=sales-2", "", "sales-1")
	if list.Code != http.StatusOK || !bytes.Contains(list.Body.Bytes(), []byte(`"salesOrderId":"order-1"`)) || bytes.Contains(list.Body.Bytes(), []byte(`"salesOrderId":"order-2"`)) {
		t.Fatalf("spoofed list scope status=%d body=%s", list.Code, list.Body.String())
	}
	createBody := `{"salesOrderId":"order-2","amountCents":100,"dueAt":"` + now.Format(time.RFC3339) + `","actorId":"sales-2"}`
	create := paymentInvoiceHTTPRequest(mux, http.MethodPost, "/v1/plugins/pharma_oa/api/payment-plans", createBody, "sales-1")
	if create.Code != http.StatusBadRequest {
		t.Fatalf("spoofed create status=%d body=%s", create.Code, create.Body.String())
	}
	run := paymentInvoiceHTTPRequest(mux, http.MethodPost, "/v1/plugins/pharma_oa/api/payment-reminder-jobs", `{"recipientId":"finance-1","actorId":"sales-2"}`, "sales-1")
	if run.Code != http.StatusCreated || !bytes.Contains(run.Body.Bytes(), []byte(`"initiatedBy":"sales-1"`)) {
		t.Fatalf("JWT actor was not retained status=%d body=%s", run.Code, run.Body.String())
	}
}

type handlerPaymentOrderReader struct {
	items map[string]*domainpharma.SalesOrder
}

func (r handlerPaymentOrderReader) GetOrder(_ context.Context, id string) (*domainpharma.SalesOrder, error) {
	item := r.items[id]
	if item == nil {
		return nil, errors.New("sales order not found")
	}
	out := *item
	return &out, nil
}

func paymentInvoiceHTTPRequest(handler http.Handler, method, target, body, actorID string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(security.WithJWTClaimsContext(req.Context(), &security.JWTClaims{Subject: actorID, Role: "sales"}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder
}
