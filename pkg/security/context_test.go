package security

import (
	"context"
	"testing"
)

func TestWithJWTClaimsContextAndExtract(t *testing.T) {
	claims := &JWTClaims{Subject: "alice", Role: "editor"}
	ctx := WithJWTClaimsContext(context.Background(), claims)

	got, ok := JWTClaimsFromContext(ctx)
	if !ok {
		t.Fatalf("expected claims in context")
	}
	if got.Subject != "alice" || got.Role != "editor" {
		t.Fatalf("unexpected claims: %+v", got)
	}
}

func TestWithJWTClaimsContextNilClaims(t *testing.T) {
	ctx := WithJWTClaimsContext(context.Background(), nil)
	if _, ok := JWTClaimsFromContext(ctx); ok {
		t.Fatalf("expected no claims")
	}
}
