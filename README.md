# Skoll

Skoll v2 是一个按分层架构重新实现的 Go 框架基线，强调职责清晰、低耦合高内聚、可测试与可扩展。

## 架构分层

- `cmd/skoll`：进程入口
- `internal/bootstrap`：配置与运行时装配
- `pkg/config`：配置加载、校验与变更通知
- `pkg/logging`：结构化日志抽象
- `pkg/errors`：统一错误模型与 HTTP 映射
- `pkg/security`：JWT、密码哈希与加密工具

## 存储层说明

- `internal/store/sql/gormrepo/model`：MySQL/PostgreSQL 共享模型定义与映射（每表独立文件）。
- `internal/store/sql/gormrepo/store`：MySQL/PostgreSQL 共享仓储实现（按 user/role/system/rbac 分文件）。
- `internal/store/sql/gormrepo/stores.go`：根包门面导出，保持上层调用稳定。
- `internal/store/sql/mysql`：MySQL 方言入口（DSN 解析、连接、迁移、键规范化）。
- `internal/store/sql/postgres`：PostgreSQL 方言入口（连接、迁移、键规范化）。
- `internal/store/sql/transaction.go`：SQL 模式下的事务边界实现（`UnitOfWork`）。

## 快速启动

默认读取优先级：`SKOLL_* 环境变量 > 配置文件 > 内置默认值`。

默认会自动查找以下配置文件（按顺序）：
- `./skoll.yaml`
- `./skoll.yml`
- `./config/skoll.yaml`
- `./config/skoll.yml`
- `./configs/skoll.yaml`
- `./configs/skoll.yml`

仓库默认配置为 `mysql + memory cache`（见 `configs/skoll.yaml`）。
当 `store.mode=mysql` 时，审计记录会持久化到 MySQL，后端重启后不会丢失。

推荐本地一键启动（强制 MySQL 模式并重启前后端）：

```powershell
powershell -ExecutionPolicy Bypass -File ./scripts/dev-up.ps1 -ForceRestart
```

`dev-up.ps1` 默认会在启动后自动执行连通性检查：
- 前端健康检查：`/skoll/health`
- 登录获取 token（默认账号 `admin`）
- 开发门户接口检查：`/skoll/v1/plugins/dev/config`、`/skoll/v1/plugins/dev/permission-catalog`

常用参数示例：

```powershell
# 跳过启动后检查
powershell -ExecutionPolicy Bypass -File ./scripts/dev-up.ps1 -ForceRestart -SkipPostChecks

# 检查失败时让脚本返回非 0（用于 CI/门禁）
powershell -ExecutionPolicy Bypass -File ./scripts/dev-up.ps1 -ForceRestart -FailOnCheckError

# 指定检查登录账号
powershell -ExecutionPolicy Bypass -File ./scripts/dev-up.ps1 -ForceRestart -AdminAccount admin -AdminPassword "Admin@123456"
```

停止本地前后端监听：

```powershell
powershell -ExecutionPolicy Bypass -File ./scripts/dev-down.ps1
```

可直接选择以下组合（你提到的 1/2/3/4）：
1. `configs/skoll.mysql-memory.yaml`
2. `configs/skoll.mysql-redis.yaml`
3. `configs/skoll.postgres-memory.yaml`
4. `configs/skoll.postgres-redis.yaml`

1. 配置文件模式（推荐本地开发）

```powershell
go run ./cmd/skoll
```

2. 指定配置文件路径

```powershell
$env:SKOLL_CONFIG_FILE="D:/workspace/3rdsrc/tinbox/skoll/configs/skoll.yaml"
go run ./cmd/skoll
```

3. 使用组合 1（mysql + memory cache）

```powershell
$env:SKOLL_CONFIG_FILE="./configs/skoll.mysql-memory.yaml"
go run ./cmd/skoll
```

4. 使用组合 2（mysql + redis cache）

```powershell
$env:SKOLL_CONFIG_FILE="./configs/skoll.mysql-redis.yaml"
go run ./cmd/skoll
```

5. 使用组合 3（postgres + memory cache）

```powershell
$env:SKOLL_CONFIG_FILE="./configs/skoll.postgres-memory.yaml"
go run ./cmd/skoll
```

6. 使用组合 4（postgres + redis cache）

```powershell
$env:SKOLL_CONFIG_FILE="./configs/skoll.postgres-redis.yaml"
go run ./cmd/skoll
```

