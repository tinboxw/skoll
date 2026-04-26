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
	if claims.RoleSource != "x-admin-jwt-role-id" {
		t.Fatalf("expected role source x-admin-jwt-role-id, got %s", claims.RoleSource)
	}
	if claims.SubjectSource != "x-admin-jwt-subject" {
		t.Fatalf("expected subject source x-admin-jwt-subject, got %s", claims.SubjectSource)
	}
	if claims.VerifiedSource != "x-admin-jwt-verified" {
		t.Fatalf("expected verified source x-admin-jwt-verified, got %s", claims.VerifiedSource)
	}
	if claims.ClaimsVersionSource != "x-admin-jwt-claims-version" {
		t.Fatalf("expected claims version source header, got %s", claims.ClaimsVersionSource)
	}
	if claims.Source != "header" {
		t.Fatalf("expected source=header, got %s", claims.Source)
	}
	if len(claims.SourceProvenance) != 1 || claims.SourceProvenance[0] != "header" {
		t.Fatalf("expected source provenance [header], got %v", claims.SourceProvenance)
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
	if claims.RoleSource != "x-admin-role-id" {
		t.Fatalf("expected role source fallback x-admin-role-id, got %s", claims.RoleSource)
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

func TestResolveAdminVerifiedClaims_SourceProvenanceChain(t *testing.T) {
	req := httptest.NewRequest("GET", "/admin/v1/system/dashboard", nil)
	req.Header.Set(HeaderAdminJWTSource, "gateway")
	req.Header.Set(HeaderAdminJWTSourceChain, "edge-auth, gateway, edge-auth")
	req.Header.Set(HeaderAdminJWTVerified, "1")

	claims := resolveAdminVerifiedClaims(req)
	if claims.Source != "gateway" {
		t.Fatalf("expected source gateway, got %s", claims.Source)
	}
	if len(claims.SourceProvenance) != 2 || claims.SourceProvenance[0] != "edge-auth" || claims.SourceProvenance[1] != "gateway" {
		t.Fatalf("unexpected source provenance chain %v", claims.SourceProvenance)
	}
}
