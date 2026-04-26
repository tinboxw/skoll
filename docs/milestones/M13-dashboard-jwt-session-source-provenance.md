# M13-dashboard-jwt-session-source-provenance

## Summary

This milestone slice adds claim source provenance fields for JWT middleware bridge payload to support multi-middleware tracing.

## Delivered

- Extended shared verified-claims adapter with source provenance extraction and normalization.
- Added bridge source headers:
  - `X-Admin-JWT-Source`
  - `X-Admin-JWT-Source-Provenance`
- Added dashboard middleware bridge fields:
  - `source_provenance`
  - `role_source`
  - `subject_source`
  - `claims_version_source`
  - `verified_source`
- Added tests for:
  - source provenance chain parsing/deduplication
  - source fallback behavior
  - bridge provenance payload assertions in dashboard route tests
- Updated roadmap, feature sequence, bridge contract, normalization policy, and README CN/EN references.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`

## Next

- M14-dashboard-jwt-session-provenance-audit-export
