# M12-dashboard-jwt-session-refresh-hints

## Summary

This milestone slice adds JWT refresh and expiry hint fields so dashboard UX can proactively guide session renewal.

## Delivered

- Extended `jwt_session_bootstrap` payload with refresh/expiry hints:
  - `session_state`
  - `expires_in_sec`
  - `refresh_recommended`
  - `refresh_reason`
  - `refresh_after_unix_sec`
- Added refresh recommendation logic for states:
  - `none`
  - `invalid`
  - `active`
  - `expiring`
  - `expired`
  - `no-exp`
- Added tests for:
  - route-level JWT hint coverage
  - expiring token refresh recommendation
  - expired token refresh recommendation
  - invalid bearer refresh recommendation
- Updated roadmap, feature sequence, contract doc, and README CN/EN references.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`

## Next

- M12-dashboard-jwt-session-verification-state
