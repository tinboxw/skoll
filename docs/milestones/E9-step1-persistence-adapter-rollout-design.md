# E9-step1 Persistence Adapter Rollout Design

## Milestone

- ID: `E9-step1`
- Name: `persistence adapter rollout design baseline`

## Delivered

- Added storage adapter mode factory and explicit mode validation.
- Added runtime configuration entrypoint for adapter mode selection in bootstrap.
- Added tests for adapter mode factory and runtime config log coverage.
- Added persistence rollout design document and phased migration strategy.

## Files

- `internal/module/storageadapter/factory.go`
- `internal/module/storageadapter/factory_test.go`
- `internal/bootstrap/run.go`
- `internal/bootstrap/run_test.go`
- `docs/planning/E9_STEP1_PERSISTENCE_ADAPTER_ROLLOUT_DESIGN.md`

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`

## Notes

- Persistent modes (`mysql`, `postgres`) are intentionally gated as planned-but-not-implemented in this step.
- Default mode remains `memory` to preserve current runtime behavior.
