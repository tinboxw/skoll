# Contributing to Skoll

Skoll is an open-source admin framework in foundation-building phase. Contributions should keep the framework reusable, testable, and free of legacy compatibility bridges unless a current contract explicitly requires one.

## Contribution Workflow

1. Fork the repository.
2. Create a focused branch.
3. Pick one issue, work item, or small improvement.
4. Implement the smallest useful change.
5. Run the validation commands that match your change.
6. Open a pull request using the PR template.

## Branch and Commit Guidance

- Keep commits focused and atomic.
- Use descriptive commit messages.
- Avoid mixing refactors and behavior changes in one commit.
- Do not add old API, old route, old data structure, or old plugin compatibility paths.
- Update docs and examples when changing contributor-facing behavior.

## Required Validation

Run the narrowest meaningful checks first, then the broader gates when your change crosses package or UI boundaries:

```bash
gofmt -w <changed-go-files>
go test ./...
```

Frontend changes should also run:

```bash
cd web
npm run typecheck
npm run build
```

Plugin manifest or marketplace changes should include:

```bash
go test ./internal/plugin/...
```

## Milestone and Evidence Rules

- Link the issue, work item, or acceptance checklist that motivated the change.
- Include command output summaries, not just "tests pass".
- Include screenshots or smoke notes for visible UI behavior.
- For performance changes, include baseline, current value, delta, and sampling command.

## Security and Secrets

- Never commit production secrets.
- Use example values in docs and store real credentials in a secret manager.
- Report vulnerabilities through [SECURITY.md](SECURITY.md), not public issues.

## Reporting Issues

Use issue templates for bug reports and feature requests. Provide:

- Skoll version or commit.
- Environment and config mode.
- Reproduction steps.
- Expected and actual behavior.
- Logs, screenshots, or API responses when useful.

## Community Expectations

Follow [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md). Be respectful, specific, and focused on making the project easier to run, extend, and review.

## Maintainers

Maintainer expectations and release responsibilities are documented in [MAINTAINERS.md](MAINTAINERS.md).
