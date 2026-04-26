# Dashboard JWT Provenance Recalibration Evidence Template

## Purpose

Provide a reusable evidence template and approval checklist for JWT provenance baseline recalibration changes.

Use this template together with:

- `docs/community/DASHBOARD_JWT_PROVENANCE_BASELINE_RECALIBRATION.md`
- `docs/community/DASHBOARD_JWT_PROVENANCE_SLO_POLICY.md`
- `docs/community/DASHBOARD_JWT_PROVENANCE_SLO_ALERT_RULES.md`

## Usage

1. Copy this template into a milestone note or change proposal.
2. Fill all required fields before canary rollout.
3. Obtain required approvals before stable rollout.
4. Attach post-rollout verification outcome within 7 days.

## Evidence Record Template

```markdown
# Recalibration Evidence Record

## Change Identity
- Change ID:
- Date:
- Owner:
- Related milestone/incident:

## Target Signal
- Signal type: (invalid_rate | unverified_rate | claims_version_missing | chain_depth_drift)
- Current threshold:
- Proposed threshold:
- Change type: (tighten | relax)
- Alert level: (warning | high | critical)

## Data Window and Sampling
- Baseline window: (e.g. 30d)
- Sample size:
- Traffic segmentation: (canary/stable, route groups)
- Data source query links:

## Statistical Summary
- P50:
- P90:
- P95:
- Current threshold percentile position:
- Proposed threshold percentile position:

## Impact Estimation
- Estimated alert volume delta:
- Detection latency impact:
- False-positive impact:
- False-negative risk:

## Safety Checks
- Critical detection sensitivity preserved: (yes/no)
- Warning-critical simultaneous relaxation avoided: (yes/no)
- Rollback criteria pre-defined: (yes/no)

## Canary Trial
- Canary start/end:
- Canary scope:
- Alert outcomes:
- Divergence from stable:

## Decision
- Decision: (approve | reject | revise)
- Decision rationale:
- Effective date:
```

## Approval Checklist

Mandatory approvers:

- Service owner
- Security on-call

Checklist:

- [ ] Evidence record completed with 30d window and segmented traffic
- [ ] Candidate threshold does not weaken critical detection sensitivity
- [ ] Rollback triggers are explicit and testable
- [ ] Canary trial result attached (>= 7d or justified exception)
- [ ] Impact estimation reviewed (alert volume and latency)
- [ ] SLO policy references remain consistent
- [ ] README CN/EN doc index sync planned
- [ ] Post-rollout verification owner assigned

## Post-Rollout Verification Template

```markdown
## Post-Rollout Verification (D+7)
- Actual alert volume delta:
- Observed false-positive trend:
- Observed false-negative signals:
- Burn-rate status change:
- Follow-up action: (keep | rollback | re-tune)
- Reviewer:
```

## Governance Notes

- Every approved recalibration must be tracked in version control.
- Keep evidence records discoverable from milestone logs.
- If approval is partial or conditional, record expiry date and re-review trigger.
