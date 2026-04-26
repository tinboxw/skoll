# Dashboard JWT Provenance Exception Metrics

## Purpose

Define observability metrics and monthly trend dashboard fields for JWT provenance exception governance.

This document complements:

- `docs/community/DASHBOARD_JWT_PROVENANCE_EXCEPTION_GOVERNANCE.md`
- `docs/community/DASHBOARD_JWT_PROVENANCE_REVIEW_AUTOMATION.md`
- `docs/community/DASHBOARD_JWT_PROVENANCE_SLO_POLICY.md`

## Metric Definitions

Recommended counters/gauges:

- `skoll_dashboard_jwt_exception_open_total`
  - Total created exception records.
- `skoll_dashboard_jwt_exception_closed_total`
  - Total closed exception records.
- `skoll_dashboard_jwt_exception_extended_total`
  - Total extension decisions.
- `skoll_dashboard_jwt_exception_expired_total`
  - Exceptions reaching expiry without valid closure.
- `skoll_dashboard_jwt_exception_active`
  - Current active exceptions (gauge).
- `skoll_dashboard_jwt_exception_overdue_postcheck_total`
  - Post-check overdue occurrences.

Labels (recommended):

- `risk_level`: `low|medium|high|critical`
- `exception_type`: taxonomy from governance matrix
- `decision`: `close|extend_24h|extend_72h|rollback|freeze`

## Monthly Trend Dashboard Fields

For monthly reporting payloads, include:

- `month`
- `exceptions_opened`
- `exceptions_closed`
- `exceptions_extended`
- `exceptions_expired`
- `exceptions_active_end_of_month`
- `overdue_postchecks`
- `critical_open_ratio`
- `median_time_to_close_hours`
- `revalidation_on_time_ratio`
- `rollback_decisions`
- `freeze_decisions`

Derived ratios:

- `critical_open_ratio = critical_opened / max(1, exceptions_opened)`
- `revalidation_on_time_ratio = on_time_revalidations / max(1, total_revalidations)`

## Suggested Monthly Trend Payload Example

```json
{
  "month": "2026-05",
  "exceptions_opened": 4,
  "exceptions_closed": 3,
  "exceptions_extended": 2,
  "exceptions_expired": 1,
  "exceptions_active_end_of_month": 2,
  "overdue_postchecks": 1,
  "critical_open_ratio": 0.25,
  "median_time_to_close_hours": 38,
  "revalidation_on_time_ratio": 0.75,
  "rollback_decisions": 1,
  "freeze_decisions": 0
}
```

## Alerting Hints for Metrics

1. `exceptions_expired` month-over-month increase > 50%: investigate process debt.
2. `critical_open_ratio` sustained > 0.3 for 2 months: trigger governance review.
3. `revalidation_on_time_ratio` < 0.8: enforce ownership correction.
4. `exceptions_active_end_of_month` rising 3 months consecutively: review exception entry criteria.

## Data Quality Rules

- Metrics must be generated from version-controlled records.
- Each decision event must map to a stable `exception_id`.
- Backfilled corrections must preserve original timestamps and append correction metadata.

## Governance

- Trend dashboards are advisory; approval remains a human decision.
- Any threshold for governance metrics must be reviewed quarterly.
- If metric sources are incomplete, monthly report must declare data confidence as `degraded`.
