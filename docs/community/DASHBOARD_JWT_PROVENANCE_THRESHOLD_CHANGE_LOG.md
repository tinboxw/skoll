# Dashboard JWT Provenance Threshold Change Log

## Purpose

Define a standard threshold-change log format and monthly review archive workflow for JWT provenance alerts.

Use with:

- `docs/community/DASHBOARD_JWT_PROVENANCE_SLO_POLICY.md`
- `docs/community/DASHBOARD_JWT_PROVENANCE_BASELINE_RECALIBRATION.md`
- `docs/community/DASHBOARD_JWT_PROVENANCE_RECALIBRATION_EVIDENCE_TEMPLATE.md`
- `docs/community/DASHBOARD_JWT_PROVENANCE_REVIEW_AUTOMATION.md`
- `docs/community/DASHBOARD_JWT_PROVENANCE_EXCEPTION_GOVERNANCE.md`

## Log Record Format

Each threshold change must create one immutable log record.

Required fields:

- `log_id`: unique identifier, recommended `jwt-prov-threshold-YYYYMM-###`
- `changed_at`: UTC timestamp
- `changed_by`: owner and team
- `signal_type`: `invalid_rate | unverified_rate | claims_version_missing | chain_depth_drift`
- `alert_level`: `warning | high | critical`
- `old_threshold`
- `new_threshold`
- `change_type`: `tighten | relax`
- `reason`
- `evidence_link`
- `approvals`: service owner + security on-call
- `effective_from`
- `rollback_criteria`
- `post_check_due`: usually D+7

## Example Record

```yaml
log_id: jwt-prov-threshold-202604-001
changed_at: 2026-04-26T09:30:00Z
changed_by: platform-observability
signal_type: invalid_rate
alert_level: high
old_threshold: 0.020
new_threshold: 0.018
change_type: tighten
reason: "Increased false negatives in canary and delayed incident pickup"
evidence_link: docs/milestones/M17-dashboard-jwt-session-provenance-recalibration-evidence-template.md
approvals:
  - service-owner: approved
  - security-oncall: approved
effective_from: 2026-04-27T00:00:00Z
rollback_criteria: "If warning volume increases > 2x for 7d without incident correlation"
post_check_due: 2026-05-04
```

## Monthly Review Archive Workflow

1. Collect records:
   - Gather all threshold log records in the target month.
2. Validate completeness:
   - Ensure every record has approvals, evidence link, and post-check status.
3. Produce monthly summary:
   - Aggregate change counts by signal and by tighten/relax type.
4. Risk review:
   - Confirm no critical detection blind spots were introduced.
5. Archive snapshot:
   - Create a monthly archive document under `docs/milestones/`.
6. Link updates:
   - Link monthly archive from the corresponding milestone record.

## Monthly Archive Template

```markdown
# JWT Provenance Threshold Change Archive (YYYY-MM)

## Summary
- Total changes:
- Tighten changes:
- Relax changes:
- Signals affected:

## Change List
| log_id | signal_type | level | old -> new | type | evidence | approvals | post-check |
| --- | --- | --- | --- | --- | --- | --- | --- |

## Risk Notes
- Critical detection blind spots:
- Incident correlation summary:

## Actions
- Keep:
- Rollback:
- Re-tune:
```

## Governance

- Do not overwrite historical log records.
- Corrections must be appended as new records referencing original `log_id`.
- Monthly archive must be committed to version control.

## First Sample Archive

- `docs/milestones/JWT_PROVENANCE_THRESHOLD_CHANGE_ARCHIVE_2026-04.md`

## Checklist Execution Example

Reference implementation:

- `docs/milestones/JWT_PROVENANCE_THRESHOLD_CHANGE_ARCHIVE_2026-04.md` (`Review Checklist Execution Example` section)
