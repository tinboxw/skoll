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

1. 内存模式（默认）

```powershell
go run ./cmd/skoll
```

2. MySQL 模式

```powershell
$env:SKOLL_STORE_MODE="mysql"
$env:SKOLL_STORE_DSN="user:pass@tcp(127.0.0.1:3306)/skoll?charset=utf8mb4&parseTime=True&loc=Local"
go run ./cmd/skoll
```

3. PostgreSQL 模式

```powershell
$env:SKOLL_STORE_MODE="postgres"
$env:SKOLL_STORE_DSN="host=127.0.0.1 user=postgres password=postgres dbname=skoll port=5432 sslmode=disable"
go run ./cmd/skoll
```

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
