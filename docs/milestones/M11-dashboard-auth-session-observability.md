# M11-dashboard-auth-session-observability

## Summary

This milestone slice adds auth/session observability counters and failure totals to the dashboard aggregate payload.

## Delivered

- Extended `GET /admin/v1/system/dashboard` with `auth_observability` section.
- Included per-mode counters for:
  - `static_token`
  - `hmac_sha256`
- Included `total_failure` counter for quick dashboard summary.
- Updated dashboard contract required sections to include `auth_observability`.
- Added/updated tests for:
  - adminauth structured snapshot
  - dashboard payload observability fields

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`

## Next

- M11-dashboard-auth-session-actionability
