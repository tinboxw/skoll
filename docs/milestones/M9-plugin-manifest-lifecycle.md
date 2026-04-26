# M9-plugin-manifest-lifecycle

## Summary

This milestone slice introduces plugin manifest lifecycle baseline capabilities, including install/lookup/list and enable/disable operations via admin APIs.

## Delivered

- Added plugin manager service in `internal/module/pluginmgr/service.go`.
- Extended storage adapter with plugin repository boundary and in-memory wiring.
- Added admin API endpoints:
  - `POST /admin/v1/plugins/manifests`
  - `GET /admin/v1/plugins`
  - `GET /admin/v1/plugins/{name}`
  - `POST /admin/v1/plugins/{name}/enable`
  - `POST /admin/v1/plugins/{name}/disable`
- Added module tests, route tests, RBAC e2e wiring updates, and adapter contract coverage.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`

## Risk Notes

- Current manifest storage is in-memory only.
- Package installation artifacts and remote registry/version checks are planned for the next slice.

## Next

- M9-extension-packaging
- M9-ecosystem-docs
