## Linked Issue or Work Item

- Closes:
- Work item or milestone:

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
- [ ] `cd web && npm run typecheck` (for frontend/shared contract changes)
- [ ] `cd web && npm run build` (for frontend/docs release gates)
- [ ] `go test -race ./...` (for concurrency-related changes)
- [ ] Benchmark snapshot captured (for performance-sensitive changes)

Validation command outputs (paste key lines):
```text
# gofmt / go test / typecheck / build / race / benchmark summary
```

## Acceptance Criteria

- [ ]
- [ ]
- [ ]

## Reproduction or Review Path

- API request, UI route, smoke script, or doc path:
- Required fixture/config:

## Performance Comparison Data (if relevant)
| Metric | Baseline | Current | Delta | Command |
|---|---:|---:|---:|---|
| Example: ns/op | 0 | 0 | 0% | `go test -bench=. -benchmem ./...` |

Conclusion:
- [ ] Acceptable
- [ ] Needs optimization before merge
- [ ] Not applicable

## DoD Checklist
- [ ] Code compiles and tests pass
- [ ] Clear package boundaries (`cmd/`, `internal/`, `pkg/`)
- [ ] At least one runnable entry or example
- [ ] Documentation reflects latest usage
- [ ] Risks and rollback notes are documented
- [ ] No old API, old route, old data structure, or old plugin compatibility path was added

## Documentation Impact

- [ ] README/docs/examples updated
- [ ] OpenAPI or schema docs updated
- [ ] Not applicable

## Risks and Rollback
- Risks:
- Rollback plan:

## Follow-ups
- Next minimal actions:
