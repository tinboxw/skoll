# Skoll Frontend Experience Target

> Work Item: `PR4-01`
> Date: 2026-07-22
> Status: Frozen design and acceptance baseline for `PR4-02` through `PR4-06`.
> Product shape: dense, calm, task-oriented open-source admin console and plugin host.

## Decisions

1. Element Plus is the only general-purpose control system. Lucide is the icon system. Shared Skoll components own page, state, table, form, drawer, confirmation, and plugin-host patterns.
2. The current UI is only partially unified. Element Plus exists, but 24 native buttons, 9 native inputs, one native table, page-local tables, and duplicated interaction styles remain.
3. The current theme implementation is a preset switch, not a complete multi-theme system. Color scheme and density must become independent axes: `light|dark` x `comfortable|compact`.
4. No compatibility UI, old theme payload migration, legacy route, dual component path, or fallback design is permitted. Replace incomplete contracts directly.
5. Operational pages use full-width work surfaces. Gradients, decorative shells, nested cards, oversized rounding, and explanatory filler are removed.
6. A state is not complete until it works in desktop and narrow viewports with keyboard, permission, bilingual, and performance acceptance.

## Measured Baseline

| Signal | Current evidence | Verdict |
| --- | --- | --- |
| Views | 35 Vue views | Broad functional surface exists |
| Shared page shell | 20 views use `PageShell`; 15 do not | Inconsistent page hierarchy |
| Shared data table | 24 views use `DataTable`; plugin and several foundation pages render local `el-table` trees | Duplicate table behavior remains |
| Native controls | 24 buttons, 9 inputs, one table | Element Plus is not yet unified |
| Theme modes | `light`, `dark`, `compact` | Compact incorrectly owns a light color scheme; dark compact is impossible |
| Theme tokens | Core colors, spacing, radius, control height | Typography, layers, focus, motion, charts, overlays, status, and component density are incomplete |
| Raw visual values | Multiple colors outside the token registry plus operational gradients | Theme leaks exist |
| Plugin console | 3,062 lines, 14 cards, many local tables, hard-coded Chinese/English copy, no `PageShell` | Highest UI complexity and drift risk |
| Static checks | i18n/a11y checks cover the 15 Pharma OA views and selected shared invariants | They do not prove the whole console or browser behavior |
| Browser smoke | Script reports fixtures as `Ready` but explicitly leaves Playwright execution pending | No executable end-to-end visual gate |
| Production preview | Built HTML requests `/skoll/assets/*`; `vite preview` serves `/assets/*` because preview uses the dev base | `/skoll/login` is blank in production preview |
| Login visual | Desktop is functional but generic; 390x844 card overflows because `92vw` ignores parent padding | Narrow layout fails |
| Mobile shell | Sidebar is always `display:none` at <=860px; header toggle only changes collapsed state | Authenticated mobile navigation is unreachable |
| Host initial bundle | About 114.5 KB gzip JavaScript plus 5.2 KB gzip CSS from `index.html` preloads | Acceptable baseline, budget still unenforced |
| Generated plugin bundle | About 337 KB gzip JavaScript and 48 KB gzip CSS | Exceeds the target entry budget |

## Target Architecture

| Layer | Owner | Target contract | Acceptance |
| --- | --- | --- | --- |
| App shell | Frontend Engineering | `AppShell`, responsive sidebar drawer, header command groups, pinned tabs, content viewport | Navigation remains reachable at 1440x900, 1024x768, and 390x844 without overlap |
| Page structure | UI Platform | `PageShell` + `PageToolbar` + optional `FilterBar` | Every operational route has one title/action hierarchy and no page-local replacement |
| Data display | UI Platform | `DataTable` with table presets, selection, actions, pagination, virtual mode, and state slots | Tables keep stable columns/actions and expose loading, empty, stale-error, forbidden, and retry states |
| Forms | UI Platform | `SchemaForm`/Element Plus forms with shared validation and save state | Labels, errors, help, disabled, saving, success, and backend failure are predictable |
| Detail/actions | UI Platform | `DetailDrawer`, Element Plus dialog, `ConfirmAction`, toast policy | Focus returns, destructive intent is explicit, duplicate submission is blocked |
| Plugin control plane | Plugin Frontend | Split overview, install, marketplace, risk, developer tasks, detail, and configuration modules | No 3,000-line page; every tab is independently testable and permission-aware |
| Theme | Design Systems | Semantic tokens plus independent scheme and density stores | Four theme/density combinations pass contrast and visual matrices |
| Copy/accessibility | Product + QA | Global locale inventory and browser accessibility suite | Zero hard-coded visible business copy; WCAG 2.2 AA workflow checks pass |
| Visual/performance | Frontend QA | Playwright screenshots, browser interaction tests, bundle and runtime budgets | CI fails on visual drift, blank render, budget regression, or inaccessible primary flow |

## Primary Workflow Inventory

