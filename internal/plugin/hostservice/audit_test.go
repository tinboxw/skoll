package hostservice

import (
	"context"
	"strings"
	"testing"

	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	"github.com/tinboxw/skoll/internal/store/clickhouse"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"github.com/tinboxw/skoll/pkg/security"
)

func TestAuditServiceBindsNamespaceAndRedactsSensitiveDetail(t *testing.T) {
	backend := auditsvc.NewService(clickhouse.NewAuditStore())
	service, err := NewAuditService("pharma_oa", backend)
	if err != nil {
		t.Fatalf("NewAuditService error: %v", err)
	}
	ctx := security.WithJWTClaimsContext(context.Background(), &security.JWTClaims{Subject: "auditor-1"})
	ctx, err = pluginsdk.WithOperationContext(ctx, pluginsdk.OperationContext{
		CorrelationID: "request-42",
		RequestID:     "request-42",
		TraceID:       "trace-42",
	})
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := service.Record(ctx, pluginsdk.AuditEntry{
		Action: "customer.update", Resource: "customer", ResourceID: "customer-1", Risk: pluginsdk.AuditRiskHigh,
		Detail: map[string]any{
			"name": "Hospital", "credentials": map[string]any{"api_token": "top-secret"},
			"headers": map[string]string{"authorization": "Bearer hidden-token", "content-type": "application/json"},
		},
	})
	if err != nil || receipt.ID == "" {
		t.Fatalf("Record receipt=%+v err=%v", receipt, err)
	}
	records, err := backend.ListByActor(context.Background(), "auditor-1", 10)
	if err != nil || len(records) != 1 {
		t.Fatalf("ListByActor records=%d err=%v", len(records), err)
	}
	record := records[0]
	if record.Action != "plugin.pharma_oa.customer.update" || record.Resource != "plugin:pharma_oa:customer" {
		t.Fatalf("audit namespace escaped: %+v", record)
	}
	if record.Detail["correlationId"] != "request-42" ||
		record.Detail["requestId"] != "request-42" ||
		record.Detail["traceId"] != "trace-42" {
		t.Fatalf("operation correlation missing from audit detail: %+v", record.Detail)
	}
	raw := strings.ToLower(string(mustJSON(t, record.Detail)))
	if strings.Contains(raw, "top-secret") || strings.Contains(raw, "hidden-token") || !strings.Contains(raw, "[redacted]") {
		t.Fatalf("audit detail was not redacted: %s", raw)
	}
}
