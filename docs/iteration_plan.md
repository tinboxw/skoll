# Skoll 项目迭代计划

> 版本: v3.0
> 最后更新: 2026-06-06
> 状态: 基于实际代码状态同步

---

## 技术栈

### 后端

| 层次 | 技术 | 版本 | 用途 |
|------|------|------|------|
| **语言** | Go | 1.24 | 核心开发语言 |
| **HTTP 框架** | net/http | 标准库 | HTTP 服务（非 Gin） |
| **ORM** | GORM | 1.31+ | 关系型数据库操作 |
| **SQL Builder** | goqu | 9.19+ | 类型安全的 SQL 构建 |
| **配置管理** | Viper | 1.20+ | 多格式配置、热更新 |
| **日志** | Zap | 1.28+ | 高性能结构化日志 |
| **认证** | JWT | 标准 | 无状态认证 |
| **密码安全** | golang.org/x/crypto | - | 密码哈希 |

### 数据库与缓存

| 层次 | 技术 | 版本 | 用途 |
|------|------|------|------|
| **主数据库** | MySQL | 8.0+ | 核心业务数据存储（GORM + gormrepo） |
| **分析数据库** | PostgreSQL | 16+ | 复杂查询、JSONB 支持（GORM adapter） |
| **时序数据库** | ClickHouse | 24+ | 审计日志、指标存储 |
| **分布式缓存** | Redis | 7.0+ | 会话管理、热点数据、事件 Pub/Sub |
| **轻量缓存** | Memcached | 1.6+ | 页面/配置缓存（远端优先 + 本地降级） |
| **本地缓存** | LRU | 进程内 | 开发/测试环境兜底 |

### 消息与事件

| 层次 | 技术 | 用途 |
|------|------|------|
| **消息队列** | Redis Pub/Sub | 事件通知、异步处理 |
| **事件总线** | 内存 + Redis | 领域事件发布订阅 |

### 前端

| 层次 | 技术 | 版本 | 用途 |
|------|------|------|------|
| **框架** | Vue | 3.4+ | 前端框架 |
| **构建工具** | Vite | 5.4+ | 开发与构建 |
| **状态管理** | Pinia | 2.1+ | 全局状态管理 |
| **路由** | Vue Router | 4.4+ | 前端路由 |
| **样式** | SCSS | - | CSS 预处理器 |
| **图标** | Lucide Vue Next | 1.0+ | 图标库 |
| **表格导出** | xlsx | 0.18+ | Excel 导出 |

### 部署与工具链

| 层次 | 技术 | 用途 |
|------|------|------|
| **容器化** | Docker | 应用容器化 |
| **编排** | Kubernetes / Docker Compose | 多环境部署 |
| **测试** | go test / vegeta / k6 | 单元/集成/性能测试 |

---

## 已完成

### M0: 基础设施搭建 ✅ 100%

Go Module 初始化、目录结构、DI 容器、安全工具（JWT / 密码哈希）、配置管理（Viper）、日志框架（Zap）、错误处理。

### M1: 领域模型 ✅ 100%

用户/角色/RBAC/审计/系统领域实体、值对象、业务规则；仓储接口定义（含事务接口）。

### M2: 存储层 ✅ 100%

Memory / MySQL / PostgreSQL / ClickHouse 多后端存储实现，gormrepo 统一仓储层，数据库迁移脚本，存储工厂。

### M3: 缓存层 ✅ 100%

Redis / Memcached / Local LRU 三层缓存实现，缓存工厂。策略为"远端优先 + 本地降级"。

### M4: 服务层 ✅ 100%

用户/角色/RBAC/审计/系统 5 个核心业务服务，事务管理器，业务校验器。

### M5: API 接口层 ✅ 100%

HTTP v1 RESTful API、统一响应封装、认证/日志/限流中间件、CLI 命令。

### M6: 事件总线 ✅ 100%

事件总线接口、发布者/订阅者、核心领域事件定义、Redis Pub/Sub 集成。

### M7: 部署交付 ✅ 100%

Docker / Kubernetes / Docker Compose 部署配置已具备，可支撑本地、容器和 K8s 场景启动。测试覆盖率、性能基线和质量门禁统一归入 M15 跟踪，避免与交付配置混用。

### M8: 插件系统基础 ✅ 100%

插件管理器、加载器、依赖解析器、生命周期管理（installed/enabled/disabled/uninstalled）、权限控制。

### M9: 插件扩展点 ✅ 100%

路由/中间件/事件/菜单/仪表盘组件/设置页 6 类扩展点注册机制，内置插件示例。

### M10: 插件 DevPortal ✅ 100%

前端插件框架、注册机制、通信体系、状态管理集成、开发者门户。

### M11: 插件发布管理 ✅ 100%

发布审批、灰度、回滚真实化：

- 三策略灰度：percent（百分比）/ tag（标签）/ canary（金丝雀版本）
- 回滚点持久化：`devRolloutPoint` 按版本号/百分比恢复
- Adapter 抽象：`DevRolloutExecutor` 接口，Mock 执行器实现
- 关键文件：`internal/handler/http/v1/plugin/dev_rollout.go`、`internal/adapter/dev_rollout.go`

