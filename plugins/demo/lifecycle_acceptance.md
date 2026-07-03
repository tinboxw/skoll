# Demo Plugin Lifecycle Acceptance

> Work Item: M7-05-05
> Scope: demo plugin install, enable, disable, upgrade, rollback, audit, and build gate coverage.

## Automated Coverage

| Area | Evidence |
|---|---|
| Manifest parsing | `TestDemoPluginManifestCoversPlatformContract` loads `plugins/demo/plugin.yaml` and validates menu, config schema, permissions, risk, and signature asset coverage. |
| Install/enable/disable | `internal/plugin/lifecycle_acceptance_test.go` covers plugin lifecycle transitions and catalog side effects. |
| Upgrade/rollback | Plugin lifecycle and rollback tests cover release ordering, rollback plans, permission/menu/config/asset restoration, and task logs. |
| Audit/query path | Full `go test ./...` includes plugin handler/service and audit-related test packages. |
| Frontend build | `cd web; npm run build` passes with the existing warning baseline. |

## Commands

```powershell
go test ./...
cd web
npm run build
```

Result on 2026-07-04: Passed.

## Manual Review Checklist

| Check | Result |
|---|---|
| Demo manifest includes structured permissions and risk levels. | Passed |
| Demo manifest includes `ui_menu` and `config_schema`. | Passed |
| Demo plugin signature asset coverage includes manifest, backend entry, and frontend dist assets. | Passed |
| Lifecycle tests cover install, enable, disable, upgrade, rollback, and catalog restoration behavior. | Passed |
| M7-05 parent task can close after M7-05-01 through M7-05-05 are Done. | Passed |

## Warning Baseline

Frontend build still reports the known Sass legacy JS API warning and Rollup PURE annotation warning from dependencies. No new build failure was introduced by the demo plugin lifecycle acceptance.
