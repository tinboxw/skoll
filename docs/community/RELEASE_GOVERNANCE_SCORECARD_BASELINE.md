# Release Governance Scorecard Baseline

## Scope

This baseline defines the release governance closure workflow for enhancement milestones.

## Evidence API

- `POST /admin/v1/release-governance/evidence`
- Required evidence fields:
  - `milestone`
  - `go_test_passed`
  - `go_race_passed`
  - `readme_synced`
- Performance evidence fields:
  - `benchmark_ns_per_op`
  - `baseline_ns_per_op`
  - `benchmark_command`

## Scorecard API

- `GET /admin/v1/release-governance/scorecard/{milestone}`
- Optional query parameter:
  - `allowed_regression` (default `0.10`)

## Gate Rules

- Quality gates pass only if latest evidence satisfies:
  - `go_test_passed=true`
  - `go_race_passed=true`
  - `readme_synced=true`
- Performance gate fails when:
  - `(benchmark - baseline) / baseline > allowed_regression`
- `release_ready` is true only when all gates pass.

## Operational Guidance

- Upload one evidence record per milestone closeout.
- Keep benchmark command stable to preserve comparability.
- Treat failed checks as release blockers until evidence is refreshed.
