# E17-step1 Control-Plane Endpoint Guardrails Baseline

## Milestone

- ID: `E17-step1`
- Name: `control-plane endpoint guardrails baseline`

## Delivered

- Added endpoint-level hardening guardrail contract with fields for:
  - `rate_limit_rpm`
  - `timeout_millis`
  - `circuit_error_threshold`
  - `circuit_open_window_sec`
  - `enabled`
- Added admin APIs:
  - `PUT /admin/v1/system/hardening/endpoint-guardrails`
  - `GET /admin/v1/system/hardening/endpoint-guardrails`
- Added validation for required endpoint and positive hardening profile values.
- Added audit event for guardrail updates (`endpoint_guardrails_updated`).
- Added route-level regression coverage for success and invalid policy payload cases.

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
