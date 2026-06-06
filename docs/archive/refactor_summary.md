# Skoll 项目重构总结

> 版本: v1.0
> 最后更新: 2026-06-06
> 基于: docs/archive/refactor.md、refactor_milestone_plan.md、refactor_milestone_plan_detailed.md、refactor_stage_review_log.md

---

## 技术栈

> 与迭代计划同步，详见 [iteration_plan.md](../iteration_plan.md#技术栈)

### 后端核心

| 层次 | 技术 | 说明 |
|------|------|------|
| 语言 | Go 1.24 | 实际版本，非 refactor.md 原始规划的 1.22 |
| HTTP 框架 | net/http | **实际实现**，refactor.md 原始规划为 Gin，代码中未使用 |
| ORM | GORM 1.31+ | 含 MySQL/PostgreSQL 驱动 |
| SQL Builder | goqu 9.19+ | 类型安全 SQL 构建 |
| 配置 | Viper 1.20+ | 多格式配置管理 |
| 日志 | Zap 1.28+ | 高性能结构化日志 |
| 缓存 | go-redis v9 | Redis 客户端 |

### 存储架构

| 层次 | 技术 | 状态 |
|------|------|------|
| 主存储 | MySQL 8.0+ (gormrepo) | ✅ 已实现 |
| 分析存储 | PostgreSQL 16+ (GORM adapter) | ✅ 已实现 |
| 时序存储 | ClickHouse 24+ | ✅ 已实现 |
| 分布式缓存 | Redis 7.0+ (远端优先+本地降级) | ✅ 已实现 |
| 轻量缓存 | Memcached 1.6+ (远端优先+本地降级) | ✅ 已实现 |
| 本地缓存 | LRU (进程内) | ✅ 已实现 |

### 前端

| 层次 | 技术 | 版本 |
|------|------|------|
| 框架 | Vue | 3.4.38 |
| 构建 | Vite | 2.9.18 |
| 状态管理 | Pinia | 2.1.7 |
| 路由 | Vue Router | 4.4.3 |
| 样式 | SCSS | - |
| 图标 | Lucide Vue Next | 1.0.0 |

---

## 已完成

### 重构阶段 P0-P7（原 M0-M7 后端核心）

| 阶段 | 目标 | 关键产出 | 状态 |
|------|------|----------|------|
| P0 | 启动与基线治理 | 执行计划、评审模板、基线报告 | ✅ |
| P1 | 核心领域模型与仓储设计 | domain/ 实体 + repository/ 接口 | ✅ |
| P2 | 多存储实现 | memory/mysql/postgres/clickhouse + gormrepo 统一层 | ✅ |
| P3 | 缓存层 | redis/memcached/local + cache factory | ✅ |
| P4 | 业务服务层 | user/role/rbac/audit/system service | ✅ |
| P5 | API/CLI 接口层 | HTTP v1 + middleware + CLI | ✅ |
| P6 | 事件总线 | bus/publisher/subscriber/events + Redis Pub/Sub | ✅ |
| P7 | 测试与优化收口 | integration/benchmark/deploy/docs | ✅ |

### 重构阶段 P8-P12（插件与前端能力）

| 阶段 | 目标 | 关键产出 | 状态 |
|------|------|----------|------|
| P8 | 插件基础架构 | plugin manager/loader/resolver | ✅ |
| P9 | 插件扩展点 | registry + builtin plugin sample | ✅ |
| P10 | 前端插件系统 | web plugin bootstrap + store + route | ✅ |
| P11 | 插件开发工具 | plugin CLI + demo + dev docs | ✅ |
| P12 | 联调收口与发布评审 | 全链路联调报告、最终验收 | ✅ |

### 分层补齐阶段

| 阶段 | 目标 | 关键产出 | 状态 |
|------|------|----------|------|
| A1 | 应用层接口闭环 | system/audit service + handler | ✅ |
| C1 | 缓存层真实适配 | Redis/Memcached 真实客户端（保留本地降级） | ✅ |
| S1 | 存储层真实持久化 | gormrepo 统一 MySQL/PostgreSQL 仓储 | ✅ |
| B1 | 基础能力补齐 | pkg/metrics / validator / utils | ✅ |

### 重构阶段门禁记录

全部 13 个阶段（P0-P12）均通过：`go fmt ./...`、`go test ./...`、`npm run build`、`go test -race ./...`。详见 [refactor_stage_review_log.md](refactor_stage_review_log.md)。

### 数据库结构重建 ✅

- 主键统一为 `BIGINT UNSIGNED`
- 引擎统一 `InnoDB`，字符集统一 `utf8mb4_unicode_ci`
- 用户模型统一 "account + name"，`displayname`/`display_name` 已清理
- 所有 `sk_id_map_*` 兼容映射表已删除
- 详见 [database_schema_refactor_plan.md](database_schema_refactor_plan.md)

---

## 规划中

### 插件安全治理（原 P3b/P3c/P4）

| 编号 | 工作项 | 优先级 | 说明 |
|------|--------|--------|------|
| P3b | 插件风险分级 | 中 | 风险评分引擎（来源/权限/UI模式/维护度）、风险等级分类 |
| P3c | 策略引擎与决策 | 中 | 组织安全策略定义、策略评估、手动审批流程、合规报告 |
| P4 | 插件市场与分发 | 低 | 租户级策略隔离、插件生态建设、性能与安全基准 |

### 插件能力扩展（原 S3）

| 编号 | 工作项 | 优先级 | 说明 |
|------|--------|--------|------|
| S3-Link | 外挂外部链接 | 高 | 快速补齐业务入口，URL配置 + 域名白名单 + 安全策略 |
| S3-Upload | 上传插件 | 中 | 上传/校验/安装异步任务，进度跟踪 |
| S3-Embed | 外挂外部页面 | 低 | iframe 嵌入 + sandbox + CSP 白名单 + 降级策略 |

### 审计与日志增强（原 S4）

| 编号 | 工作项 | 优先级 | 说明 |
|------|--------|--------|------|
| S4-1 | 统一审计体系 | 中 | 用户行为/权限/配置/数据全覆盖审计 |
| S4-2 | 查询与导出 | 中 | 多条件筛选、高风险模板、异步导出 |
| S4-3 | 异常行为监控 | 低 | 规则引擎 + 告警分级 + 自动抑制 |
| S4-4 | 可视化看板 | 低 | 操作趋势、失败率、风险分布、事件时间线 |

### 数据库统一化（原 S2）

| 编号 | 工作项 | 优先级 | 说明 |
|------|--------|--------|------|
| S2-1 | 插件管理数据库化 | 中 | sk_plugins/sk_plugin_releases/sk_plugin_routes 表实现 |
| S2-2 | 读写路径切换 | 中 | 管理端插件列表改为数据库驱动，启停/安装/卸载写库 |
| S2-3 | 一致性巡检 | 低 | 定时任务校验运行时与数据库状态 |

---

## 与原 refactor.md 的关键差异

| 差异项 | refactor.md 规划 | 实际实现 |
|--------|-----------------|---------|
| HTTP 框架 | Gin 1.9+ | net/http（标准库） |
| Go 版本 | 1.22+ | 1.24 |
| 存储层模式 | store/memory、store/sql/mysql 等独立实现 | gormrepo 统一仓储层，消除 MySQL/PostgreSQL 重复 |
| 前端 Element Plus | 规划 2.6+ | 未引入，使用自定义 CSS（待替换） |
| Vite 版本 | 规划 5.2+ | 实际 2.9（待升级） |
| Gin-Vue-Admin 契约 | 规划兼容 | 未实施，使用自有接口契约 |

---

**文档维护者**: 开发团队
**下次审查**: 2026-07-01
