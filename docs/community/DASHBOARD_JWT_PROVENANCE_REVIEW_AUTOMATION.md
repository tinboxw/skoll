# Dashboard JWT Provenance Monthly Review Automation

## Purpose

Define a practical automation checklist and ownership rotation guidance for monthly JWT provenance threshold reviews.

This guide extends:

- `docs/community/DASHBOARD_JWT_PROVENANCE_THRESHOLD_CHANGE_LOG.md`
- `docs/community/DASHBOARD_JWT_PROVENANCE_BASELINE_RECALIBRATION.md`
- `docs/community/DASHBOARD_JWT_PROVENANCE_RECALIBRATION_EVIDENCE_TEMPLATE.md`

## Automation Scope

Monthly automation focuses on:

- collecting threshold-change records
- validating evidence/approval completeness
- generating monthly archive summaries
- assigning post-check owners and due dates

It does not auto-approve threshold changes.

## Monthly Automation Checklist

1. Trigger and window
   - Run on the first business day of each month.
   - Scope previous calendar month data.
2. Record discovery
   - Enumerate all `jwt-prov-threshold-YYYYMM-###` records.
   - Ensure records map to archive month.
3. Completeness validation
   - Verify each record has evidence link, approval identities, and post-check due.
4. Risk guard validation
   - Flag relax changes without rollback criteria.
   - Flag warning+critical simultaneous relax attempts.
5. Archive generation
   - Render monthly summary and change table.
   - Output archive under `docs/milestones/`.
6. Owner assignment
   - Assign follow-up owner for each open post-check.
7. Review publication
   - Open review issue/note with archive link and unresolved items.

## Suggested Automation Output Contract

```yaml
month: 2026-04
records_total: 2
records_incomplete: 0
risk_flags: 0
archive_file: docs/milestones/JWT_PROVENANCE_THRESHOLD_CHANGE_ARCHIVE_2026-04.md
open_followups:
  - log_id: jwt-prov-threshold-202604-002
    owner: sre-weekly-primary
    due: 2026-05-08
status: ready_for_review
```

## Ownership Rotation Guidance

Roles:

- `review_primary`: executes monthly automation checks
- `review_secondary`: validates risk flags and approval completeness
- `security_reviewer`: validates relax-change risk posture

Rotation policy:

1. Rotate `review_primary` monthly.
2. `review_secondary` must differ from primary.
3. `security_reviewer` rotates every quarter or incident cycle.
4. No owner can approve their own relax-change proposal alone.

## Rotation Roster Template

```markdown
# JWT Provenance Review Rotation (YYYY-QX)

| Month | review_primary | review_secondary | security_reviewer |
| --- | --- | --- | --- |
| 2026-04 | sre-a | sre-b | sec-a |
| 2026-05 | sre-b | sre-c | sec-a |
| 2026-06 | sre-c | sre-a | sec-b |
```

## Escalation Rules

1. Any incomplete critical-level change record blocks monthly sign-off.
2. Any missing security approval on relax changes triggers security escalation.
3. Any unresolved post-check beyond D+14 escalates to service owner.

## Governance

- Automation outputs are evidence, not approvals.
- Monthly archive and rotation roster updates must be version-controlled.
- Exceptions must include reason, approver, and expiry date.

## Quarterly Roster Sample

- `docs/milestones/JWT_PROVENANCE_REVIEW_ROTATION_2026-Q2.md`

## Escalation Handoff Template

Reference implementation:

- `docs/milestones/JWT_PROVENANCE_REVIEW_ROTATION_2026-Q2.md` (`Escalation Handoff Template` section)
