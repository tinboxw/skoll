# FE5 Frontend Typecheck Gate

> Work Item: FE5-01  
> Date: 2026-06-19  
> Scope: Vue 3 + TypeScript frontend type safety gate

## Gate

Every frontend code change that touches `web/src`, `web/package.json`, `web/tsconfig.json`, frontend API clients, stores, router, permissions, or generated Vue code must pass:

```powershell
cd web
npm run typecheck
```

The script is defined in `web/package.json`:

```json
"typecheck": "vue-tsc --noEmit"
```

## CI

`.github/workflows/ci.yml` includes a `frontend-typecheck` job.

The job:

1. Checks whether `web/package.json` exists.
2. Uses Node.js 20.
3. Installs dependencies with `npm ci` from `web/package-lock.json`.
4. Runs `npm run typecheck` in `web`.

## Failure Policy

Typecheck failures block frontend task acceptance.

Failure records must include:

- Failing command.
- First relevant TypeScript or Vue template diagnostic.
- Affected file and route or workflow when known.
- Retry result after the fix.

## Current Result

Local validation on 2026-06-19:

```powershell
cd web
npm run typecheck
```

Result: Passed.

## Known Limits

Typecheck does not replace:

- `npm run build` for production bundle validation.
- Browser smoke checks for visible workflows.
- Permission-state checks for admin and restricted roles.
