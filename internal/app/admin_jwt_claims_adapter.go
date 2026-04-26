package app

import (
	"net/http"
	"strconv"
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
	verified := normalizeAdminJWTVerified(verifiedRaw)
	subject := normalizeAdminJWTSubject(strings.TrimSpace(r.Header.Get(HeaderAdminJWTSubject)))
	jwtRoleID := strings.TrimSpace(r.Header.Get(HeaderAdminJWTRoleID))
	legacyRoleID := strings.TrimSpace(r.Header.Get(HeaderAdminRoleID))
	claimsVersion := normalizeAdminJWTClaimsVersion(strings.TrimSpace(r.Header.Get(HeaderAdminJWTClaimsVersion)))

	roleID := normalizeAdminJWTRoleID(jwtRoleID, legacyRoleID)

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

func normalizeAdminJWTRoleID(jwtRoleID, legacyRoleID string) string {
	if normalized, ok := normalizePositiveIntString(jwtRoleID); ok {
		return normalized
	}
	if normalized, ok := normalizePositiveIntString(legacyRoleID); ok {
		return normalized
	}
	return ""
}

func normalizeAdminJWTSubject(subject string) string {
	return strings.TrimSpace(subject)
}

func normalizeAdminJWTClaimsVersion(version string) string {
	return strings.ToLower(strings.TrimSpace(version))
}

func normalizeAdminJWTVerified(raw string) bool {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func normalizePositiveIntString(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", false
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return "", false
	}
	return strconv.FormatInt(parsed, 10), true
}
