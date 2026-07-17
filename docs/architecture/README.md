# Skoll 架构概览

> 版本: v2.0
> 日期: 2026-06-06

## 1. 整体架构

Skoll 采用分层架构设计，核心依赖方向为：

```
handler(HTTP) → service(业务) → repository(接口) ← store(实现)
```

`domain` 作为纯领域模型被上层依赖，不依赖任何外层。

### 1.1 分层架构图

```
┌──────────────────────────────────────────────────────┐
│                   cmd/skoll                          │
│         main.go → newServerRunner() → Run()          │
└──────────────────────┬───────────────────────────────┘
                       │
┌──────────────────────▼───────────────────────────────┐
│              internal/bootstrap                      │
│   RuntimeConfig(env) → buildDependencies() → Runner  │
│   装配: storeBundle + services + plugins + router    │
└──┬────────┬────────┬────────┬──────────┬────────────┘
   │        │        │        │          │
   ▼        ▼        ▼        ▼          ▼
┌──────┐┌──────┐┌──────┐┌──────┐┌──────────────┐
│handler││service││ event││plugin││  middleware   │
│http/v1││ user  ││ bus  ││ mgr  ││ auth/log/rate │
│user   ││ role  ││(mem/ ││      ││              │
│role   ││ rbac  ││redis)││      ││              │
│rbac   ││ audit ││      ││      ││              │
│audit  ││ system││      ││      ││              │
│system ││common ││      ││      ││              │
│plugin │└───┬───┘└──────┘└──────┘└──────────────┘
│       │    │
│       │    │         ┌──────────────────────────┐
│       │    │         │      adapter             │
│       │    │         │ DevRolloutExecutor       │
│       │    │         │ (灰度执行层接口抽象)       │
│       │    │         └──────────────────────────┘
│       │    ▼
│       │  ┌──────────┐     ┌─────────────────────┐
│       │  │repository │────▶│       store         │
│       │  │(接口层)   │     │ factory.go → Bundle │
│       │  │user/role/ │     │ memory│mysql│pg│ck  │
│       │  │rbac/audit/│     │   gormrepo (SQL)    │
│       │  │system/    │     │   clickhouse (时序) │
│       │  │plugin     │     └─────────────────────┘
│       │  └──────────┘
│       │
│       ▼
┌──────────┐    ┌──────────────────────────────┐
│  domain  │    │            pkg/               │
│  user    │    │ config│errors│logging│metrics │
│  role    │    │ security│utils│validator      │
│  rbac    │    │ version                       │
│  audit   │    └──────────────────────────────┘
│  system  │
│  shared  │
└──────────┘
```

---

## 2. 技术栈

### 2.1 后端

| 类别 | 技术选型 | 说明 |
|------|---------|------|
| 语言 | Go 1.24 | - |
| HTTP 框架 | Go 1.22 `net/http` 增强路由（`http.ServeMux`） | 使用 `METHOD /path/{param}` 模式匹配 |
| ORM | GORM 1.31 | `gormrepo/` 统一实现 |
| SQL Builder | goqu v9 | `internal/store/sql/goqu_builder.go` |
| 配置管理 | Viper | `pkg/config/loader.go`，支持 YAML + 环境变量 |
| 日志 | Zap | `pkg/logging/zap_adapter.go`，结构化日志 |
| 认证 | JWT (Bearer) | `pkg/security/jwt.go`，`middleware/auth.go` |
| 密码加密 | bcrypt | `pkg/security/` |
| 数据加密 | AES | `pkg/security/` |
| 指标 | Prometheus | `pkg/metrics/` |

### 2.2 存储

| 存储 | 用途 | 实现 |
|------|------|------|
| MySQL | 生产主库 | `internal/store/sql/mysql/` + `gormrepo/` |
| PostgreSQL | 生产备选 | `internal/store/sql/postgres/` + `gormrepo/` |
| ClickHouse | 审计/指标时序数据 | `internal/store/clickhouse/` |
| Memory | 开发/测试 | `internal/store/memory/` |

### 2.3 缓存

| 缓存 | 实现 |
|------|------|
| Redis | `internal/cache/redis/` |
| Memcached | `internal/cache/memcached/` |
| Local LRU | `internal/cache/local/` |

### 2.4 前端

| 类别 | 技术选型 |
|------|---------|
| 框架 | Vue 3.4 |
| 状态管理 | Pinia 2.1 |
| 路由 | Vue Router 4.4 |
| 构建 | Vite 2.9 |
| 样式 | SCSS |
| 图标 | Lucide Vue Next |
| 组件库 | 自定义 CSS（规划使用 Element Plus，未引入） |

