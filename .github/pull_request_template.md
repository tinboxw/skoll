## Milestone
- Milestone label (required): `M0/M1/M2/M3-简短内容`

## Background
- Why this change is needed:

## Scope
- In scope:
- Out of scope:

## Changes
- Key implementation points:
- Affected modules/files:

## Validation
- [ ] `go test ./...`
- [ ] `go test -race ./...` (required for concurrency-related changes)
- [ ] Benchmark snapshot captured

Validation command outputs (paste key lines):
```text
# go fmt / go test / race / benchmark summary
```

## Performance Comparison Data (required)
| Metric | Baseline | Current | Delta | Command |
|---|---:|---:|---:|---|
| Example: ns/op | 0 | 0 | 0% | `go test -bench=. -benchmem ./...` |

Conclusion:
- [ ] Acceptable
- [ ] Needs optimization before merge

## DoD Checklist
- [ ] Code compiles and tests pass
- [ ] Clear package boundaries (`cmd/`, `internal/`, `pkg/`)
- [ ] At least one runnable entry or example
- [ ] Documentation reflects latest usage
- [ ] Risks and rollback notes are documented

## README Parity (required)
- [ ] `README.md` updated
- [ ] `README.en.md` updated
- [ ] If not synced in this PR, parity gap and follow-up issue are documented

## Risks and Rollback
- Risks:
- Rollback plan:

## Follow-ups
- Next minimal actions:
