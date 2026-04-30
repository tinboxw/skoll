# E18-step1 Parity Closure Checkpoints and Evidence Report Baseline

## Milestone

- ID: `E18-step1`
- Name: `parity closure checkpoints and evidence report baseline`

## Delivered

- Added parity closure checkpoint contract with fields for:
  - `reference_project`
  - `capability`
  - `status` (`completed|partial|planned`)
  - `evidence_links`
  - `known_gap`
  - `owner`
- Added release governance APIs:
  - `PUT /admin/v1/release-governance/parity-closure/checkpoints`
  - `GET /admin/v1/release-governance/parity-closure/checkpoints`
  - `GET /admin/v1/release-governance/parity-closure/report`
- Added parity closure report aggregation fields:
  - reference project set
  - total/completed/known-gap/evidence-linked checkpoint counters
  - compatibility statement
  - known-gap summary
- Added route-level regression assertions for invalid payload rejection and report totals.

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
