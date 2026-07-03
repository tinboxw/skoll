# Skoll Progress Inspection 2026-07-04

> Scope: project progress inspection and task refinement.
> Rule: Skoll is still in open-source foundation construction. Do not add legacy API, legacy data, legacy plugin, legacy route compatibility, or dual-path transition plans.

## 进度总览

| Item | Result |
| --- | --- |
| Formal execution table | `docs/refactor/work_items.md` |
| Total formal work items | 252 |
| Completed formal work items | 242 |
| Remaining formal work items | 10 |
| Trigger threshold | Original remaining work items were 5, so the `< 10` refinement threshold was triggered |
| Candidate pool | `docs/refactor/next_work_items.md` remains a historical candidate source, not the current execution source |
| Parent task table | `docs/refactor/task_board.md` |
| Completed parent tasks | 61 |
| Remaining parent tasks | 3 |
| Recent progress | M7 quality gates, coverage target, backend/frontend performance baselines, and quick start have been completed |

Completed delivery quality is generally acceptable:

- M0-N0 established architecture, task acceptance, quality baseline, release scope freeze, and the no-compatibility rule.
- M1-M2 completed permission/menu catalog and audit observability with API, frontend, and smoke acceptance.
- M3-M6 completed file storage, dictionary/config/organization/data scope, generator v1, and plugin marketplace/lifecycle capabilities.
- FE0-FE6 completed frontend foundation, design standards, frontend architecture, core-page UX, plugin portal UX, testing gates, and performance baseline.
- M7 has completed CI gates, coverage targets, performance baselines, and quick start.

Current deviation from the original plan:

- The project has moved faster than the original `next_work_items.md` candidate pool. Many candidate rows are now stale because the formal execution table has already absorbed and completed them.
- `task_board.md` had one stale parent state: `M4-05` was still `Todo` while `work_items.md` showed its children completed. It has been corrected to `Done`.
- The remaining M7 tail tasks were too broad for one-task-one-commit execution. They have been split from 5 broad items into 12 atomic items; parallel development has already completed 2 of those atomic items, leaving 10 Todo items.

## 问题与风险

| Risk | Severity | Evidence | Action |
| --- | --- | --- | --- |
| Candidate pool stale | Medium | `next_work_items.md` still contains old Todo rows for N0/M3/M4 tasks that are already represented in `work_items.md` | Do not use it for execution; use only `work_items.md` until final release |
| Parent/task state drift | Medium | `M4-05` parent state lagged behind completed work items | Corrected in `task_board.md`; final release inspection must recheck parent-child consistency |
| Tail tasks too broad | High | Original remaining 5 tasks mixed docs, examples, templates, release, tests, and manual reproduction | Split into 12 atomic tasks with clear verification; current remaining count is 10 |
| Release proof still incomplete | High | Quick start exists, but operations manual, examples, open-source templates, and release checklist are not yet done | Execute M7-04 through M7-06 in order |
| No-compat rule needs final audit | Medium | Code search still contains `legacy` and `compatibility` strings; some are valid negative tests or plugin version constraints, but each must be classified before release | Add final release inspection task `M7-06-04` |
| Acceptance log encoding display | Low | Console output shows mojibake in older `acceptance_log.md` tail content | Do not rewrite history now; final release inspection should validate source encoding and readable rendered docs |

Direction check:

- Architecture direction remains correct: Modular Monolith + Clean/Hexagonal boundaries + Tactical DDD only where domain complexity justifies it.
- The project remains aligned with the target of exceeding gin-vue-admin through plugin lifecycle, generator, audit, data scope, frontend quality, and open-source operations.
- No backward-compatibility scheme should be added in the remaining phase. Version constraints for Skoll/plugin compatibility are allowed only as current contract validation, not legacy bridge behavior.

## 剩余任务微调清单

The tail tasks have been refined in `docs/refactor/work_items.md`. Parallel development has already completed `M7-04-02` and `M7-04-03`, so only rows with `Todo` status should be claimed next.

| Work Item | Status | Priority | Dependency | Deliverable | Acceptance |
| --- | --- | --- | --- | --- | --- |
| M7-04-02 | Done | P0 | M7-04-01 | Operations manual: configuration and deployment | Env vars, Docker/Compose, start/stop paths are documented |
| M7-04-03 | Done | P0 | M7-04-02 | Operations manual: backup, logs, troubleshooting | Backup/restore, log location, common failure recovery steps are executable |
| M7-04-04 | Done | P0 | M7-04-03 | Operations manual reproduction record | A new environment can follow quick start + operations docs to start, log in, stop, and recover |
| M7-05-01 | Done | P0 | M7-04-04, M5 Done | Demo product spec and generated file list | Spec, generated files, permission keys, and menu declarations are complete |
| M7-05-02 | Done | P0 | M7-05-01 | Demo product backend acceptance | Backend API, permission, audit, store/service flows pass `go test ./...` |
| M7-05-03 | Todo | P0 | M7-05-02 | Demo product frontend acceptance | List/form/empty/error/permission states pass typecheck and build |
| M7-05-04 | Todo | P0 | M7-05-03, M6 Done | Demo plugin manifest and assets | Manifest, permission, menu, config schema, signature/risk fields are complete |
| M7-05-05 | Todo | P0 | M7-05-04 | Demo plugin lifecycle acceptance | Install/enable/disable/upgrade/rollback paths pass tests and build |
| M7-06-01 | Todo | P1 | M7-05-05 | Open-source contribution guide | Contribution, security, conduct, and maintainer notes are complete |
| M7-06-02 | Todo | P1 | M7-06-01 | Issue and PR templates | Bug, feature, task, and PR templates include reproduction, acceptance, and no-compat confirmation |
| M7-06-03 | Todo | P0 | M7-06-02 | Release checklist | Version, migration, image, OpenAPI, examples, and quality gates are checkable |
| M7-06-04 | Todo | P0 | M7-06-03 | Final release readiness inspection | `work_items.md`, `task_board.md`, `acceptance_log.md`, `git log`, and quality gates are consistent |

Rejected or deferred adjustments:

- Do not add compatibility tasks.
- Do not reopen completed M3-M6 scope unless the final inspection finds a concrete failing acceptance item.
- Do not execute from `next_work_items.md`; it is now a source/history document only.

## 后续里程碑细化表

| Milestone | Goal | Work Items | Priority | Dependencies | Expected Deliverables | Milestone Acceptance |
| --- | --- | --- | --- | --- | --- | --- |
| M7-A: 运维可复现 | Make the project operable by a new user and maintainer | M7-04-02, M7-04-03, M7-04-04 | P0 | M7-04-01 | Operations manual and reproduction record | A clean environment can start, log in, stop, inspect logs, and follow backup/restore guidance |
| M7-B: 示例证明框架能力 | Prove generator and plugin platform with runnable examples | M7-05-01, M7-05-02, M7-05-03, M7-05-04, M7-05-05 | P0 | M7-A, M5 Done, M6 Done | Demo product, demo plugin, tests, frontend build evidence | `go test ./...`, frontend typecheck/build, and plugin lifecycle checks pass |
| M7-C: 开源发布收口 | Make the repository ready for external contributors and release | M7-06-01, M7-06-02, M7-06-03, M7-06-04 | P0/P1 | M7-B | Contribution templates, release checklist, final inspection report | All formal work items are Done, parent tasks are Done, acceptance log and git log are consistent, no compatibility drift remains |

Execution order:

1. Finish M7-A before example work, because examples need stable run and operations guidance.
2. Finish M7-B before release checklist, because release checklist must reference real examples.
3. Finish M7-C last, with `M7-06-04` as the final readiness gate.
