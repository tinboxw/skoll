# E16-step2 Retry Backoff Dead-Letter And Replay Baseline

## Milestone

- ID: `E16-step2`
- Name: `retry backoff, dead-letter lifecycle, and replay baseline`

## Delivered

- Extended scheduler reliability model with retry policy contracts:
  - `max_retries`
  - `backoff_base_millis`
  - `backoff_max_millis`
  - `jitter_percent`
- Added retry scheduling model with computed delay + jitter evidence.
- Added dead-letter lifecycle model and replay operation with idempotent replay behavior.
- Added admin APIs:
  - `PUT /admin/v1/jobs/{id}/retry-policy`
  - `GET /admin/v1/jobs/{id}/retry-policy`
  - `POST /admin/v1/jobs/{id}/retries/schedule`
  - `POST /admin/v1/jobs/{id}/dead-letters`
  - `GET /admin/v1/jobs/dead-letters?limit=20`
  - `POST /admin/v1/jobs/dead-letters/{execution_key}/replay`
- Added unit and route regression tests for retry policy + DLQ + replay flows.

## Files

- `internal/module/jobscheduler/service.go`
- `internal/module/jobscheduler/service_test.go`
- `internal/app/admin_modules.go`
- `internal/app/admin_modules_test.go`
- `internal/module/storageadapter/adapter.go`
- `docs/planning/FEATURE_PARITY_PLAN.md`
- `docs/community/MULTI_INSTANCE_CONSISTENCY_BASELINE.md`
- `README.md`
- `README.en.md`

## Validation

- `go fmt ./...`
- `go test ./internal/module/jobscheduler ./internal/app ./internal/module/storageadapter`
- `go test ./...`
- `go test -race ./...`
- `Push-Location internal/module/jobscheduler; go test -bench=. -benchmem; Pop-Location`
