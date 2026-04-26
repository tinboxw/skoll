# M15-dashboard-jwt-session-provenance-ops-runbook

## Summary

This milestone slice adds a production operations runbook and alert triage guidance for dashboard JWT provenance export.

## Delivered

- Added JWT provenance operations runbook:
  - `docs/community/DASHBOARD_JWT_PROVENANCE_OPS_RUNBOOK.md`
- Added alert classification and triage workflow for:
  - `verification_state_invalid`
  - `verification_state_unverified`
  - `claims_not_verified`
  - `claims_version_missing`
  - `provenance_chain_depth_high`
- Added mitigation and recovery verification checklist for on-call workflows.
- Linked runbook into provenance guide and README CN/EN docs index.
- Updated roadmap and feature sequence to mark M15-step2 complete.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`

## Next

- M16-dashboard-jwt-session-provenance-slo-dashboards
