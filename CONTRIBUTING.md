# Contributing to Skoll

## Workflow

1. Fork the repository.
2. Create a feature branch.
3. Implement the smallest focused change.
4. Run required validation commands.
5. Open a pull request using the PR template.

## Branch and Commit Guidance

- Keep commits focused and atomic.
- Use descriptive commit messages.
- Avoid mixing refactors and behavior changes in one commit.

## Required Validation

Run locally before opening a PR:

```bash
go fmt ./...
go test ./...
go test -race ./...
go test -bench=. -benchmem ./...
```

If your change is not concurrency-related, still include test and benchmark evidence where applicable.

## Milestone and Evidence Rules

- Use milestone labels in the format `M0/M1/M2/M3-brief-topic` for PR context.
- Include performance comparison data: baseline, current, delta, and sampling command.
- Keep `README.md` and `README.en.md` synchronized for usage-facing changes.

## Security and Secrets

- Never commit production secrets.
- Use `.env.production.example` and secret managers for deployment.
- Follow `docs/planning/ADMIN_AUTH_SECURITY_RUNBOOK.md` for auth operations.

## Reporting Issues

Use issue templates for bug reports and feature requests. Provide reproduction steps and expected behavior.

## Code of Conduct

Be respectful, constructive, and focused on technical outcomes.
