# Skoll Plugin Runtime Task Board

> Batch: `skoll-plugin-runtime-2026-07-21`
> Goal: make Skoll a business-plugin platform that can deliver Pharma OA and unrelated industries without changing core code.
> Status source: `plugin_runtime_work_items_2026-07-21.md`
> Rule: current contracts only. Do not add legacy compatibility, dual paths, or transition adapters.

## Milestones

| ID | Goal | Exit Criteria | Dependencies | Status |
| --- | --- | --- | --- | --- |
| PR0 | Establish the official execution batch | Board, Work Items, acceptance log, indexes, and collaboration rules are complete | Closed hardening batch | Done |
| PR1 | Execute independent plugins through a real runtime | External plugin routes execute real backends; lifecycle, health, migrations, events, and Pharma OA separation pass | PR0 | Done |
| PR2 | Provide persistent platform services to plugins | Workflow, notifications, jobs, transactions, files, audit, config, and data scope survive restart and expose tested SDK contracts | PR1 | Done |
| PR3 | Generate installable full-stack plugins | Generator output installs and runs without hand editing or core-code changes | PR2 | Done |
| PR4 | Deliver a unified premium frontend experience | Element Plus, themes, i18n, accessibility, responsive states, visual regression, and performance budgets pass | PR1, PR2 | Done |
| PR5 | Prove framework generality with a second industry | An equipment-maintenance plugin is built only on public contracts and completes lifecycle E2E with zero core-code diff | PR3, PR4 | Done |

## Milestone Acceptance

### PR1 Real Plugin Runtime

- Enabled plugin API requests reach the declared backend and return its real status/body.
- Disabled, unhealthy, undeclared, or unreachable plugins fail closed with stable errors.
- Install, migration, enable, health, event delivery, disable, and uninstall are covered by automated tests.
- Pharma OA uses the same public runtime contract as every other plugin; no unconditional core wiring remains.
- Current plugin manifest and runtime format are the only supported format.

### PR2 Persistent Platform Services

- Workflow, notifications, scheduled jobs, retries, and plugin state survive process restart.
- Plugins consume host transactions, data scope, files, audit, config, and events through public ports.
- Contract tests prove tenant/organization isolation, idempotency, atomicity, and failure behavior.

### PR3 Full-Stack Plugin Generator

- One command generates backend, frontend, manifest, migration, permissions, audit, tests, and packaging.
- Generated output passes formatting, tests, typecheck, build, install, enable, route execution, disable, and uninstall.
- No generated file requires manual edits before the first successful run.

### PR4 Frontend Experience System

- Product UI consistently uses Element Plus and the shared Skoll UI kit.
- Light, dark, and compact modes cover every main workflow without raw color leaks.
- Chinese and English, keyboard access, responsive layouts, complete states, visual regression, and performance budgets pass.

### PR5 Independent Industry Proof

- The equipment-maintenance plugin covers assets, work orders, inspections, spare parts, and dashboards.
- It uses only published plugin/runtime/SDK contracts.
- Installation and removal require no core backend or host frontend source changes.

## Execution Rules

1. Take the first dependency-ready `Todo` Work Item and set it to `Doing` before editing.
2. A developer owns only one `Doing` Work Item at a time unless the board explicitly records parallel ownership.
3. Run the listed verification and set the item to `Review` only after all commands pass.
4. Failed acceptance must be recorded as `Failed`, then the same Work Item returns to `Doing`; do not skip ahead.
5. After independent acceptance, set the item to `Done`, append evidence, and create exactly one commit named `<work-item-id>: <short summary>`.
6. API, permission, audit, migration, frontend client, i18n, and documentation impacts belong to the same Work Item.
7. Never implement compatibility layers, legacy formats, dual routes, or temporary bridges.
