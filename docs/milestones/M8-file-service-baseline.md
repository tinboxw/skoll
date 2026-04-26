# M8-file-service-baseline

## Summary

This milestone slice introduces a file upload/download module with a pluggable storage backend contract and a local filesystem backend as the first implementation.

## Delivered

- Added file module service in `internal/module/fileservice/service.go`.
- Added local filesystem backend (`LocalBackend`) implementing pluggable backend contract.
- Extended storage adapter with file repository boundary and in-memory wiring.
- Added admin API endpoints:
  - `POST /admin/v1/files` (multipart field: `file`)
  - `GET /admin/v1/files`
  - `GET /admin/v1/files/{id}`
  - `GET /admin/v1/files/{id}/download`
- Added module tests, route tests, RBAC e2e wiring updates, and adapter contract coverage.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`

## Risk Notes

- Current backend is local filesystem only; object storage adapters are pending.
- Upload API currently reads file into memory before save; large-file streaming optimization is not implemented yet.

## Next

- M8-job-scheduler-baseline
- M8-module-generator
