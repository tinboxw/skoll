package app

import (
	"fmt"
	"strings"
	"sync/atomic"
)

const (
	provenanceHintVerificationInvalid    = "verification_state_invalid"
	provenanceHintVerificationUnverified = "verification_state_unverified"
	provenanceHintClaimsNotVerified      = "claims_not_verified"
	provenanceHintClaimsVersionMissing   = "claims_version_missing"
	provenanceHintChainDepthHigh         = "provenance_chain_depth_high"
)

type dashboardJWTProvenanceOperationalMetrics struct {
	ExportsTotal    uint64 `json:"exports_total"`
	EnabledTotal    uint64 `json:"enabled_total"`
	DisabledTotal   uint64 `json:"disabled_total"`
	VerifiedTotal   uint64 `json:"verified_total"`
	UnverifiedTotal uint64 `json:"unverified_total"`
	InvalidTotal    uint64 `json:"invalid_total"`
	AlertHintsTotal uint64 `json:"alert_hints_total"`
}

type dashboardJWTProvenanceSLODashboard struct {
	Window              string  `json:"window"`
	TargetReliability   float64 `json:"target_reliability"`
	ObservedReliability float64 `json:"observed_reliability"`
	ErrorRate           float64 `json:"error_rate"`
	BurnRate            float64 `json:"burn_rate"`
	Status              string  `json:"status"`
}

type dashboardJWTProvenanceErrorBudgetPolicy struct {
	Window              string  `json:"window"`
	BudgetRatio         float64 `json:"budget_ratio"`
	ConsumedRatio       float64 `json:"consumed_ratio"`
	RemainingRatio      float64 `json:"remaining_ratio"`
	Action              string  `json:"action"`
	FreezeRecommended   bool    `json:"freeze_recommended"`
	EscalateRecommended bool    `json:"escalate_recommended"`
}

const (
	provenanceSLOWindow            = "30d"
	provenanceSLOTargetReliability = 0.99
)