7. 按环境变量覆盖任意组合（例如覆盖 Redis 地址）

```powershell
$env:SKOLL_CONFIG_FILE="./configs/skoll.mysql-redis.yaml"
$env:SKOLL_CACHE_REDIS_ADDR="10.0.0.5:6379"
go run ./cmd/skoll
```

8. 统一日志输出到单文件（推荐）

```powershell
$env:SKOLL_LOG_DIR="log"
$env:SKOLL_LOG_FILE="skoll.log"
go run ./cmd/skoll
```

说明：设置 `SKOLL_LOG_FILE` 后，应用日志与插件操作日志会统一写入 `log/skoll.log`。

9. 默认 stdout（不指定日志文件）

```powershell
$env:SKOLL_LOG_FILE=""
$env:SKOLL_LOG_PLUGIN_PER_FILE="false"
go run ./cmd/skoll
```

说明：默认会输出到 stdout，插件日志带 `plugin_id` 字段，便于区分来源。

10. 插件按插件ID输出到各自文件

```powershell
$env:SKOLL_LOG_FILE=""
$env:SKOLL_LOG_PLUGIN_PER_FILE="true"
$env:SKOLL_LOG_DIR="log"
go run ./cmd/skoll
```

说明：插件日志将写入 `log/<pluginId>.log`。

## 默认登录账号

- 默认内置管理员账号：`admin`
- 默认密码：`Admin@123456`
- 内置种子账号密码长度策略为至少 8 位，因此不支持 `admin/admin`。
- 如果你使用持久化 MySQL 数据，服务启动时会自动校准内置账号到上述默认密码（建议首次登录后立即修改）。

## 测试

```powershell
go fmt ./...
go test ./...
go test -race ./...

# 可选：本地 MySQL 集成验证
$env:SKOLL_TEST_MYSQL_DSN="root:root@tcp(127.0.0.1:3306)/skoll?parseTime=true"
go test ./...
```

## 设计文档

- `docs/planning/FRAMEWORK_REDESIGN_BLUEPRINT_2026-05-09.md`
- `docs/planning/FRAMEWORK_REDESIGN_EXECUTION_PLAN_2026-05-09.md`

## API

- `GET /skoll/health`
- `GET /skoll/v1/plugins`
- `POST /skoll/v1/plugins/install`（请求体：`{ "path": "plugins/demo" }`）
- `POST /skoll/v1/plugins/validate`（请求体：`{ "path": "plugins/demo" }`）
- `POST /skoll/v1/plugins/{id}/enable`
- `POST /skoll/v1/plugins/{id}/disable`
- `DELETE /skoll/v1/plugins/{id}`
- `GET /skoll/v1/plugins/{id}/debug`
- `GET /skoll/v1/plugins/{id}/logs`
- `POST /skoll/v1/plugins/dev/scaffold`（仅 `super_admin`，请求体：`{ "pluginsRoot": "plugins", "pluginId": "crm-order", "pluginName": "CRM Order", "appId": "crm", "mode": "workspace|repository" }`）
- `POST /skoll/v1/plugins/dev/validate-all`（仅 `super_admin`，请求体：`{ "pluginsRoot": "plugins" }`）
- `GET /skoll/v1/plugins/dev/config`（仅 `super_admin`）
- `GET /skoll/v1/plugins/dev/manifest?pluginsRoot=plugins&pluginId=crm-order`（仅 `super_admin`）
- `POST /skoll/v1/plugins/dev/manifest/validate`（仅 `super_admin`，请求体：`{ "pluginsRoot": "plugins", "pluginId": "crm-order", "manifest": "...yaml..." }`）
- `PUT /skoll/v1/plugins/dev/manifest`（仅 `super_admin`，请求体：`{ "pluginsRoot": "plugins", "pluginId": "crm-order", "manifest": "...yaml..." }`）
- `POST /skoll/v1/plugins/dev/projects`（仅 `super_admin`，请求体：`{ "pluginsRoot": "plugins" }`）
- `POST /skoll/v1/plugins/dev/remove`（仅 `super_admin`，请求体：`{ "pluginsRoot": "plugins", "pluginId": "crm-order", "removeFiles": true }`）
- `POST /skoll/v1/plugins/dev/package`（仅 `super_admin`，请求体：`{ "pluginsRoot": "plugins", "pluginId": "crm-order", "outputDir": "plugins/_dist" }`）
- `POST /skoll/v1/plugins/dev/pipeline`（仅 `super_admin`，请求体：`{ "pluginsRoot": "plugins", "pluginId": "crm-order", "outputDir": "plugins/_dist" }`）
- `POST /skoll/v1/plugins/dev/rollout`（仅 `super_admin`，请求体：`{ "pluginId": "crm-order", "rolloutPercent": 20 }`）
- `POST /skoll/v1/plugins/dev/rollback`（仅 `super_admin`，请求体：`{ "pluginId": "crm-order" }`）
- `POST /skoll/v1/plugins/dev/release-orders`（仅 `super_admin`，请求体：`{ "pluginsRoot": "plugins", "pluginId": "crm-order", "releaseVersion": "1.2.0", "changelog": "..." }`）
- `GET /skoll/v1/plugins/dev/release-orders?pluginsRoot=plugins&pluginId=crm-order`（仅 `super_admin`）
- `POST /skoll/v1/plugins/dev/release-orders/{orderId}/approve`（仅 `super_admin`，请求体：`{ "pluginsRoot": "plugins", "pluginId": "crm-order", "comment": "approved" }`）
- `POST /skoll/v1/plugins/dev/release-orders/{orderId}/reject`（仅 `super_admin`，请求体：`{ "pluginsRoot": "plugins", "pluginId": "crm-order", "comment": "need more checks" }`）

