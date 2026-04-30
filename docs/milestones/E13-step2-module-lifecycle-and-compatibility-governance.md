# E13-step2 Module Lifecycle And Compatibility Governance

## Milestone

- ID: `E13-step2`
- Name: `module lifecycle and compatibility governance`

## Delivered

- Added plugin lifecycle governance capability:
  - idempotent remove operation (`Remove`)
  - compatibility precheck report (`CheckCompatibility`)
- Added admin governance APIs:
  - `POST /admin/v1/plugins/{name}/remove`
  - `POST /admin/v1/plugins/compatibility-check`
- Compatibility check blocks unsafe combinations via machine-readable blockers:
  - invalid name/version
  - missing dependencies
  - dependency minimum-version mismatch
  - self-dependency cycle
- Added governance audit linkage:
  - `plugin_remove`
  - `plugin_compatibility_check`

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

- This step provides idempotent lifecycle primitives and unsafe-combination prechecks.
- E13 final step will focus on bounded retry/dead-letter execution diagnostics.
