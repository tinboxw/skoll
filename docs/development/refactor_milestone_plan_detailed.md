# Skoll 重构实施里程碑计划（详细执行版）

> 基准文档：`docs/refactor.md`
> 计划版本：v2.0（执行版）
> 制定日期：2026-05-09
> 执行原则：阶段完成 -> 门禁通过 -> 阶段评审通过 -> 才可进入下一阶段

## 1. 执行总则

- 采用“里程碑 + 质量门禁 + 阶段评审”三层控制。
- 每个阶段必须输出：代码、测试证据、文档更新、风险与回滚说明。
- 每个阶段评审通过后，更新阶段评审记录，再启动下一阶段。

### 1.1 固定门禁（每阶段都执行）

```bash
go fmt ./...
go test ./...
go test -race ./...
```

### 1.2 前端门禁（涉及 web 变更时）

```bash
cd web
npm install
npm run build
```

### 1.3 阶段评审通过条件

- 任务清单完成率 100%。
- 门禁全部通过。
- 文档同步完成（至少包含变更说明、使用方式、已知限制）。
- 无 P0/P1 级阻塞缺陷。

## 2. 阶段排期总览（按 refactor.md 要求）

| 阶段 | 目标映射 | 计划时间 | 关键交付物 | 阶段状态 |
| --- | --- | --- | --- | --- |
| P0 | 启动与基线治理 | 2026-05-11 ~ 2026-05-13 | 执行计划、评审模板、基线报告 | 进行中 |
| P1 | M1 核心领域与仓储 | 2026-05-14 ~ 2026-05-20 | domain/repository 契约 | 已完成（待复核） |
| P2 | M2 多存储实现 | 2026-05-21 ~ 2026-05-30 | memory/mysql/postgres/clickhouse + factory | 已完成（待复核） |
| P3 | M3 缓存层 | 2026-05-31 ~ 2026-06-04 | redis/memcached/local + factory | 已完成（待复核） |
| P4 | M4 服务层 | 2026-06-05 ~ 2026-06-11 | user/role/rbac/audit service | 已完成（待复核） |
| P5 | M5 API/CLI 接口层 | 2026-06-12 ~ 2026-06-18 | v1 API、middleware、CLI 基础 | 已完成（待复核） |
| P6 | M6 事件总线 | 2026-06-19 ~ 2026-06-23 | bus/publisher/subscriber/events | 已完成（待复核） |
| P7 | M7 测试与优化 | 2026-06-24 ~ 2026-07-01 | integration/benchmark/deploy/docs | 部分完成 |
| P8 | M8 插件基础架构 | 2026-07-02 ~ 2026-07-08 | plugin manager/loader/resolver | 已完成（待复核） |
| P9 | M9 插件扩展点 | 2026-07-09 ~ 2026-07-15 | registry + builtin plugin sample | 已完成（待复核） |
| P10 | M10 前端插件系统 | 2026-07-16 ~ 2026-07-24 | web plugin bootstrap + store + route | 已完成（待联调收口） |
| P11 | M11 插件开发工具 | 2026-07-25 ~ 2026-07-31 | plugin CLI + demo + dev docs | 已完成（待集成接线） |
| P12 | 联调收口与发布评审 | 2026-08-01 ~ 2026-08-08 | 全链路联调报告、最终验收报告 | 进行中 |

## 3. 分阶段详细任务

## P0 启动与基线治理

### 目标
建立可持续推进机制与评审闭环，确认后续阶段执行规则。

### 任务
- 输出详细里程碑计划（本文件）。
- 建立阶段评审记录文件与模板。
- 执行当前基线健康检查（后端启动、健康检查、全量测试门禁）。

### 交付物
- `docs/development/refactor_milestone_plan_detailed.md`
- `docs/development/refactor_stage_review_log.md`
- 基线命令执行结果（附于评审记录）

### 验收标准
- 文档可直接指导执行，不依赖口头补充。
- 阶段评审模板可复用。
- 基线可运行且门禁通过。

## P1 ~ P6（后端核心能力）

### 目标
保持 `docs/refactor.md` 的核心后端架构目标一致，确保领域、存储、缓存、服务、API、事件总线全链路具备可用实现。

### 每阶段固定任务
- 对照 `docs/refactor.md` 对应章节逐条核对目录与职责。
- 完成代码 + 测试 + 文档同步。
- 输出阶段评审项：完成项、未完成项、风险项、回滚方案。

