# E1-policy-engine-and-data-scope

## Summary

This milestone delivers the E1 baseline for policy-engine style authorization and role-based data-scope enforcement.

## Delivered

- RBAC module enhancements:
  - role policy rules (`allow`/`deny`, claim constraints)
  - role data scope (`tenant_ids`, `require_owner_match`)
- Admin API additions:
  - `PUT/GET /admin/v1/roles/{id}/policies`
  - `PUT/GET /admin/v1/roles/{id}/data-scope`
- Runtime authorization update:
  - startup bootstrap now composes admin auth verifier with role API/policy/data-scope authorizer middleware.
- Documentation:
  - `docs/community/ADMIN_RBAC_POLICY_DATA_SCOPE_BASELINE.md`

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`
- `go test -bench=BenchmarkSetRoleMenus -benchmem ./internal/module/rbac`

## Key Test Coverage

- `internal/module/rbac/service_test.go`:
  - policy normalization and retrieval
  - data-scope normalization and retrieval
- `internal/app/admin_modules_test.go`:
  - role policy/data-scope route set/get behavior
- `internal/app/admin_rbac_e2e_test.go`:
  - verified-claims constrained allow policy
  - tenant scope deny
  - owner-match enforcement

## Next

- E2-dynamic-route-menu-button-permission
