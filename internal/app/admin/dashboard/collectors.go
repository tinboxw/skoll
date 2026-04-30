package dashboard

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"runtime"
	"sort"
	"strings"
	"time"

	admincontracts "github.com/tinboxw/skoll/internal/app/admin/contracts"
	adminsecurity "github.com/tinboxw/skoll/internal/app/admin/security"
	"github.com/tinboxw/skoll/internal/integration/adminauth"
)

var moduleStartTime = time.Now().UTC()

const contractVersion = "v1"

const jwtRefreshLeadWindow = 5 * time.Minute

func CollectContractDescriptor() ContractDescriptor {
	return ContractDescriptor{
		Name:      "dashboard-ui-bootstrap",
		Version:   contractVersion,
		Stability: "stable",
		RequiredSections: []string{
			"auth_session",
			"auth_observability",
			"auth_actionability",
			"jwt_session_bootstrap",
			"status",
			"runtime_metrics",
			"node_health",
			"scheduler_reliability",
			"hardening_posture",
		},
	}
}

func CollectJWTSessionBootstrap(r *http.Request, now time.Time) JWTSessionBootstrap {
	middlewareBridge := collectJWTMiddlewareBridge(r)
	claimsTrusted := middlewareBridge.Verified
	buildAuditExport := func(verificationState string) JWTProvenanceAuditExport {
		return collectJWTProvenanceAuditExport(middlewareBridge, verificationState, claimsTrusted)
	}

	authorization := strings.TrimSpace(r.Header.Get("Authorization"))
	if authorization == "" {
		verificationState, verificationHint, trustLevel, trustMessage := DeriveJWTVerificationHints("none", "none", claimsTrusted, "")
		return JWTSessionBootstrap{
			TokenPresent:          false,
			TokenFormat:           "none",
			ClaimsTrusted:         claimsTrusted,
			MiddlewareBridge:      middlewareBridge,
			ProvenanceAuditExport: buildAuditExport(verificationState),
			SessionState:          "none",
			VerificationState:     verificationState,
			VerificationHint:      verificationHint,
			TrustLevel:            trustLevel,
			TrustMessage:          trustMessage,
			Expired:               false,
			RefreshRecommended:    false,
		}
	}

	parts := strings.Fields(authorization)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		verificationState, verificationHint, trustLevel, trustMessage := DeriveJWTVerificationHints("unsupported", "invalid", claimsTrusted, "authorization header must use bearer scheme")
		return JWTSessionBootstrap{
			TokenPresent:          true,
			TokenFormat:           "unsupported",
			ClaimsTrusted:         claimsTrusted,
			MiddlewareBridge:      middlewareBridge,
			ProvenanceAuditExport: buildAuditExport(verificationState),
			SessionState:          "invalid",
			VerificationState:     verificationState,
			VerificationHint:      verificationHint,
			TrustLevel:            trustLevel,
			TrustMessage:          trustMessage,
			RefreshRecommended:    true,
			RefreshReason:         "authorization_header_invalid",
			ParseError:            "authorization header must use bearer scheme",
		}
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		verificationState, verificationHint, trustLevel, trustMessage := DeriveJWTVerificationHints("unsupported", "invalid", claimsTrusted, "bearer token is empty")
		return JWTSessionBootstrap{
			TokenPresent:          true,
			TokenFormat:           "unsupported",
			ClaimsTrusted:         claimsTrusted,
			MiddlewareBridge:      middlewareBridge,
			ProvenanceAuditExport: buildAuditExport(verificationState),
			SessionState:          "invalid",
			VerificationState:     verificationState,
			VerificationHint:      verificationHint,
			TrustLevel:            trustLevel,
			TrustMessage:          trustMessage,
			RefreshRecommended:    true,
			RefreshReason:         "authorization_header_invalid",
			ParseError:            "bearer token is empty",
		}
	}

	if strings.Count(token, ".") != 2 {
		verificationState, verificationHint, trustLevel, trustMessage := DeriveJWTVerificationHints("bearer-non-jwt", "invalid", claimsTrusted, "token does not match jwt compact format")
		return JWTSessionBootstrap{
			TokenPresent:          true,
			TokenFormat:           "bearer-non-jwt",
			ClaimsTrusted:         claimsTrusted,
			MiddlewareBridge:      middlewareBridge,
			ProvenanceAuditExport: buildAuditExport(verificationState),
			SessionState:          "invalid",
			VerificationState:     verificationState,
			VerificationHint:      verificationHint,
			TrustLevel:            trustLevel,
			TrustMessage:          trustMessage,
			RefreshRecommended:    true,
			RefreshReason:         "token_not_jwt",
			ParseError:            "token does not match jwt compact format",
		}
	}

	claims, err := decodeJWTClaims(token)
	if err != nil {
		verificationState, verificationHint, trustLevel, trustMessage := DeriveJWTVerificationHints("bearer-jwt", "invalid", claimsTrusted, err.Error())
		return JWTSessionBootstrap{
			TokenPresent:          true,
			TokenFormat:           "bearer-jwt",
			ClaimsTrusted:         claimsTrusted,
			MiddlewareBridge:      middlewareBridge,
			ProvenanceAuditExport: buildAuditExport(verificationState),
			SessionState:          "invalid",
			VerificationState:     verificationState,
			VerificationHint:      verificationHint,
			TrustLevel:            trustLevel,
			TrustMessage:          trustMessage,
			RefreshRecommended:    true,
			RefreshReason:         "jwt_parse_error",
			ParseError:            err.Error(),
		}
	}

	expiresAt := claimInt64(claims, "exp")
	expiresInSec := int64(0)
	refreshAfterUnixSec := int64(0)
	refreshRecommended := false
	refreshReason := ""
	sessionState := "active"

	if expiresAt <= 0 {
		sessionState = "no-exp"
		refreshRecommended = true
		refreshReason = "exp_claim_missing"
	} else {
		expiresInSec = expiresAt - now.Unix()
		if expiresInSec <= 0 {
			sessionState = "expired"
			refreshRecommended = true
			refreshReason = "token_expired"
		} else if expiresInSec <= int64(jwtRefreshLeadWindow/time.Second) {
			sessionState = "expiring"
			refreshRecommended = true
			refreshReason = "token_expiring_soon"
			refreshAfterUnixSec = now.Unix()
		} else {
			refreshAfterUnixSec = expiresAt - int64(jwtRefreshLeadWindow/time.Second)
		}
	}

	verificationState, verificationHint, trustLevel, trustMessage := DeriveJWTVerificationHints("bearer-jwt", sessionState, claimsTrusted, "")

	return JWTSessionBootstrap{
		TokenPresent:          true,
		TokenFormat:           "bearer-jwt",
		ClaimsTrusted:         claimsTrusted,
		MiddlewareBridge:      middlewareBridge,
		ProvenanceAuditExport: buildAuditExport(verificationState),
		SessionState:          sessionState,
		VerificationState:     verificationState,
		VerificationHint:      verificationHint,
		TrustLevel:            trustLevel,
		TrustMessage:          trustMessage,
		Subject:               claimString(claims, "sub"),
		Issuer:                claimString(claims, "iss"),
		Audience:              claimAudience(claims, "aud"),
		IssuedAtUnixSec:       claimInt64(claims, "iat"),
		ExpiresAtUnixSec:      expiresAt,
		ExpiresInSec:          expiresInSec,
		RefreshAfterUnixSec:   refreshAfterUnixSec,
		RefreshRecommended:    refreshRecommended,
		RefreshReason:         refreshReason,
		Expired:               expiresAt > 0 && now.Unix() >= expiresAt,
	}
}

