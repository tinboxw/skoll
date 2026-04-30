# E18 Release Governance and Parity Closure Plan

## Objective

Finalize full parity closure with capability checklist evidence, automated release scorecards, and blocking regression governance.

## Scope

- Capability-by-capability closure checklist for both reference projects.
- Release scorecard automation with benchmark threshold blocking.
- Public parity closure report and compatibility statement.
- DoD and regression-risk review package.

## Acceptance

- All E10-E18 capability checkpoints have evidence links.
- Scorecard blocks threshold-breaking regressions by policy.
- Compatibility statement and known-gap section are published.
- Final milestone review signs off parity closure.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`
- `go test -bench=. -benchmem ./...`
