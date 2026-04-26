# Dashboard JWT Provenance Exception Governance

## Purpose

Define an exception governance matrix and expiry revalidation workflow for JWT provenance threshold controls.

This document is used with:

- `docs/community/DASHBOARD_JWT_PROVENANCE_REVIEW_AUTOMATION.md`
- `docs/community/DASHBOARD_JWT_PROVENANCE_THRESHOLD_CHANGE_LOG.md`
- `docs/community/DASHBOARD_JWT_PROVENANCE_RECALIBRATION_EVIDENCE_TEMPLATE.md`

## Exception Governance Matrix

| Exception Type | Example | Risk Level | Required Approvals | Max Validity | Mandatory Safeguards |
| --- | --- | --- | --- | --- | --- |
| Missing non-critical evidence field | late post-check owner assignment | Medium | review_primary + review_secondary | 14d | owner assigned within 2 business days |
| Relax change without complete canary window | canary ran 4d instead of 7d | High | service owner + security reviewer | 7d | rollback criteria tightened + daily monitoring |
| Missing security approval on relax change | approval pending during incident | Critical | service owner + security reviewer + on-call manager | 72h | relax change freeze until approval completed |
| Overdue post-check | D+7 verification not completed | High | review_primary + service owner | 7d | escalated tracking and sign-off block |
| Archive publication delay | monthly archive not published on time | Low | review_primary | 5 business days | publish with incident note |

## Exception Record Template

```markdown
# JWT Provenance Exception Record

## Identity
- exception_id:
- created_at:
- owner:
- related_log_ids:

## Exception Details
- exception_type:
- reason:
- scope:
- risk_level:

## Approvals
- required_approvers:
- approvals_received:
- approved_until:

## Safeguards
- active_controls:
- rollback_or_freeze_condition:

## Revalidation Plan
- revalidation_due:
- revalidation_owner:
- revalidation_criteria:
- closure_conditions:
```

## Expiry Revalidation Workflow

1. Register exception
   - Create exception record and attach related threshold/evidence links.
2. Validate validity window
   - Ensure requested validity does not exceed matrix `Max Validity`.
3. Enforce safeguards
   - Activate required controls before exception becomes effective.
4. Pre-expiry reminder
   - Notify owners and approvers 48h before `approved_until`.
5. Revalidation review
   - Decide one of: close, extend (new approval), or rollback/freeze.
6. Closure and archive
   - Record outcome and link from monthly archive or milestone note.

## Revalidation Decision Rules

1. Extend only if risk trend is stable or reduced.
2. Never extend Critical exceptions without explicit security reviewer approval.
3. Any second extension requires service owner written rationale.
4. If revalidation evidence is missing at expiry, default action is rollback/freeze.

## Sample Exception Records

- `docs/milestones/JWT_PROVENANCE_EXCEPTION_RECORDS_2026-05.md`

## Revalidation Decision Log Template

```markdown
| decision_id | exception_id | decided_at | decision | rationale | approvers | next_due |
| --- | --- | --- | --- | --- | --- | --- |
```

Decision values:

- `close`
- `extend_24h`
- `extend_72h`
- `rollback`
- `freeze`

## Operational Metrics Suggestions

- `exceptions_open_total`
- `exceptions_expired_total`
- `exceptions_extended_total`
- `exceptions_closed_total`

## Governance

- Exceptions are temporary and must be traceable.
- Every extension is a new decision event with new timestamp and approver set.
- Exception records and outcomes must be version-controlled.
