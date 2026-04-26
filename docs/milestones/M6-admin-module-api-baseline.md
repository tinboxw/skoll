# M6-admin-module-api-baseline

## Summary

This milestone delivers the first M6 vertical slice for feature parity: unified admin module APIs under `/admin/v1/*` for user/role/menu/audit scaffolds, and shared admin auth wrapper wiring so all module routes can be protected consistently.

## Delivered

- Added admin module route mounting in `internal/app/admin_modules.go`.
- Added API endpoints:
  - `POST /admin/v1/users`
  - `GET /admin/v1/users`
  - `GET /admin/v1/users/{id}`
  - `POST /admin/v1/roles`
  - `GET /admin/v1/roles`
  - `GET /admin/v1/roles/{id}`
  - `POST /admin/v1/menus`
  - `GET /admin/v1/menus`
  - `GET /admin/v1/menus/{id}`
  - `POST /admin/v1/audit-logs`
  - `GET /admin/v1/audit-logs?limit=...`
- Added route-level optional wrapper support, allowing reuse of existing admin verifier for all `/admin/v1` APIs.
- Wired module services into startup path in `cmd/skoll/main.go`.
- Added API tests and baseline benchmark in `internal/app/admin_modules_test.go`.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`

## Risk Notes

- Data is still in-memory only; restart loses all admin module state.
- API envelope is still simple and not yet unified with error code taxonomy.

## Next

- M6-role-menu-binding
- M6-role-api-binding
- M6-rbac-e2e-smoke
