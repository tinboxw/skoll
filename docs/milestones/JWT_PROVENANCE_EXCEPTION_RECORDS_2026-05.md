# JWT Provenance Exception Records (2026-05)

## Sample Exception Records

### Exception 1

- exception_id: jwt-prov-ex-202605-001
- created_at: 2026-05-03T08:30:00Z
- owner: sre-b
- related_log_ids: jwt-prov-threshold-202605-002
- exception_type: Relax change without complete canary window
- reason: Canary verification completed for 4 days due to emergency deployment freeze
- scope: claims_version_missing warning threshold
- risk_level: High
- required_approvers: service owner, security reviewer
- approvals_received: service owner (approved), security reviewer (approved)
- approved_until: 2026-05-10T08:30:00Z
- active_controls: daily burn-rate checks, rollback criteria tightened
- rollback_or_freeze_condition: if false negatives rise for 24h, freeze relax change
- revalidation_due: 2026-05-09T08:30:00Z
- revalidation_owner: review_primary
- revalidation_criteria: no critical blind spots, stable alert volume
- closure_conditions: canary expanded to >= 7d and verified

### Exception 2

- exception_id: jwt-prov-ex-202605-002
- created_at: 2026-05-12T10:00:00Z
- owner: sre-c
- related_log_ids: jwt-prov-threshold-202605-004
- exception_type: Overdue post-check
- reason: D+7 verification delayed by regional incident response workload
- scope: invalid_rate high threshold post-check
- risk_level: High
- required_approvers: review_primary, service owner
- approvals_received: review_primary (approved), service owner (approved)
- approved_until: 2026-05-19T10:00:00Z
- active_controls: escalated tracking and sign-off block
- rollback_or_freeze_condition: auto-freeze if verification absent at expiry
- revalidation_due: 2026-05-18T10:00:00Z
- revalidation_owner: review_secondary
- revalidation_criteria: post-check evidence complete and incident impact bounded
- closure_conditions: verification completed and archive updated

## Revalidation Decision Log

| decision_id | exception_id | decided_at | decision | rationale | approvers | next_due |
| --- | --- | --- | --- | --- | --- | --- |
| jwt-prov-ex-dec-202605-001 | jwt-prov-ex-202605-001 | 2026-05-09T07:00:00Z | extend_72h | canary evidence still incomplete but risk stable | service owner + security reviewer | 2026-05-12T07:00:00Z |
| jwt-prov-ex-dec-202605-002 | jwt-prov-ex-202605-001 | 2026-05-11T09:15:00Z | close | 7d canary evidence complete and no blind-spot findings | service owner + security reviewer | n/a |
| jwt-prov-ex-dec-202605-003 | jwt-prov-ex-202605-002 | 2026-05-18T08:40:00Z | rollback | post-check evidence missing at expiry threshold | review_primary + service owner | n/a |
