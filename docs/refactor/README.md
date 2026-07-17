# Skoll Refactor Docs

> Status: M0-M7/FE/N0 foundation refactor is complete. The active direction is feature development, UI polish, and the pharma OA business-plugin roadmap.
> Rule: do not design legacy API, legacy data, legacy plugin, or legacy page compatibility plans.

## How To Read This Directory

Use this directory in order:

1. `current/`: active plans and candidate tasks for the next development batch.
2. `governance/`: architecture rules, execution principles, and document source mapping.
3. `reference/`: reusable technical references that still help new implementation.
4. `old/`: completed M0-M7/FE/N0 evidence. Read only for history or audit.

Completed documents are intentionally archived under `old/`. They are not the current execution entry.

## Current Execution

| Document | Purpose |
| --- | --- |
| [current/README.md](current/README.md) | Current task intake, progress update, acceptance, and commit rules |
| [current/hardening_task_board_2026-07-18.md](current/hardening_task_board_2026-07-18.md) | Active hardening parent task board |
| [current/hardening_work_items_2026-07-18.md](current/hardening_work_items_2026-07-18.md) | Active hardening Work Item table |
| [current/hardening_acceptance_log_2026-07-18.md](current/hardening_acceptance_log_2026-07-18.md) | Active hardening acceptance evidence |
| [current/pharma_oa_milestone_closeout_2026-07-17.md](current/pharma_oa_milestone_closeout_2026-07-17.md) | Closed Pharma OA batch evidence and hardening source |
| [current/business_plugin_capability_plan_2026-07-04.md](current/business_plugin_capability_plan_2026-07-04.md) | Business-plugin capability gaps, pharma OA function blueprint, and framework infrastructure matrix |
| [current/pharma_oa_milestone_plan_2026-07-04.md](current/pharma_oa_milestone_plan_2026-07-04.md) | F6-F12 pharma OA milestones and atomic candidate work items |
| [current/progress_inspection_pharma_oa_iteration_2026-07-04.md](current/progress_inspection_pharma_oa_iteration_2026-07-04.md) | Latest progress inspection, risks, task tuning, and milestone refinement |
| [current/feature_ui_milestone_plan_2026-07-04.md](current/feature_ui_milestone_plan_2026-07-04.md) | Feature development and UI optimization milestone plan |
| [current/feature_ui_work_items_2026-07-04.md](current/feature_ui_work_items_2026-07-04.md) | Candidate work items for feature and UI improvements |

Recommended next execution order:

1. F6 business-plugin platform foundation.
2. F7 workflow, dynamic forms, todo, and notification platform.
3. F8 pharma master-data plugin.
4. F9 pharma purchase/sales/inventory minimum closed loop.

## Governance

| Document | Purpose |
| --- | --- |
| [governance/architecture_and_execution_plan.md](governance/architecture_and_execution_plan.md) | Architecture model, no-compatibility rule, milestone governance |
| [governance/source_map.md](governance/source_map.md) | Document absorption, archive mapping, and maintenance rules |

Architecture position:

- Modular Monolith + Clean/Hexagonal boundaries + Tactical DDD for complex domains.
- Simple CRUD remains generator-friendly.
- Every Work Item must have deliverables, acceptance criteria, verification commands, and one commit after passing acceptance.

## Reference

| Document | Purpose |
| --- | --- |
| [reference/generator_template_output_spec.md](reference/generator_template_output_spec.md) | Generator template output rules |
| [reference/m4_data_scope_acceptance_checklist.md](reference/m4_data_scope_acceptance_checklist.md) | Data-scope acceptance reference for future business modules |

## Archive

| Directory | Purpose |
| --- | --- |
| [old/completed-m0-m7-2026-07-04](old/completed-m0-m7-2026-07-04) | Completed M0-M7/FE/N0 tasks, acceptance logs, quality baselines, and release-readiness evidence |

Archived files keep historical evidence. Do not update archived task status for new work. Create a new current task board/work-item file when the next batch starts.

## Collaboration Rule

All developers must read [current/README.md](current/README.md) before taking work. It defines where tasks are read, where progress is updated, where acceptance is recorded, and when commits are created.

## Maintenance Rules

1. Keep the top level of `docs/refactor` small: `README.md`, `current/`, `governance/`, `reference/`, `old/`.
2. New active plans go to `current/`.
3. Completed batches go to `old/<batch-name>/`.
4. Reusable implementation references go to `reference/`.
5. Governance and source mapping go to `governance/`.
6. Update this README whenever a document is added, moved, or archived.
7. Prefer short, action-oriented docs with exact paths, commands, deliverables, and acceptance criteria.
