# E12-step1 Bulk Atomic Operations Baseline

## Milestone

- ID: `E12-step1`
- Name: `admin domain bulk atomic operations baseline`

## Delivered

- Added all-or-nothing bulk APIs:
  - `POST /admin/v1/users/bulk`
  - `POST /admin/v1/configs/bulk`
  - `POST /admin/v1/dictionaries/bulk`
- Bulk handlers perform full pre-validation before applying writes.
- Responses include `atomic=true` and processed `count`.
- Added test coverage for success and validation-failure paths.

## Files

- `internal/app/admin_modules.go`
- `internal/app/admin_modules_test.go`

## Validation

- `go fmt ./...`
- `go test ./internal/app ./internal/module/rbac ./internal/module/storageadapter`
- `go test ./...`
- `go test -race ./...`

## Decisions Applied

- Bulk operations use all-or-nothing semantics.
