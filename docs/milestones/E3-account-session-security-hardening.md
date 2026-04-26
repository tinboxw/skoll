# E3-account-session-security-hardening

## Summary

This milestone delivers account/session hardening baseline with lock and revoke controls, MFA extension point, and audit event coverage.

## Delivered

- User module enhancements:
  - password rotation with minimal interval guard
  - failed login lock strategy and reset
  - MFA extension point (`enabled`, `provider`)
  - session revoke/status and anomaly reporting
- Admin API additions:
  - `POST /admin/v1/users/{id}/password/rotate`
  - `POST /admin/v1/users/{id}/login-failures`
  - `POST /admin/v1/users/{id}/lock/reset`
  - `POST /admin/v1/users/{id}/mfa`
  - `POST /admin/v1/sessions/revoke`
  - `GET /admin/v1/sessions/{session_id}/status`
  - `POST /admin/v1/sessions/anomalies`
- Security audit linkage via actor `security`.
- Documentation:
  - `docs/community/ADMIN_ACCOUNT_SESSION_SECURITY_BASELINE.md`

## Validation

- `go fmt ./...`
- `go test ./internal/module/user ./internal/module/storageadapter ./internal/app`

## Key Test Coverage

- `internal/module/user/service_test.go`:
  - password rotation interval checks
  - failed-login lock behavior
  - MFA toggling
  - session revoke/anomaly tracking
- `internal/module/storageadapter/contract_test.go`:
  - user security/session adapter contract checks
- `internal/app/admin_modules_test.go`:
  - account/session security route behavior
  - security audit event visibility

## Next

- E4-generator-ecosystem-depth
