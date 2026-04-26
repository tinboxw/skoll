# M6-rbac-e2e-smoke

## Summary

This slice validates the baseline RBAC authorization chain in an end-to-end smoke scenario: admin identity verification, role resolution, and API-level permission checks.

## Delivered

- Added reusable RBAC API authorizer middleware in `internal/app/admin_rbac_authorizer.go`.
- Added e2e smoke test in `internal/app/admin_rbac_e2e_test.go`.
- Covered authorization chain:
  - missing token -> unauthorized
  - token without role identity -> unauthorized
  - token + role + bound API -> allowed
  - token + role + unbound API -> forbidden

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`

## Risk Notes

- Role identity is currently provided via header (`X-Admin-Role-ID`) for testability baseline and has not been bound to signed JWT claims yet.
- RBAC policy remains in-memory and non-persistent.

## Next

- M7-storage-adapter-contract
- M7-config-and-dictionary