var dashboardJWTProvenanceMetrics = struct {
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

func deriveDashboardJWTProvenanceAlertingHints(export dashboardJWTProvenanceAuditExport) []string {
	hints := make([]string, 0, 4)

	switch export.VerificationState {
	case "invalid":
		hints = append(hints, provenanceHintVerificationInvalid)
	case "unverified":
		hints = append(hints, provenanceHintVerificationUnverified)
	}

	if export.Enabled && !export.Verified {
		hints = append(hints, provenanceHintClaimsNotVerified)
	}
	if export.Enabled && export.ClaimsVersion == "" {
		hints = append(hints, provenanceHintClaimsVersionMissing)
	}
	if len(export.SourceProvenance) >= 3 {
		hints = append(hints, provenanceHintChainDepthHigh)
	}

	return hints
}

func observeDashboardJWTProvenanceAuditExport(export dashboardJWTProvenanceAuditExport, hints []string) {
	dashboardJWTProvenanceMetrics.exportsTotal.Add(1)
	if export.Enabled {
		dashboardJWTProvenanceMetrics.enabledTotal.Add(1)
	} else {
		dashboardJWTProvenanceMetrics.disabledTotal.Add(1)
	}
	if export.Verified {
		dashboardJWTProvenanceMetrics.verifiedTotal.Add(1)
	}
	switch export.VerificationState {
	case "unverified":
		dashboardJWTProvenanceMetrics.unverifiedTotal.Add(1)
	case "invalid":
		dashboardJWTProvenanceMetrics.invalidTotal.Add(1)
	}

	for _, hint := range hints {
		dashboardJWTProvenanceMetrics.alertHintsTotal.Add(1)
		switch hint {
		case provenanceHintVerificationInvalid:
			dashboardJWTProvenanceMetrics.hintVerificationInvalid.Add(1)
		case provenanceHintVerificationUnverified:
			dashboardJWTProvenanceMetrics.hintVerificationUnverified.Add(1)
		case provenanceHintClaimsNotVerified:
			dashboardJWTProvenanceMetrics.hintClaimsNotVerified.Add(1)
		case provenanceHintClaimsVersionMissing:
			dashboardJWTProvenanceMetrics.hintClaimsVersionMissing.Add(1)
		case provenanceHintChainDepthHigh:
			dashboardJWTProvenanceMetrics.hintChainDepthHigh.Add(1)
		}
	}
}

func snapshotDashboardJWTProvenanceOperationalMetrics() dashboardJWTProvenanceOperationalMetrics {
	return dashboardJWTProvenanceOperationalMetrics{
		ExportsTotal:    dashboardJWTProvenanceMetrics.exportsTotal.Load(),
		EnabledTotal:    dashboardJWTProvenanceMetrics.enabledTotal.Load(),
		DisabledTotal:   dashboardJWTProvenanceMetrics.disabledTotal.Load(),
		VerifiedTotal:   dashboardJWTProvenanceMetrics.verifiedTotal.Load(),
		UnverifiedTotal: dashboardJWTProvenanceMetrics.unverifiedTotal.Load(),
		InvalidTotal:    dashboardJWTProvenanceMetrics.invalidTotal.Load(),
		AlertHintsTotal: dashboardJWTProvenanceMetrics.alertHintsTotal.Load(),
	}
}

func dashboardJWTProvenanceMetricsPrometheus() string {
	var b strings.Builder

	b.WriteString("# HELP skoll_dashboard_jwt_provenance_exports_total Total number of dashboard JWT provenance exports by enabled state.\n")
	b.WriteString("# TYPE skoll_dashboard_jwt_provenance_exports_total counter\n")
	writeProvenanceCounter(&b, "skoll_dashboard_jwt_provenance_exports_total{enabled=\"true\"}", dashboardJWTProvenanceMetrics.enabledTotal.Load())
	writeProvenanceCounter(&b, "skoll_dashboard_jwt_provenance_exports_total{enabled=\"false\"}", dashboardJWTProvenanceMetrics.disabledTotal.Load())

	b.WriteString("# HELP skoll_dashboard_jwt_provenance_verification_states_total Total number of dashboard JWT provenance exports by verification state.\n")
	b.WriteString("# TYPE skoll_dashboard_jwt_provenance_verification_states_total counter\n")
	writeProvenanceCounter(&b, "skoll_dashboard_jwt_provenance_verification_states_total{state=\"invalid\"}", dashboardJWTProvenanceMetrics.invalidTotal.Load())
	writeProvenanceCounter(&b, "skoll_dashboard_jwt_provenance_verification_states_total{state=\"unverified\"}", dashboardJWTProvenanceMetrics.unverifiedTotal.Load())
	writeProvenanceCounter(&b, "skoll_dashboard_jwt_provenance_verification_states_total{state=\"verified\"}", dashboardJWTProvenanceMetrics.verifiedTotal.Load())

	b.WriteString("# HELP skoll_dashboard_jwt_provenance_alert_hints_total Total number of emitted dashboard JWT provenance alerting hints by hint type.\n")
	b.WriteString("# TYPE skoll_dashboard_jwt_provenance_alert_hints_total counter\n")
	writeProvenanceCounter(&b, fmt.Sprintf("skoll_dashboard_jwt_provenance_alert_hints_total{hint=%q}", provenanceHintVerificationInvalid), dashboardJWTProvenanceMetrics.hintVerificationInvalid.Load())
	writeProvenanceCounter(&b, fmt.Sprintf("skoll_dashboard_jwt_provenance_alert_hints_total{hint=%q}", provenanceHintVerificationUnverified), dashboardJWTProvenanceMetrics.hintVerificationUnverified.Load())
	writeProvenanceCounter(&b, fmt.Sprintf("skoll_dashboard_jwt_provenance_alert_hints_total{hint=%q}", provenanceHintClaimsNotVerified), dashboardJWTProvenanceMetrics.hintClaimsNotVerified.Load())
	writeProvenanceCounter(&b, fmt.Sprintf("skoll_dashboard_jwt_provenance_alert_hints_total{hint=%q}", provenanceHintClaimsVersionMissing), dashboardJWTProvenanceMetrics.hintClaimsVersionMissing.Load())
	writeProvenanceCounter(&b, fmt.Sprintf("skoll_dashboard_jwt_provenance_alert_hints_total{hint=%q}", provenanceHintChainDepthHigh), dashboardJWTProvenanceMetrics.hintChainDepthHigh.Load())

	return b.String()
}

func writeProvenanceCounter(b *strings.Builder, metric string, total uint64) {
	if total == 0 {
		return
	}
	b.WriteString(fmt.Sprintf("%s %d\n", metric, total))
}

func resetDashboardJWTProvenanceMetricsForTest() {
	dashboardJWTProvenanceMetrics = struct {
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

func buildDashboardJWTProvenanceSLODashboard(metrics dashboardJWTProvenanceOperationalMetrics) dashboardJWTProvenanceSLODashboard {
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

	return dashboardJWTProvenanceSLODashboard{
		Window:              provenanceSLOWindow,
		TargetReliability:   provenanceSLOTargetReliability,
		ObservedReliability: observedReliability,
		ErrorRate:           errorRate,
		BurnRate:            burnRate,
		Status:              status,
	}
}

func buildDashboardJWTProvenanceErrorBudgetPolicy(slo dashboardJWTProvenanceSLODashboard) dashboardJWTProvenanceErrorBudgetPolicy {
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

	return dashboardJWTProvenanceErrorBudgetPolicy{
		Window:              slo.Window,
		BudgetRatio:         budgetRatio,
		ConsumedRatio:       consumedRatio,
		RemainingRatio:      remainingRatio,
		Action:              action,
		FreezeRecommended:   freeze,
		EscalateRecommended: escalate,
	}
}
