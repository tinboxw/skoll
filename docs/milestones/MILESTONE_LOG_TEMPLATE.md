# Milestone Log Template

## Basic Info
- Milestone: `M0/M1/M2/M3-简短内容`
- Date range:
- Owner:
- Related PRs/issues:

## Objectives
- Primary objective:
- Non-goals:

## Delivered
- Completed items:
- Deferred items:

## Validation Evidence
- Commands:
```text
go test ./...
go test -race ./...   # if concurrency related
go test -bench=. -benchmem ./...
```
- Key output summary:

## Performance Comparison Data (required)
| Metric | Baseline | Current | Delta | Command | Verdict |
|---|---:|---:|---:|---|---|
| p95 latency | TBD | TBD | TBD | your command | TBD |
| throughput/QPS | TBD | TBD | TBD | your command | TBD |
| bench ns/op | TBD | TBD | TBD | `go test -bench=. -benchmem ./...` | TBD |

Notes:
- If no measurable metric is available, write `N/A` with reason.

## DoD Review (5 points)
- [ ] Code compiles and tests pass
- [ ] Package boundaries are clear
- [ ] Runnable entry/example exists
- [ ] Documentation is updated
- [ ] Risks/assumptions are documented

Score: `X/5`

## Regression Risks
- High:
- Medium:
- Low:

## README Parity
- `README.md` status:
- `README.en.md` status:
- Parity gap (if any):

## Decision
- Result: `Pass / Conditional Pass / Fail`
- Blocking items:
- Next minimal steps:
