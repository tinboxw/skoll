# E11-step3 Policy Persistence Governance Baseline

## Milestone

- ID: `E11-step3`
- Name: `rbac policy persistence governance baseline`

## Delivered

- Added policy persistence export/import APIs:
  - `GET /admin/v1/roles/{id}/policies/persistence/export`
  - `POST /admin/v1/roles/{id}/policies/persistence/import`
- Export bundle includes policy-governance state:
  - role menus, APIs, policy rules, data-scope, route permission contract, snapshots
- Import flow enforces validations before apply:
  - role existence
  - referenced menu existence
  - API format and API registry membership
  - policy effect validity (`allow`/`deny`)
  - operator required for auditability
- Import writes produce audit evidence event:
  - `policy_persistence_import`

## Files

- `internal/app/admin_modules.go`
- `internal/app/admin_modules_test.go`

## Validation

- `go fmt ./...`
- `go test ./internal/app ./internal/module/rbac ./internal/module/storageadapter`
- `go test ./...`
- `go test -race ./...`
