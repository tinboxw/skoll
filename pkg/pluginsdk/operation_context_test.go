package pluginsdk

import (
	"context"
	"testing"
)

func TestOperationContextUsesValidRequestIdentityAsCorrelation(t *testing.T) {
	operation, err := NewOperationContext("request-42", "trace-42")
	if err != nil {
		t.Fatalf("new operation context: %v", err)
	}
	if operation.CorrelationID != "request-42" || operation.RequestID != "request-42" || operation.TraceID != "trace-42" {
		t.Fatalf("unexpected operation context: %+v", operation)
	}
	bound, err := WithOperationContext(context.Background(), operation)
	if err != nil {
		t.Fatalf("bind operation context: %v", err)
	}
	stored, ok := OperationContextFromContext(bound)
	if !ok || stored != operation {
		t.Fatalf("operation context was not preserved: %+v", stored)
	}
}

func TestOperationContextReplacesInvalidExternalIdentities(t *testing.T) {
	operation, err := NewOperationContext("request with spaces", "trace with spaces")
	if err != nil {
		t.Fatalf("new operation context: %v", err)
	}
	if operation.CorrelationID == "" || operation.RequestID != operation.CorrelationID || operation.TraceID != "" {
		t.Fatalf("invalid identities were not replaced safely: %+v", operation)
	}
}

func TestWithOperationContextRejectsMissingOrMalformedCorrelation(t *testing.T) {
	for _, operation := range []OperationContext{
		{},
		{CorrelationID: "*"},
		{CorrelationID: "request-1", RequestID: "not valid"},
	} {
		if _, err := WithOperationContext(context.Background(), operation); err == nil {
			t.Fatalf("expected invalid operation context to fail: %+v", operation)
		}
	}
}
