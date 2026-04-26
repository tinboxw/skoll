# M12-dashboard-jwt-session-middleware-bridge

## Summary

This milestone slice exposes JWT middleware bridge fields in dashboard bootstrap payload to connect verified claims context from upstream middleware.

## Delivered

- Extended `jwt_session_bootstrap` with middleware bridge fields:
  - `claims_trusted`
  - `middleware_bridge.present`
  - `middleware_bridge.verified`
  - `middleware_bridge.source`
  - `middleware_bridge.subject`
  - `middleware_bridge.role_id`
  - `middleware_bridge.claims_version`
- Added header-based middleware bridge extraction for:
  - `X-Admin-JWT-Verified`
  - `X-Admin-JWT-Subject`
  - `X-Admin-JWT-Claims-Version`
  - existing `X-Admin-Role-ID` alignment
- Wired bridge verification into JWT trust hint derivation (`verified` path).
- Added tests for:
  - default bridge presence/source behavior
  - verified middleware bridge trust promotion
  - existing JWT bootstrap behavior compatibility
- Updated roadmap, feature sequence, dashboard contract doc, and README CN/EN references.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`

## Next

- M12-dashboard-jwt-session-bridge-contract-docs
