# M12-dashboard-jwt-session-bootstrap

## Summary

This milestone slice introduces dashboard JWT session bootstrap fields so frontend can initialize identity/session UX with stable metadata.

## Delivered

- Extended `GET /admin/v1/system/dashboard` with `jwt_session_bootstrap` section.
- Added bootstrap fields:
  - `token_present`
  - `token_format`
  - `claims_trusted`
  - `subject`
  - `issuer`
  - `audience`
  - `issued_at_unix_sec`
  - `expires_at_unix_sec`
  - `expired`
  - `parse_error`
- Added non-validating Bearer JWT claim parsing for bootstrap hints only.
- Updated dashboard contract required sections to include `jwt_session_bootstrap`.
- Added tests for:
  - dashboard route payload coverage of `jwt_session_bootstrap`
  - bearer JWT bootstrap extraction
  - invalid bearer token parsing behavior
- Updated roadmap, feature sequence, and README CN/EN references.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`

## Next

- M12-dashboard-jwt-session-refresh-hints
