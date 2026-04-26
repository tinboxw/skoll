# JWT Provenance Review Rotation (2026-Q2)

## Quarterly Rotation Roster

| Month | review_primary | review_secondary | security_reviewer |
| --- | --- | --- | --- |
| 2026-04 | sre-a | sre-b | sec-a |
| 2026-05 | sre-b | sre-c | sec-a |
| 2026-06 | sre-c | sre-a | sec-b |

## Escalation Handoff Template

Use this template when monthly review cannot be signed off due to risk/completeness blockers.

```markdown
# JWT Provenance Escalation Handoff

## Handoff Identity
- Month:
- Triggered at:
- Handoff owner:
- Receiving owner:

## Blocker Summary
- Blocker type: (incomplete_critical_record | missing_security_approval | overdue_post_check | other)
- Affected log_id(s):
- Severity:
- Current operational impact:

## Evidence and Context
- Archive link:
- Evidence links:
- Prior actions attempted:
- Why unresolved:

## Required Decision
- Decision needed by:
- Decision owner:
- Options:
  - keep
  - rollback
  - temporary exception (with expiry)

## Immediate Safeguards
- Mitigation currently active:
- Additional guard requested:
- Communication channel:

## Exit Criteria
- Conditions to close escalation:
- Verification owner:
- Verification deadline:
```

## Example Handoff Snapshot

- Month: 2026-05
- Blocker type: overdue_post_check
- Affected log_id(s): jwt-prov-threshold-202605-002
- Decision owner: service-owner-platform
- Decision needed by: 2026-06-03 12:00 UTC
- Immediate safeguard: freeze relax changes for `claims_version_missing` until post-check completion

## Governance

- Keep each handoff record immutable after publishing.
- Any correction is appended as a follow-up entry with timestamp and owner.
- Link final decision outcome back to the monthly archive record.
