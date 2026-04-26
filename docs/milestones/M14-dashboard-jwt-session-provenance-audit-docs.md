# M14-dashboard-jwt-session-provenance-audit-docs

## Summary

This milestone slice publishes compatibility and SIEM mapping guidance for `jwt_session_bootstrap.provenance_audit_export`.

## Delivered

- Added dedicated provenance audit export guide:
  - `docs/community/DASHBOARD_JWT_PROVENANCE_AUDIT_EXPORT_GUIDE.md`
- Documented compatibility rules for additive evolution under current dashboard contract `v1`.
- Added field-by-field SIEM mapping guidance with a stable key namespace:
  - `skoll.jwt.provenance.*`
- Added export example and operational usage guidance for SOC/SIEM pipelines.
- Linked the guide from middleware bridge contract documentation.
- Updated roadmap, feature sequence, and README CN/EN documentation index.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`

## Next

- M15-dashboard-jwt-session-provenance-ops-metrics
