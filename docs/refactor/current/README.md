# Current Refactor Execution Rules

> Scope: active business-plugin data, reusable OA infrastructure, plugin UX, and medical OA implementation after the closed plugin-runtime batch.
> Status: M0-M7/FE/N0 are complete and archived. Do not update archived task status for new work.
> Pharma OA batch: F6-F12 closed on 2026-07-17. Its files remain closed evidence, not task intake.
> Latest closed batch: `skoll-plugin-runtime-2026-07-21` (closed 2026-07-22).
> Active batch: `skoll-business-plugin-foundation-2026-07-22`.

## Single Source Of Truth

| Need | Read Or Update |
| --- | --- |
| Current task intake and progress rules | This file |
| Active parent task board | `business_plugin_task_board_2026-07-22.md` |
| Active Work Item table | `business_plugin_work_items_2026-07-22.md` |
| Active acceptance log | `business_plugin_acceptance_log_2026-07-22.md` |
| Latest closed parent task board | `plugin_runtime_task_board_2026-07-21.md` |
| Latest closed Work Item table | `plugin_runtime_work_items_2026-07-21.md` |
| Latest closed acceptance log | `plugin_runtime_acceptance_log_2026-07-21.md` |
| Latest plugin-runtime closeout | `plugin_runtime_closeout_2026-07-22.md` |
| Latest closed parent task board | `hardening_task_board_2026-07-18.md` |
| Latest closed Work Item table | `hardening_work_items_2026-07-18.md` |
| Latest closed acceptance log | `hardening_acceptance_log_2026-07-18.md` |
| Latest hardening closeout | `hardening_closeout_2026-07-21.md` |
| Active performance baseline | `performance_capacity_baseline_2026-07-21.md` |
| Frozen frontend experience target | `frontend_experience_target_2026-07-22.md` |
| Active deployment and recovery rehearsal | `deployment_recovery_rehearsal_2026-07-21.md` |
| Active H2-05 database acceptance report | `pharma_oa_database_acceptance_2026-07-18.md` |
| Active batch source | `pharma_oa_milestone_closeout_2026-07-17.md` |
| Closed Pharma OA parent board | `pharma_oa_task_board.md` |
| Closed Pharma OA Work Items | `pharma_oa_work_items.md` |
| Closed Pharma OA acceptance log | `pharma_oa_acceptance_log.md` |
| Candidate task source | `pharma_oa_milestone_plan_2026-07-04.md` |
| Business-plugin and pharma OA context | `business_plugin_capability_plan_2026-07-04.md` |
| Feature/UI candidate pool | `feature_ui_work_items_2026-07-04.md` |
| Latest progress inspection | `progress_inspection_pharma_oa_iteration_2026-07-04.md` |
| Pharma OA performance and permission report | `pharma_oa_performance_permission_report.md` |
| Pharma OA milestone closeout | `pharma_oa_milestone_closeout_2026-07-17.md` |
| Completed historical evidence | `../old/completed-m0-m7-2026-07-04/` |

## How Developers Read Tasks

1. Start from `docs/refactor/README.md`.
2. Read this file before taking any task.
3. Read `business_plugin_task_board_2026-07-22.md` for active milestone boundaries.
4. Take only the first dependency-ready `Todo` from `business_plugin_work_items_2026-07-22.md`.
5. Do not take tasks from `../old/`; archived files are historical evidence only.

## Active Task Files

The active batch uses these files under `docs/refactor/current/`:

```text
business_plugin_task_board_2026-07-22.md
business_plugin_work_items_2026-07-22.md
business_plugin_acceptance_log_2026-07-22.md
```

The latest closed plugin-runtime batch remains available as evidence:

```text
plugin_runtime_task_board_2026-07-21.md
plugin_runtime_work_items_2026-07-21.md
plugin_runtime_acceptance_log_2026-07-21.md
plugin_runtime_closeout_2026-07-22.md
frontend_experience_target_2026-07-22.md
```

The plugin-runtime, hardening, and earlier Pharma OA files are closed evidence. Do not reopen them for new progress.

## Progress Update Rules

Developers update progress only in the official active business-plugin batch:

| Action | Required Update |
| --- | --- |
| Take a task | Set the Work Item status to `Doing` in `business_plugin_work_items_2026-07-22.md` |
| Need review | Set status to `Review` and add verification results |
| Acceptance passes | Set status to `Done`, append acceptance evidence, then commit once |
| Acceptance fails | Set status to `Failed`, record reason and rerun the same Work Item |
| Blocked | Set status to `Blocked`, record blocker owner, condition, and next check time |

Status flow:

```text
Todo -> Doing -> Review -> Done
              -> Failed -> Doing
              -> Blocked
```

## Work Item Requirements

Every Work Item must include:

- Work Item ID.
- Skill.
- Task description.
- Dependencies.
- Deliverables.
- Acceptance criteria.
- Verification commands.
- Status.

Do not mark a task `Done` unless acceptance passed and a commit was created.

## Commit Rule

Each passed Work Item gets one commit:

```text
<work-item-id>: <short summary>
```

If a task touches API, permission, audit, migration, frontend client, or docs, those impacts must be updated in the same Work Item before acceptance.

## No-Compatibility Rule

Do not design or implement:

- legacy API compatibility layers
- legacy data compatibility
- legacy plugin-format compatibility
- legacy page-route compatibility
- dual-path transitional behavior

Skoll remains in clean open-source foundation construction.
