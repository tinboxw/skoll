# M7-config-and-dictionary

## Summary

This milestone slice delivers config center and dictionary APIs on top of the storage adapter boundary, with in-memory implementation and contract coverage for persistence-ready evolution.

## Delivered

- Added config module service in `internal/module/config/service.go`.
- Added dictionary module service in `internal/module/dictionary/service.go`.
- Extended storage adapter contracts to include config and dictionary repositories.
- Wired config and dictionary repositories into runtime startup path.
- Added admin API endpoints:
  - `POST /admin/v1/configs`
  - `GET /admin/v1/configs`
  - `GET /admin/v1/configs/{key}`
  - `POST /admin/v1/dictionaries`
  - `GET /admin/v1/dictionaries`
  - `GET /admin/v1/dictionaries/{id}`
- Added route and module tests for config/dictionary CRUD baseline behavior.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`

## Risk Notes

- Config and dictionary storage remains in-memory in this slice.
- Config key path currently assumes path-safe keys and does not support slash segments.

## Next

- M7-durable-audit-log
- M8-file-service-baseline
