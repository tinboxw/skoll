# M16-dashboard-jwt-session-provenance-slo-dashboards

## Summary

This milestone slice adds provenance SLO dashboard fields and error-budget policy to dashboard JWT provenance export.

## Delivered

- Added `provenance_audit_export.slo_dashboard` with:
  - `window`
  - `target_reliability`
  - `observed_reliability`
  - `error_rate`
  - `burn_rate`
  - `status`
- Added `provenance_audit_export.error_budget_policy` with:
  - `window`
  - `budget_ratio`
  - `consumed_ratio`
  - `remaining_ratio`
  - `action`
  - `freeze_recommended`
  - `escalate_recommended`
- Added computation helpers and policy thresholds for SLO/error budget evaluation.
- Added unit tests for:
  - SLO status transitions (`at_risk` / `critical`)
  - error budget policy critical action mapping
  - dashboard payload presence of `slo_dashboard` and `error_budget_policy`
- Added dedicated community policy doc:
  - `docs/community/DASHBOARD_JWT_PROVENANCE_SLO_POLICY.md`
- Updated roadmap, feature sequence, bridge/provenance docs, and README CN/EN indexes.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`

## Next

- M16-dashboard-jwt-session-provenance-slo-alert-rules
