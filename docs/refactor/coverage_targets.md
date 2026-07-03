# M7 Coverage Targets and Exceptions

> Updated: 2026-07-03
> Scope: M7-02 coverage gate planning for Go packages and CI evidence.

## Baseline

- Command: `go test ./... -covermode=atomic -coverprofile=coverage.out`
- Summary command: `go tool cover -func=coverage.out`
- Current total coverage: 66.6%
- CI evidence: `.github/workflows/ci.yml` uploads `go-coverage` with `coverage.out` and `coverage.txt`.

## Targets

| Scope | Target | Enforcement |
|---|---:|---|
| Repository total | 70% | Track in CI artifact during M7; hard failure can be introduced after exception list burns down. |
| Domain packages | 80% | New or changed domain behavior should include unit tests unless listed as an exception. |
| Service packages | 75% | New service branches should include success and error-path tests. |
| HTTP handlers and middleware | 70% | New endpoints should include `httptest` coverage for success and primary failure paths. |
| Plugin lifecycle/platform | 70% | Plugin install, validation, lifecycle, and marketplace changes should include fixture-isolated tests. |
| Store implementations | 65% | Store behavior should prefer sqlite-backed or memory-backed tests with no tracked fixture mutation. |

## Current Low-Coverage Focus

| Package or area | Current signal | Target | Plan |
|---|---:|---:|---|
| `internal/domain/role` | 37.2% | 80% | Add validation and permission mutation tests. |
| `internal/domain/user` | 38.2% | 80% | Add account/email/password normalization and validation tests. |
| `pkg/utils` | 46.7% | 70% | Add conversion edge cases and time/string helper tests. |
| `internal/store` | 51.9% | 65% | Add transaction and bundle mode tests. |
| `internal/bootstrap` | 55.8% | 70% | Add config/auth policy/middleware branch tests. |
| `internal/service/system` | 68.5% | 75% | Add reset and setting validation error tests. |

## Temporary Exceptions

| Package or area | Reason | Exit Criteria |
|---|---|---|
| `cmd/skoll` | CLI entrypoint and process lifecycle are better covered by smoke tests. | Add command smoke or runner unit tests before enabling package target. |
| `internal/adapter` | Development rollout mock has no production behavior boundary yet. | Add rollout executor contract tests when real adapters are introduced. |
| `internal/cache/local`, `internal/cache/redis`, `internal/cache/memcached` | Adapter behavior needs isolated backend fixtures or fakes. | Add backend-independent adapter contract tests. |
| `internal/event/events` | Event constants/types only. | Cover when event catalog gains parsing or validation behavior. |
| `internal/handler/http/v1/rbac`, `internal/handler/http/v1/role` | Route behavior is currently covered through aggregate router tests, not package-local handler tests. | Add package-local handler tests for bind/policy/check and role grant/revoke paths. |
| `internal/store/sql/mysql`, `internal/store/sql/postgres`, `internal/store/clickhouse` | External database integrations need service containers or explicit integration profile. | Add container-backed integration job or adapter contract fake. |
| `pkg/errors`, `pkg/version` | Small wrappers/constant surfaces with low behavior density. | Add direct tests when error translation/version behavior expands. |
| plugin demo `main` packages | Entrypoints only; demo behavior is covered in service/handler tests. | Add smoke tests if demo binaries become release artifacts. |

## M7 Improvement Plan

1. Protect the current baseline by keeping `go-coverage` artifact in CI for every push and pull request.
2. Raise repository total coverage from 66.6% to 70% by targeting `internal/domain/role`, `internal/domain/user`, `pkg/utils`, and `internal/store`.
3. Add focused handler tests for RBAC and role package-local routes before turning their package targets into hard gates.
4. Convert temporary exceptions into tracked work items when a package gains production behavior or a stable fixture strategy.
5. Revisit hard-fail thresholds after M7-03 performance baselines and M7-04 docs are complete, so quality gates do not block framework bootstrap work without clear remediation.
