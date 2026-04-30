# E11-step2 Permission Diff and Audit Linkage

## Milestone

- ID: `E11-step2`
- Name: `permission diff/check and audit linkage`

## Delivered

- Added role permission diff endpoint:
  - `POST /admin/v1/roles/{id}/permissions/diff`
- Added role permission blocking-check endpoint:
  - `POST /admin/v1/roles/{id}/permissions/check`
- Added blocking evaluation rules for permission expansion:
  - API expansion
  - allow-policy addition
  - tenant scope expansion
  - owner-match relaxation
  - cross-tenant whitelist expansion
- Added cross-tenant admin whitelist in data scope model and API contract.
- Enforced rollback approval field:
  - `approver` is now mandatory in `POST /admin/v1/roles/{id}/policies/rollback`.
- Linked rollback audit records with approver identity.

## Files

- `internal/module/rbac/service.go`
- `internal/module/rbac/service_test.go`
- `internal/module/storageadapter/adapter.go`
- `internal/app/admin_modules.go`
- `internal/app/admin_modules_test.go`
- `internal/app/admin_rbac_authorizer.go`

## Validation

- `go fmt ./...`
- `go test ./internal/module/rbac ./internal/app ./internal/module/storageadapter`
- `go test ./...`
- `go test -race ./...`

## Decisions Applied

- Blocking-level permission check is enabled.
- Cross-tenant admin whitelist capability is enabled.
- Rollback approver field is mandatory.
