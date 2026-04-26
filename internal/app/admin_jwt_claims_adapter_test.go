package app

import (
	"net/http/httptest"
	"testing"
)

func TestResolveAdminVerifiedClaims_PrioritizesJWTSpecificRoleHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/admin/v1/system/dashboard", nil)
	req.Header.Set(HeaderAdminRoleID, "1")
	req.Header.Set(HeaderAdminJWTRoleID, "09")
	req.Header.Set(HeaderAdminJWTVerified, "true")
	req.Header.Set(HeaderAdminJWTSubject, "bridge-user")
	req.Header.Set(HeaderAdminJWTClaimsVersion, " V2 ")

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
	if claims.ClaimsVersion != "v2" {
		t.Fatalf("expected normalized claims version v2, got %s", claims.ClaimsVersion)
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

func TestResolveAdminVerifiedClaims_FallbackWhenJWTRoleInvalid(t *testing.T) {
	req := httptest.NewRequest("GET", "/admin/v1/system/dashboard", nil)
	req.Header.Set(HeaderAdminJWTRoleID, "abc")
	req.Header.Set(HeaderAdminRoleID, " 7 ")

	claims := resolveAdminVerifiedClaims(req)
	if claims.RoleID != "7" {
		t.Fatalf("expected fallback role id 7, got %s", claims.RoleID)
	}
}

func TestResolveAdminVerifiedClaims_VerifiedAlias(t *testing.T) {
	req := httptest.NewRequest("GET", "/admin/v1/system/dashboard", nil)
	req.Header.Set(HeaderAdminJWTVerified, "YES")

	claims := resolveAdminVerifiedClaims(req)
	if !claims.Verified {
		t.Fatalf("expected verified=true for alias YES")
	}
}
