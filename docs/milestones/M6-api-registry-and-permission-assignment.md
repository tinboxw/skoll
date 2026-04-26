# M6-api-registry-and-permission-assignment

## Summary

This milestone slice adds an API registry endpoint and enforces registered-API policy when assigning role API permissions.

## Delivered

- Added in-memory API registry service in `internal/module/apiregistry/service.go`.
- Added API registry tests in `internal/module/apiregistry/service_test.go`.
- Route mount now auto-registers mounted admin API patterns.
- Added endpoint `GET /admin/v1/apis` for permission assignment discovery.
- Updated role API binding endpoint to reject unregistered APIs.
- Added route tests for registry listing and invalid API binding rejection.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`

## Risk Notes

- Registry is currently process-local in-memory state.
- Binding expression format is currently method+path string and not yet a typed resource model.

## Next

- M6-rbac-e2e-smoke
- M7-storage-adapter-contract
