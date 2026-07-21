# Current Refactor Execution Rules

> Scope: active plugin-runtime and frontend-experience implementation after the closed Pharma OA hardening batch.
> Status: M0-M7/FE/N0 are complete and archived. Do not update archived task status for new work.
> Pharma OA batch: F6-F12 closed on 2026-07-17. Its files remain closed evidence, not task intake.
> Latest closed batch: `skoll-hardening-2026-07-18` (closed 2026-07-21).
> Active batch: `skoll-plugin-runtime-2026-07-21`.

## Single Source Of Truth

| Need | Read Or Update |
| --- | --- |
| Current task intake and progress rules | This file |
| Active parent task board | `plugin_runtime_task_board_2026-07-21.md` |
| Active Work Item table | `plugin_runtime_work_items_2026-07-21.md` |
| Active acceptance log | `plugin_runtime_acceptance_log_2026-07-21.md` |
| Latest closed parent task board | `hardening_task_board_2026-07-18.md` |
| Latest closed Work Item table | `hardening_work_items_2026-07-18.md` |
| Latest closed acceptance log | `hardening_acceptance_log_2026-07-18.md` |
| Latest hardening closeout | `hardening_closeout_2026-07-21.md` |
| Active performance baseline | `performance_capacity_baseline_2026-07-21.md` |
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
3. Read `plugin_runtime_task_board_2026-07-21.md` for milestone boundaries.
4. Take only a dependency-ready `Todo` from `plugin_runtime_work_items_2026-07-21.md`.
5. Do not take tasks from `../old/`; archived files are historical evidence only.

## Active Task Files

The active batch uses these files under `docs/refactor/current/`:

```text
plugin_runtime_task_board_2026-07-21.md
plugin_runtime_work_items_2026-07-21.md
plugin_runtime_acceptance_log_2026-07-21.md
```

The hardening and Pharma OA files are closed evidence. Do not reopen them for new progress.

## Progress Update Rules

Developers update progress only in the official next-batch files:

| Action | Required Update |
| --- | --- |
| Take a task | Set the Work Item status to `Doing` in `plugin_runtime_work_items_2026-07-21.md` |
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
