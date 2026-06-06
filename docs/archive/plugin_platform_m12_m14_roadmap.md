# 插件平台 M12-M14 路线图（可执行版）

## 1. 背景与目标

> 基建阶段执行原则（强约束）
>
> - 不做兼容实现：不保留旧接口、旧字段、旧流程并行。
> - 不做兜底实现：不允许 silent fallback、默认降级成功、自动切回旧路径。
> - 不做双写双读：新模型上线后只允许单一事实来源（SSOT）。
> - 不做灰色开关逃逸：除明确的环境总开关外，不增加“临时旁路开关”。
> - 违反以上任一条，视为未完成，不得进入下一里程碑。

当前插件平台已具备基础开发与管理能力：
- 插件管理：列表、启停、卸载、配置、调试、日志、访问。
- 开发者门户：脚手架、项目扫描、Manifest 编辑、打包、流水线、灰度、发布单审批入口。

但仍存在关键缺口：
- 发布链路未闭环（审批通过后缺少自动发布任务）。
- 灰度/回滚更多是配置层行为，未接入真实流量层。
- 发布流程持久化与可追踪性不足（历史和状态在本地文件/内存侧偏重）。
- 平台观测与门禁能力弱（缺少可视化发布态与硬门禁策略）。

本路线图聚焦三段式推进：
- M12：发布闭环最小可用（MVP）。
- M13：灰度与回滚真实化、持久化与可观测。
- M14：平台化与治理能力补齐。

---

## 2. 里程碑总览

| 里程碑 | 周期建议 | 核心目标 | 结果形态 |
|---|---|---|---|
| M12 | 2 周 | 审批到发布任务闭环 | 能从发布单直接触发并跟踪发布任务 |
| M13 | 2 周 | 灰度/回滚真实化与状态持久化 | 灰度策略接入执行层，回滚可重复可审计 |
| M14 | 2 周 | 治理与平台化 | 门禁、观测、文档契约、运行手册完整 |

---

## 3. M12（发布闭环 MVP）

### 3.1 范围

后端：
- 新增发布任务域模型（发布单 -> 发布任务）。
- 审批通过触发发布任务创建（异步执行）。
- 发布任务状态机：pending -> running -> success/failed/cancelled。
- 提供任务查询与日志查询 API。

前端（Developer Portal）：
- 发布单列表增加“触发发布”动作与“任务追踪”面板。
- 任务进度、失败原因、产物信息可视化。

基建硬约束（M12）：
- execute 链路只走发布任务执行器，不保留“审批后同步直发”旧路径。
- 发布失败必须显式失败返回，不允许自动改为“仅记录不执行”。
- 页面仅消费新任务接口，不读取旧状态拼装结果。

### 3.2 建议接口

- `POST /skoll/v1/plugins/dev/release-orders/{orderId}/execute`
	- 入参：`pluginsRoot`、`pluginId`、`targetEnv`、`artifactPath`。
	- 出参：`taskId`、`status`。
- `GET /skoll/v1/plugins/dev/release-tasks`
	- 支持按 `pluginId`、`orderId`、`status` 过滤。
- `GET /skoll/v1/plugins/dev/release-tasks/{taskId}`
	- 返回阶段进度与结果。
- `GET /skoll/v1/plugins/dev/release-tasks/{taskId}/logs`
	- 返回发布步骤日志（结构化优先）。

### 3.3 任务拆分

后端：
1. 定义 `ReleaseTask` 领域模型与存储接口。
2. 将 `approve` 与 `execute` 解耦，审批通过不直接阻塞请求线程。
3. 引入后台任务执行器（简单 worker 池即可）。
4. 给每个任务步骤打审计点（开始/结束/失败）。

前端：
1. Developer Portal 增加“发布任务”模块。
2. 发布单列表联动任务状态。
3. 失败任务支持“一键查看失败步骤”。

测试：
1. 任务状态机单元测试。
2. 审批通过触发任务的集成测试。
3. 任务日志与查询接口的契约测试。

### 3.4 M12 本周执行清单（零兼容版）

后端泳道（P0）：
1. 在发布单审批逻辑中删除旧同步发布分支，只保留“创建任务并入队”。
2. 新建 `release_tasks` 持久化模型与仓储，发布状态以任务状态为唯一来源。
3. 新增 execute/list/detail/logs 四个 API，并统一错误码与审计字段。
4. 任务执行器步骤固定化：`prepare` -> `verify_artifact` -> `publish` -> `post_check`。
5. 明确并发策略：同插件同环境存在 running 任务时直接 `task_conflict`。

