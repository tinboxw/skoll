# E13-step3 Runtime Isolation And Dead-Letter Diagnostics

## Milestone

- ID: `E13-step3`
- Name: `runtime isolation and dead-letter diagnostics`

## Delivered

- Added bounded-retry hook execution diagnostics:
  - `ExecuteHookDiagnostic(name, namespace, failTimes)`
  - max attempts derived from runtime policy (`retry_limit + 1`)
  - deterministic diagnostics per attempt
- Added dead-letter diagnostics records:
  - failed executions with exhausted retries are recorded when `dead_letter=true`
  - dead-letter list query API available for governance visibility
- Added admin APIs:
  - `POST /admin/v1/plugins/hooks/{namespace}/{name}/execute-diagnostic`
  - `GET /admin/v1/plugins/hooks/dead-letters`
- Added audit linkage for diagnostic execution actions.

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

- With step1/step2/step3 completed, E13 hook and module governance parity baseline is closed.
