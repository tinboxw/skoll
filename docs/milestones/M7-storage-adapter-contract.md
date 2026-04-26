# M7-storage-adapter-contract

## Summary

This milestone slice introduces a unified storage adapter contract for admin modules and validates the in-memory implementation via shared contract tests.

## Delivered

- Added storage adapter boundary in `internal/module/storageadapter/adapter.go`.
- Added shared contract tests in `internal/module/storageadapter/contract_test.go`.
- Added in-memory adapter implementation that composes existing module services.
- Updated runtime wiring in `cmd/skoll/main.go` to inject module dependencies through the adapter.
- Refactored admin route mounting boundary in `internal/app/admin_modules.go` to consume service interfaces instead of concrete service pointers.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`

## Risk Notes

- Contract currently covers baseline CRUD/binding behavior only; pagination/filter semantics will be extended with persistent adapters.
- Only in-memory adapter is implemented in this slice.

## Next

- M7-config-and-dictionary
- M7-durable-audit-log
