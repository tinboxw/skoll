# Skoll 强化批次结项报告

> Batch: `skoll-hardening-2026-07-18`
> Date: 2026-07-21
> Result: 25/25 Work Items accepted; all P0 hardening gates passed.
> Scope: repository hardening batch closeout, not approval to publish a tagged production release.

## 结论

H1-H6 已逐项完成独立验收。插件权限、医药 OA 持久化与组织数据范围、中英文与无障碍、容量和长任务基线、部署恢复、发布边界及最终运行时审查均有可重复证据。未引入旧接口、旧数据结构、旧插件格式或旧页面路径兼容方案，`docs/refactor/old/` 未更新。

最终门禁未发现 P0 阻断问题。当前仓库可作为下一开发批次的稳定基线；实际发布镜像、版本标签、SBOM、签名和目标集群上线仍须由具体发布流程完成。

## 验收环境

| Item | Value |
| --- | --- |
| OS | Windows 11 10.0.26200, amd64 |
| Go | 1.24.1 |
| Node.js / npm | 22.14.0 / 11.1.0 |
| Database | Local MySQL 5.7 service; MySQL 8.0.31 client |
| Frontend | Vite 5.4.21, 3,612 modules in production build |
| Runtime | Isolated in-memory Skoll process on a random loopback port |
| Baseline commit | `b728952` before H6-03 closeout commit |

## 最终门禁

| Gate | Result | Evidence |
| --- | --- | --- |
| Work Item consistency | Pass | 25 rows recognized; H1-H5 and H6-01/H6-02 complete; every completed item has acceptance evidence. |
| Go quality | Pass | Tracked Go files are formatted; `go test ./... -count=1` and `go vet ./...` pass. |
| API and plugin contracts | Pass | OpenAPI copies stay synchronized; plugin manifests validate; lifecycle smoke passes in Chinese and English. |
| Frontend and i18n | Pass | 1,365 locale keys, 1,256 references, accessibility and large-list checks, typecheck, and production build pass; Chinese remains the default locale. |
| Pharma OA flow | Pass | Chinese and English end-to-end smoke, critical permission mapping, audit behavior, pagination, and benchmark checks pass. |
| Database | Pass | SQLite and live MySQL migration, seed, restart, backup/restore, master, order, and workflow repository contracts pass against `skoll_acceptance`. |
| Jobs and performance | Pass | Both soak reports, performance report audit, bundle and allocation baseline evidence pass. |
| Deployment recovery | Pass | Clean deployment creates 47 tables; health/readiness, backup/restore, forward upgrade, and restore-point rollback pass in isolated MySQL databases. |
| Runtime review | Pass | Health, readiness, served OpenAPI, login/token/profile, and `pharma_oa` discovery pass in an isolated process. |
| Docs and boundaries | Pass | Chinese-default and English release boundaries, links, licensing, sample/regulatory position, data safety, secrets, and support limits pass automated review. |
| Repository boundary | Pass | CodeGraph is synchronized, `git diff --check` passes, and no archived hardening evidence was changed. |

Machine-readable evidence is under [`evidence/h6-03/`](evidence/h6-03/README.md).

## 重试记录

| Attempt | Status | Failure evidence | Retry action |
| --- | --- | --- | --- |
| 1 | Failed -> Doing | Runtime review exception handling read a still-locked stderr file. | Stop the child process before reading diagnostics; rerun. |
| 2 | Failed -> Doing | Windows PowerShell returned served OpenAPI content as bytes, causing a false `/ready` miss. | Decode response bytes explicitly as UTF-8; rerun. |
| 3 | Failed -> Doing | The combined frontend/build/smoke command exceeded the outer 124-second command limit before a complete result was available. | Split build and smoke groups, assign independent limits, and rerun both groups successfully. |
| 4 | Failed -> Doing | Database acceptance refused to reset the ordinary `skoll` database. | Use the protected `skoll_acceptance` database and rerun without touching the existing application database. |

## 问题清单与限制

No P0 issues remain.

| Priority | Item | Disposition |
| --- | --- | --- |
| P1 | No live PostgreSQL DSN was available during this final local run. | PostgreSQL tests skipped locally; existing H2 multi-database contract evidence remains valid. Add PostgreSQL as a required CI service for release candidates. |
| P1 | Docker daemon was unavailable. | Compose configuration and persistence topology were validated; no container launch is claimed. Run image build and Compose startup in release CI. |
| P1 | No live Kubernetes cluster was attached. | Client-side Kustomize render, probes, security context, ConfigMap, Service, and PVCs were validated. Rehearse on a staging cluster before production. |
| P1 | Tag, changelog, SBOM, third-party license report, image signature, and release owner are release-specific. | Required before publishing artifacts; this batch closeout does not manufacture a release identity. |
| P2 | Vite reports known Dart Sass legacy API and VueUse annotation warnings. | Accepted baseline; fail future gates on new warning classes or bundle regression. |
| Boundary | Pharma OA is an industry sample, not regulatory certification or a production SLA. | Follow the bilingual release boundary documents and perform adopter-specific validation. |

## 发布检查快照

| Area | Batch result | Release-time action |
| --- | --- | --- |
| Source, tests, OpenAPI, plugin manifests, frontend build | Pass | Re-run from the tagged commit. |
| Browser, accessibility, responsive, loading/empty/error/no-permission/saving/destructive states | Pass by H4-04/H5-02 evidence | Re-run against the release image. |
| Database, backup, restore, upgrade, rollback | Pass on SQLite/MySQL | Add live PostgreSQL and target-managed database rehearsal. |
| Runtime probes, authentication, profile, plugin discovery | Pass | Re-run in the target topology. |
| License, regulatory, data safety, secrets, support boundaries | Pass | Include the boundary docs in release notes. |
| Version/tag/changelog/SBOM/signing/image publication | Pending release owner | Must complete before public artifact publication. |

## 下一批建议

1. Make MySQL and PostgreSQL service containers mandatory in release CI and retain migration evidence as artifacts.
2. Build, scan, generate SBOMs for, and sign container images from a tagged commit.
3. Add a live staging-cluster deploy, probe, backup/restore, and rollback gate.
4. Continue tenant isolation, security review, production-scale load tests, and operator alert validation.
5. Create a new parent board, Work Item table, and acceptance log under `docs/refactor/current/` before taking more implementation work.

## English Summary

The `skoll-hardening-2026-07-18` batch completed all 25 Work Items and all P0 hardening gates. Go, frontend, bilingual plugin and Pharma OA flows, live MySQL persistence, job/performance evidence, deployment recovery, documentation, and an isolated runtime review passed. Live PostgreSQL, Docker daemon execution, a real Kubernetes cluster, and release-specific identity/SBOM/signing remain explicit release-time work; this closeout is not regulatory certification or approval to publish production artifacts.
