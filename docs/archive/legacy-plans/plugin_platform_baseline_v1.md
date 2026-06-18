# Skoll 插件平台基建蓝图 V1

更新时间：2026-05-20
状态：Draft（可执行）

## 1. 背景与目标

面向“会持续孵化多个子应用”的场景，Skoll 需要从“可装插件”升级到“可治理插件平台”。

V1 目标：
- 子应用可以以独立项目形式交付，并通过插件方式接入 Skoll。
- 平台具备统一的安装、升级、权限、审计、回滚与安全门禁能力。
- 新插件接入成本可控（脚手架化、标准化、门禁自动化）。

## 2. 架构原则

- 一插件一项目：插件代码、发布节奏、团队职责独立。
- 平台与业务解耦：Skoll 管平台能力，插件管业务域能力。
- 契约先行：先定义 Manifest/API/权限/观测标准，再放量接入。
- 默认隔离：数据隔离、权限隔离、故障隔离、发布隔离。
- 可回滚优先：所有插件变更都必须有回滚路径。

## 3. 推荐接入模式

推荐默认模式：`ui_mode=separated`

- 插件前端：作为插件页面入口与静态资源。
- 插件后端：独立服务（建议独立进程/容器），提供业务 API。
- Skoll：
  - 插件注册与生命周期管理
  - 登录态与权限体系
  - 审计汇聚
  - 插件治理能力（启停、升级、回滚、日志）

不建议：把大型业务子应用直接做成 monolith 嵌入宿主进程。

## 4. 标准化契约（Plugin Contract V1）

### 4.1 Manifest 必填字段（最小集）

- `id`
- `name`
- `version`
- `ui_mode`
- `level`
- `mount_policy`
- `permissions`
- `dependencies`

建议新增（V1.1）：
- `api_version`：插件契约版本
- `compatibility.skoll`：兼容的 Skoll Core 版本范围
- `service.health_url`：后端健康检查地址（separated 模式）
- `service.base_url`：后端服务入口（separated 模式）
- `migrations`：迁移版本信息

Schema 草案位置：
- `docs/schemas/plugin-manifest.schema.json`

### 4.2 权限命名规范

采用命名空间：`<pluginId>.<domain>.<action>`

示例：
- `oa.leave.submit`
- `oa.leave.approve`
- `oa.attendance.read`

禁止：跨插件复用未加前缀的权限名。

### 4.3 审计规范

插件关键操作必须写审计（至少包含）：
- actor
- action
- resource
- result
- metadata（插件 ID、业务对象 ID、请求追踪 ID）

### 4.4 数据边界规范

- 插件必须使用独立 schema 或独立数据库。
- 禁止直接写入 Skoll 核心业务表。
- 跨插件数据交互通过 API 或事件总线完成。

## 5. 交付与仓库规范

每个插件独立仓库建议结构：

- `/plugin-manifest/plugin.yaml`
- `/backend`（可选，separated 或 backend_only）
- `/frontend`（可选，frontend_only 或 separated）
- `/migrations`
- `/api-contract`（OpenAPI / protobuf）
- `/deploy`（容器与部署清单）
- `/docs`（运行手册、回滚手册）

每个仓库 CI 最低门禁：
- lint
- unit test
- contract test
- security scan（依赖漏洞）
- manifest schema validation

## 6. 平台能力建设路线图

### P0（必须先完成）

- 插件 SDK 与脚手架（生成标准目录、模板、CI）
- Manifest Schema 与校验器（含兼容性字段）
- 插件迁移执行器（install/upgrade/rollback）
- 权限命名空间校验与冲突检测
- 插件健康检查与启动探针

### P1（规模化前完成）

- 插件市场/注册中心（版本管理、依赖图、审计轨迹）
- 灰度发布与自动回滚
- 统一观测面板（插件级别日志/指标/告警）
- 签名校验与来源信任链

### P2（企业级增强）

- 多租户策略
- 插件 SLA 分级治理
- 成本与资源配额管理
- 合规审计增强（留痕与证据链）

## 7. 当前缺口盘点（基于现状）

已具备：
- 基础生命周期管理（install/enable/disable/uninstall/list/get）
- Manifest 读取与基础校验
- 插件页面与静态资源路由
- 插件持久化（MySQL）

关键缺口：
- 缺少插件迁移生命周期（install/upgrade/rollback）标准执行器
- 缺少插件兼容矩阵（插件版本 vs Skoll Core 版本）
- 缺少插件健康探针与服务发现契约（separated 模式）
- 缺少发布签名校验闭环（字段有，流程未完全打通）
- 缺少插件 CI 门禁模板与一键脚手架

## 8. 验收标准（DoD）

满足以下条件才允许大规模孵化子应用插件：

- 新插件从脚手架创建到本地接入 <= 30 分钟
- 插件安装/升级失败可一键回滚
- 插件权限无冲突且有命名空间校验
- 插件健康状态可被平台实时观测
- 插件关键行为可审计、可追踪

## 9. 建议的下一步（两周内）

第 1 周：
- 完成 Manifest Schema V1 与校验器
- 完成插件脚手架 MVP（含 CI 模板）
- 输出权限命名规范并在 CI 中强校验

第 2 周：
- 完成迁移执行器 MVP（install/upgrade/rollback）
- 完成 separated 模式 service health 契约
- 选一个试点插件（建议 OA）跑通全链路