| Workflow | Current surface | Main gaps | Owner | Target components | Work Item |
| --- | --- | --- | --- | --- | --- |
| Login and profile | `Login`, `Profile` | Native controls, generic hierarchy, mobile overflow, missing forbidden/empty semantics | Frontend Engineering | Element Plus form, auth shell, `StateBlock` | PR4-02, PR4-05 |
| Global navigation | `App`, `Header`, `Sidebar`, `PinnedTabs` | Mobile navigation unreachable, ad-hoc segmented buttons/menu, gradients, raw colors | UI Platform | Responsive app shell, `el-segmented`, dropdown, drawer | PR4-02, PR4-03, PR4-05 |
| Dashboard | `Dashboard` | Static store counters dominate; no loading state; local metric/panel patterns | Product + Frontend | Operational metric strip, attention queue, shared state and links | PR4-02, PR4-05 |
| Identity and access | User, Role, Permission, Menu, Organization | Mixed page shells, repeated tables/forms, dense permission flows lack one hierarchy | IAM Frontend | Page shell, table presets, scope indicator, detail drawer | PR4-02, PR4-04 |
| Workflow and workbench | Workflow, Todo Center, Form Builder | Native summary buttons, hard-coded English copy, repeated summary tile CSS | Workflow Frontend | `el-segmented`, work queue table, schema form, timeline drawer | PR4-02, PR4-04 |
| Foundation data | Dictionary, File, Setting, Audit | Mixed shells, local tables, native file input, uneven permission and empty states | Platform Frontend | Page shell, upload, table/filter presets, audit drawer | PR4-02, PR4-05 |
| Plugin operations | Plugin overview/install/lifecycle/config | Monolithic page, nested cards, hard-coded copy, duplicate tables, mixed states | Plugin Frontend | Plugin workspace modules, install wizard, lifecycle command bar | PR4-02, PR4-04, PR4-05 |
| Marketplace and risk | Local marketplace and risk tabs | Hidden inside the monolith, local labels and tables, weak task recovery hierarchy | Plugin Frontend | Marketplace table, risk matrix, retry drawer | PR4-02, PR4-05 |
| Developer portal | Projects, releases, rollout, logs, rollback | Many nested tabs/tables, no independent route/test boundary | Plugin Frontend | Developer workspace routes, task table, log viewer | PR4-02, PR4-05, PR4-06 |
| Pharma OA overview | Pharma dashboards/compliance | Local summary controls, raw colors, dashboard state inconsistency | Pharma Product + Frontend | Industry dashboard kit, status/risk tokens | PR4-03, PR4-05 |
| Pharma OA HR/CRM | Employee, customer, follow-up, opportunity | Large local forms, some raw colors, repeated list/detail behavior | Pharma Frontend | Domain table presets, schema forms, detail drawer | PR4-02, PR4-03 |
| Pharma OA supply chain | Purchase inbound, sales, cold chain, qualification | Missing empty state in purchase/sales; critical statuses need one semantic system | Pharma Frontend | Inventory tables, batch/temperature status, exception drawer | PR4-03, PR4-05 |
| Pharma OA quality/finance/content | Complaints, recalls, contracts, invoices, announcements | Dense actions and status semantics need visual/theme/browser matrix | Pharma Frontend + QA | Quality timeline, finance table, guarded actions | PR4-03, PR4-04, PR4-05 |

## Required State Matrix

`Required` means the workflow must have executable evidence for the state. `N/A` requires a documented reason in the test fixture.

| Workflow group | Loading | Empty | Error/retry | Forbidden | Offline/degraded | Saving/progress | Success | Destructive | Narrow |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Auth/profile | Required | N/A | Required | Required | Required | Required | Required | Required | Required |
| Shell/navigation | Required | Required | Required | Required | Required | N/A | Required | N/A | Required |
| Dashboard | Required | Required | Required | Required | Required | N/A | N/A | N/A | Required |
| IAM | Required | Required | Required | Required | Required | Required | Required | Required | Required |
| Workflow/workbench | Required | Required | Required | Required | Required | Required | Required | Required | Required |
| Foundation data | Required | Required | Required | Required | Required | Required | Required | Required | Required |
| Plugin control plane | Required | Required | Required | Required | Required | Required | Required | Required | Required |
| Pharma OA | Required | Required | Required | Required | Required | Required | Required | Required | Required |

## Inconsistency Register