前端泳道（P0）：
1. 发布单区域接入 execute API，触发后仅展示任务视图。
2. 任务列表支持按状态筛选与详情抽屉。
3. 任务日志按步骤分段展示，失败步骤置顶。
4. 删除/下线页面里对旧发布状态字段的依赖展示。

测试泳道（P0）：
1. 状态机覆盖：合法流转全覆盖，非法流转必须失败。
2. 并发覆盖：重复 execute 返回 `task_conflict`。
3. 失败覆盖：任一步骤失败，任务最终状态必须是 failed 且含失败步骤。
4. 端到端覆盖：发布单 -> execute -> 查询任务 -> 查询日志。

文档泳道（P0）：
1. 在 `plugin_dev_tools.md` 增补 release-task API 使用说明。
2. 在 openapi 中补齐 execute/list/detail/logs 的请求响应结构。
3. 标注“无兼容路径”升级说明，明确这是基建破坏性调整。

### 3.5 DoD

- 审批通过后可创建发布任务并可查询到状态流转。
- 失败任务能定位到明确失败步骤与错误信息。
- 发布任务与发布单存在稳定关联（可追溯）。
- API 与页面文案、文档保持一致。
- 不存在旧路径可触发发布（代码检索与集成测试双重证明）。
- 不存在 fallback 到旧状态字段的前端渲染逻辑。

### 3.6 验收用例

1. 创建发布单 -> 审批通过 -> 执行发布 -> 任务成功。
2. 制造执行失败 -> 任务失败 -> 页面可见失败原因。
3. 同一插件并发两个任务时，策略符合预期（排队或拒绝）。
4. 强制检查旧 API/旧字段调用，必须全部为 0（否则验收失败）。

---

## 4. M13（灰度/回滚真实化 + 持久化）

### 4.1 范围

后端：
- 灰度状态从内存历史迁移到持久化存储。
- 引入灰度策略执行接口（对接网关/流量层抽象）。
- 回滚策略从“上次值”扩展为“回滚点版本”。

前端：
- 灰度策略展示升级：当前流量、目标流量、执行中状态、历史回滚点。
- 回滚确认弹窗展示影响范围与预计恢复时间。

基建硬约束（M13）：
- 禁止“执行失败时仅写配置成功”的伪成功行为。
- rollout/rollback 结果仅以执行层回执为准，不做本地推断成功。

### 4.2 建议接口

- `POST /skoll/v1/plugins/dev/rollout`
	- 扩展：`strategyType`（percent/tag/canary）、`targetEnv`。
- `POST /skoll/v1/plugins/dev/rollback`
	- 扩展：`rollbackTo`（last_success | taskId | version）。
- `GET /skoll/v1/plugins/dev/rollout/history`
	- 返回每次灰度执行记录与操作者。

### 4.3 任务拆分

后端：
1. 灰度历史持久化表设计与迁移。
2. `rollout` 执行适配层（先做 mock adapter，后接真实网关）。
3. `rollback` 支持基于回滚点恢复。
4. 发布任务与灰度任务统一任务框架。

前端：
1. 灰度历史时间线组件。
2. 回滚点选择器与安全确认。
3. 将“仅配置写入”提示替换为“执行层状态”。

测试：
1. 灰度历史持久化回归测试。
2. 回滚点可重放测试。
3. 执行层异常下的幂等与重试策略测试。

### 4.4 DoD

- 重启服务后灰度/回滚历史不丢失。
- 回滚可指定历史点并可审计。
- 页面可看到执行层真实状态，不再仅反映配置字段。

### 4.5 验收用例

1. 灰度 10% -> 30% -> 回滚到 10%。
2. 服务重启后历史可查询且可继续回滚。
3. 执行层故障时系统返回可诊断错误且不污染状态。

---

## 5. M14（治理与平台化）

### 5.1 范围

门禁治理：
- 版本门禁（SemVer 规则、跳版策略）。
- 变更门禁（changelog 必填、breaking change 标记）。
- 依赖门禁（插件依赖冲突检测）。

可观测：
- 发布任务看板（成功率、平均时长、失败类型分布）。
- 插件健康状态统一指标（安装态、运行态、发布态）。