func DeriveJWTVerificationHints(tokenFormat, sessionState string, claimsTrusted bool, parseError string) (string, string, string, string) {
	if claimsTrusted {
		return "verified", "Claims are verified by trusted middleware.", "trusted", "JWT claims verified; safe for authorization-adjacent UX hints."
	}

	if parseError != "" || sessionState == "invalid" || tokenFormat == "unsupported" || tokenFormat == "bearer-non-jwt" {
		return "invalid", "Token cannot be verified from dashboard bootstrap context.", "untrusted", "Treat token payload as untrusted input and request re-authentication."
	}

	if tokenFormat == "none" || sessionState == "none" {
		return "not_present", "No bearer token provided for JWT bootstrap context.", "none", "No JWT trust context available."
	}

	return "unverified", "JWT payload is parsed without signature validation in bootstrap mode.", "low", "Display identity hints only; defer privileged actions until backend-verified session checks succeed."
}

func CollectAuthSessionContext(r *http.Request) AuthSessionContext {
	tokenPresent := strings.TrimSpace(r.Header.Get(adminauth.HeaderToken)) != ""
	signaturePresent := strings.TrimSpace(r.Header.Get(adminauth.HeaderSignature)) != ""
	mode := "none"
	authenticated := false

	if signaturePresent {
		mode = "hmac-sha256"
		authenticated = true
	} else if tokenPresent {
		mode = "static-token"
		authenticated = true
	}

	roleID := strings.TrimSpace(r.Header.Get(adminsecurity.HeaderAdminRoleID))

	return AuthSessionContext{
		Authenticated:          authenticated,
		AuthModeHint:           mode,
		RoleID:                 roleID,
		HasRoleBinding:         roleID != "",
		TokenHeaderPresent:     tokenPresent,
		SignatureHeaderPresent: signaturePresent,
	}
}

