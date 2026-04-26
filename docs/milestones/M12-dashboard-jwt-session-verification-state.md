# M12-dashboard-jwt-session-verification-state

## Summary

This milestone slice adds JWT verification-state hints in dashboard bootstrap payload to improve trust messaging in frontend UX.

## Delivered

- Extended `jwt_session_bootstrap` with verification-state fields:
  - `verification_state`
  - `verification_hint`
  - `trust_level`
  - `trust_message`
- Added derivation logic for trust messaging states:
  - `not_present`
  - `invalid`
  - `unverified`
  - `verified` (reserved for middleware-verified path)
- Updated tests for:
  - route-level verification-state assertions
  - invalid bearer verification-state assertions
  - verification hint derivation behavior
- Updated roadmap, feature sequence, dashboard contract doc, and README CN/EN references.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`

## Next

- M12-dashboard-jwt-session-middleware-bridge
