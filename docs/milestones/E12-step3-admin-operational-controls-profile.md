# E12-step3 Admin Operational Controls Profile

## Milestone

- ID: `E12-step3`
- Name: `admin operational controls profile baseline`

## Delivered

- Added operational control profile API:
  - `GET /admin/v1/admin-ops/control-profile`
- Profile payload includes retention and control hints:
  - audit retention days
  - archive window and batch size
  - actor/action rate-guard hint list
- Added test coverage for control profile endpoint.

## Files

- `internal/app/admin_modules.go`
- `internal/app/admin_modules_test.go`

## Validation

- `go fmt ./...`
- `go test ./internal/app`
- `go test ./...`
- `go test -race ./...`