func CollectAuthObservability() AuthObservability {
	snapshot := adminauth.Snapshot()
	return AuthObservability{
		StaticToken: AuthModeCounters{
			Success: snapshot.StaticToken.Success,
			Failure: snapshot.StaticToken.Failure,
			Reasons: snapshot.StaticToken.Reasons,
		},
		HMACSHA256: AuthModeCounters{
			Success: snapshot.HMACSHA256.Success,
			Failure: snapshot.HMACSHA256.Failure,
			Reasons: snapshot.HMACSHA256.Reasons,
		},
		TotalFailure: snapshot.StaticToken.Failure + snapshot.HMACSHA256.Failure,
	}
}

func CollectAuthActionability(authSession AuthSessionContext, authObs AuthObservability) AuthActionability {
	totalSuccess := authObs.StaticToken.Success + authObs.HMACSHA256.Success
	totalAttempts := totalSuccess + authObs.TotalFailure
	failureRate := 0.0
	if totalAttempts > 0 {
		failureRate = float64(authObs.TotalFailure) / float64(totalAttempts)
		failureRate = math.Round(failureRate*10000) / 10000
	}

	severity := "info"
	switch {
	case authObs.TotalFailure >= 20 || failureRate >= 0.30:
		severity = "critical"
	case authObs.TotalFailure > 0 || failureRate >= 0.10:
		severity = "warning"
	}

	nextActions := make([]string, 0, 4)
	switch authSession.AuthModeHint {
	case "none":
		nextActions = append(nextActions,
			"Enable admin auth for dashboard entry traffic before external exposure.",
			"Use hmac-sha256 as the preferred auth mode for production environments.",
		)
		if severity == "info" {
			severity = "warning"
		}
	case "static-token":
		nextActions = append(nextActions,
			"Plan migration from static-token to hmac-sha256 for stronger replay-resistant protection.",
			"Rotate static token on a fixed cadence and after any suspicious access pattern.",
		)
		if severity == "info" {
			severity = "warning"
		}
	case "hmac-sha256":
		nextActions = append(nextActions,
			"Keep SKOLL_ADMIN_AUTH_HMAC_SECRET rotation on the defined runbook cadence.",
		)
	}

	if authSession.Authenticated && !authSession.HasRoleBinding {
		nextActions = append(nextActions,
			"Attach role binding context for authenticated dashboard requests to unlock RBAC-aware UX decisions.",
		)
		if severity == "info" {
			severity = "warning"
		}
	}

	if authObs.TotalFailure > 0 {
		nextActions = append(nextActions,
			"Review top auth failure reasons and tune alerts for replay_nonce/missing_headers spikes.",
		)
	}

	if len(nextActions) == 0 {
		nextActions = append(nextActions, "No immediate auth/session action required.")
	}

	return AuthActionability{
		Severity:            severity,
		RecommendedAuthMode: "hmac-sha256",
		FailureRate:         failureRate,
		TopFailureReasons:   topFailureReasons(authObs),
		NextActions:         nextActions,
		Docs: []string{
			"docs/community/DASHBOARD_AUTH_SESSION_POLICY.md",
			"docs/planning/ADMIN_AUTH_SECURITY_RUNBOOK.md",
		},
	}
}

