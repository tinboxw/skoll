# M2 Regression Record

> Work Item: M2-06-03  
> Date: 2026-06-19  
> Environment: Windows, PowerShell, backend tests with `CGO_ENABLED=0`

## Commands

```powershell
$env:CGO_ENABLED='0'
go test ./...

cd web
npm run build
```

## Results

| Gate | Result | Notes |
| --- | --- | --- |
| Go full test suite | Passed | `go test ./...` passed with `CGO_ENABLED=0`. |
| Frontend build | Passed | Vite production build completed. |
| Frontend warnings | Accepted | Existing Rollup pure annotation warnings from `@vueuse/core`; existing Dart Sass legacy JS API warnings. |

## Regression Scope

The Go test run covered:

- bootstrap and middleware
- audit domain, service, handler, repository, memory store, SQL store
- permission/menu domain, service, handler, stores
- plugin manager and plugin handlers
- frontend-independent integration tests

The frontend build covered:

- Vue type-aware production compilation through Vite
- route/page chunk generation
- audit page bundle integration

## Follow-Up Notes

1. Keep using `CGO_ENABLED=0` on this Windows environment unless a task explicitly needs SQLite-backed cgo tests.
2. Frontend warning cleanup can be tracked later as FE6 performance/build hygiene; it is not blocking M2.
