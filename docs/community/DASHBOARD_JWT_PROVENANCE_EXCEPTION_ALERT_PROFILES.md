# Dashboard JWT Provenance Exception Alert Profiles

## Purpose

Define governance metric alert profiles and escalation thresholds for JWT provenance exception operations.

Use with:

- `docs/community/DASHBOARD_JWT_PROVENANCE_EXCEPTION_METRICS.md`
- `docs/community/DASHBOARD_JWT_PROVENANCE_EXCEPTION_GOVERNANCE.md`
- `docs/community/DASHBOARD_JWT_PROVENANCE_REVIEW_AUTOMATION.md`
- `docs/community/DASHBOARD_JWT_PROVENANCE_SLO_POLICY.md`
- `docs/community/DASHBOARD_JWT_PROVENANCE_GOVERNANCE_SCORECARD_TEMPLATE.md`

## Alert Profile Matrix

| profile_id | primary_signal | trigger_condition | severity | initial_owner | dedup_window | evaluation_window |
| --- | --- | --- | --- | --- | --- | --- |
| `jwt-exc-expired-growth` | `exceptions_expired` | month-over-month increase > 50% and current month >= 2 | P2 | `review_primary` | 24h | monthly |
| `jwt-exc-critical-ratio` | `critical_open_ratio` | > 0.30 for 2 consecutive months and opened >= 3 | P1 | `security_reviewer` | 12h | monthly |
| `jwt-exc-revalidation-lag` | `revalidation_on_time_ratio` | < 0.80 and `total_revalidations >= 5` | P2 | `review_secondary` | 24h | monthly |
| `jwt-exc-active-backlog` | `exceptions_active_end_of_month` | increasing for 3 consecutive months and end-of-month >= 5 | P3 | `review_primary` | 72h | monthly |
| `jwt-exc-overdue-postcheck` | `overdue_postchecks` | >= 3 in a month | P1 | `service_owner` | 12h | monthly |

## Escalation Threshold Policy

### Acknowledgement SLA

- P1: acknowledge within 30 minutes.
- P2: acknowledge within 4 hours.
- P3: acknowledge within 1 business day.

### Mitigation Start SLA

- P1: mitigation action started within 2 hours.
- P2: mitigation action started within 1 business day.
- P3: mitigation action started within 3 business days.

### Escalation Path

1. Primary owner acknowledgement timeout:
   - Escalate to `service_owner` and `security_reviewer`.
2. Mitigation start timeout:
   - Escalate to on-call manager and add change freeze recommendation.
3. Repeated profile trigger (same profile in 2 consecutive periods):
   - Require governance review meeting and written corrective plan.

## Notification Routing

- P1: paging + chat + issue ticket.
- P2: chat + issue ticket.
- P3: issue ticket and monthly review queue.

Required metadata for each alert event:

- `profile_id`
- `month`
- `trigger_value`
- `threshold`
- `risk_level`
- `owner`
- `ack_deadline`
- `mitigation_deadline`

## Suppression and Freeze Rules

1. Temporary suppression requires explicit approval and expiry timestamp.
2. P1 alerts cannot be suppressed beyond 24 hours.
3. If the same P1 profile is suppressed twice in a quarter, enforce threshold review and change freeze until sign-off.

## Change Control for Profiles

- Any threshold/profile adjustment must create a record in:
  - `docs/community/DASHBOARD_JWT_PROVENANCE_THRESHOLD_CHANGE_LOG.md`
- Include rationale, expected impact, and rollback criteria.
- Review profile thresholds at least once per quarter.

## Sample Monthly Alert Summary

```markdown
# JWT Provenance Exception Alert Summary (2026-06)

| profile_id | triggers | max_observed | threshold | escalations | final_state |
| --- | --- | --- | --- | --- | --- |
| jwt-exc-critical-ratio | 1 | 0.36 | 0.30 | 1 | mitigated |
| jwt-exc-revalidation-lag | 1 | 0.72 | 0.80 | 0 | monitoring |
```

## Governance

- Alert profiles guide operational response, but final governance decisions remain human-approved.
- Alert thresholds must be traceable, version-controlled, and auditable.
- If source metrics have degraded confidence, alerts are marked `advisory_only` and require manual corroboration.
