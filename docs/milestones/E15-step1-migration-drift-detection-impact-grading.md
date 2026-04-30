# E15-step1 Migration Drift Detection Impact Grading

## Milestone

- ID: `E15-step1`
- Name: `migration drift detection and impact grading`

## Delivered

- Added migration drift detection API:
  - `POST /admin/v1/db/migrations/drift-detect`
- Implemented drift report generation with:
  - missing expected migration steps
  - unexpected applied migration steps
  - impact grading (`none`, `medium`, `high`)
  - actionable recommendations
- Added audit linkage for drift reports (`actor=dbops`, `action=migration_drift_detect`).
- Added route-level regression tests for success and validation failure paths.

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