契约与文档：
- OpenAPI 补齐 Dev Portal 全量接口。
- 运行手册：故障场景、回滚 SOP、发布值班检查单。

基建硬约束（M14）：
- 门禁检查失败必须阻断发布，不允许“告警但放行”。
- 指标采集失败必须显式暴露，不允许静默吞错。

### 5.2 建议接口与报表

- `GET /skoll/v1/plugins/dev/metrics/release`
- `GET /skoll/v1/plugins/dev/metrics/health`
- `GET /skoll/v1/plugins/dev/gates/check?pluginId=...`

### 5.3 任务拆分

1. 门禁引擎（规则可配置）与错误码规范。
2. 插件发布看板页（趋势 + 明细 + drill-down）。
3. OpenAPI 与页面实际能力对齐。
4. 编写与演练发布故障手册。

### 5.4 DoD

- 任何发布动作必须经过门禁检查并给出可读失败原因。
- 发布/灰度/回滚三类任务都有指标与看板可查。
- 文档与接口实现一致，抽样验证通过。

### 5.5 验收用例

1. 缺少 changelog 的发布被门禁拒绝。
2. 依赖冲突插件无法发布并给出冲突详情。
3. 发布失败类型可在看板中按维度筛选定位。

---

## 6. 跨里程碑通用规范

### 6.1 API 规范

- 错误码统一：`invalid_request`、`forbidden`、`not_found`、`task_conflict`、`gate_failed`。
- 所有 Dev API 继续限制 `super_admin`。
- 所有写操作接口必须写审计日志。
- 所有失败必须显式返回失败码，不允许 200 + message 伪失败。

### 6.2 数据规范

- 发布单、发布任务、灰度历史三类数据采用稳定主键关联。
- 所有时间字段使用 UTC RFC3339。
- 所有状态字段有限状态机化，避免自由字符串。

### 6.3 测试门禁

- 每个里程碑至少新增：
	- 单元测试 >= 10 个关键用例。
	- 集成测试 >= 3 条端到端主链路。
	- 回归场景 >= 3 条（重启、并发、失败重试）。

### 6.4 基建禁令（CI 强校验）

- 禁止出现 fallback 关键路径：
  - CI grep 关键字：`fallback`、`degrade`、`兼容`、`兜底`（允许注释白名单需评审）。
- 禁止旧接口引用：
  - 通过契约扫描与前端静态检查确保旧 API 调用为 0。
- 禁止双源状态：
  - 发布状态字段来源检查必须唯一（任务表/任务 API）。
- 任一禁令命中，流水线直接失败。

---

## 7. 交付清单模板（每个里程碑）

1. 代码交付
- 后端 API、服务层、存储层、前端页面功能。

2. 测试交付
- 单测、集成测试、手工验收脚本。

3. 文档交付
- 开发文档、接口文档、运行手册。

4. 运维交付
- 启动参数说明、故障排查清单、回滚预案。

---

## 8. 建议执行顺序（两周节奏）

第 1 周（M12）：
- 后端任务模型 + execute API + 最小任务执行器。
- 前端任务跟踪面板。
- 跑通“审批 -> 执行 -> 成功/失败可追踪”。

第 2 周（M13）：
- rollout/rollback 持久化。
- 执行层 adapter 抽象与 mock 对接。
- 灰度历史可视化与回滚点选择。

第 3 周（M14）：
- 门禁引擎与发布看板。
- OpenAPI 补齐与运行手册。
- 里程碑联合验收。

---

## 9. 当前代码定位参考

- 插件开发与 Dev API：
	- [internal/handler/http/v1/plugin/handler.go](internal/handler/http/v1/plugin/handler.go)
	- [internal/handler/http/v1/plugin/dev_permission_catalog.go](internal/handler/http/v1/plugin/dev_permission_catalog.go)
	- [internal/handler/http/v1/plugin/dev_release_order.go](internal/handler/http/v1/plugin/dev_release_order.go)

- 开发者门户前端：
	- [plugins/developer-portal/static/index.html](plugins/developer-portal/static/index.html)
	- [plugins/developer-portal/static/app.js](plugins/developer-portal/static/app.js)

- 插件管理页面：
	- [web/src/views/Plugin/index.vue](web/src/views/Plugin/index.vue)

- Manifest 结构与约束：
	- [internal/plugin/types.go](internal/plugin/types.go)
	- [docs/schemas/plugin-manifest.schema.json](docs/schemas/plugin-manifest.schema.json)

