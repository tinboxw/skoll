# Dashboard JWT Provenance Governance Scorecard Template

## Purpose

Define a monthly governance scorecard template and decision readiness indicators for JWT provenance exception operations.

Use with:

- `docs/community/DASHBOARD_JWT_PROVENANCE_EXCEPTION_METRICS.md`
- `docs/community/DASHBOARD_JWT_PROVENANCE_EXCEPTION_ALERT_PROFILES.md`
- `docs/community/DASHBOARD_JWT_PROVENANCE_EXCEPTION_GOVERNANCE.md`
- `docs/community/DASHBOARD_JWT_PROVENANCE_REVIEW_AUTOMATION.md`

## Monthly Scorecard Template

```markdown
# JWT Provenance Governance Scorecard (YYYY-MM)

## Executive Summary
- reporting_month:
- data_confidence: high | medium | degraded
- overall_readiness: ready | conditional | not_ready
- primary_risk_theme:
- recommended_governance_action:

## Metrics Snapshot
| metric | value | threshold | trend | status |
| --- | --- | --- | --- | --- |
| exceptions_opened |  |  |  |  |
| exceptions_expired |  |  |  |  |
| critical_open_ratio |  | <= 0.30 |  |  |
| revalidation_on_time_ratio |  | >= 0.80 |  |  |
| overdue_postchecks |  | <= 2 |  |  |
| exceptions_active_end_of_month |  | <= 5 |  |  |

## Alert Profile Outcomes
| profile_id | triggers | escalations | final_state | owner |
| --- | --- | --- | --- | --- |

## Exception Lifecycle Summary
- opened:
- closed:
- extended:
- expired:
- median_time_to_close_hours:

## Readiness Indicators
| indicator | result | evidence | blocker |
| --- | --- | --- | --- |
| P1 alerts acknowledged in SLA | pass/fail |  |  |
| P1 alerts mitigation started in SLA | pass/fail |  |  |
| critical exceptions fully approved | pass/fail |  |  |
| post-check backlog under limit | pass/fail |  |  |
| threshold changes with complete evidence | pass/fail |  |  |
| unresolved expiry exceptions | pass/fail |  |  |

## Decision Board
- release_policy: proceed | proceed_with_guardrails | freeze_changes
- approval_required_from:
- follow_up_actions:
- next_review_due:
```

## Decision Readiness Indicator Rules

1. `ready`
   - No failed critical indicators.
   - `critical_open_ratio <= 0.30`.
   - `revalidation_on_time_ratio >= 0.80`.
   - No unresolved expiry exceptions older than 7 days.
2. `conditional`
   - At most one non-critical indicator failed.
   - No unresolved P1 escalation breach.
   - Guardrail actions clearly assigned with due dates.
3. `not_ready`
   - Any critical indicator failed, or
   - Any unresolved P1 escalation breach, or
   - Data confidence is `degraded` with missing core metrics.

Critical indicators:

- P1 acknowledgement SLA compliance
- P1 mitigation-start SLA compliance
- Critical exception approval completeness
- Unresolved expiry exception count

## Suggested Status Mapping

- `status=green`: indicator result `pass`, no immediate action.
- `status=amber`: near-threshold or one-off deviation, owner follow-up required.
- `status=red`: threshold breach or SLA miss, escalation required.

## Evidence Requirements

Each monthly scorecard should link:

- exception metrics payload and source commit
- alert profile summary and escalation logs
- exception decision logs
- threshold-change records
- review publication note

## First Scorecard Sample

- `docs/milestones/JWT_PROVENANCE_GOVERNANCE_SCORECARD_2026-06.md`

## Review Sign-Off Example

Reference implementation:

- `docs/milestones/JWT_PROVENANCE_GOVERNANCE_SCORECARD_2026-06.md` (`Review Sign-Off Example` section)

## Governance

- Scorecards are decision-support artifacts and do not replace human approvals.
- Any template/threshold update must be version-controlled and reviewed.
- If data confidence is `degraded`, scorecard decisions must include manual corroboration notes.
