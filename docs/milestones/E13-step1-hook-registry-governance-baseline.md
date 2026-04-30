# E13-step1 Hook Registry Governance Baseline

## Milestone

- ID: `E13-step1`
- Name: `hook registry governance baseline`

## Delivered

- Added hook registry governance capability in plugin manager:
  - register hook with namespace/version/order
  - enable/disable hook lifecycle control
  - runtime budget policy update (`timeout_millis`, `retry_limit`, `dead_letter`)
  - deterministic hook listing by namespace/order/name
- Added admin APIs for hook governance:
  - `POST /admin/v1/plugins/hooks/register`
  - `GET /admin/v1/plugins/hooks`
  - `POST /admin/v1/plugins/hooks/{namespace}/{name}/enable`
  - `POST /admin/v1/plugins/hooks/{namespace}/{name}/disable`
  - `POST /admin/v1/plugins/hooks/{namespace}/{name}/order`
  - `POST /admin/v1/plugins/hooks/{namespace}/{name}/runtime`
- Added audit event linkage for hook governance actions.
- Added module boundary interface updates for storage adapter compatibility.

## Files

- `internal/module/pluginmgr/service.go`
- `internal/module/pluginmgr/service_test.go`
- `internal/app/admin_modules.go`
- `internal/app/admin_modules_test.go`
- `internal/module/storageadapter/adapter.go`

## Validation

- `go fmt ./...`
- `go test ./internal/module/pluginmgr ./internal/app ./internal/module/storageadapter`
- `go test ./...`
- `go test -race ./...`

## Notes

- This step establishes registry governance and isolation controls.
- E13 follow-up steps will extend lifecycle recoverability and compatibility gates.
