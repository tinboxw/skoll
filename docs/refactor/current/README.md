# Current Refactor Execution Rules

> Scope: next-batch feature development, UI optimization, business-plugin platform work, and pharma OA planning.
> Status: M0-M7/FE/N0 are complete and archived. Do not update archived task status for new work.

## Single Source Of Truth

| Need | Read Or Update |
| --- | --- |
| Current task intake and progress rules | This file |
| Official parent task board | `pharma_oa_task_board.md` |
| Official work-item table | `pharma_oa_work_items.md` |
| Official acceptance log | `pharma_oa_acceptance_log.md` |
| Candidate task source | `pharma_oa_milestone_plan_2026-07-04.md` |
| Business-plugin and pharma OA context | `business_plugin_capability_plan_2026-07-04.md` |
| Feature/UI candidate pool | `feature_ui_work_items_2026-07-04.md` |
| Latest progress inspection | `progress_inspection_pharma_oa_iteration_2026-07-04.md` |
| Completed historical evidence | `../old/completed-m0-m7-2026-07-04/` |

## How Developers Read Tasks

1. Start from `docs/refactor/README.md`.
2. Read this file before taking any task.
3. Read `pharma_oa_milestone_plan_2026-07-04.md` for F6-F12 candidate tasks.
4. When an official next-batch task table exists, take work only from that table.
5. Do not take tasks from `../old/`; archived files are historical evidence only.

## Official Next-Batch Task Files

The current batch uses these files under `docs/refactor/current/`:

```text
pharma_oa_task_board.md
pharma_oa_work_items.md
pharma_oa_acceptance_log.md
```

Because these files exist, take implementation work only from `pharma_oa_work_items.md`. `pharma_oa_milestone_plan_2026-07-04.md` is now a candidate pool and planning reference, not a progress ledger.

## Progress Update Rules

Developers update progress only in the official next-batch files:

| Action | Required Update |
| --- | --- |
| Take a task | Set the Work Item status to `Doing` in `pharma_oa_work_items.md` |
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
