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
	HeaderAdminJWTSource        = "X-Admin-JWT-Source"
	HeaderAdminJWTSourceChain   = "X-Admin-JWT-Source-Provenance"
)

type adminVerifiedClaims struct {
	Present             bool
	Verified            bool
	Subject             string
	RoleID              string
	ClaimsVersion       string
	Source              string
	SourceProvenance    []string
	RoleSource          string
	SubjectSource       string
	VerifiedSource      string
	ClaimsVersionSource string
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
	claimsVersionRaw := strings.TrimSpace(r.Header.Get(HeaderAdminJWTClaimsVersion))
	claimsVersion := normalizeAdminJWTClaimsVersion(claimsVersionRaw)

	roleID, roleSource := normalizeAdminJWTRoleID(jwtRoleID, legacyRoleID)

	subjectSource := ""
	if subject != "" {
		subjectSource = strings.ToLower(HeaderAdminJWTSubject)
	}
	verifiedSource := ""
	if strings.TrimSpace(verifiedRaw) != "" {
		verifiedSource = strings.ToLower(HeaderAdminJWTVerified)
	}
	claimsVersionSource := ""
	if claimsVersion != "" {
		claimsVersionSource = strings.ToLower(HeaderAdminJWTClaimsVersion)
	}

	present := verifiedRaw != "" || subject != "" || jwtRoleID != "" || legacyRoleID != "" || claimsVersion != ""
	source := "none"
	if present {
		source = "header"
	}
	sourceHint := normalizeAdminJWTSource(strings.TrimSpace(r.Header.Get(HeaderAdminJWTSource)))
	if sourceHint != "" {
		source = sourceHint
	}
	sourceProvenance := normalizeAdminJWTSourceProvenance(strings.TrimSpace(r.Header.Get(HeaderAdminJWTSourceChain)), source)

	return adminVerifiedClaims{
		Present:             present,
		Verified:            verified,
		Subject:             subject,
		RoleID:              roleID,
		ClaimsVersion:       claimsVersion,
		Source:              source,
		SourceProvenance:    sourceProvenance,
		RoleSource:          roleSource,
		SubjectSource:       subjectSource,
		VerifiedSource:      verifiedSource,
		ClaimsVersionSource: claimsVersionSource,
	}
}

func normalizeAdminJWTRoleID(jwtRoleID, legacyRoleID string) (string, string) {
	if normalized, ok := normalizePositiveIntString(jwtRoleID); ok {
		return normalized, strings.ToLower(HeaderAdminJWTRoleID)
	}
	if normalized, ok := normalizePositiveIntString(legacyRoleID); ok {
		return normalized, strings.ToLower(HeaderAdminRoleID)
	}
	return "", ""
}

func normalizeAdminJWTSubject(subject string) string {
	return strings.TrimSpace(subject)
}

func normalizeAdminJWTClaimsVersion(version string) string {
	return strings.ToLower(strings.TrimSpace(version))
}

func normalizeAdminJWTSource(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return ""
	}
	return value
}

func normalizeAdminJWTSourceProvenance(raw, fallback string) []string {
	parts := strings.Split(raw, ",")
	seen := make(map[string]struct{})
	out := make([]string, 0, len(parts)+1)
	for _, part := range parts {
		value := normalizeAdminJWTSource(part)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	if len(out) == 0 && fallback != "none" {
		out = append(out, fallback)
	}
	if len(out) == 0 {
		return []string{}
	}
	return out
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
