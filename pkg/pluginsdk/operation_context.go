package pluginsdk

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

const MaxOperationIdentitySize = 128

var operationIdentityPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}$`)

type OperationContext struct {
	CorrelationID string
	RequestID     string
	TraceID       string
}

type operationContextKey struct{}

func NewOperationContext(requestID, traceID string) (OperationContext, error) {
	requestID = strings.TrimSpace(requestID)
	traceID = strings.TrimSpace(traceID)
	if requestID != "" && !validOperationIdentity(requestID) {
		requestID = ""
	}
	if traceID != "" && !validOperationIdentity(traceID) {
		traceID = ""
	}
	correlationID := requestID
	if correlationID == "" {
		generated, err := newOperationIdentity()
		if err != nil {
			return OperationContext{}, err
		}
		correlationID = generated
		requestID = generated
	}
	return OperationContext{CorrelationID: correlationID, RequestID: requestID, TraceID: traceID}, nil
}

func WithOperationContext(ctx context.Context, operation OperationContext) (context.Context, error) {
	operation.CorrelationID = strings.TrimSpace(operation.CorrelationID)
	operation.RequestID = strings.TrimSpace(operation.RequestID)
	operation.TraceID = strings.TrimSpace(operation.TraceID)
	if operation.CorrelationID == "" || !validOperationIdentity(operation.CorrelationID) ||
		operation.RequestID != "" && !validOperationIdentity(operation.RequestID) ||
		operation.TraceID != "" && !validOperationIdentity(operation.TraceID) {
		return nil, fmt.Errorf("plugin operation context is invalid")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, operationContextKey{}, operation), nil
}

func EnsureOperationContext(ctx context.Context) (context.Context, OperationContext, error) {
	if operation, ok := OperationContextFromContext(ctx); ok {
		return ctx, operation, nil
	}
	operation, err := NewOperationContext("", "")
	if err != nil {
		return nil, OperationContext{}, err
	}
	bound, err := WithOperationContext(ctx, operation)
	return bound, operation, err
}

func OperationContextFromContext(ctx context.Context) (OperationContext, bool) {
	if ctx == nil {
		return OperationContext{}, false
	}
	operation, ok := ctx.Value(operationContextKey{}).(OperationContext)
	if !ok || operation.CorrelationID == "" {
		return OperationContext{}, false
	}
	return operation, true
}

func validOperationIdentity(value string) bool {
	value = strings.TrimSpace(value)
	return len(value) <= MaxOperationIdentitySize && operationIdentityPattern.MatchString(value)
}

func newOperationIdentity() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate plugin operation identity: %w", err)
	}
	return "op-" + hex.EncodeToString(raw[:]), nil
}
