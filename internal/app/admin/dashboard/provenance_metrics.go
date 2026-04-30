package dashboard

import (
	"fmt"
	"strings"
	"sync/atomic"
)

const (
	JWTProvenanceHintVerificationInvalid    = "verification_state_invalid"
	JWTProvenanceHintVerificationUnverified = "verification_state_unverified"
	JWTProvenanceHintClaimsNotVerified      = "claims_not_verified"
	JWTProvenanceHintClaimsVersionMissing   = "claims_version_missing"
	JWTProvenanceHintChainDepthHigh         = "provenance_chain_depth_high"
)

const (
	provenanceSLOWindow            = "30d"
	provenanceSLOTargetReliability = 0.99
)

var jwtProvenanceMetrics = struct {
	exportsTotal    atomic.Uint64
	enabledTotal    atomic.Uint64
	disabledTotal   atomic.Uint64
	verifiedTotal   atomic.Uint64
	unverifiedTotal atomic.Uint64
	invalidTotal    atomic.Uint64
	alertHintsTotal atomic.Uint64

	hintVerificationInvalid    atomic.Uint64
	hintVerificationUnverified atomic.Uint64
	hintClaimsNotVerified      atomic.Uint64
	hintClaimsVersionMissing   atomic.Uint64
	hintChainDepthHigh         atomic.Uint64
}{}

func DeriveJWTProvenanceAlertingHints(export JWTProvenanceAuditExport) []string {
	hints := make([]string, 0, 4)

	switch export.VerificationState {
	case "invalid":
		hints = append(hints, JWTProvenanceHintVerificationInvalid)
	case "unverified":
		hints = append(hints, JWTProvenanceHintVerificationUnverified)
	}

	if export.Enabled && !export.Verified {
		hints = append(hints, JWTProvenanceHintClaimsNotVerified)
	}
	if export.Enabled && export.ClaimsVersion == "" {
		hints = append(hints, JWTProvenanceHintClaimsVersionMissing)
	}
	if len(export.SourceProvenance) >= 3 {
		hints = append(hints, JWTProvenanceHintChainDepthHigh)
	}

	return hints
}

func ObserveJWTProvenanceAuditExport(export JWTProvenanceAuditExport, hints []string) {
	jwtProvenanceMetrics.exportsTotal.Add(1)
	if export.Enabled {
		jwtProvenanceMetrics.enabledTotal.Add(1)
	} else {
		jwtProvenanceMetrics.disabledTotal.Add(1)
	}
	if export.Verified {
		jwtProvenanceMetrics.verifiedTotal.Add(1)
	}
	switch export.VerificationState {
	case "unverified":
		jwtProvenanceMetrics.unverifiedTotal.Add(1)
	case "invalid":
		jwtProvenanceMetrics.invalidTotal.Add(1)
	}

	for _, hint := range hints {
		jwtProvenanceMetrics.alertHintsTotal.Add(1)
		switch hint {
		case JWTProvenanceHintVerificationInvalid:
			jwtProvenanceMetrics.hintVerificationInvalid.Add(1)
		case JWTProvenanceHintVerificationUnverified:
			jwtProvenanceMetrics.hintVerificationUnverified.Add(1)
		case JWTProvenanceHintClaimsNotVerified:
			jwtProvenanceMetrics.hintClaimsNotVerified.Add(1)
		case JWTProvenanceHintClaimsVersionMissing:
			jwtProvenanceMetrics.hintClaimsVersionMissing.Add(1)
		case JWTProvenanceHintChainDepthHigh:
			jwtProvenanceMetrics.hintChainDepthHigh.Add(1)
		}
	}
}

func SnapshotJWTProvenanceOperationalMetrics() JWTProvenanceOperationalMetrics {
	return JWTProvenanceOperationalMetrics{
		ExportsTotal:    jwtProvenanceMetrics.exportsTotal.Load(),
		EnabledTotal:    jwtProvenanceMetrics.enabledTotal.Load(),
		DisabledTotal:   jwtProvenanceMetrics.disabledTotal.Load(),
		VerifiedTotal:   jwtProvenanceMetrics.verifiedTotal.Load(),
		UnverifiedTotal: jwtProvenanceMetrics.unverifiedTotal.Load(),
		InvalidTotal:    jwtProvenanceMetrics.invalidTotal.Load(),
		AlertHintsTotal: jwtProvenanceMetrics.alertHintsTotal.Load(),
	}
}

