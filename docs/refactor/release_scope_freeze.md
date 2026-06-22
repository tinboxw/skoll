# Release Scope Freeze

> Status: Active after N0 closeout  
> Date: 2026-06-22  
> Scope: Post-M2 / FE0-FE6 closeout, before M3-M7 execution

This note freezes the release-preparation scope after the N0 quality gate. It defines where future tasks are taken from, what may change before the next milestone begins, and what is explicitly out of scope.

## Frozen Baseline

| Area | Baseline |
| --- | --- |
| Formal task source | `docs/refactor/work_items.md` |
| Candidate/source record | `docs/refactor/next_work_items.md` |
| Parent milestone view | `docs/refactor/task_board.md` |
| Acceptance evidence | `docs/refactor/acceptance_log.md` |
| Latest completed quality gates | `go test ./...`, `cd web && npm run typecheck`, `cd web && npm run build` |
| Work Item snapshot after N0-05 | 244 total, 177 Done, 67 Todo |

`next_work_items.md` remains a source and candidate record. It is not the execution table once its tasks have been merged into `work_items.md`.

## Execution Rule

1. Future implementation must take the first eligible `Todo` from `docs/refactor/work_items.md`.
2. New post-M2 work may only be appended or inserted into `work_items.md` and `task_board.md` when it follows the existing table format, status enum, skill naming, deliverable, acceptance, and verification-command rules.
3. Each Work Item must be implemented, verified, logged in `acceptance_log.md`, marked `Done`, and committed before moving to the next Work Item.
4. If verification fails, the same Work Item stays active until the failure is fixed or explicitly recorded as blocked by an external condition.
5. M3-M7 must be executed in the merged order unless a quality-gate failure requires a narrowly scoped fix item to be inserted ahead of the next feature item.

## Allowed Before M3 Starts

- Fix documentation index drift.
- Correct acceptance-log omissions.
- Record quality-gate reruns.
- Add narrow fix tasks caused by failed gates.
- Update source maps when archived evidence is summarized into current docs.

## Out Of Scope

- Adding compatibility layers for old API paths, old data structures, old plugin manifest formats, or old page routes.
- Adding broad new feature scope outside the merged M3-M7 Work Items.
- Treating archived process documents as active execution sources.
- Marking a Work Item done without its verification command or a recorded, justified limitation.
- Skipping `acceptance_log.md` updates for completed Work Items.

## Quality Gate Carry-Forward

The N0 gates established the current release-preparation baseline:

```powershell
$env:CC='D:\workspace\mingw64\bin\gcc.exe'
go test ./...

cd web
npm run typecheck
npm run build
```

On this Windows machine, full sqlite-backed Go tests need a valid cgo compiler. If the default `CC` points to a missing compiler, rerun with `D:\workspace\mingw64\bin\gcc.exe` and record the retry.

The frontend build currently passes with non-blocking warnings:

- Dart Sass legacy JS API deprecation warning.
- Rollup removal of misplaced `/* #__PURE__ */` annotations from `@vueuse/core`.

These warnings do not block M3, but any new warning introduced by a future Work Item must be recorded in that Work Item's acceptance entry.

## Next Formal Task

After this freeze note is accepted, the next formal Work Item is:

| ID | Skill | Task | Verification |
| --- | --- | --- | --- |
| M3-01-01 | `skoll-file-storage-refactor` | FileObject 模型 | `go test ./internal/domain/file/...` |
