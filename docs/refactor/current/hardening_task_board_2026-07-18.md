# Skoll Hardening Task Board

> Batch: `skoll-hardening-2026-07-18`
> Status source: `hardening_work_items_2026-07-18.md`
> Source: `pharma_oa_milestone_closeout_2026-07-17.md`
> Rule: execute only from the official Work Item table; do not update `../old/` with new progress.

## Parent Tasks

| ID | Skill | Goal | Dependencies | Deliverables | Acceptance | Status |
| --- | --- | --- | --- | --- | --- | --- |
| H1 | `skoll-plugin-platform` / `skoll-permission-rbac` / `skoll-security-hardening` | Close plugin route authentication and permission registration | Closed F6-F12 batch | Route permission descriptor, resolver, middleware enforcement, lifecycle/audit/OpenAPI checks | Every declared plugin business route authenticates and enforces its declared permission; denial is audited; public page/assets remain bounded | Done |
| H2 | `skoll-database-development` / `skoll-upgrade-migration` | Persist Pharma OA business data and inventory transactions | H1, closed F8-F12 | SQL repositories, migrations, indexes, transaction boundaries, seed/restart checks | MySQL, PostgreSQL, and SQLite contracts pass; restart preserves data; inventory ledger and balances remain atomic | Done |
| H3 | `skoll-permission-rbac` / `skoll-security-hardening` | Add trusted organization data scope | H1, H2 | JWT organization claims, scope policy, frontend scope UX, permission matrix | Self, organization, organization tree, and all-data scopes cannot be broadened by request parameters | Done |
| H4 | `skoll-i18n-accessibility` / `skoll-vue-frontend` | Complete Chinese-default i18n and accessibility | H1-H3 | Locale inventory, Chinese/English resources, accessible interactions, browser matrix | Pharma OA main workflows default to Chinese, switch to English, and pass keyboard, focus, ARIA, state, and responsive checks | Done |
| H5 | `skoll-performance-scaling` / `skoll-frontend-performance-refactor` | Establish production-oriented capacity and stability baselines | H2-H4 | Database benchmarks, server pagination, frontend large-list UX, long-run jobs, performance report | P95/P99, slow-query, resource, bundle, and long-running stability budgets are measured and enforced | Doing |
| H6 | `skoll-devops-deployment` / `skoll-quality-gate` / `skoll-open-source-framework` | Prepare reproducible release and operations evidence | H1-H5 | Deployment, backup/restore, upgrade rehearsal, regulatory boundary, release checklist | Clean deploy, restore, upgrade, rollback, docs, licensing, and final quality gates pass | Todo |

## Execution Rules

1. Take the first dependency-ready `Todo` from `hardening_work_items_2026-07-18.md`.
2. Set it to `Doing` before implementation.
3. Set it to `Review` only after its verification commands pass.
4. On failure use `Failed -> Doing`, record the retry, and do not advance.
5. Set it to `Done`, append acceptance evidence, and commit exactly once using `<work-item-id>: <short summary>`.
6. Runtime changes must synchronize API/OpenAPI, permission, audit, migration/seed, frontend client, UI states, and Chinese-default i18n impacts as applicable.
