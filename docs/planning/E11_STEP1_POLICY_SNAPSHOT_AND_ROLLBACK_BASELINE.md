# E11-step1 Policy Snapshot and Rollback Baseline

## Objective

Deliver the first executable governance slice for E11 parity: role policy snapshotting and rollback contracts.

## Delivered Scope

- RBAC service now supports policy snapshot lifecycle:
  - create snapshot
  - list snapshots
  - rollback policies by snapshot version
- Admin APIs added:
  - `POST /admin/v1/roles/{id}/policies/snapshots`
  - `GET /admin/v1/roles/{id}/policies/snapshots`
  - `POST /admin/v1/roles/{id}/policies/rollback`
- Audit events added for snapshot create and rollback actions.
- Added tests for service and admin API contracts.

## Current Constraints

- Snapshot persistence is currently in-memory baseline only.
- Cross-instance durable snapshot storage will be completed in next E11 steps.

## Validation Commands

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`
- `go test -run ^$ -bench BenchmarkSetRoleMenus -benchmem ./internal/module/rbac`

## Next Steps

- Add persistent snapshot storage adapter support.
- Add permission diff/check report APIs (route/menu/button/API/data-scope).
- Add rollback evidence binding to release governance scorecard.
