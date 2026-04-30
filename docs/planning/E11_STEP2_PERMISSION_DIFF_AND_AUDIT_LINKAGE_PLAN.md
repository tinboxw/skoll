# E11-step2 Permission Diff and Audit Linkage Plan

## Objective

Complete E11 governance depth by adding permission diff/check tooling across route/menu/button/API/data-scope and binding rollback actions to audit evidence.

## Scope

- Permission diff model:
  - role current vs target policies
  - route/menu/button contract drift
  - API binding drift
  - data-scope drift
- Diff API contracts:
  - `POST /admin/v1/roles/{id}/permissions/diff`
  - `POST /admin/v1/roles/{id}/permissions/check`
- Rollback evidence linkage:
  - rollback response includes snapshot version, operator, and evidence id
  - release governance can query rollback evidence history

## Acceptance

- Diff output is deterministic and machine-readable.
- Check endpoint returns `pass/warn/fail` with reason set.
- Rollback and snapshot operations produce auditable evidence records.
- Regression tests cover false-positive and privilege-escalation edge cases.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`
- `go test -run ^$ -bench BenchmarkSetRoleMenus -benchmem ./internal/module/rbac`
