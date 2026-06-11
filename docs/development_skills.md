# Development Skills

This project uses Codex skills to keep repeated Skoll development work consistent.

## Prepared Skills

| Skill | Purpose |
| --- | --- |
| `skoll-web-ui-design` | Admin UI/UX design, layout review, plugin-console experience, browser review criteria. |
| `skoll-web-development` | Full-stack feature workflow across Go backend, Vue frontend, local review, and milestone commits. |
| `skoll-vue-frontend` | Vue 3 + TypeScript + Element Plus frontend implementation patterns. |
| `skoll-go-backend` | Go backend API, service, store, plugin, auth/RBAC, and test workflow. |

## Usage Rules

- Use `skoll-web-ui-design` when changing visible admin pages, plugin UI, developer portal pages, schema forms, navigation, or responsive layout.
- Use `skoll-web-development` when a task crosses backend and frontend or requires local service review.
- Use `skoll-vue-frontend` when editing `web/src`.
- Use `skoll-go-backend` when editing `cmd`, `internal`, backend plugin code, or backend tests.

## Project Workflow

After each completed milestone:

1. Run targeted validation.
2. Start backend and frontend.
3. Open the frontend and backend review URLs.
4. Commit with a focused message.

Default review URLs:

- Frontend: `http://127.0.0.1:5173/`
- Backend Swagger: `http://127.0.0.1:18080/skoll/docs/swagger`