### 关键交付物
- `internal/domain/**`
- `internal/repository/**`
- `internal/store/**`
- `internal/cache/**`
- `internal/service/**`
- `internal/handler/http/**`
- `internal/event/**`

### 验收标准
- 固定门禁全部通过。
- 关键 API 与服务具备可验证行为（含失败路径）。
- 无破坏性回归。

## P7 测试与优化收口

### 目标
完成测试、性能、部署与文档体系收口，匹配 `docs/refactor.md` 对 M7 的要求。

### 任务
- 补齐 `tests/unit` 关键场景覆盖，形成覆盖率报表。
- 完成 benchmark 基线采集与对比。
- 校验 `deploy/docker`、`deploy/k8s`、`deploy/compose` 可用性。
- 同步 `docs/api`、`docs/user`、`docs/architecture`。

### 关键交付物
- 测试覆盖率报告
- benchmark 报告
- 部署验证记录
- 文档同步清单

### 验收标准
- 覆盖率达到规划阈值（目标 >= 80%）。
- 性能基线可复现。
- 部署配置可执行。

## P8 ~ P11（插件与前端能力）

### 目标
完成插件基础架构、扩展点、前端插件系统、插件开发工具并实现可联调。

### 任务
- P8：插件 manager/loader/resolver 生命周期与依赖控制。
- P9：扩展点 route/middleware/event/menu/widget/setting 完整注册能力。
- P10：前端插件注册、状态管理、动态路由、后端插件同步机制。
- P11：`list/debug/logs/validate` 开发工具 + 示例插件 + 文档。

### 关键交付物
- `internal/plugin/**`
- `internal/handler/cli/plugin_cmd.go`
- `web/src/plugins/**`
- `web/src/stores/plugins.ts`
- `docs/development/plugin_dev_tools.md`
- `plugins/demo/**`

### 验收标准
- 插件生命周期可操作且受依赖约束。
- 前端插件系统可注册并渲染。
- 插件工具命令可执行。
- 前端构建通过。

## P12 联调收口与发布评审

### 目标
完成前后端联调闭环，产出可发布的验收结论。

### 任务
- 打通后端插件同步接口（如 `/v1/plugins?enabled=true`）与前端调用。
- 补齐 Vite 代理与联调脚本。
- 形成完整联调用例：健康检查、插件列表、插件详情、错误回退。
- 发布前评审：风险、回滚、发布窗口、验收签字。

### 当前进展（2026-05-10）

- [x] 前端插件同步状态机落地：`loading/success/error`、同步次数与最后同步时间。
- [x] 插件同步降级模式落地：后端同步失败时保留内置插件并展示可重试提示。
- [x] 插件页刷新动作接入统一后端同步入口，形成“加载/错误/重试/降级”闭环。
- [x] 前端门禁通过：`npm run build`、`npm run smoke:auth`、`npm run smoke:auth-actor`。
- [x] Vite 代理支持环境变量切换（联调环境无需改代码即可切换后端地址）。
- [x] 联调脚本补齐：新增 `smoke:health` 与 `smoke:all` 串行校验入口。
- [x] 插件联调用例补齐：`/v1/plugins` 列表、`/debug` 详情、`/logs` 与不存在插件 `404` 回退断言。

### 当前验证证据（2026-05-10）

- `npm run build` 通过。
- `npm run smoke:plugins` 通过（列表/详情/日志/404 回退）。
- `npm run smoke:all` 通过（`smoke:health` + `smoke:auth` + `smoke:plugins` + `smoke:auth-actor`）。

### 关键交付物
- 联调报告
- 发布评审记录
- 最终验收结论

### 验收标准
- 联调用例通过率 100%。
- 无阻塞级缺陷。
- 评审结论为“通过”。

## 4. 阶段评审机制（强制）

每阶段结束必须在评审记录中填写：

- 阶段编号与时间窗口
- 完成项（含文件/模块）
- 未完成项与原因
- 执行门禁命令与结果
- 质量风险与缓解措施
- 是否允许进入下一阶段（Pass/Block）

## 5. 当前执行策略（按“先评审后推进”）

- 当前立即执行：P0 基线评审。
- P1~P11：对已完成阶段执行“复核评审补证”，补齐证据链。
- P12：作为当前主推进阶段，重点完成前后端插件同步接口与联调闭环。

## 6. 分层补齐推进计划（2026-05-10 启动）

### 6.1 现状差距基线（来自本轮全仓扫描）

