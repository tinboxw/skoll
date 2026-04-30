# E16-step1 Dispatch Claim Lease Renewal Baseline

## Milestone

- ID: `E16-step1`
- Name: `dispatch claim lease renewal baseline`

## Delivered

- Extended scheduler claim model with lease metadata:
  - `lease_until_unix_sec`
  - `lease_renewal_count`
- Added lease renewal service operation:
  - `RenewClaimLease(executionKey, instanceID, leaseTTLSeconds, now)`
- Added admin API for lease renewal:
  - `POST /admin/v1/job-dispatch-claims/{execution_key}/renew`
- Added ownership and validation guards:
  - unknown claim -> `404`
  - lease owner mismatch -> `403`
  - invalid TTL -> `400`
- Added unit and route regression tests for lease renewal success and ownership rejection.

## Files

- `internal/module/jobscheduler/service.go`
- `internal/module/jobscheduler/service_test.go`
- `internal/app/admin_modules.go`
- `internal/app/admin_modules_test.go`
- `internal/module/storageadapter/adapter.go`
- `docs/planning/FEATURE_PARITY_PLAN.md`
- `README.md`
- `README.en.md`

## Validation

- `go fmt ./...`
- `go test ./internal/module/jobscheduler ./internal/app ./internal/module/storageadapter`
- `go test ./...`
- `go test -race ./...`
- `Push-Location internal/module/jobscheduler; go test -bench=. -benchmem; Pop-Location`
