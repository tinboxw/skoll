# M11-dashboard-auth-session-actionability

## Summary

This milestone slice adds actionable auth/session guidance fields for dashboard bootstrap UX decisions.

## Delivered

- Extended `GET /admin/v1/system/dashboard` with `auth_actionability` section.
- Added actionability fields:
  - `severity`
  - `recommended_auth_mode`
  - `failure_rate`
  - `top_failure_reasons`
  - `next_actions`
  - `docs`
- Updated dashboard contract required sections to include `auth_actionability`.
- Added tests for:
  - dashboard route payload coverage of `auth_actionability`
  - actionability derivation logic for mode guidance and high-failure severity
- Updated roadmap, feature sequence, and README CN/EN references.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`

## Next

- M12-dashboard-jwt-session-bootstrap