func CollectSystemStatus(services admincontracts.AdminModuleServices) SystemStatusResponse {
	return SystemStatusResponse{
		Version:       "v1",
		UserCount:     len(services.Users.List()),
		RoleCount:     len(services.Roles.List()),
		MenuCount:     len(services.Menus.List()),
		ConfigCount:   len(services.Configs.List()),
		DictCount:     len(services.Dictionaries.List()),
		FileCount:     len(services.Files.List()),
		JobCount:      len(services.Jobs.List()),
		PluginCount:   len(services.Plugins.List()),
		APIEntryCount: len(services.APIs.List()),
	}
}

func CollectRuntimeMetrics() RuntimeMetricsResponse {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	lastPause := uint64(0)
	if mem.NumGC > 0 {
		lastPause = mem.PauseNs[(mem.NumGC-1)%uint32(len(mem.PauseNs))]
	}
	now := time.Now().UTC()
	uptime := now.Sub(moduleStartTime)
	if uptime < 0 {
		uptime = 0
	}
	return RuntimeMetricsResponse{
		UptimeSeconds:   int64(uptime / time.Second),
		Goroutines:      runtime.NumGoroutine(),
		MemoryAlloc:     mem.Alloc,
		MemorySys:       mem.Sys,
		HeapAlloc:       mem.HeapAlloc,
		HeapSys:         mem.HeapSys,
		TotalAlloc:      mem.TotalAlloc,
		GCCount:         mem.NumGC,
		LastGCPauseNs:   lastPause,
		SnapshotUnixSec: now.Unix(),
	}
}

func CollectNodeHealth(services admincontracts.AdminModuleServices) NodeHealthResponse {
	now := time.Now().UTC().Unix()
	deps := []HealthCheckItem{
		{Name: "users", Status: healthStatus(services.Users != nil), Detail: "admin user service", CheckedAt: now},
		{Name: "roles", Status: healthStatus(services.Roles != nil), Detail: "admin role service", CheckedAt: now},
		{Name: "menus", Status: healthStatus(services.Menus != nil), Detail: "admin menu service", CheckedAt: now},
		{Name: "audit", Status: healthStatus(services.Audit != nil), Detail: "admin audit service", CheckedAt: now},
		{Name: "configs", Status: healthStatus(services.Configs != nil), Detail: "admin config service", CheckedAt: now},
		{Name: "dictionaries", Status: healthStatus(services.Dictionaries != nil), Detail: "admin dictionary service", CheckedAt: now},
		{Name: "files", Status: healthStatus(services.Files != nil), Detail: "admin file service", CheckedAt: now},
		{Name: "jobs", Status: healthStatus(services.Jobs != nil), Detail: "admin job service", CheckedAt: now},
		{Name: "generator", Status: healthStatus(services.Generator != nil), Detail: "admin generator service", CheckedAt: now},
		{Name: "plugins", Status: healthStatus(services.Plugins != nil), Detail: "admin plugin service", CheckedAt: now},
		{Name: "rbac", Status: healthStatus(services.RBAC != nil), Detail: "admin rbac service", CheckedAt: now},
		{Name: "api_registry", Status: healthStatus(services.APIs != nil), Detail: "admin api registry service", CheckedAt: now},
		{Name: "release_governance", Status: healthStatus(services.Releases != nil), Detail: "admin release governance service", CheckedAt: now},
	}
	nodeStatus := "up"
	for _, item := range deps {
		if item.Status != "up" {
			nodeStatus = "degraded"
			break
		}
	}
	return NodeHealthResponse{NodeStatus: nodeStatus, CheckedAt: now, Dependencies: deps}
}

