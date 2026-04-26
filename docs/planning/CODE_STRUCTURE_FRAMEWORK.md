# Skoll Code Structure Framework (Post-M22 Baseline)

## Purpose

Define the repository structure and bootstrap boundaries for the next enhancement phase, so feature growth does not overload `cmd/skoll/main.go`.

## Design Principles

- Keep `main` tiny and orchestration-only.
- Move startup wiring and runtime policy checks into dedicated internal bootstrap package.
- Preserve clear boundaries:
  - transport in `internal/app`
  - business modules in `internal/module/*`
  - external integration adapters in `internal/integration/*`
  - startup/runtime composition in `internal/bootstrap`

## Current Baseline Structure

```text
.
├── cmd/skoll/
│   └── main.go
├── internal/
│   ├── app/
│   ├── bootstrap/
│   ├── domain/
│   ├── integration/
│   ├── module/
│   └── service/
├── pkg/
├── docs/
└── examples/
```

## Bootstrap Boundary

### `cmd/skoll/main.go`

Only responsibilities:

- call bootstrap runner
- handle top-level fatal exit behavior

### `internal/bootstrap`

Responsibilities:

- parse env/flag runtime configuration
- validate runtime constraints and auth policy
- initialize external dependencies (go-admin bootstrap, nonce store)
- compose app server and mount module routes
- coordinate graceful shutdown lifecycle

## Extension Guidelines

1. New feature wiring should prefer `internal/bootstrap` composition code, not `main`.
2. HTTP handler and route behavior should stay in `internal/app`.
3. Module-specific service logic belongs in `internal/module/<feature>`.
4. Any reusable cross-cutting utility should move into `pkg/` only if it is truly public/reusable.
5. Add tests at the same boundary where logic is introduced:
   - bootstrap behavior in `internal/bootstrap/*_test.go`
   - handler behavior in `internal/app/*_test.go`
   - module behavior in `internal/module/*/*_test.go`

## Iteration-Readiness Checklist

Before starting E1 and beyond:

- `main` remains minimal and stable.
- startup policy and shutdown sequencing are test-covered in bootstrap tests.
- new directories are introduced only when a clear boundary emerges.
- roadmap, feature plan, and bilingual README remain synchronized.
