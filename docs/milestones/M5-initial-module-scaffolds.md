# M5-initial-module-scaffolds

## Basic Info

- Milestone: M5-initial-module-scaffolds
- Date range: 2026-04
- Owner: skoll contributors
- Related PRs/issues: TBD

## Objectives

- Start M5 with first generic module scaffolds.
- Keep package boundaries stable while introducing module-level service packages.
- Add baseline tests and benchmark evidence for later regression comparison.

## Non-Goals

- Full business workflows and persistence integration.
- API routing and controller exposure for new modules.
- Permission model completion.

## Delivered

- Added scaffold services under `internal/module/`:
  - `user`
  - `role`
  - `menu`
  - `audit`
- Added unit tests and benchmark functions for each scaffold service.
- Kept existing runtime paths untouched (`cmd`, `internal/app`, `internal/service`).

## Validation Evidence

Commands:

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`
- `go test -run ^$ -bench "." -benchmem ./internal/module/...`

Key output summary:

- `go test ./...`: PASS
- `go test -race ./...`: PASS
- benchmark snapshot:
  - `internal/module/audit BenchmarkServiceRecent-16`: `1915 ns/op`, `8192 B/op`, `1 allocs/op`
  - `internal/module/menu BenchmarkServiceList-16`: `119119 ns/op`, `57481 B/op`, `4 allocs/op`
  - `internal/module/role BenchmarkServiceList-16`: `124632 ns/op`, `49274 B/op`, `4 allocs/op`
  - `internal/module/user BenchmarkServiceList-16`: `126115 ns/op`, `73880 B/op`, `4 allocs/op`

## Performance Comparison Data (required)

| Metric | Baseline | Current | Delta | Command | Verdict |
| --- | ---: | ---: | ---: | --- | --- |
| audit BenchmarkServiceRecent ns/op | N/A | 1915 | N/A | go test -run ^$ -bench "." -benchmem ./internal/module/audit | Baseline established |
| menu BenchmarkServiceList ns/op | N/A | 119119 | N/A | go test -run ^$ -bench "." -benchmem ./internal/module/menu | Baseline established |
| role BenchmarkServiceList ns/op | N/A | 124632 | N/A | go test -run ^$ -bench "." -benchmem ./internal/module/role | Baseline established |
| user BenchmarkServiceList ns/op | N/A | 126115 | N/A | go test -run ^$ -bench "." -benchmem ./internal/module/user | Baseline established |

## DoD Review (5 points)

- [x] Code compiles and tests pass
- [x] Package boundaries are clear
- [x] Runnable entry/example exists
- [x] Documentation is updated
- [x] Risks/assumptions are documented

Score: 5/5

## Regression Risks

- High: none identified.
- Medium: in-memory scaffolds are not production persistence implementations.
- Low: list operations are O(n log n) due to sorting in current scaffolds.

## README Parity

- README.md status: synced
- README.en.md status: synced
- Parity gap: none
