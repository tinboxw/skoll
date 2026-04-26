# M13-dashboard-jwt-session-verified-claims-adapter

## Summary

This milestone slice aligns verified JWT claims extraction between dashboard bootstrap and RBAC authorization through a shared adapter.

## Delivered

- Added shared verified-claims adapter in app layer:
  - `resolveAdminVerifiedClaims(*http.Request)`
- Added JWT-specific role header support:
  - `X-Admin-JWT-Role-ID`
  - precedence over legacy `X-Admin-Role-ID`
- Aligned dashboard middleware bridge extraction with shared adapter.
- Aligned `WithRoleAPIAuthorizer` role-id extraction with shared adapter.
- Added tests for:
  - adapter role-id precedence/fallback behavior
  - RBAC E2E authorization via JWT role bridge header
- Updated roadmap, feature sequence, middleware bridge contract, and README CN/EN references.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`

## Next

- M13-dashboard-jwt-session-claims-normalization