---

## 3. 核心模块

### 3.1 domain（领域模型）

纯领域模型层，定义核心实体和值对象。

| 子包 | 核心实体 | 关键字段 |
|------|---------|---------|
| `user` | `User` + `Email`/`PasswordHash` 值对象 | ID, Account, Name, Email, Status(active/disabled), AuditMeta |
| `role` | `Role` | ID, Name, Key, Description, Permissions[], BuiltIn, AuditMeta |
| `rbac` | `Binding` / `PolicyRule` / `DataScope` | SubjectType(user/role), Effect(allow/deny), Scope(self/dept/dept_tree/all/custom) |
| `audit` | `Record` | ID, ActorID, Action, Resource, ResourceID, Detail(map), OccurredAt |
| `system` | `Setting` | ID, Key, Value, Encrypted, AuditMeta |
| `shared` | `ID`(string) / `TimeRange` / `AuditMeta` | 通用类型：CreatedAt/UpdatedAt 审计时间戳 |

### 3.2 repository（仓储接口）

纯契约层，定义数据访问接口。

| 接口 | 方法数 | 核心方法 |
|------|--------|---------|
| `UserRepository` | 6 | GetByID, GetByAccount, GetByEmail, List, Save, Delete |
| `RoleRepository` | 5 | GetByID, GetByKey, List, Save, Delete |
| `RBACRepository` | 5 | CreateBinding, DeleteBinding, ListBindingsBySubject, ListPolicyRulesByRoleID, ReplacePolicyRules |
| `AuditRepository` | 5 | Append, GetByID, ListByActor, ListByTimeRange, DeleteByTimeRange |
| `SystemRepository` | 5 | GetSettingByID, GetSettingByKey, ListSettings, SaveSetting, DeleteSetting |
| `PluginRepository` | 5+ | List, Get, Save, Delete |
| 通用 | `Tx` / `UnitOfWork` | 事务上下文与事务边界 |

### 3.3 store（存储实现）

实现了所有 `repository` 接口。通过 `factory.go` 的 `NewBundle()` 根据运行模式返回聚合了所有 repo 实现的 Bundle。

SQL 持久化统一使用 `internal/store/sql/gormrepo/`，采用 `*_model.go`（GORM 模型定义）+ `*_store.go`（CRUD 实现 + Model↔Domain 转换）同目录模式。

| 模式 | 实现路径 | 用途 |
|------|---------|------|
| `memory` | `internal/store/memory/` | 开发/测试，纯内存 map + 同步 |
| `mysql` | `internal/store/sql/mysql/` + `gormrepo/` | 生产主库 |
| `postgres` | `internal/store/sql/postgres/` + `gormrepo/` | 生产备选 |
| `clickhouse` | `internal/store/clickhouse/` | 审计/指标时序存储 |

### 3.4 service（业务服务）

| 服务 | 核心能力 |
|------|---------|
| `user.Service` | Create/CreateBatch/Get/List/Update/UpdateEmail/Delete/Disable，含事务 + 审计 |
| `role.Service` | Create/Get/List/Update/Delete/Grant/Revoke |
| `rbac.Service` | BindRole/UnbindBinding/ListBindings/SetRolePolicies/CheckPermission/ListBindingsByUser |
| `audit.Service` | Append/GetByID/ListByActor/ListByTimeRange/ClearByTimeRange |
| `system.Service` | Upsert/GetByKey/List/Reset |
| `common.TransactionManager` | 封装 `UnitOfWork` 提供事务编排 |

### 3.5 handler（HTTP 层）

- **框架**：Go 1.22 `net/http` 增强路由（`http.ServeMux`），使用 `METHOD /path/{param}` 模式匹配
- **路由注册**：`internal/handler/http/router.go` 集中注册 v1 路由 + 插件扩展路由 + Swagger/OpenAPI 文档端点
- **统一响应**：`{code, message, data}` JSON 格式，`code="ok"` 表示成功，`code="error"` 表示失败
- **OpenAPI**：通过 `go:embed` 内嵌 `openapi.yaml`，暴露 `/docs/openapi.yaml` 和 `/docs/swagger` 端点

#### v1 端点覆盖

| 模块 | 端点数 | 路径前缀 |
|------|--------|---------|
| 用户管理 | 10 | `/v1/users` |
| 角色管理 | 8 | `/v1/roles` |
| RBAC | 5 | `/v1/rbac` |
| 审计 | 5 | `/v1/audit` |
| 系统设置 | 4 | `/v1/system/settings` |
| 插件管理 | 15+ (含 DevPortal) | `/v1/plugins` |

