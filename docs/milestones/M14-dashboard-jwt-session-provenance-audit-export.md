# M14-dashboard-jwt-session-provenance-audit-export

## Summary

This milestone slice adds JWT provenance audit export fields to `jwt_session_bootstrap` for security operations and external audit pipelines.

## Delivered

- Added `jwt_session_bootstrap.provenance_audit_export` payload object.
- Added audit export fields:
  - `enabled`
  - `source_provenance`
  - `source_path`
  - `source`
  - `verified`
  - `claims_trusted`
  - `verification_state`
  - `subject`
  - `role_id`
  - `claims_version`
  - `role_source`
  - `subject_source`
  - `claims_version_source`
  - `verified_source`
- Wired audit export generation consistently across all bootstrap branches:
  - no token
  - invalid/unsupported bearer
  - parsed bearer JWT
- Added dashboard tests for default and verified-bridge audit export assertions.
- Updated roadmap, feature sequence, middleware bridge contract docs, and README CN/EN references.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`

## Next

- M14-dashboard-jwt-session-provenance-audit-docs
