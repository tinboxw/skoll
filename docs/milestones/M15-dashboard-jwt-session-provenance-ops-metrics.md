# M15-dashboard-jwt-session-provenance-ops-metrics

## Summary

This milestone slice adds operational metrics and alerting hints for dashboard JWT provenance export.

## Delivered

- Added provenance operational metrics in dashboard payload:
  - `jwt_session_bootstrap.provenance_audit_export.operational_metrics`
  - Fields: `exports_total`, `enabled_total`, `disabled_total`, `verified_total`, `unverified_total`, `invalid_total`, `alert_hints_total`
- Added provenance alerting hints in dashboard payload:
  - `jwt_session_bootstrap.provenance_audit_export.alerting_hints`
  - Hints: `verification_state_invalid`, `verification_state_unverified`, `claims_not_verified`, `claims_version_missing`, `provenance_chain_depth_high`
- Added Prometheus metrics exposure for provenance operations:
  - `skoll_dashboard_jwt_provenance_exports_total`
  - `skoll_dashboard_jwt_provenance_verification_states_total`
  - `skoll_dashboard_jwt_provenance_alert_hints_total`
- Added/updated tests for:
  - alerting hint derivation
  - provenance metrics exposure in Prometheus payload
  - dashboard payload operational metrics fields
- Updated roadmap, feature sequence, bridge contract and provenance guide, and README CN/EN references.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`

## Next

- M15-dashboard-jwt-session-provenance-ops-runbook
