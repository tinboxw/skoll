# E18-step2 Release Blocking Policy and Automated Decision Baseline

## Milestone

- ID: `E18-step2`
- Name: `release blocking policy and automated decision baseline`

## Delivered

- Added release blocking policy contract with fields for:
  - `allowed_regression_ratio`
  - `block_on_go_test_failure`
  - `block_on_go_race_failure`
  - `block_on_readme_not_synced`
  - `block_on_missing_evidence`
- Added release governance APIs:
  - `PUT /admin/v1/release-governance/blocking-policy`
  - `GET /admin/v1/release-governance/blocking-policy`
  - `GET /admin/v1/release-governance/block-decision/{milestone}`
- Added automated block decision derivation from scorecard failed checks and policy toggles.
- Added route-level regression assertions for:
  - policy update/get
  - block decision evaluation with strict regression threshold

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
