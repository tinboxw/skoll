package app

import (
	"net/http/httptest"
	"testing"
)

func TestResolveAdminVerifiedClaims_PrioritizesJWTSpecificRoleHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/admin/v1/system/dashboard", nil)
	req.Header.Set(HeaderAdminRoleID, "1")
	req.Header.Set(HeaderAdminJWTRoleID, "9")
	req.Header.Set(HeaderAdminJWTVerified, "true")
	req.Header.Set(HeaderAdminJWTSubject, "bridge-user")
	req.Header.Set(HeaderAdminJWTClaimsVersion, "v2")

	claims := resolveAdminVerifiedClaims(req)
	if !claims.Present {
		t.Fatalf("expected claims present")
	}
	if !claims.Verified {
		t.Fatalf("expected claims verified")
	}
	if claims.RoleID != "9" {
		t.Fatalf("expected jwt-specific role id precedence, got %s", claims.RoleID)
	}
	if claims.Source != "header" {
		t.Fatalf("expected source=header, got %s", claims.Source)
	}
}

func TestResolveAdminVerifiedClaims_FallbackToLegacyRoleHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/admin/v1/system/dashboard", nil)
	req.Header.Set(HeaderAdminRoleID, "3")

	claims := resolveAdminVerifiedClaims(req)
	if claims.RoleID != "3" {
		t.Fatalf("expected fallback role id 3, got %s", claims.RoleID)
	}
	if !claims.Present {
		t.Fatalf("expected claims present due to legacy role header")
	}
}