func JWTProvenanceMetricsPrometheus() string {
	var b strings.Builder

	b.WriteString("# HELP skoll_dashboard_jwt_provenance_exports_total Total number of dashboard JWT provenance exports by enabled state.\n")
	b.WriteString("# TYPE skoll_dashboard_jwt_provenance_exports_total counter\n")
	writeProvenanceCounter(&b, "skoll_dashboard_jwt_provenance_exports_total{enabled=\"true\"}", jwtProvenanceMetrics.enabledTotal.Load())
	writeProvenanceCounter(&b, "skoll_dashboard_jwt_provenance_exports_total{enabled=\"false\"}", jwtProvenanceMetrics.disabledTotal.Load())

	b.WriteString("# HELP skoll_dashboard_jwt_provenance_verification_states_total Total number of dashboard JWT provenance exports by verification state.\n")
	b.WriteString("# TYPE skoll_dashboard_jwt_provenance_verification_states_total counter\n")
	writeProvenanceCounter(&b, "skoll_dashboard_jwt_provenance_verification_states_total{state=\"invalid\"}", jwtProvenanceMetrics.invalidTotal.Load())
	writeProvenanceCounter(&b, "skoll_dashboard_jwt_provenance_verification_states_total{state=\"unverified\"}", jwtProvenanceMetrics.unverifiedTotal.Load())
	writeProvenanceCounter(&b, "skoll_dashboard_jwt_provenance_verification_states_total{state=\"verified\"}", jwtProvenanceMetrics.verifiedTotal.Load())

	b.WriteString("# HELP skoll_dashboard_jwt_provenance_alert_hints_total Total number of emitted dashboard JWT provenance alerting hints by hint type.\n")
	b.WriteString("# TYPE skoll_dashboard_jwt_provenance_alert_hints_total counter\n")
	writeProvenanceCounter(&b, fmt.Sprintf("skoll_dashboard_jwt_provenance_alert_hints_total{hint=%q}", JWTProvenanceHintVerificationInvalid), jwtProvenanceMetrics.hintVerificationInvalid.Load())
	writeProvenanceCounter(&b, fmt.Sprintf("skoll_dashboard_jwt_provenance_alert_hints_total{hint=%q}", JWTProvenanceHintVerificationUnverified), jwtProvenanceMetrics.hintVerificationUnverified.Load())
	writeProvenanceCounter(&b, fmt.Sprintf("skoll_dashboard_jwt_provenance_alert_hints_total{hint=%q}", JWTProvenanceHintClaimsNotVerified), jwtProvenanceMetrics.hintClaimsNotVerified.Load())
	writeProvenanceCounter(&b, fmt.Sprintf("skoll_dashboard_jwt_provenance_alert_hints_total{hint=%q}", JWTProvenanceHintClaimsVersionMissing), jwtProvenanceMetrics.hintClaimsVersionMissing.Load())
	writeProvenanceCounter(&b, fmt.Sprintf("skoll_dashboard_jwt_provenance_alert_hints_total{hint=%q}", JWTProvenanceHintChainDepthHigh), jwtProvenanceMetrics.hintChainDepthHigh.Load())

	return b.String()
}

func ResetJWTProvenanceMetricsForTest() {
	jwtProvenanceMetrics = struct {
		exportsTotal    atomic.Uint64
		enabledTotal    atomic.Uint64
		disabledTotal   atomic.Uint64
		verifiedTotal   atomic.Uint64
		unverifiedTotal atomic.Uint64
		invalidTotal    atomic.Uint64
		alertHintsTotal atomic.Uint64

		hintVerificationInvalid    atomic.Uint64
		hintVerificationUnverified atomic.Uint64
		hintClaimsNotVerified      atomic.Uint64
		hintClaimsVersionMissing   atomic.Uint64
		hintChainDepthHigh         atomic.Uint64
	}{}
}

func BuildJWTProvenanceSLODashboard(metrics JWTProvenanceOperationalMetrics) JWTProvenanceSLODashboard {
	total := float64(metrics.EnabledTotal)
	errorCount := float64(metrics.InvalidTotal + metrics.UnverifiedTotal)
	observedReliability := 1.0
	errorRate := 0.0
	if total > 0 {
		errorRate = errorCount / total
		observedReliability = 1.0 - errorRate
	}
	if observedReliability < 0 {
		observedReliability = 0
	}

	errorBudget := 1.0 - provenanceSLOTargetReliability
	burnRate := 0.0
	if errorBudget > 0 {
		burnRate = errorRate / errorBudget
	}

	status := "healthy"
	switch {
	case burnRate >= 2:
		status = "critical"
	case burnRate >= 1:
		status = "at_risk"
	}

	return JWTProvenanceSLODashboard{
		Window:              provenanceSLOWindow,
		TargetReliability:   provenanceSLOTargetReliability,
		ObservedReliability: observedReliability,
		ErrorRate:           errorRate,
		BurnRate:            burnRate,
		Status:              status,
	}
}

func BuildJWTProvenanceErrorBudgetPolicy(slo JWTProvenanceSLODashboard) JWTProvenanceErrorBudgetPolicy {
	budgetRatio := 1.0 - slo.TargetReliability
	consumedRatio := 0.0
	if budgetRatio > 0 {
		consumedRatio = slo.ErrorRate / budgetRatio
	}
	remainingRatio := 1.0 - consumedRatio
	if remainingRatio < 0 {
		remainingRatio = 0
	}

	action := "monitor"
	freeze := false
	escalate := false
	switch {
	case consumedRatio >= 2:
		action = "mitigate_immediately"
		freeze = true
		escalate = true
	case consumedRatio >= 1:
		action = "stabilize_and_reduce_risk"
		escalate = true
	}

	return JWTProvenanceErrorBudgetPolicy{
		Window:              slo.Window,
		BudgetRatio:         budgetRatio,
		ConsumedRatio:       consumedRatio,
		RemainingRatio:      remainingRatio,
		Action:              action,
		FreezeRecommended:   freeze,
		EscalateRecommended: escalate,
	}
}

func writeProvenanceCounter(b *strings.Builder, metric string, total uint64) {
	if total == 0 {
		return
	}
	b.WriteString(fmt.Sprintf("%s %d\n", metric, total))
}
