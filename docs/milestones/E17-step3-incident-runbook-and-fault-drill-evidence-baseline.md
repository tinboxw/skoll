# E17-step3 Incident Runbook and Fault-Drill Evidence Baseline

## Milestone

- ID: `E17-step3`
- Name: `incident runbook and fault-drill evidence baseline`

## Delivered

- Added incident runbook profile contract with fields for:
  - `domain`
  - `severity`
  - `runbook`
  - `owner`
  - `escalation`
  - `mitigation_sla_seconds`
- Added fault-drill evidence contract with fields for:
  - `scenario`
  - `domain`
  - `injector`
  - `mitigation_evidence`
  - `residual_risk`
- Added admin APIs:
  - `PUT /admin/v1/system/hardening/incident-runbooks`
  - `GET /admin/v1/system/hardening/incident-runbooks`
  - `POST /admin/v1/system/hardening/fault-drills`
  - `GET /admin/v1/system/hardening/fault-drills`
- Added validation and sorting/listing logic for runbooks and drills.
- Added audit events:
  - `incident_runbooks_updated`
  - `fault_drill_recorded`
- Added route-level regression coverage for invalid and valid submissions.

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
