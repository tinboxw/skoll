# Skoll Plugin Runtime Acceptance Log

> Batch: `skoll-plugin-runtime-2026-07-21`
> Append evidence only after the Work Item reaches `Review` and passes independent acceptance.
> Failed acceptance must be recorded before the same Work Item returns to `Doing`.

## Entry Template

```text
## <Work Item ID> <Title>

- Date:
- Owner:
- Status flow:
- Scope:

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |

### Verification Commands

Result:

### Impact Review

- API/OpenAPI:
- Permission/audit:
- Migration/seed:
- Frontend/i18n:
- Documentation:
- Compatibility: none; current contracts only.

### Commit

`<work-item-id>: <short summary>`
```

## PR0-00 Establish Official Plugin Runtime Batch

- Date: 2026-07-21
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Done`
- Scope: Create the official parent board, atomic Work Item table, acceptance log, and current document indexes.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Batch files | Pass | Board, Work Items, and acceptance log exist under `docs/refactor/current/` |
| Atomic structure | Pass | Every Work Item includes priority, Skill, description, dependencies, deliverables, acceptance, verification, and status |
| Milestones | Pass | PR1-PR5 cover runtime, platform services, generator, frontend experience, and independent-industry proof |
| Collaboration | Pass | Task claim, review, failure retry, acceptance, and one-commit rules are explicit |
| Project direction | Pass | The batch forbids compatibility layers, legacy formats, dual paths, and transition adapters |

### Verification Commands

Result: document links, Work Item field/count/status checks, archive boundary check, and `git diff --check` passed.

### Impact Review

- API/OpenAPI: no runtime contract changed.
- Permission/audit: planned as explicit Work Items.
- Migration/seed: planned as explicit Work Items.
- Frontend/i18n: planned as explicit Work Items.
- Documentation: current execution indexes now point to this batch.
- Compatibility: none; current contracts only.

### Commit

`PR0-00: establish plugin runtime batch`

## PR1-01 Execute Declared External Plugin Routes

- Date: 2026-07-21
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Done`
- Scope: Replace placeholder route success with declared-route validation and real HTTP execution against `service_base_url`.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Real execution | Pass | Full Router test reaches an `httptest` plugin backend through the `/skoll` prefix and manifest-declared route |
| Request preservation | Pass | Method, base-path join, declared path, query, body, and ordinary request header assertions pass |
| Response preservation | Pass | Backend HTTP 201, response header, and raw JSON body reach the caller unchanged |
| Fail closed | Pass | Missing backend returns 503, unreachable backend returns 502, disabled plugin returns 503, and missing executor returns 502; no placeholder 200 remains |
| Declaration boundary | Pass | A method/path pair absent from `api.routes` is not executed |
| Contract documentation | Pass | Chinese-default and English plugin API references define `service_base_url`, forwarding behavior, and stable failure codes |
| Quality gates | Pass | Focused tests, race tests, vet, full Go suite, diff check, and CodeGraph sync pass |

### Verification Commands

```powershell
go test ./internal/bootstrap ./internal/handler/http/... ./internal/plugin/... -count=1
go test -race ./internal/bootstrap ./internal/handler/http ./internal/plugin/... -count=1
go vet ./internal/bootstrap ./internal/handler/http/... ./internal/plugin/...
go test ./... -count=1
codegraph sync .
git diff --check
```

Result: all commands passed; CodeGraph synchronized 4 changed Go files.

### Impact Review

- API/OpenAPI: no new path or schema; execution semantics for existing declared plugin routes changed from placeholder success to real backend response.
- Permission/audit: existing JWT, route permission, and audit middleware remain in front of the executor.
- Migration/seed: no impact.
- Frontend/i18n: no client change; stable Chinese-default runtime error messages added.
- Documentation: Chinese and English plugin API contracts updated.
- Compatibility: none; current manifest contracts only.

### Commit

`PR1-01: execute external plugin routes`
