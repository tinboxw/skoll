# M8-job-scheduler-baseline

## Summary

This milestone slice introduces a baseline job scheduler module with job definition APIs, manual run trigger, and execution history query for operational visibility.

## Delivered

- Added job scheduler module in `internal/module/jobscheduler/service.go`.
- Extended storage adapter with job repository boundary and in-memory wiring.
- Added admin API endpoints:
  - `POST /admin/v1/jobs`
  - `GET /admin/v1/jobs`
  - `POST /admin/v1/jobs/{id}/run`
  - `GET /admin/v1/jobs/{id}/history`
- Added module tests, route tests, RBAC e2e wiring updates, and adapter contract coverage.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`

## Risk Notes

- Current schedule field is stored as metadata only; cron parsing and real-time dispatch loop are not implemented in this baseline.
- Current run action is manual API trigger and synchronous completion.

## Next

- M8-module-generator
- M9-plugin-manifest-lifecycle