func collectJWTProvenanceAuditExport(bridge JWTMiddlewareBridge, verificationState string, claimsTrusted bool) JWTProvenanceAuditExport {
	sourcePath := ""
	if len(bridge.SourceProvenance) > 0 {
		sourcePath = strings.Join(bridge.SourceProvenance, ">")
	}
	enabled := bridge.Present || len(bridge.SourceProvenance) > 0

	export := JWTProvenanceAuditExport{
		Enabled:             enabled,
		SourceProvenance:    append([]string(nil), bridge.SourceProvenance...),
		SourcePath:          sourcePath,
		Source:              bridge.Source,
		Verified:            bridge.Verified,
		ClaimsTrusted:       claimsTrusted,
		VerificationState:   verificationState,
		Subject:             bridge.Subject,
		RoleID:              bridge.RoleID,
		ClaimsVersion:       bridge.ClaimsVersion,
		RoleSource:          bridge.RoleSource,
		SubjectSource:       bridge.SubjectSource,
		ClaimsVersionSource: bridge.ClaimsVersionSource,
		VerifiedSource:      bridge.VerifiedSource,
	}

	hints := DeriveJWTProvenanceAlertingHints(export)
	ObserveJWTProvenanceAuditExport(export, hints)
	opsMetrics := SnapshotJWTProvenanceOperationalMetrics()
	export.OperationalMetrics = opsMetrics
	export.SLODashboard = BuildJWTProvenanceSLODashboard(opsMetrics)
	export.ErrorBudgetPolicy = BuildJWTProvenanceErrorBudgetPolicy(export.SLODashboard)
	export.AlertingHints = hints
	return export
}

func collectJWTMiddlewareBridge(r *http.Request) JWTMiddlewareBridge {
	claims := adminsecurity.ResolveVerifiedClaims(r)
	return JWTMiddlewareBridge{
		Present:             claims.Present,
		Verified:            claims.Verified,
		Source:              claims.Source,
		SourceProvenance:    claims.SourceProvenance,
		Subject:             claims.Subject,
		SubjectSource:       claims.SubjectSource,
		RoleID:              claims.RoleID,
		RoleSource:          claims.RoleSource,
		ClaimsVersion:       claims.ClaimsVersion,
		ClaimsVersionSource: claims.ClaimsVersionSource,
		VerifiedSource:      claims.VerifiedSource,
	}
}

func topFailureReasons(authObs AuthObservability) []string {
	reasons := make([]string, 0, len(authObs.StaticToken.Reasons)+len(authObs.HMACSHA256.Reasons))
	for reason, count := range authObs.StaticToken.Reasons {
		if count > 0 {
			reasons = append(reasons, fmt.Sprintf("static-token:%s=%d", reason, count))
		}
	}
	for reason, count := range authObs.HMACSHA256.Reasons {
		if count > 0 {
			reasons = append(reasons, fmt.Sprintf("hmac-sha256:%s=%d", reason, count))
		}
	}
	if len(reasons) == 0 {
		return []string{}
	}
	sort.Strings(reasons)
	if len(reasons) > 5 {
		return reasons[:5]
	}
	return reasons
}

func decodeJWTClaims(token string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid jwt segments")
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid jwt payload encoding")
	}
	claims := make(map[string]any)
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, fmt.Errorf("invalid jwt payload json")
	}
	return claims, nil
}

func claimString(claims map[string]any, key string) string {
	if claims == nil {
		return ""
	}
	if value, ok := claims[key].(string); ok {
		return value
	}
	return ""
}

func claimInt64(claims map[string]any, key string) int64 {
	if claims == nil {
		return 0
	}
	switch value := claims[key].(type) {
	case float64:
		return int64(value)
	case json.Number:
		n, err := value.Int64()
		if err == nil {
			return n
		}
	}
	return 0
}

func claimAudience(claims map[string]any, key string) []string {
	if claims == nil {
		return nil
	}
	value, ok := claims[key]
	if !ok {
		return nil
	}
	switch typed := value.(type) {
	case string:
		if typed == "" {
			return nil
		}
		return []string{typed}
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		if len(out) == 0 {
			return nil
		}
		return out
	}
	return nil
}

func healthStatus(ok bool) string {
	if ok {
		return "up"
	}
	return "down"
}
