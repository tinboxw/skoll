# E8 Release Governance Closure

## Milestone

- ID: `E8-step1`
- Name: `release governance closure baseline`

## Delivered

- Added release governance service for evidence ingestion and scorecard computation.
- Added release governance APIs:
  - `POST /admin/v1/release-governance/evidence`
  - `GET /admin/v1/release-governance/scorecard/{milestone}`
- Added storage adapter and bootstrap wiring for release governance service.
- Added route and contract tests for scorecard gate behavior.
- Added baseline guide: `docs/community/RELEASE_GOVERNANCE_SCORECARD_BASELINE.md`.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`
- `go test -bench=BenchmarkAdminUsersListEndpoint -benchmem ./internal/app`

## Notes

- Scorecard evaluates the latest evidence for each milestone.
- Performance regression uses ratio: `(current - baseline) / baseline`.
