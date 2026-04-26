# E2-dynamic-route-menu-button-permission

## Summary

This milestone delivers dynamic route/menu/button permission contracts with backend-side versioning and consistency checks.

## Delivered

- RBAC module enhancements:
  - role route/menu/button permission contract model
  - contract version normalization (`v1`/`v2`, unknown fallback to `v1`)
  - role contract consistency checker against role-menu bindings
- Admin API additions:
  - `PUT/GET /admin/v1/roles/{id}/permission-contract`
  - `POST /admin/v1/roles/{id}/permission-contract/consistency-check`
- Documentation:
  - `docs/community/ADMIN_DYNAMIC_PERMISSION_CONTRACT.md`

## Validation

- `go fmt ./...`
- `go test ./internal/module/rbac ./internal/module/storageadapter ./internal/app`

## Key Test Coverage

- `internal/module/rbac/service_test.go`:
  - version fallback and normalization
  - route/button deduplication
  - consistency fail/pass path
- `internal/module/storageadapter/contract_test.go`:
  - adapter contract coverage for route permission APIs
- `internal/app/admin_modules_test.go`:
  - permission-contract set/get/consistency API behavior

## Next

- E3-account-session-security-hardening
