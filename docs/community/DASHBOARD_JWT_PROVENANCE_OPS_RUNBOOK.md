# Dashboard JWT Provenance Ops Runbook

## Purpose

Provide a practical incident-response workflow for `jwt_session_bootstrap.provenance_audit_export` operational metrics and alerting hints.

This runbook is for production operations, SOC, and on-call engineers.

## Scope

This runbook covers:

- Prometheus provenance counters:
  - `skoll_dashboard_jwt_provenance_exports_total`
  - `skoll_dashboard_jwt_provenance_verification_states_total`
  - `skoll_dashboard_jwt_provenance_alert_hints_total`
- Dashboard payload fields:
  - `provenance_audit_export.operational_metrics`
  - `provenance_audit_export.alerting_hints`

It does not replace authorization controls. Access decisions must still rely on backend authorization outcomes.

## Alert Classes

### P1: Verification Invalid Surge

Trigger examples:

- `verification_state_invalid` hint rate spikes above baseline.
- `skoll_dashboard_jwt_provenance_verification_states_total{state="invalid"}` slope increases sharply.

Immediate actions:

1. Check recent changes in auth gateway/middleware rollout.
2. Validate `Authorization` header formatting and JWT parsing paths.
3. Confirm clock drift and signing/verification dependencies for upstream services.

### P2: Unverified/Not Verified Drift

Trigger examples:

- Sustained increase in `verification_state_unverified` and `claims_not_verified`.
- `enabled_total` remains high while `verified_total` drops unexpectedly.

Immediate actions:

1. Verify whether trusted verification middleware is bypassed for some routes.
2. Check route-level middleware ordering and conditional feature flags.
3. Confirm canary/blue-green split is not routing around verifier components.

### P2: Claims Version Gaps

Trigger examples:

- Increase in `claims_version_missing` hints.
- Downstream SIEM entries missing `claims_version` during active deployment windows.

Immediate actions:

1. Verify header/metadata propagation for claims version.
2. Check mixed-version middleware deployment state.
3. Coordinate rollout completion or temporary compatibility fallback.

### P3: Chain Depth or Topology Drift

Trigger examples:

- Increase in `provenance_chain_depth_high` hints.
- Unexpected source path patterns appear in exported events.

Immediate actions:

1. Inspect ingress/middleware chain for duplicated hops.
2. Validate reverse-proxy and service-mesh path changes.
3. Compare expected path templates versus observed `source_path`.

## Triage Workflow

1. Identify dominant hint type using:
   - `skoll_dashboard_jwt_provenance_alert_hints_total{hint=...}`
2. Correlate with verification-state counters:
   - `...verification_states_total{state="invalid|unverified|verified"}`
3. Sample latest dashboard payloads and inspect:
   - `source_provenance`, `source_path`, `claims_version`, `role_source`, `verified_source`
4. Determine blast radius:
   - all tenants, specific route groups, specific ingress region
5. Apply mitigations and watch counters return toward baseline.

## Mitigation Playbook

1. If invalid spikes after deploy:
   - rollback verifier/middleware changes first.
2. If unverified dominates:
   - restore verifier middleware order and re-enable trust bridge path.
3. If claims version missing:
   - enforce default claims version emission or rollback partial rollout.
4. If chain depth drifts:
   - remove duplicate middleware registrations and normalize ingress chain.

## Recovery Verification

Recovery can be considered complete when:

1. Invalid/unverified hint rates return to normal baseline window.
2. `verified_total / enabled_total` ratio stabilizes to expected range.
3. No new unknown source paths are observed in SIEM exports.
4. On-call and security review agree on root cause and closure evidence.

## Post-Incident Checklist

1. Record root cause, impacted windows, and recovered version in incident log.
2. Add regression tests or rollout guards for the failing path.
3. Update alert thresholds if historical baselines changed materially.
4. Update this runbook with new triage heuristics if needed.
