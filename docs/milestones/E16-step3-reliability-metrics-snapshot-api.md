# E16-step3 Reliability Metrics Snapshot API

## Milestone

- ID: `E16-step3`
- Name: `scheduler reliability metrics snapshot API`

## Delivered

- Added scheduler reliability snapshot model aggregating:
  - active claim count
  - retry schedule count
  - dead-letter count
  - replayed dead-letter count
  - total replay actions
- Added admin API:
  - `GET /admin/v1/jobs/reliability/metrics`
- Added service-level and route-level regression assertions for reliability metrics outputs.

## Files

- `internal/module/jobscheduler/service.go`
- `internal/module/jobscheduler/service_test.go`
- `internal/app/admin_modules.go`
- `internal/app/admin_modules_test.go`
- `internal/module/storageadapter/adapter.go`
- `docs/planning/FEATURE_PARITY_PLAN.md`
- `README.md`
- `README.en.md`
- `docs/community/MULTI_INSTANCE_CONSISTENCY_BASELINE.md`

## Validation

- `go fmt ./...`
- `go test ./internal/module/jobscheduler ./internal/app ./internal/module/storageadapter`
- `go test ./...`
- `go test -race ./...`
- `Push-Location internal/module/jobscheduler; go test -bench=. -benchmem; Pop-Location`
