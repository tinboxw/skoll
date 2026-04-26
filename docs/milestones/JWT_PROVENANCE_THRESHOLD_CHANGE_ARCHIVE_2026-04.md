# JWT Provenance Threshold Change Archive (2026-04)

## Summary

- Total changes: 2
- Tighten changes: 1
- Relax changes: 1
- Signals affected: invalid_rate, claims_version_missing

## Change List

| log_id | signal_type | level | old -> new | type | evidence | approvals | post-check |
| --- | --- | --- | --- | --- | --- | --- | --- |
| jwt-prov-threshold-202604-001 | invalid_rate | high | 0.020 -> 0.018 | tighten | docs/milestones/M17-dashboard-jwt-session-provenance-recalibration-evidence-template.md | service-owner/security-oncall approved | due 2026-05-04 |
| jwt-prov-threshold-202604-002 | claims_version_missing | warning | 0.100 -> 0.120 | relax | docs/milestones/M17-dashboard-jwt-session-provenance-recalibration-evidence-template.md | service-owner/security-oncall approved | due 2026-05-08 |

## Risk Notes

- Critical detection blind spots: none observed in April canary/stable comparisons.
- Incident correlation summary: one canary incident correlated with log 001 and resolved after threshold tighten.

## Review Checklist Execution Example

- [x] All records include evidence links and approval identities.
- [x] Every relax change has explicit rollback criteria.
- [x] No simultaneous warning+critical relaxation in same signal.
- [x] Post-check due date assigned for each record.
- [x] No unresolved critical blind-spot findings.

## Actions

- Keep: `jwt-prov-threshold-202604-001`
- Rollback: none
- Re-tune: observe `jwt-prov-threshold-202604-002` until D+7 post-check