- 应用层：`system` 业务线缺失 service/handler，`audit` 缺查询 API。
- 缓存层：`redis/memcached` 适配器仍为本地 LRU 模拟实现。
- 存储层：已切换为 `gormrepo` 共享仓储层，`mysql/postgres/system repo` 持久化已打通。
- 基础能力：`pkg/metrics`、`pkg/validator`、`pkg/utils` 存在空实现文件。

### 6.2 分层推进顺序（严格串行）

1. A1 应用层接口闭环（当前阶段）
2. C1 缓存层真实适配（Redis/Memcached）
3. S1 存储层真实持久化（MySQL/Postgres/SystemRepo）
4. B1 基础能力补齐（metrics/validator/utils）

### 6.3 A1 阶段任务清单（进行中）

- [x] 将“缺失实现清单”固化到执行计划。
- [x] 新增 `system` 内存仓储实现并接入 `store.Bundle`。
- [x] 新增 `internal/service/system`（upsert/get/list）。
- [x] 新增 `v1/system` HTTP 接口并接入主路由。
- [x] 新增 `v1/audit` 查询接口并接入主路由。
- [x] `bootstrap` 注入链路接入 system/audit service。
- [x] 执行门禁：`go fmt ./...`、`go test ./...`（`-race` 留在并发专项阶段执行）。

### 6.4 A1 阶段验收标准

- `memory` 模式下 system/audit API 可直接调用并返回结构化结果。
- 新增路由不影响既有 `user/role/rbac/plugin` 路径行为。
- 门禁通过且无新增阻塞回归。

### 6.5 C1 阶段任务清单（进行中）

- [x] Redis 适配器接入真实客户端（保留本地回退保障）。
- [x] Memcached 适配器接入真实协议实现（保留本地回退保障）。
- [x] 保持 `cache.Bundle` 对外接口不变，避免应用层改造耦合。
- [x] 执行门禁：`go mod tidy`、`go fmt ./...`、`go test ./...`。

### 6.6 C1 阶段说明

- 当前策略为“远端优先 + 本地降级”，用于在开发环境无 Redis/Memcached 服务时保持可运行。
- 下一阶段（S1）完成后，再评估是否切换为“严格远端依赖”模式。

### 6.7 S1 阶段任务清单（进行中）

- [x] MySQL adapter 切换为真实 GORM 连接，并完成 DSN 解析。
- [x] MySQL user/role/rbac/system 仓储从 memory 占位替换为持久化实现。
- [x] MySQL schema 自动迁移已打通（兼容索引长度限制）。
- [x] 本地 MySQL 直连验证通过（`SKOLL_TEST_MYSQL_DSN`）。
- [x] PostgreSQL 仓储持久化实现（切换为真实 GORM adapter + 自动迁移）。
- [x] 事务层从 pass-through 升级为真实 DB transaction 边界。
- [x] 提取 `internal/store/sql/gormrepo` 统一 MySQL/PostgreSQL 重复仓储实现。

### 6.8 S1 阶段验证证据

- `go test ./internal/store -run TestNewBundleMySQLIntegration -count=1` 通过。
- `go test ./...`（带 `SKOLL_TEST_MYSQL_DSN`）通过。
- `go fmt ./...`、`go test ./...`（默认环境）通过。

### 6.9 注意事项（存储层扩展约束）

- `internal/store/sql/gormrepo/model` 必须保持“每个表一个模型文件”，避免回归为单一 `models.go` 聚合文件。
- 新增/变更 SQL 模型时必须同步更新 `model/all_models.go`，确保 MySQL/PostgreSQL 自动迁移清单一致。

### 6.10 B1 阶段任务清单（已完成）

- [x] `pkg/utils` 补齐字符串、时间与类型转换工具函数。
- [x] `pkg/validator` 补齐规则定义与通用校验器。
- [x] `pkg/metrics` 补齐内存指标采集与导出（JSON/Text/Prometheus）。
- [x] 为 `metrics/validator/utils` 新增最小单元测试覆盖。
- [x] 执行门禁：`go fmt ./...`、`go test ./...`。

### 6.11 B1 阶段验证证据

- `go test ./pkg/utils ./pkg/validator ./pkg/metrics` 通过。
- `go fmt ./...`、`go test ./...` 通过。

---

> 说明：本计划是 `docs/refactor.md` 的执行化版本。若里程碑内容变更，必须先更新本计划并通过评审后再执行代码变更。
