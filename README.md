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

## 基础设施配置补充

- 统一 API 基础前缀：`SKOLL_API_BASE_PREFIX`（默认 `/skoll`，前端构建时注入，后端运行时读取）
- 前端页面基础路径：`SKOLL_WEB_BASE_PATH`（默认 `/skoll`，前端页面入口默认挂载到该路径）
- 兼容保留：`SKOLL_SERVER_API_PREFIX` 仍可作为回退配置
- 事件总线模式：`SKOLL_EVENT_MODE`（`memory` 或 `redis`）
- Redis Pub/Sub 地址：`SKOLL_EVENT_REDIS_ADDR`（当 `SKOLL_EVENT_MODE=redis` 时必填）
- Swagger UI：`GET {API_BASE_PREFIX}/docs/swagger`
- OpenAPI：`GET {API_BASE_PREFIX}/docs/openapi.yaml`（`servers.url` 会按当前前缀动态生成）

