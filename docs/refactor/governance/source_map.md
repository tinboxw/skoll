# Refactor Document Source Map

> Purpose: record which documents are active, which are reusable references, and which are archived evidence.

## Current Sources

| Topic | Current Document |
| --- | --- |
| Refactor entry | `docs/refactor/README.md` |
| Active feature and UI planning | `docs/refactor/current/feature_ui_milestone_plan_2026-07-04.md` |
| Active feature/UI work candidates | `docs/refactor/current/feature_ui_work_items_2026-07-04.md` |
| Business-plugin capability plan | `docs/refactor/current/business_plugin_capability_plan_2026-07-04.md` |
| Pharma OA milestone plan | `docs/refactor/current/pharma_oa_milestone_plan_2026-07-04.md` |
| Latest progress inspection | `docs/refactor/current/progress_inspection_pharma_oa_iteration_2026-07-04.md` |
| Architecture and governance | `docs/refactor/governance/architecture_and_execution_plan.md` |
| Generator output reference | `docs/refactor/reference/generator_template_output_spec.md` |
| Data-scope acceptance reference | `docs/refactor/reference/m4_data_scope_acceptance_checklist.md` |

## Archived Completed Batch

Completed M0-M7/FE/N0 evidence now lives in:

`docs/refactor/old/completed-m0-m7-2026-07-04/`

| Archived Content | Notes |
| --- | --- |
| `work_items.md` | 252/252 Done. Historical execution ledger. |
| `task_board.md` | 65/65 Done. Historical parent task board. |
| `acceptance_log.md` | Historical acceptance evidence. |
| `final_release_readiness_2026-07-04.md` | Final readiness inspection for completed foundation batch. |
| `progress_inspection_2026-07-04.md` | Earlier inspection, superseded by current pharma OA inspection. |
| `next_work_items.md` | Previous candidate pool, superseded by current F6-F12 planning. |
| quality and performance baselines | Historical quality evidence for completed work. |
| frontend foundation/quality docs | Historical FE0-FE6 evidence and guidelines. |
| release freeze and closeout docs | Historical closeout evidence. |

## Archive Rules

1. Archive completed batches only after active content has been summarized into `current/` or `governance/`.
2. Archived files are evidence, not execution entry points.
3. Do not rewrite archived task status for new work.
4. If archived content becomes relevant again, extract the valid part into a current or reference document instead of editing the archived original.
5. When moving documents, update `docs/refactor/README.md`, this source map, and top-level `docs/README.md` if it links to the moved paths.

## Not Adopted

The project continues to reject:

- Legacy API compatibility layers.
- Legacy data-structure compatibility.
- Legacy plugin-format compatibility.
- Legacy page-route compatibility.
- Static UI demos as acceptance evidence without real API, permission, audit, state, and build validation.
