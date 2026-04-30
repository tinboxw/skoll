# E15-step3 Controlled SQL Classes And Dual Confirmation

## Milestone

- ID: `E15-step3`
- Name: `controlled SQL classes and destructive dual confirmation`

## Delivered

- Upgraded controlled SQL governance to class-based execution policy:
  - `read_only`
  - `write_guarded`
  - `destructive_confirmed`
- Added automatic SQL class inference when `sql_class` is omitted.
- Enforced safeguards:
  - read-only class blocks write/destructive SQL.
  - write-guarded class requires confirmation token for write statements.
  - destructive-confirmed class requires explicit dual confirmation:
    - `confirm_token = I_UNDERSTAND`
    - `confirm_token_dual = CONFIRM_DESTRUCTIVE_SQL`
    - `allow_dangerous = true`
- Extended API response with resolved SQL class and safety result.
- Added route-level regression coverage for all SQL classes and confirmation paths.

## Files

- `internal/app/admin_modules.go`
- `internal/app/admin_modules_test.go`
- `docs/community/DATABASE_OPS_GOVERNANCE_BASELINE.md`
- `docs/planning/FEATURE_PARITY_PLAN.md`
- `README.md`
- `README.en.md`

## Validation

- `go fmt ./...`
- `go test ./internal/app`
- `go test ./...`
- `go test -race ./...`
- `Push-Location internal/app; go test -bench=. -benchmem; Pop-Location`
