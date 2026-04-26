# AGENTS.md

## Scope
- These instructions apply to the whole repository.
- Keep guidance minimal and rely on linked docs when details already exist.

## Project Context
- Project name: Skoll.
- Intended direction from docs: Go project focused on high performance and high concurrency.
- Current state: repository scaffold only (no Go source files or module metadata yet).

## Source Docs
- Chinese README: [README.md](README.md)
- English README: [README.en.md](README.en.md)

## Agent Workflow
- Before implementing features, check whether the requested structure already exists.
- Do not assume hidden architecture; if missing, propose and implement the smallest sensible Go layout.
- Keep changes focused on the user request and avoid unrelated refactors.

## Go Conventions (When Bootstrapping)
- Prefer standard Go tooling first: `go fmt ./...`, `go test ./...`.
- For concurrency-related work, validate with `go test -race ./...`.
- For performance-sensitive paths, add benchmarks and use `go test -bench=. -benchmem ./...`.
- If a module is not initialized yet, confirm module path with the user before running `go mod init`.

## Suggested Layout (As Codebase Grows)
- `cmd/` for executable entry points.
- `pkg/` for public library packages.
- `internal/` for private implementation details.
- `docs/` for architecture/design notes.
- `examples/` for usage examples.

## Documentation Updates
- When adding build/test commands or structure, update both README files or clearly note language parity gaps.

## Contribution Notes
- Follow documented contribution flow from README files: fork, create feature branch, commit, open PR.