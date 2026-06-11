# Development Skills

This project uses Codex skills to keep repeated Skoll development work consistent.

## Prepared Skills

| Skill | Purpose |
| --- | --- |
| `skoll-web-ui-design` | Admin UI/UX design, layout review, plugin-console experience, browser review criteria. |
| `skoll-web-development` | Full-stack feature workflow across Go backend, Vue frontend, local review, and milestone commits. |
| `skoll-vue-frontend` | Vue 3 + TypeScript + Element Plus frontend implementation patterns. |
| `skoll-go-backend` | Go backend API, service, store, plugin, auth/RBAC, and test workflow. |
| `skoll-plugin-platform` | Plugin manifest, config schema, UI menu, lifecycle, developer portal, marketplace, rollout, and rollback. |
| `skoll-permission-rbac` | Roles, permissions, API auth policy, menu permission, route guards, data scope, and super_admin behavior. |
| `skoll-quality-gate` | Milestone validation, test command selection, local review checks, and commit readiness. |
| `skoll-open-source-framework` | Open-source framework positioning, starter experience, examples, compatibility, and contribution surface. |
| `skoll-code-generator` | Model-to-CRUD generation, backend/frontend scaffolding, menu/permission generation, and plugin templates. |
| `skoll-devops-deployment` | Docker, Compose, Kubernetes, CI, release packaging, environment variables, health probes, and upgrades. |
| `skoll-docs-writer` | User, developer, plugin, API, roadmap, changelog, and index documentation. |
| `skoll-security-hardening` | JWT/auth, plugin signatures, upload validation, risk controls, secrets, audit, CORS, and rate limits. |
| `skoll-api-contracts` | HTTP API design, OpenAPI, frontend API clients, response envelopes, error codes, and Swagger behavior. |
| `skoll-observability-audit` | Audit logs, operation logs, plugin task logs, metrics, health/readiness, and admin monitoring UI. |
| `skoll-data-dictionary-config` | Dictionaries, system settings, plugin config, schema forms, validation metadata, and config audit. |
| `skoll-testing-automation` | Unit, integration, smoke, browser, API contract, plugin fixture, and CI regression tests. |

## Usage Rules

- Use `skoll-web-ui-design` when changing visible admin pages, plugin UI, developer portal pages, schema forms, navigation, or responsive layout.
- Use `skoll-web-development` when a task crosses backend and frontend or requires local service review.
- Use `skoll-vue-frontend` when editing `web/src`.
- Use `skoll-go-backend` when editing `cmd`, `internal`, backend plugin code, or backend tests.
- Use specialized skills when a task touches a named subsystem such as plugin platform, RBAC, security, API contracts, docs, deployment, or code generation.

## Additional Candidates

The current 16 skills cover the main infrastructure phase. Consider adding these later when the matching subsystem becomes active:

| Candidate | Purpose |
| --- | --- |
| `skoll-file-storage` | File upload, object storage, local/S3 adapters, file permissions, thumbnails, and cleanup jobs. |
| `skoll-workflow-jobs` | Background jobs, workflow approvals, scheduled tasks, queues, retries, and task UI. |
| `skoll-i18n-accessibility` | Internationalization, locale resources, keyboard support, accessibility checks, and UI copy consistency. |
| `skoll-performance-scaling` | Backend profiling, frontend bundle budgets, caching, pagination, query tuning, and large-table UX. |
| `skoll-upgrade-migration` | Version compatibility, database migrations, plugin migration hooks, deprecation policy, and release upgrades. |
| `skoll-community-governance` | Issue templates, PR templates, contribution rules, maintainership, roadmap communication, and release governance. |

## Project Workflow

After each completed milestone:

1. Run targeted validation.
2. Start backend and frontend.
3. Open the frontend and backend review URLs.
4. Commit with a focused message.

Default review URLs:

- Frontend: `http://127.0.0.1:5173/`
- Backend Swagger: `http://127.0.0.1:18080/skoll/docs/swagger`
