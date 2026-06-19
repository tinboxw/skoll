# FE6 Bundle Baseline

## Metadata

- Work Item: FE6-01
- Date: 2026-06-19
- Executor: Codex
- Scope: frontend production bundle baseline
- Status: Build passed

## Command

```powershell
cd web
npm run build
```

## Build Result

- Result: Passed
- Tool: Vite 5.4.21
- Modules transformed: 3501
- Build time: 52.65s
- Output directory: `web/dist`

## Largest JavaScript Assets

| Asset | Size | Gzip | Notes |
|---|---:|---:|---|
| `assets/xlsx-DLNWaC59.js` | 332.45 kB | 113.83 kB | Largest chunk; tied to spreadsheet export/import behavior. |
| `assets/index-DOS68aUq.js` | 126.82 kB | 36.60 kB | Large app/page chunk; track route ownership in FE6-02. |
| `assets/el-alert-6BvFKBbP.js` | 120.08 kB | 42.13 kB | Element Plus alert-related chunk. |
| `assets/vue-DfxjscMg.js` | 109.79 kB | 42.86 kB | Vue runtime/vendor chunk. |
| `assets/index-D5tDRDcz.js` | 100.24 kB | 31.71 kB | Large app/page chunk; track route ownership in FE6-02. |
| `assets/index-C4jX92Vs.js` | 99.49 kB | 28.08 kB | Large app/page chunk; track route ownership in FE6-02. |
| `assets/el-table-column-qEkp7vsl.js` | 91.24 kB | 31.39 kB | Table-related Element Plus chunk; feeds FE6-03 table risk work. |

## Largest CSS Assets

| Asset | Size | Gzip | Notes |
|---|---:|---:|---|
| `assets/el-alert-4vtwGh_z.css` | 44.37 kB | 5.51 kB | Element Plus alert styles. |
| `assets/index-BGrcA6Br.css` | 40.18 kB | 5.95 kB | Large page/app stylesheet. |
| `assets/el-table-column-CkuDla-c.css` | 26.05 kB | 3.79 kB | Table styles; feeds FE6-03 table risk work. |
| `assets/index-ihcur7Vx.css` | 23.97 kB | 4.35 kB | Large page/app stylesheet. |
| `assets/el-tab-pane-CFSGIcRU.css` | 22.97 kB | 3.42 kB | Tab panel styles; relevant to plugin/audit panels. |

## Warning Baseline

Current warning set:

- `node_modules/@vueuse/core/dist/index.js` pure annotation comments cannot be interpreted by Rollup and are removed.
- Dart Sass `legacy-js-api` deprecation warning appears twice.

These warnings are treated as existing dependency/toolchain warnings for FE6-01. Later FE6 tasks must record whether they introduce new warnings or only preserve this baseline.

## Performance Risks To Track

| Risk | Follow-up |
|---|---|
| `xlsx` is the largest JS chunk | FE6-02 should confirm whether spreadsheet code is lazy-loaded with the export/import route that needs it. |
| Multiple anonymous `index-*` chunks near or above 100 kB | FE6-02 should map route chunk ownership and document any eager routes. |
| Element Plus table chunk is large | FE6-03 should define table pagination, stable dimensions, and virtualization thresholds. |
| Element Plus alert/tab CSS chunks are visible in the largest CSS list | FE6-05 should confirm plugin/audit heavy panels are deferred and do not force early panel cost. |

## Baseline Acceptance

- Build output is measured.
- Largest JS chunks are recorded.
- Largest CSS chunks are recorded.
- Existing warnings are recorded.
- Follow-up risks are assigned to FE6-02, FE6-03, and FE6-05.