开发者页面插件（可安装/卸载）：
- 插件目录：`plugins/developer-portal`
- 安装：`POST /skoll/v1/plugins/install`，请求体 `{"path":"plugins/developer-portal"}`
- 访问方式：在插件管理中启用后，点击该插件的「访问」进入开发者页面
- 生产建议：无需开发能力时可直接卸载该插件

Dev Portal 返回契约（`data` 字段）：
- scaffold：`{ operation, status, pluginsRoot, pluginId, pluginName, pluginDir, mode }`
- validate-all：`{ operation, status, pluginsRoot, summary:{total,valid,invalid}, results:[...] }`
- config：`{ enabled, defaultRoot, allowedRoots[] }`
- manifest_get/manifest_put/manifest_validate：`{ operation, status, pluginsRoot, pluginId, pluginDir, manifestPath, manifest, validation, error? }`
- projects：`{ operation, status, pluginsRoot, projects:[{ pluginId,name,version,path,mode,installed,enabled,previewUrl,buildHint,packageHint,publishHint,lastUpdatedAt }] }`
- remove：`{ operation, status, pluginsRoot, pluginId, pluginDir, uninstalled, filesRemoved }`
- package：`{ operation, status, pluginsRoot, pluginId, pluginDir, artifactPath }`
- pipeline：`{ operation, status, pluginsRoot, pluginId, pluginDir, startedAt, finishedAt, steps:[{name,status,message?,artifactPath?,durationMs}] }`
- rollout/rollback：`{ operation, status, pluginId, rolloutPercent, persisted, message? }`
- release order create/list/approve/reject：`{ operation, status, order:{ orderId, pluginId, releaseVersion, orderStatus, changelog?, createdBy, createdAt, approvedBy?, approvedAt?, rejectedBy?, rejectedAt?, reviewComment? }, orders?[] }`

## 基础设施配置补充

- 统一 API 基础前缀：`SKOLL_API_BASE_PREFIX`（默认 `/skoll`，前端构建时注入，后端运行时读取）
- 前端页面基础路径：`SKOLL_WEB_BASE_PATH`（默认 `/skoll`，前端页面入口默认挂载到该路径）
- 兼容保留：`SKOLL_SERVER_API_PREFIX` 仍可作为回退配置
- Dev Portal 开关：`SKOLL_DEV_PORTAL_ENABLED`（默认 `false`，开启后注册 `/v1/plugins/dev/*`）
- Dev Portal 根目录白名单：`SKOLL_DEV_PLUGINS_ROOT`（默认 `plugins`，支持 `;` 或 `,` 分隔多个目录；`pluginsRoot` 必须命中其中一个目录）
- 事件总线模式：`SKOLL_EVENT_MODE`（`memory` 或 `redis`）
- Redis Pub/Sub 地址：`SKOLL_EVENT_REDIS_ADDR`（当 `SKOLL_EVENT_MODE=redis` 时必填）
- Swagger UI：`GET {API_BASE_PREFIX}/docs/swagger`
- OpenAPI：`GET {API_BASE_PREFIX}/docs/openapi.yaml`（`servers.url` 会按当前前缀动态生成）

