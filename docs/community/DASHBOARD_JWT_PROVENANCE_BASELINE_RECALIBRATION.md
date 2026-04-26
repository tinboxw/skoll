# Dashboard JWT Provenance Baseline Recalibration

## Purpose

Define a periodic workflow and review cadence to recalibrate dashboard JWT provenance alert baselines without weakening security posture.

This document complements:

- `docs/community/DASHBOARD_JWT_PROVENANCE_SLO_POLICY.md`
- `docs/community/DASHBOARD_JWT_PROVENANCE_SLO_ALERT_RULES.md`
- `docs/community/DASHBOARD_JWT_PROVENANCE_OPS_RUNBOOK.md`
- `docs/community/DASHBOARD_JWT_PROVENANCE_RECALIBRATION_EVIDENCE_TEMPLATE.md`

## Recalibration Scope

Recalibration applies to baseline-driven alert thresholds for:

- Invalid verification rate
- Unverified drift rate
- Claims version missing ratio
- Provenance chain depth drift ratio

It does not change authorization decisions or trust boundaries.

## Review Cadence

1. Weekly light review:
   - Check 7d trend direction and major outliers.
2. Monthly baseline review:
   - Recompute baseline windows using latest 30d production data.
3. Quarterly policy review:
   - Re-validate threshold suitability against seasonality and release patterns.
4. Event-driven review:
   - Trigger immediately after significant auth/middleware architecture changes.

## Recalibration Workflow

1. Collect evidence window:
   - Export the latest 30d metric samples for all provenance signals.
2. Segment traffic:
   - Split by canary/stable and major route groups to avoid blended false confidence.
3. Compute candidate thresholds:
   - Derive P50/P90/P95 for each ratio and compare with current threshold.
4. Safety check:
   - Reject candidate if it weakens critical detection sensitivity.
5. Canary trial:
   - Apply candidate threshold to canary alert route for at least 7d.
6. Review board sign-off:
   - Require service owner plus security on-call approval.
7. Rollout:
   - Promote threshold to stable and annotate dashboard with change marker.
8. Post-rollout verify:
   - Track false-positive and false-negative changes for another 7d.

## Decision Rules

1. Keep or tighten threshold when burn-rate critical alerts still correlate with real incidents.
2. Relax threshold only when sustained false positives are evidenced across at least two monthly windows.
3. Never relax both warning and critical thresholds in the same rollout.
4. Any threshold relaxation requires rollback criteria documented in advance.

## Evidence Checklist

For each threshold change, capture:

- Current threshold and proposed threshold
- Baseline period and sample size
- Estimated alert volume delta
- Expected impact on `at_risk` and `critical` detection latency
- Canary outcome summary
- Reviewer and approval timestamp

## Rollback Triggers

Rollback to previous baseline when one of the following is true:

1. Critical incident occurs without matching critical alert.
2. Canary/stable divergence exceeds agreed tolerance for 24h.
3. Error-budget consumed ratio increases materially after threshold relaxation.

## Governance

- All baseline changes must be version-controlled.
- Link each change to a milestone record or incident postmortem.
- Keep CN/EN README indexes in sync with new process docs.