### M12: 前端后台框架 ✅ 100%

Vue 3.4 + Vite 5.4 项目初始化，Pinia 2.1 状态管理，Vue Router 4.4，SCSS 样式系统，Layout 组件（Sidebar + Header），Lucide Vue Next 图标库，Element Plus 按需导入，登录页面，API 请求封装。

### M13: 前端页面开发 ✅ 100%

6 个核心管理页面全部实现：

- Dashboard 仪表盘页面
- User 用户管理页面（含批量创建、角色分配、禁用）
- Role 角色管理页面（含权限授予/撤销、角色用户查询）
- Permission 权限管理页面
- Plugin 插件管理页面（含安装/启停/卸载/配置/DevPortal）
- Setting 系统设置页面

### M14: 前端集成联调 ✅ 100%

前后端 API 对接完成，认证流程联调（登录/JWT/刷新），JWT 认证中间件（含白名单放行），插件前端页面静态资源服务，插件 iframe 嵌入页面。

---

## 规划中

### 规划原则

当前主线从单点功能收敛转向开源 Admin 基建：先补动态菜单/动态路由/权限同源模型，再推进 Element Plus 全量迁移和插件 UI 声明能力，随后补齐字典、配置、文件、代码生成、质量门禁与发布文档。插件能力与 UI 能力是第一梯队，不再作为远期附加项。

### 下一步执行计划

| 顺序 | 编号 | 工作项 | 优先级 | 预估工期 | 验收标准 |
|------|------|--------|--------|----------|----------|
| 1 | P0-0 | 状态校准与基线报告 | P0 | 0.5 天 | 执行并记录 `go test ./...`、覆盖率统计、`cd web && npm run build`、现有 CI 门禁；产出当前失败项和覆盖率缺口 |
| 2 | P0-1 | Vite 升级到 5.x | P0 | 已完成 | 升级 `vite`、`@vitejs/plugin-vue`、TypeScript 兼容项；`npm run build` 通过 |
| 3 | P0-2 | Element Plus 分阶段接入 | P0 | 基础完成 | 已按需接入 Element Plus，并完成插件管理页、权限矩阵页、菜单管理页、User/Role 列表页、Role 编辑页、User 新增/编辑/批量页、Audit/Setting 页面和 Common 组件首批迁移；已新增 schema 表单渲染器首版并接入插件配置面板 |
| 4 | P0-5 | 动态菜单/路由/权限基线 | P0 | 进行中 | 前端菜单注册中心、插件 `ui_menu` 解析输出、`canAccess`、`v-permission`、权限矩阵页、后端 `/v1/system/menus`、侧栏远端菜单加载、菜单管理页首版、同级排序、静态页面级路由权限、插件路由权限继承和 User/Role 首批按钮权限已接入；下一步做 UI 迁移 |
| 5 | P1-1 | 字典/配置/文件基础 | P1 | 1-2 周 | 字典管理、配置 schema、上传存储接口进入 API/UI 主线 |
| 6 | M15-1 | 覆盖率提升计划 | P1 | 1 周 | 建立 `coverage.out` 生成方式；优先补齐 service、handler、store/sql 低覆盖包；核心业务包覆盖率目标 ≥80% |
| 7 | M15-2 | API 性能基准与报告 | P1 | 1 周 | 选定 k6 或 vegeta；覆盖登录、用户列表、角色列表、插件列表、审计查询；输出 QPS、P95/P99、错误率和环境参数 |
| 8 | M15-3 | CI/CD 质量门禁增强 | P1 | 0.5-1 周 | CI 增加前端 build、覆盖率产物、benchmark artifact；明确 race test 触发规则和失败处理 |
| 9 | M16-1 | 发布文档收口 | P2 | 持续 | 将基线报告、性能报告、部署、配置、插件开发教程统一链接到 `docs/README.md` |

### P0: 开源 Admin 基建与 UI 收敛 🔄 55%

