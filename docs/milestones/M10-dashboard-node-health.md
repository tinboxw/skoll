# M10-dashboard-node-health

## Summary

This milestone slice adds node and dependency health summary API support for admin observability dashboard readiness.

## Delivered

- Added endpoint: `GET /admin/v1/system/node-health`
- Added response payload with:
  - overall `node_status`
  - `checked_at_unix_sec`
  - dependency list with name/status/detail/check timestamp
- Added route test for response status and dependency list validation.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`

## Notes

- Current dependency checks validate service wiring availability (up/down) at process level.
- External dependencies (database/redis/object storage) active probing will be added in later slices.

## Next

- Define M10 dashboard aggregation and frontend integration plan.
