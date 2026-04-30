# E12-step2 Admin Query Contract And Audit SLA Profile

## Milestone

- ID: `E12-step2`
- Name: `admin query contract and audit paging SLA profile`

## Delivered

- Added audit query profile endpoint:
  - `GET /admin/v1/audit-logs/profile`
  - Exposes default page/size, max size, and supported filters.
- Added typed query endpoints for admin operational data:
  - `GET /admin/v1/configs/query`
  - `GET /admin/v1/dictionaries/query`
- Added stable pagination envelope for query endpoints:
  - fields: `items`, `page`, `size`, `total`, `has_next`
- Added max-size guardrails for audit query parameters (`size`/`limit` <= 200).
- Added tests for profile and query filtering/pagination behavior.

## Files

- `internal/app/admin_modules.go`
- `internal/app/admin_modules_test.go`

## Validation

- `go fmt ./...`
- `go test ./internal/app ./internal/module/rbac ./internal/module/storageadapter`
- `go test ./...`
- `go test -race ./...`

## Notes

- Existing list endpoints remain backward compatible.
- Query endpoints provide persistence-ready contract shape for later storage adapters.
