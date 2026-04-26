# JWT Provenance Governance Scorecard (2026-06)

## Executive Summary

- reporting_month: `2026-06`
- data_confidence: `high`
- overall_readiness: `conditional`
- primary_risk_theme: elevated critical exception ratio
- recommended_governance_action: proceed with guardrails and weekly checkpoint

## Metrics Snapshot

| metric | value | threshold | trend | status |
| --- | --- | --- | --- | --- |
| exceptions_opened | 6 | n/a | up | amber |
| exceptions_expired | 1 | <= 1 | flat | green |
| critical_open_ratio | 0.33 | <= 0.30 | up | red |
| revalidation_on_time_ratio | 0.86 | >= 0.80 | up | green |
| overdue_postchecks | 2 | <= 2 | down | green |
| exceptions_active_end_of_month | 4 | <= 5 | flat | green |

## Alert Profile Outcomes

| profile_id | triggers | escalations | final_state | owner |
| --- | --- | --- | --- | --- |
| jwt-exc-critical-ratio | 1 | 1 | mitigated | security_reviewer |
| jwt-exc-revalidation-lag | 0 | 0 | n/a | review_secondary |
| jwt-exc-overdue-postcheck | 0 | 0 | n/a | service_owner |

## Exception Lifecycle Summary

- opened: 6
- closed: 5
- extended: 2
- expired: 1
- median_time_to_close_hours: 41

## Readiness Indicators

| indicator | result | evidence | blocker |
| --- | --- | --- | --- |
| P1 alerts acknowledged in SLA | pass | escalation log 2026-06-09 | none |
| P1 alerts mitigation started in SLA | pass | mitigation start 2026-06-09 11:20 UTC | none |
| critical exceptions fully approved | pass | exception approvals IDs EXC-2026-06-001/004 | none |
| post-check backlog under limit | pass | backlog report week 26 | none |
| threshold changes with complete evidence | pass | threshold log ids 202606-001..003 | none |
| unresolved expiry exceptions | fail | EXC-2026-05-009 remains open | close by 2026-07-05 |

## Decision Board

- release_policy: `proceed_with_guardrails`
- approval_required_from: service_owner, security_reviewer
- follow_up_actions:
  - close EXC-2026-05-009 and publish closure note
  - hold weekly governance sync until critical_open_ratio <= 0.30
  - run threshold recheck after next monthly archive
- next_review_due: `2026-07-05`

## Review Sign-Off Example

| role | reviewer | decision | signed_at | notes |
| --- | --- | --- | --- | --- |
| review_primary | sre-a | approve_with_guardrails | 2026-07-01T09:00:00Z | monitor critical ratio weekly |
| review_secondary | sre-b | approve | 2026-07-01T09:20:00Z | backlog within range |
| security_reviewer | sec-a | approve_with_guardrails | 2026-07-01T09:35:00Z | unresolved expiry must be closed |
| service_owner | platform-owner | approve | 2026-07-01T10:00:00Z | action owners assigned |

## Evidence Links

- scorecard template: `docs/community/DASHBOARD_JWT_PROVENANCE_GOVERNANCE_SCORECARD_TEMPLATE.md`
- metrics source: `docs/community/DASHBOARD_JWT_PROVENANCE_EXCEPTION_METRICS.md`
- alert profiles: `docs/community/DASHBOARD_JWT_PROVENANCE_EXCEPTION_ALERT_PROFILES.md`
- exception governance matrix: `docs/community/DASHBOARD_JWT_PROVENANCE_EXCEPTION_GOVERNANCE.md`
