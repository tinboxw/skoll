# E10 Auth and Session Parity Plan

## Objective

Deliver Gin-Vue-Admin-style admin auth lifecycle parity for login, refresh, logout/revoke, and session introspection, while preserving existing admin verifier compatibility.

## Delivered Scope

- Added auth lifecycle routes:
  - `POST /admin/v1/auth/login`
  - `POST /admin/v1/auth/refresh`
  - `POST /admin/v1/auth/logout`
  - `GET /admin/v1/auth/sessions/{session_id}`
- Added in-memory JWT session lifecycle implementation in `user.Service`:
  - access token issuance (HS256 JWT)
  - refresh token rotation
  - logout/revoke invalidation
  - session status query
- Kept existing admin verifier flow (`static-token`/`hmac-sha256`) unchanged.
- Added route tests and service tests for end-to-end lifecycle behavior.

## Contract Notes

- `login` returns access + refresh token pair and session metadata.
- `refresh` rotates refresh token and returns a new access token.
- `logout` revokes session and invalidates refresh token.
- `session` endpoint reports revocation and token window metadata.
- Auth lifecycle routes are mounted as public admin routes so bootstrap login does not require pre-existing admin wrapper credentials.

## Failure Reason Normalization

- Invalid user at login: `invalid_credentials`.
- Invalid/expired/revoked refresh token: `invalid_refresh_token`.

## Validation Commands

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`
- `go test -run ^$ -bench BenchmarkAdminAuthLoginEndpoint -benchmem ./internal/app`
