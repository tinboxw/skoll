# E14-step1 Marketplace Trust Root And Signed Index Baseline

## Milestone

- ID: `E14-step1`
- Name: `marketplace trust-root and signed index ingestion baseline`

## Delivered

- Added trust-root governance in plugin manager:
  - set trust roots
  - list trust roots
- Added signed index ingestion with rejection controls:
  - reject unknown trust root signer
  - reject expired index
  - reject tampered signature
- Added marketplace index source state tracking.
- Added admin APIs:
  - `PUT /admin/v1/plugins/marketplace/trust-roots`
  - `GET /admin/v1/plugins/marketplace/trust-roots`
  - `POST /admin/v1/plugins/marketplace/index/ingest`
  - `GET /admin/v1/plugins/marketplace/index/sources`
- Added audit linkage for trust-root updates and index ingestion.

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
