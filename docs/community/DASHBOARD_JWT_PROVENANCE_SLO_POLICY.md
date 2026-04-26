# Dashboard JWT Provenance SLO Policy

## Purpose

Define SLO dashboard semantics and error-budget policy for JWT provenance export reliability in `jwt_session_bootstrap.provenance_audit_export`.

## SLO Dashboard Fields

`provenance_audit_export.slo_dashboard` currently includes:

- `window`: fixed evaluation window, current value `30d`
- `target_reliability`: SLO target, current value `0.99`
- `observed_reliability`: computed as `1 - error_rate`
- `error_rate`: `(invalid_total + unverified_total) / enabled_total`
- `burn_rate`: `error_rate / (1 - target_reliability)`
- `status`: `healthy` | `at_risk` | `critical`

Status thresholds (current policy):

- `critical`: `burn_rate >= 2`
- `at_risk`: `1 <= burn_rate < 2`
- `healthy`: `burn_rate < 1`

## Error Budget Policy Fields

`provenance_audit_export.error_budget_policy` currently includes:

- `window`
- `budget_ratio`
- `consumed_ratio`
- `remaining_ratio`
- `action`
- `freeze_recommended`
- `escalate_recommended`

Computation:

- `budget_ratio = 1 - target_reliability`
- `consumed_ratio = error_rate / budget_ratio`
- `remaining_ratio = max(0, 1 - consumed_ratio)`

Action mapping (current policy):

- `consumed_ratio >= 2`: `mitigate_immediately`, freeze+escalate
- `1 <= consumed_ratio < 2`: `stabilize_and_reduce_risk`, escalate
- `consumed_ratio < 1`: `monitor`

## Operational Use

1. Use `status` as primary SLO health indicator on dashboards.
2. Use `action` and freeze/escalation flags to drive release gates.
3. Track trend direction, not only point-in-time values, before rollback decisions.
4. Pair with provenance ops runbook for triage procedures.

## Related Docs

- `docs/community/DASHBOARD_JWT_PROVENANCE_AUDIT_EXPORT_GUIDE.md`
- `docs/community/DASHBOARD_JWT_PROVENANCE_OPS_RUNBOOK.md`
