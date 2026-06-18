# FE5 Frontend Build Gate

> Work Item: FE5-02  
> Date: 2026-06-19  
> Scope: Vue frontend production bundle gate

## Gate

Every frontend Work Item must pass:

```powershell
cd web
npm run build
```

The script is defined in `web/package.json`:

```json
"build": "vite build"
```

## CI

`.github/workflows/ci.yml` includes a `frontend-build` job.

The job:

1. Checks whether `web/package.json` exists.
2. Uses Node.js 20.
3. Installs dependencies with `npm ci` from `web/package-lock.json`.
4. Runs `npm run build` in `web`.

## Failure Policy

Build failures block frontend task acceptance.

Failure records must include:

- Failing command.
- First relevant Vite, Vue, TypeScript, CSS, or dependency diagnostic.
- Whether the failure happens during transform, render, or chunk generation.
- Retry result after the fix.

## Bundle Notes

Build output must be reviewed when a task adds dependencies, route chunks, heavy tables, uploads/downloads, plugin panels, or generated pages.

Record at least:

- Largest JavaScript chunk.
- New warnings.
- New dependency or asset reason when bundle size changes materially.

## Current Result

Local validation on 2026-06-19:

```powershell
cd web
npm run build
```

Result: Passed.

Known warnings remain:

- Dart Sass legacy JS API deprecation warning.
- Rollup removal of non-positioned `@vueuse/core` pure annotations.

## Known Limits

Build does not replace:

- `npm run typecheck` for full Vue template type validation.
- Browser smoke checks for visible workflows.
- Permission-state checks for admin and restricted roles.
