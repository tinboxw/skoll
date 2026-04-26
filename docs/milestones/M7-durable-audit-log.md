# M7-durable-audit-log

## Summary

This milestone slice upgrades admin audit logs from a simple recent-list endpoint to a paginated, filterable query API while keeping compatibility with existing `limit` usage.

## Delivered

- Extended audit module with query contract in `internal/module/audit/service.go`:
  - page/size paging
  - actor/action/target exact filtering
  - keyword search via `q`
- Kept `Recent(limit)` compatibility by delegating to query semantics.
- Extended storage adapter audit repository contract with `Query(...)`.
- Upgraded `/admin/v1/audit-logs` endpoint to support:
  - `page`, `size`
  - `actor`, `action`, `target`
  - `q`
  - backward-compatible `limit`
- Added module, route, and adapter contract tests.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`

## Risk Notes

- Current persistence remains in-memory; process restart still clears audit history.
- Query filter semantics are exact-match for actor/action/target and substring-match for `q`.

## Next

- M8-file-service-baseline
- M8-job-scheduler-baseline
