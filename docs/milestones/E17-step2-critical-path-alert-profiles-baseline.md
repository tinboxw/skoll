# E17-step2 Critical-Path Alert Profiles Baseline

## Milestone

- ID: `E17-step2`
- Name: `critical-path alert profile baseline`

## Delivered

- Added critical-path alert profile contract with fields for:
  - `metric`
  - `warn_threshold`
  - `critical_threshold`
  - `window_seconds`
  - `runbook`
  - `owner`
  - `enabled`
- Added admin APIs:
  - `PUT /admin/v1/system/hardening/alert-profiles`
  - `GET /admin/v1/system/hardening/alert-profiles`
- Added validation for metric/threshold/window/runbook/owner with threshold ordering checks.
- Added audit event for alert profile updates (`alert_profiles_updated`).
- Added route-level regression coverage for valid and invalid profile submissions.

## Files

- `internal/app/admin_modules.go`
- `internal/app/admin_modules_test.go`
- `docs/planning/FEATURE_PARITY_PLAN.md`
- `README.md`
- `README.en.md`

## Validation

- `go fmt ./...`
- `go test ./internal/app`
- `go test ./...`
- `go test -race ./...`
- `Push-Location internal/app; go test -bench=. -benchmem; Pop-Location`
