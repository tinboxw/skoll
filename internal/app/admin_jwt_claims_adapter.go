package app

import (
	"net/http"
	"strings"
)

const (
	HeaderAdminJWTVerified      = "X-Admin-JWT-Verified"
	HeaderAdminJWTSubject       = "X-Admin-JWT-Subject"
	HeaderAdminJWTClaimsVersion = "X-Admin-JWT-Claims-Version"
	HeaderAdminJWTRoleID        = "X-Admin-JWT-Role-ID"
)

type adminVerifiedClaims struct {
	Present       bool
	Verified      bool
	Subject       string
	RoleID        string
	ClaimsVersion string
	Source        string
}

func resolveAdminVerifiedClaims(r *http.Request) adminVerifiedClaims {
	if r == nil {
		return adminVerifiedClaims{Source: "none"}
	}

	verifiedRaw := strings.TrimSpace(r.Header.Get(HeaderAdminJWTVerified))
	verified := strings.EqualFold(verifiedRaw, "true") || verifiedRaw == "1"
	subject := strings.TrimSpace(r.Header.Get(HeaderAdminJWTSubject))
	jwtRoleID := strings.TrimSpace(r.Header.Get(HeaderAdminJWTRoleID))
	legacyRoleID := strings.TrimSpace(r.Header.Get(HeaderAdminRoleID))
	claimsVersion := strings.TrimSpace(r.Header.Get(HeaderAdminJWTClaimsVersion))

	roleID := jwtRoleID
	if roleID == "" {
		roleID = legacyRoleID
	}

	present := verifiedRaw != "" || subject != "" || jwtRoleID != "" || legacyRoleID != "" || claimsVersion != ""
	source := "none"
	if present {
		source = "header"
	}

	return adminVerifiedClaims{
		Present:       present,
		Verified:      verified,
		Subject:       subject,
		RoleID:        roleID,
		ClaimsVersion: claimsVersion,
		Source:        source,
	}
}
