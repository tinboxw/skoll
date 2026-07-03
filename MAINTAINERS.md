# Maintainer Guide

## Responsibilities

Maintainers keep Skoll usable as an open-source admin framework. The practical duties are:

- triage issues and pull requests;
- keep task board, work items, and acceptance logs consistent;
- protect current API, plugin, permission, audit, and frontend contracts;
- reject accidental legacy compatibility paths;
- keep examples, docs, and release checklists current;
- respond to security reports through the private process in `SECURITY.md`.

## Review Expectations

Every pull request should have:

- a clear problem statement;
- focused code or docs changes;
- validation commands and result summaries;
- updated docs/examples when behavior changes;
- no unrelated formatting churn.

Prefer small PRs. Ask contributors to split broad work when review risk becomes unclear.

## Release Expectations

Before a release candidate, maintainers should verify:

- `go test ./...`;
- `cd web && npm run typecheck`;
- `cd web && npm run build`;
- OpenAPI and plugin manifest schema checks;
- examples and quick start still work;
- release checklist, task board, work items, acceptance log, and git history agree.

## Decision Style

Be lightweight and explicit. Record decisions in issues, pull requests, docs, or acceptance logs instead of relying on private context.