### 3.6 event（事件总线）

支持两种后端实现：

| 实现 | 文件 | 适用场景 |
|------|------|---------|
| 内存 | `internal/event/inmemory_bus.go` | 单实例/开发 |
| Redis Pub/Sub | `internal/event/redis_bus.go` | 多实例/生产 |

### 3.7 plugin（插件系统）

- **核心组件**：Manager 接口（Install/Uninstall/Enable/Disable/List/Get）、RuntimeManager（文件加载+拓扑排序）、MemoryRegistry（扩展点注册）
- **内置插件**：`builtin-auth`（登录/JWT）、`builtin-logger`（日志查询）、`builtin-dashboard`（仪表盘）
- **DevPortal**：完整的插件开发生命周期 API（脚手架/清单管理/验证/打包/发布审批/灰度回滚真实化），由 `v1/plugin/handler.go` 提供
- **灰度执行层**：通过 `internal/adapter/` 的 `DevRolloutExecutor` 接口抽象灰度/回滚的真实执行，`MockDevRolloutExecutor` 为默认实现，预留对接真实网关的能力

### 3.8 adapter（外部集成适配层）

`internal/adapter/` 定义与外部系统集成的抽象接口，解耦业务逻辑与外部依赖。

| 组件 | 文件 | 说明 |
|------|------|------|
| `DevRolloutExecutor` 接口 | `internal/adapter/dev_rollout.go` | 定义 `ApplyRollout` / `ApplyRollback` / `VerifyRollout` 三个方法，抽象灰度流量切换的真实执行 |
| `MockDevRolloutExecutor` | `internal/adapter/dev_rollout_mock.go` | 默认实现，打印结构化日志模拟网关调用，为未来对接真实 API 网关/流量层预留接口 |

**adapter 模式**：`PluginHandler` 持有 `DevRolloutExecutor` 接口字段，`applyDevRollout` / `applyDevRollback` 通过接口调用执行器，实现业务层与执行层的解耦。默认注入 `MockDevRolloutExecutor`，生产环境可替换为对接真实网关的实现（如 Kong/Nginx/APISIX 等）。

### 3.9 cache（缓存层）

通过 `internal/cache/factory.go` 统一创建，支持 Redis、Memcached、Local LRU 三种后端。

---

## 4. 启动流程

1. `cmd/skoll/main.go`：捕获系统信号 → `newServerRunner()` → `runner.Run(ctx)`
2. `internal/bootstrap/run.go`：加载环境配置 → `buildDependencies()` → 启动 HTTP Server（优雅关闭）
3. `internal/bootstrap/di.go`（约 700 行核心装配）：
   - 初始化 Logger (Zap)
   - 创建 Store Bundle (MySQL/Postgres/Memory)
   - 构建 5 个 Service (user/role/rbac/audit/system)
   - 创建 PluginManager（扫描 `plugins/` 目录 + 内置插件）
   - 构建 HTTP Router（注册 v1 路由 + 插件扩展路由）
   - 装配中间件链 (Logger → RateLimit → Auth)

---

## 5. 中间件

| 中间件 | 实现文件 | 职责 |
|--------|---------|------|
| Auth | `middleware/auth.go` | JWT Bearer 验证，白名单放行 health/ready/plugins/auth |
| Logger | `middleware/logger.go` | 请求日志 |
| RateLimit | `middleware/ratelimit.go` | 请求限速 |

---

## 6. 前端架构

```
web/
├── src/
│   ├── components/     # Layout, Sidebar, Header
│   ├── stores/         # Pinia 状态管理
│   ├── router/         # Vue Router 路由
│   ├── views/          # 6 个页面
│   │   ├── Login/
│   │   ├── Dashboard/
│   │   ├── User/
│   │   ├── Role/
│   │   ├── Permission/
│   │   ├── Plugin/
│   │   └── Setting/
│   └── assets/         # 静态资源
├── package.json        # Vue 3.4 + Pinia 2.1 + Vue Router 4.4 + Vite 2.9
└── vite.config.ts
```

---

## 7. 部署

| 方式 | 路径 |
|------|------|
| Docker | `deploy/docker/Dockerfile` |
| Kubernetes | `deploy/k8s/` |
| Docker Compose | `deploy/compose/docker-compose.yaml` |

## 8. 相关文档

| 文档 | 用途 |
| --- | --- |
| [database.md](database.md) | 核心数据库与存储模式 |
| [pharma-oa-persistence.md](pharma-oa-persistence.md) | 医药 OA repository、schema、事务和 migration 基线 |
| [rbac.md](rbac.md) | 权限与数据范围模型 |
