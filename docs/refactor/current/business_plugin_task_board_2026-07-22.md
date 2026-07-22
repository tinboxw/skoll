# Skoll Business Plugin Foundation Task Board

> Batch: `skoll-business-plugin-foundation-2026-07-22`
> Goal: make independent plugins efficient enough to deliver a complete medical-industry OA system without core-code changes.
> Status source: `business_plugin_work_items_2026-07-22.md`
> Scope: functionality, interaction quality, visual quality, and performance. Release, deployment, and documentation-only milestones are excluded.
> Rule: current contracts only. Do not add compatibility modes, legacy formats, dual paths, or fallback implementations.

## Milestones

| ID | Goal | Exit Criteria | Dependencies | Status |
| --- | --- | --- | --- | --- |
| BF0 | Establish the official business-plugin batch | Board, atomic Work Items, acceptance log, indexes, skills, and execution rules resolve | Closed plugin-runtime batch | Done |
| BF1 | Provide a safe relational datastore to independent plugins | Public scoped data contracts, host SQL adapter, external client, lifecycle policy, and real E2E pass | BF0 | Done |
| BF2 | Provide reusable business-document and approval primitives | Numbering, forms, documents, workflow, attachments, comments, timeline, search, and export are reusable by plugins | BF1 | Done |
| BF3 | Deliver a complete plugin control-center experience | Runtime, capabilities, data, migrations, jobs, audit, errors, and lifecycle actions are understandable and responsive | BF1 | Done |
| BF4 | Deliver medical OA master-data plugins | Employees, organization extensions, customers, suppliers, products, manufacturers, qualifications, and validity alerts pass | BF1, BF2 | Todo |
| BF5 | Deliver medical OA transaction and quality workflows | Approval, CRM, purchasing, sales, inventory, batches, quality, contracts, finance coordination, and dashboards pass E2E | BF2, BF3, BF4 | Todo |

## Milestone Acceptance

### BF1 Plugin Datastore

- Plugins use one public structured datastore contract, never `internal/` packages or direct host database handles.
- The host owns plugin namespace, typed parameters, tenant/organization/owner scope, query limits, transaction binding, audit attribution, and lifecycle cleanup.
- SQLite, PostgreSQL, and MySQL behavior is covered at the contract boundary; unsupported expressions fail closed.
- External-process restart, migration failure, transaction rollback, disable, and uninstall pass automated tests.

### BF2 Business Documents And Approval

- Plugins can declare document types, fields, numbering, state transitions, workflow bindings, attachments, comments, and timeline events.
- Mutation is idempotent and transactional; validation and permission failures publish no partial document state.
- Element Plus components render list, form, detail, approval, timeline, print, and export workflows with complete states.

### BF3 Plugin Control Center

- Operators can understand installed state, process health, capabilities, routes, data policy, migrations, jobs, audit, and current failures from one workspace.
- Destructive actions require explicit confirmation and show durable results; disabled plugins expose no active capability.
- Light/dark themes, compact density, Chinese/English, keyboard use, mobile/desktop layouts, visual regression, and performance budgets pass.

### BF4 Medical OA Master Data

- The medical OA plugin owns employees, customer/supplier relationships, product and manufacturer catalogs, qualifications, and validity alerts through public contracts.
- Every list and mutation enforces trusted tenant and organization scope, permission, audit, and attachment ownership.
- Data entry and review workflows are usable on desktop and mobile with no plugin-specific host source changes.

### BF5 Medical OA Transactions And Quality

- OA approvals, CRM, purchasing, inbound/outbound, sales, returns, transfers, inventory counts, batches, expiry, quality inspection, quarantine, recall, contracts, and finance coordination form traceable workflows.
- Stock and document transitions are atomic, idempotent, auditable, and blocked when qualification, batch, expiry, or quality rules fail.
- End-to-end scenarios cover normal, rejected, duplicate, concurrent, no-permission, cross-scope, restart, and mobile workflows.

## Execution Rules

1. Take the first dependency-ready `Todo` Work Item and set it to `Doing` before implementation.
2. One developer owns one `Doing` item; parallel work must use different files or an explicit ownership note.
3. API, permission, audit, migration, frontend client, i18n, and documentation impacts stay in the same Work Item.
4. Failed acceptance is recorded as `Failed`, then the same item returns to `Doing`; it is never skipped.
5. After independent acceptance, set the item to `Done`, append evidence, and create exactly one commit named `<work-item-id>: <short summary>`.
6. Do not reopen closed plugin-runtime tasks; new discoveries become atomic BF Work Items with explicit dependencies.
7. Do not implement compatibility, legacy, dual-route, transition, or fallback behavior.
