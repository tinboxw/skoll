# M6-role-bindings

## Summary

This milestone slice delivers role binding capabilities in admin APIs, including role-menu and role-api bindings with in-memory storage and route-level tests.

## Delivered

- Added in-memory RBAC binding service in `internal/module/rbac/service.go`.
- Added RBAC unit tests in `internal/module/rbac/service_test.go`.
- Added role binding endpoints under `/admin/v1/roles/{id}`:
  - `PUT /menus`
  - `GET /menus`
  - `PUT /apis`
  - `GET /apis`
- Added route tests covering deduplication/sorting behavior and read-after-write flow.
- Wired RBAC service into startup route mount path.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`

## Risk Notes

- Binding storage is still in-memory and non-persistent.
- API binding values are plain strings and not yet linked to registry metadata.

## Next

- M6-rbac-e2e-smoke
- M6-api-registry-and-permission-assignment
