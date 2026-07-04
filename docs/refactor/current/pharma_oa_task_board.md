# Pharma OA Task Board

> Batch: pharma-oa-2026-07-04
> Status source: `pharma_oa_work_items.md`
> Rule: execute only from the official work-item table. Do not update `../old/` for new progress.

## Parent Tasks

| ID | Skill | Goal | Dependencies | Deliverables | Acceptance | Status |
| --- | --- | --- | --- | --- | --- | --- |
| F6 | `skoll-plugin-platform` / `skoll-vue-frontend` | Business-plugin platform foundation | Completed M0-M7/FE/N0 | Theme Engine, Plugin UI Kit, Host SDK, data contract, API/OpenAPI contract, event bus, business plugin generator | Sample business plugin can declare data, APIs, permissions, menus, UI, and host capability calls; tests/build pass | Done |
| F7 | `skoll-workflow-jobs` / `skoll-data-dictionary-config` | Workflow, dynamic forms, todo, and notification platform | F6 | Workflow domain/API/UI, form builder schema/UI, todo/notification center, audit fixtures | Leave, purchase request, and contract approval flows run end to end with audit timeline | Done |
| F8 | `skoll-plugin-platform` / `skoll-go-backend` / `skoll-vue-frontend` | Pharma master-data plugin | F6, F7 | `pharma_oa` plugin skeleton, employee, product, supplier, customer, warehouse, import/export | Master data CRUD, permissions, audit, attachment, import/export, and plugin lifecycle checks pass | Doing |
| F9 | `skoll-database-development` / `skoll-workflow-jobs` | Pharma purchase/sales/inventory minimum closed loop | F7, F8 | Inventory model, purchase request/order/inbound, sales outbound, stocktake/transfer, alerts, smoke | Purchase inbound to sales outbound closed loop passes with accurate stock ledger, batch, expiry, and qualification blocking | Todo |
| F10 | `skoll-workflow-jobs` / `skoll-file-storage` | Pharma OA collaboration and compliance | F7, F8, F9 | Announcements, contracts, qualification management, quality complaints, recalls, cold-chain records, compliance dashboard | Qualification expiry, complaints, recalls, and compliance risks are remindable, approvable, traceable, and auditable | Todo |
| F11 | `skoll-api-contracts` / `skoll-vue-frontend` | Pharma CRM and business dashboard | F8, F9, F10 | Customer follow-up, sales opportunities, payments/invoices, metric API, dashboard widgets, report export | Customer follow-up, sales funnel, stock alerts, qualification expiry, and approval efficiency are visible with correct permissions | Todo |
| F12 | `skoll-testing-automation` / `skoll-quality-gate` | Pharma OA sample acceptance | F6-F11 | Demo data, E2E smoke, industry plugin README, performance/permission validation, closeout report | New employee onboarding, purchase request, inbound, sales outbound, qualification alert, and customer follow-up pass end to end | Todo |

## Execution Rules

1. Take the first `Todo` from `pharma_oa_work_items.md`.
2. Set it to `Doing` before implementation.
3. Set it to `Review` when verification is ready.
4. Mark it `Done` only after all acceptance items pass and evidence is added to `pharma_oa_acceptance_log.md`.
5. Commit exactly once per accepted Work Item with `<work-item-id>: <short summary>`.
