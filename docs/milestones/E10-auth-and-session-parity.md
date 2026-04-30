# E10 Auth and Session Parity

## Milestone

- ID: `E10`
- Name: `auth and session parity`

## Delivered

- Implemented JWT auth lifecycle API contracts for login, refresh, logout, and session introspection.
- Added in-memory auth session engine with:
  - HS256 JWT access tokens
  - refresh token rotation
  - session revocation and refresh invalidation
- Added route-level public mounting for auth bootstrap endpoints under `/admin/v1/auth/*`.
- Added test coverage in module and admin API layers for lifecycle flow.

## Files

- `internal/module/user/service.go`
- `internal/module/user/service_test.go`
- `internal/module/storageadapter/adapter.go`
- `internal/app/admin_modules.go`
- `internal/app/admin_modules_test.go`
- `docs/planning/E10_AUTH_AND_SESSION_PARITY_PLAN.md`

## Validation Evidence

Commands:

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`
- `go test -run ^$ -bench BenchmarkAdminAuthLoginEndpoint -benchmem ./internal/app`

Key output summary:

- `go test ./...`: PASS
- `go test -race ./...`: PASS
- `BenchmarkAdminAuthLoginEndpoint`: `1384 ns/op`, `2313 B/op`, `18 allocs/op`

## Regression Risks

- High: current JWT signing secret is in-memory default; production-grade secret rotation and external KMS integration are deferred to later parity steps.
- Medium: auth lifecycle routes are public by design for login bootstrap; ingress/rate-limit policy must enforce abuse controls.
- Low: in-memory auth session store is process-local and intended as baseline until persistent adapter rollout.

## README Parity

- README.md status: synced
- README.en.md status: synced