| 编号 | 工作项 | 当前状态 | 下一步 |
|------|--------|----------|--------|
| P0-0 | 状态校准与基线报告 | 未开始 | 先跑最小门禁，确认真实失败项；不要凭历史归档估算 |
| P0-1 | Vite 版本升级 | 已完成 | 已从 Vite 2.9 升到 5.x，并通过 `npm run build` |
| P0-2 | Element Plus UI 替换 | 基础完成 | 已按需接入，完成插件管理页、权限矩阵页、菜单管理页、User/Role 列表页、Role 编辑页、User 新增/编辑/批量页、Audit/Setting 页面和 Common 组件首批迁移；已新增 `SchemaForm` 并接入插件 `config_schema` 配置面板与系统设置常用配置区；字段级校验已支持 required/min/max/min_length/max_length/pattern；下一步补齐确认弹窗和空状态 |
| P0-3 | 插件管理页信息架构优化 | 已完成首批 | 已改为状态卡片、系统/应用插件分区、DevPortal 工作台、操作下拉、检查器和危险动作确认；DevPortal 已支持脚手架、校验、打包、发布审批、执行、percent/tag/canary 灰度、回滚、任务详情/日志 |
| P0-4 | 路由懒加载 | 已完成 | 静态后台页面已改为动态导入，降低首屏业务包体 |
| P0-5 | 动态菜单/路由/权限基线 | 进行中 | 已新增前端菜单注册中心，统一系统菜单、插件菜单、排序和权限过滤；已补齐插件 `ui_menu` schema、后端解析、签名 canonical、列表输出；已新增插件 `config_schema` manifest 声明、后端解析/API 输出/签名 canonical、字段级校验和前端配置表单渲染；系统设置页已复用 `SchemaForm` 渲染常用配置；已新增 `canAccess` 与 `v-permission` 并在插件页首批接入；权限管理页已升级为 Element Plus 权限矩阵；已新增后端 `/v1/system/menus` 菜单树 API，并在认证后加载远端菜单驱动侧栏；菜单管理页首版已支持树形编辑、显隐、同级上移/下移排序、权限字段和保存；静态后台页面已补齐页面级路由权限并复用 `canAccess`/隐含权限规则；插件路由已继承 `ui_menu.required_roles/required_permissions` 并修正无权默认页回退；User/Role 列表、角色编辑页、User 新增/编辑/批量新增表单已接入创建、编辑、删除、授权等按钮权限；下一步做可选拖拽排序和统一确认弹窗 |

**决策说明**: Vite 升级已先于 Element Plus 迁移完成。Element Plus 使用按需导入，避免全量注册造成入口包过大。

### M15: 测试与质量 🔄 60%

| 编号 | 工作项 | 当前状态 | 下一步 |
|------|--------|----------|--------|
| M15-1 | 测试覆盖率统计 | 未形成当前报告 | 运行 `go test ./... -coverprofile=coverage.out`，按包排序覆盖率缺口 |
| M15-2 | 单元/集成测试补齐 | 已有 71 个 `_test.go` 文件 | 优先补 handler、service、store/sql 的边界错误、权限失败、事务失败场景 |
| M15-3 | 完整 API 性能基准 | 仅有 Go benchmark 和 CI benchmark snapshot | 新增 k6/vegeta 脚本，固定测试账号、数据规模和运行参数 |
| M15-4 | CI/CD 流水线 | 已有 Go CI、gofmt、go test、按需 race、benchmark artifact | 增加前端 build、覆盖率产物、OpenAPI/manifest 校验入口 |
| M15-5 | 代码质量门禁 | 已有 gofmt 和 no-fallback 检查 | 评估是否引入 golangci-lint；前端补 `vue-tsc` 或 ESLint 前先升级 Vite/TS |

### M16: 文档与发布 🔄 70%

| 编号 | 工作项 | 当前状态 | 下一步 |
|------|--------|----------|--------|
| M16-1 | 开发环境搭建指南 | 已有 `development/getting-started.md` | 补充常见失败排查和前后端联调路径 |
| M16-2 | API 详细使用文档 | 已有 `api/README.md` 与 `api/openapi.yaml` | 增加认证、分页、错误码、插件 DevPortal 示例请求 |
| M16-3 | 数据库 Schema 文档 | 已有 `architecture/database.md` | 与最新 migration/gormrepo 字段再校准一次 |
| M16-4 | RBAC 权限模型说明 | 已有 `architecture/rbac.md` | 补充典型授权链路和失败判定示例 |
| M16-5 | 部署运维文档 | 已有 `user/deployment.md` | 增加生产配置检查清单、日志/健康检查/回滚说明 |
| M16-6 | 插件开发教程 | 已有 `development/plugin-guide.md` 和 `plugin_dev_tools.md` | 增加从脚手架到发布审批、灰度、回滚的完整 walkthrough |
| M16-7 | 完整配置参考文档 | 已有 `configuration.md` | 修正文件路径细节，补充多环境配置示例 |
| M16-8 | 历史文档归档与整理 | 已归档 archive | 保留归档索引，避免历史计划与当前计划混读 |

### 下一轮产品增强

| 编号 | 工作项 | 优先级 | 启动条件 |
|------|--------|--------|----------|
| F-1 | 插件风险分级 | 中 | M15 质量门禁稳定，插件安装/发布主流程有性能与安全基线 |
| F-2 | 策略引擎与决策 | 中 | 风险分级模型确定，已有审批流可复用 |
| F-3 | 审计与日志增强 | 中 | 当前审计查询/导出压测结果明确，能定位容量瓶颈 |
| F-4 | 插件市场与分发 | 低 | 插件安全、版本、灰度、回滚链路稳定后再启动 |
| F-5 | 插件能力扩展 | 低 | Upload/Link/Embed 三类形态的权限和风险模型完成 |
| F-6 | 安全基准与渗透测试 | 低 | P0 前端工程和 M15 性能基线完成后执行专项 |

---

**文档维护者**: 开发团队
**下次审查**: 2026-07-01
