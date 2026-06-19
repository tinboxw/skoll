# FE4 Plugin Portal Risk Map

日期: 2026-06-19
范围: ADJ-FE-20260619-06
状态: Accepted

## Purpose

FE4 插件门户涉及安装、启停、配置、日志、发布、回滚和任务状态。后续 FE4 页面改造必须把风险、权限、状态、确认和失败原因放在操作者第一眼可扫的位置。

## Risk Matrix

| Area | Primary risk | Current anchors | Required UI signal | Acceptance |
|---|---|---|---|---|
| Install | 安装未知包可能引入权限、菜单、迁移和签名风险 | `installPluginPath()`、`validatePluginPath()`、安装/校验按钮 | 预检结果、签名状态、权限 diff、菜单 diff、迁移影响、阻断原因 | Install 前必须先能读取预检结果；失败时保留错误、风险和可重试路径。 |
| Enable/Disable | 启停会改变菜单、路由、默认首页和可见权限 | `runAction()`、`confirmPluginAction()`、默认首页回退逻辑 | enabled 状态、默认首页影响、访问可达性、危险确认 | Disable/Uninstall 必须二次确认；影响默认首页时必须提示回退。 |
| Config | 配置错误可能导致插件不可用或暴露敏感信息 | `openConfig()`、`saveConfig()`、`SchemaForm`、JSON fallback | schema 字段、保存中、校验失败、敏感配置提示、保存结果 | Config 保存必须有 loading、错误、成功和无权限态；schema 与 JSON fallback 分区清楚。 |
| Logs | 日志缺失或错误会影响排障 | `openLogs()`、inspector logs panel | 空态、加载失败、插件 ID、日志来源 | Logs 面板必须区分暂无日志、读取失败和未选择插件。 |
| Release | 打包、发布单、流水线可能失败或需要审批 | `runDevAction("package"|"pipeline")`、`reviewDevReleaseOrder()`、`executeDevReleaseOrder()` | 发布单状态、审批动作、产物路径、失败原因 | Release 任务必须展示 pending/approved/executed/failed，并保留审批与执行条件。 |
| Rollback | 回滚会影响线上路由、菜单和版本 | `runDevAction("rollback")`、`confirmDevAction()` | danger 确认、目标插件、环境、比例或版本、结果摘要 | Rollback 必须二次危险确认，并展示操作响应和任务日志。 |
| Task status | 任务详情、步骤、日志缺失会降低可恢复性 | `openDevTaskDrawer()`、release/rollout task tables | detail、steps、logs、failureReason、duration | Task status 必须支持详情、步骤和日志三层查看；失败行必须有 failureReason。 |

## Acceptance Gate

- Install: 预检、签名、权限 diff、菜单 diff、迁移影响和失败原因可以在页面验收中定位。
- Enable/Disable: 启用、停用、卸载、默认首页回退和危险确认均有状态记录。
- Config: schema 表单、JSON fallback、保存中、错误、成功、无权限态都可验证。
- Logs: 未选择插件、暂无日志、读取失败和日志内容四类状态都可区分。
- Release: 发布单、发布任务、审批、执行、失败原因和产物路径可读。
- Rollback: 回滚动作必须被危险确认保护，并在任务列表/日志中可追踪。
- Task status: release 与 rollout 任务均有 detail、steps、logs 验收路径。

## Validation Command

```powershell
rg -n "Install|Enable/Disable|Config|Logs|Release|Rollback|Task status|Acceptance" docs/refactor/fe4_plugin_portal_risk_map.md
```
