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

- `GET /health`
- `GET /v1/plugins`
- `POST /v1/plugins/install`（请求体：`{ "path": "plugins/demo" }`）
- `POST /v1/plugins/validate`（请求体：`{ "path": "plugins/demo" }`）
- `POST /v1/plugins/{id}/enable`
- `POST /v1/plugins/{id}/disable`
- `DELETE /v1/plugins/{id}`
- `GET /v1/plugins/{id}/debug`
- `GET /v1/plugins/{id}/logs`