| ID | Priority | Evidence | Owner | Target | Measurable acceptance | Work Item |
| --- | --- | --- | --- | --- | --- | --- |
| UI-001 | P0 | Sidebar disappears below 860px and cannot be reopened | UI Platform | Responsive app shell | All primary routes are keyboard/touch reachable at 390x844 | PR4-02 |
| UI-002 | P0 | Production preview serves asset paths inconsistent with built base | Frontend QA | Vite base/preview contract | Built `/skoll/login` renders nonblank and every referenced asset returns its declared MIME type | PR4-06 |
| UI-003 | P0 | Plugin page is 3,062 lines and owns six product areas | Plugin Frontend | Routed plugin workspace modules | No route component exceeds 800 lines; each area has focused tests | PR4-02 |
| UI-004 | P0 | Browser smoke does not execute a browser runner | Frontend QA | Playwright workflow suite | Login, dashboard, plugin, permission denial, audit, and Pharma flow execute in CI | PR4-06 |
| UI-005 | P0 | Compact mode forces light scheme | Design Systems | Independent scheme/density state | Light/dark each run in comfortable/compact and persist one current payload | PR4-03 |
| UI-006 | P0 | Generated plugin entry is about 337 KB gzip JS | Generator + Performance | On-demand Element Plus and route splitting | Generated plugin entry JavaScript <=180 KB gzip | PR4-06 |
| UI-007 | P1 | 24 native buttons and 9 native inputs remain | UI Platform | Element Plus/shared controls | Zero unjustified native controls; file selection uses `el-upload` | PR4-02 |
| UI-008 | P1 | 15 views do not use `PageShell` | UI Platform | Shared page hierarchy | Every operational route uses the shared shell or an approved auth/plugin-host shell | PR4-02 |
| UI-009 | P1 | Hard-coded copy remains outside Pharma OA checks | Product + QA | Global locale inventory | Zero hard-coded visible copy across all Vue views and plugin host surfaces | PR4-04 |
| UI-010 | P1 | Accessibility scan covers Pharma views and source invariants only | Frontend QA | Browser a11y matrix | Keyboard, focus, name/role/value, error association, contrast, reduced motion pass globally | PR4-04 |
| UI-011 | P1 | Raw colors and gradients escape semantic tokens | Design Systems | Theme tokens | Zero raw product colors outside the token registry and zero decorative operational gradients | PR4-03 |
| UI-012 | P1 | Shared high-impact components have no component tests | Frontend QA | Component harness | Page shell, table, state, toolbar, drawer, confirmation, schema form have interaction tests | PR4-02, PR4-06 |
| UI-013 | P1 | Login card overflows a 390px viewport | Auth Frontend | Responsive auth shell | No horizontal overflow at 320, 390, 768, and 1440 widths | PR4-05 |
| UI-014 | P1 | Purchase inbound and sales lack explicit empty evidence | Pharma Frontend | Domain table state | Both workflows show actionable, localized empty states | PR4-05 |
| UI-015 | P2 | Radius reaches 10-12px and pills are used beyond compact status/segmented controls | Design Systems | Shape tokens | Cards/panels <=8px; pills only for status, avatars, and segmented controls | PR4-03 |

## Frozen Acceptance Budgets

| Dimension | Required threshold |
| --- | --- |
| Control consistency | Zero unjustified native controls and zero page-local replacements for shared interaction patterns |
| Theme | 100% of primary workflows pass light/dark x comfortable/compact screenshots |
| Color | Zero raw product colors outside the semantic token registry |
| Copy | Zero hard-coded visible business copy in Vue/TypeScript product surfaces |
| Accessibility | WCAG 2.2 AA contrast; complete keyboard flow; visible focus; named controls; associated validation errors |
| Responsive | No horizontal page overflow or overlapping controls at 320, 390, 768, 1024, and 1440 CSS pixels |
| State coverage | Every `Required` cell in the matrix has a deterministic fixture and screenshot/interaction assertion |
| Host entry | Initial host JavaScript <=130 KB gzip and initial CSS <=20 KB gzip |
| Route chunks | Normal route chunk <=100 KB gzip; explicitly lazy heavy tool chunk <=120 KB gzip |
| Generated plugin entry | JavaScript <=180 KB gzip and CSS <=60 KB gzip |
| Interaction | Primary action response <=100ms locally; route usable <=2.5s on the agreed throttled profile; no long task >200ms |
| Large lists | 10,000-row supported views scroll without layout shift and keep p95 interaction <=100ms locally |
| Visual regression | Desktop, tablet, and mobile baselines pass with reviewed deterministic masks only |

## PR4 Execution Handoff

| Work Item | Must consume | Exit evidence |
| --- | --- | --- |
| PR4-02 | UI-001, UI-003, UI-007, UI-008, UI-012 | Shared component tests, control scan, mobile navigation smoke, typecheck/build |
| PR4-03 | UI-005, UI-011, UI-015 | Token scan and four-combination visual/contrast matrix |
| PR4-04 | UI-009, UI-010 | Global locale scan and executable browser accessibility workflows |
| PR4-05 | State matrix, UI-013, UI-014 | Viewport/state Playwright matrix with no overlap or layout shift |
| PR4-06 | UI-002, UI-004, UI-006 and all frozen budgets | Visual regression, asset MIME check, bundle/runtime budget gates |

## Audit Evidence

```powershell
codegraph explore "Vue frontend architecture primary admin workflows shared components Element Plus themes responsive states"
codegraph impact PageShell
codegraph impact DataTable
rg --files web/src
rg -n --glob '*.vue' '<(button|input|table)\b' web/src
rg -n --glob '*.vue' --glob '*.scss' '#[0-9A-Fa-f]{3,8}\b|rgba?\(' web/src
npm run typecheck
npm run build
```

The baseline is intentionally current-only. Subsequent tasks change the present contracts directly and must not add compatibility branches for the UI being replaced.
