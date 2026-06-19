# FE0 Frontend Build Baseline

> Work Item: FE0-03  
> Date: 2026-06-19  
> Command: `cd web && npm run build`

## Result

Passed. Vite production build completed and emitted route/component chunks.

## Build Summary

| Metric | Value |
| --- | --- |
| Vite version | 5.4.21 |
| Modules transformed | 3497 |
| Build time | 48.93s |
| Output directory | `web/dist` |

## Largest JavaScript Chunks

| Chunk | Size | Gzip |
| --- | ---: | ---: |
| `assets/xlsx-DLNWaC59.js` | 332.45 kB | 113.83 kB |
| `assets/el-alert-DBKJ8-g9.js` | 165.66 kB | 57.70 kB |
| `assets/index-xTGWmmQg.js` | 125.14 kB | 36.39 kB |
| `assets/vue-DdkF7qVP.js` | 109.78 kB | 42.86 kB |
| `assets/el-table-column-BbKeg47Z.js` | 91.21 kB | 31.43 kB |
| `assets/index-BfW-PdVP.js` | 86.92 kB | 28.29 kB |
| `assets/index-kDuRNV6q.js` | 78.77 kB | 23.28 kB |

## Largest CSS Chunks

| Chunk | Size | Gzip |
| --- | ---: | ---: |
| `assets/el-alert-BuLWHs_T.css` | 41.01 kB | 5.79 kB |
| `assets/index-h5Y-nuOP.css` | 40.79 kB | 6.05 kB |
| `assets/el-table-column-CkuDla-c.css` | 26.05 kB | 3.79 kB |
| `assets/el-tab-pane-CFSGIcRU.css` | 22.97 kB | 3.42 kB |
| `assets/index-B22CYC-z.css` | 21.80 kB | 4.01 kB |

## Warnings

| Warning | Source | Risk | Follow-up |
| --- | --- | --- | --- |
| Dart Sass legacy JS API deprecation | Sass toolchain | Medium-term build compatibility risk before Dart Sass 2.0.0 | Track under FE6 build hygiene. |
| Rollup cannot interpret misplaced `/* #__PURE__ */` comments | `node_modules/@vueuse/core/dist/index.js` | Low; Rollup removes the comments | Track dependency upgrade or Vite/Rollup warning cleanup under FE6. |

## Dependency Risk Notes

1. `xlsx` is the largest gzip chunk and is tied to batch user import. FE6 should consider route/component-level isolation if it leaks into unrelated routes.
2. Element Plus component chunks are expected for the current UI stack, but heavy Plugin/Audit/Permission routes should keep route-level lazy loading.
3. Build output currently has no hard size budget. FE6 should define route-level warning thresholds.

## Verification

```powershell
cd web
npm run build
```

Result: Passed with the warnings listed above.
