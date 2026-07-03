# M7 Frontend Performance Sampling

> Updated: 2026-07-03
> Scope: M7-03-02 frontend build, route loading, table/list, and plugin panel baseline.

## Build Baseline

- Command: `cd web && npm run build`
- Result: Passed
- Build time observed: 51.10s
- Transformed modules: 3513
- Existing warnings:
  - Dart Sass legacy JS API deprecation warning.
  - Rollup removes two misplaced `/* #__PURE__ */` comments from `@vueuse/core`.

## Largest Assets

| Asset | Size | Gzip |
|---|---:|---:|
| `assets/xlsx-DLNWaC59.js` | 332.45 kB | 113.83 kB |
| `assets/index-bzYb7Cus.js` | 126.61 kB | 36.50 kB |
| `assets/el-alert-Dja3qNHd.js` | 120.08 kB | 42.13 kB |
| `assets/index-BcLxRDxN.js` | 110.74 kB | 30.22 kB |
| `assets/vue-CFvUB0Vb.js` | 109.79 kB | 42.86 kB |
| `assets/index-Bwi2zZwq.js` | 103.95 kB | 32.72 kB |
| `assets/el-table-column-DpxvMuDz.js` | 91.23 kB | 31.38 kB |

## First-Screen Baseline

| Area | Baseline |
|---|---|
| Admin shell | `web/src/router/index.ts` uses route-level dynamic imports for all major pages. |
| Login route | Login page is lazily imported as `LoginPage`. |
| Default route | `/` and `/skoll` redirect through `resolveSafeDefaultHomePath()` without importing heavy pages directly. |
| Heavy dependency risk | `xlsx` remains the largest emitted JS asset and should stay isolated from first-screen workflows unless import analysis proves otherwise. |

## Route-Switch Baseline

| Route Group | Import Mode | Notes |
|---|---|---|
| Dashboard | Dynamic import | Major dashboard route is lazy-loaded. |
| User | Dynamic import | List/add/batch/edit are split into separate route chunks. |
| Role | Dynamic import | List and edit routes are split. |
| Permission/Menu/Dictionary/Organization | Dynamic import | Admin registry pages are lazy-loaded. |
| Audit | Dynamic import | Audit route is separated from shell and login. |
| Plugin | Dynamic import | Plugin page is lazy-loaded; plugin home paths wait for plugin bootstrap before resolving. |

## Heavy Table/List Baseline

| Surface | Current Baseline | Risk |
|---|---|---|
| User list | Dedicated route chunk and API-backed page. | Large table performance depends on server pagination and stable table dimensions. |
| Role list | Dedicated route chunk. | Grant/revoke interactions should avoid full-page reload patterns. |
| Audit query | Dedicated route chunk. | Audit filters and export must avoid client-side full-history scans. |
| Permission/menu registries | Dedicated route chunks. | Tree/table views should keep bounded data and predictable loading states. |
| Plugin list | Dedicated route chunk. | Marketplace/risk/lifecycle panels should avoid eager loading all heavy details. |

## Plugin Panel Baseline

| Panel/Flow | Baseline |
|---|---|
| Plugin list/detail | Loaded through the plugin route chunk. |
| Plugin route resolution | `waitForPluginBootstrap()` only runs for plugin home paths. |
| Risk/config/log panels | Treat as heavy panels; continue deferring detail work until the plugin page is opened. |

## Follow-Up Thresholds

| Signal | Follow-up Trigger |
|---|---|
| Build time | Build exceeds 60s on local baseline without dependency explanation. |
| Largest JS asset | Any non-vendor route chunk exceeds 150 kB gzip. |
| `xlsx` asset | Spreadsheet feature imports leak into first-screen or unrelated admin routes. |
| Route lazy loading | A major page switches from dynamic import to eager import. |
| Large tables | User, role, plugin, audit, or permission pages fetch unbounded lists by default. |
| Plugin panels | Risk/config/log/history panels load before the plugin route is opened. |

## Validation Commands

```powershell
cd web
npm run build
```

```powershell
rg -n "const .*Page = \\(\\) => import" web/src/router/index.ts
rg -n "xlsx-DLNWaC59|el-table-column|assets/vue" docs/refactor/frontend_performance_sampling.md
```
