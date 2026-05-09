# 重构后续里程碑计划（M1-M7）

## 说明
本计划基于 [docs/refactor.md](../refactor.md) 的分阶段实施章节整理，当前默认 M0 已完成，以下为后续执行里程碑清单。

## 当前状态
- 已完成：M0 项目初始化与基础设施搭建。
- 进行方式：按里程碑顺序推进，每阶段完成后执行门禁测试并记录结果。

## 执行任务清单（按顺序推进）
- [x] T0 优先恢复历史删除代码并校验工作区无 `D` 状态。✓
- [x] T1 完成 M1 核心领域模型与仓储设计。✓
- [x] T2 完成 M2 多数据库存储层实现。✓
- [x] T3 完成 M3 缓存层实现。✓
- [x] T4 完成 M4 业务服务层开发。✓
- [x] T5 完成 M5 API 接口层开发。✓
- [ ] T6 完成 M6 事件总线与异步处理。
- [ ] T7 完成 M7 集成测试与优化。
- [ ] T8 每阶段完成后执行并记录门禁（`go fmt ./...`、`go test ./...`、`go test -race ./...`）。

## 里程碑总览

| 里程碑 | 周期 | 核心目标 | 关键交付 |
| --- | --- | --- | --- |
| M1 | 第3-5周 | 核心领域模型与仓储设计 | domain 实体与 repository 接口 |
| M2 | 第6-10周 | 多数据库存储层实现 | memory/mysql/postgres/clickhouse + migrations |
| M3 | 第11-13周 | 缓存层实现 | redis/memcached/local + cache factory |
| M4 | 第14-17周 | 业务服务层开发 | user/role/rbac/audit 服务 + 事务管理 |
| M5 | 第18-20周 | API 接口层开发 | HTTP v1、中间件、CLI |
| M6 | 第21-22周 | 事件总线与异步处理 | bus/publisher/subscriber/events |
| M7 | 第23-24周 | 集成测试与优化 | 覆盖率、性能基线、部署配置与文档 |

## M1 核心领域模型与仓储设计
### 目标
建立用户、角色、RBAC、审计、系统等核心领域模型，并定义统一仓储契约。

### 目录范围
- internal/domain/user
- internal/domain/role
- internal/domain/rbac
- internal/domain/audit
- internal/domain/system
- internal/domain/shared
- internal/repository

### 退出标准
- 领域对象与规则具备最小可用实现。
- 仓储接口齐备（含事务接口）。
- `go fmt ./...`、`go test ./...`、`go test -race ./...` 通过。

## M2 存储层实现（多数据库）
### 目标
实现 memory/mysql/postgres/clickhouse 存储适配与存储工厂。

### 目录范围
- internal/store/memory
- internal/store/sql/mysql
- internal/store/sql/postgres
- internal/store/clickhouse
- internal/store/factory.go
- migrations/mysql
- migrations/postgres

### 退出标准
- 各存储实现可通过接口契约测试。
- 迁移脚本可执行。
- 相关门禁通过（含 race）。

## M3 缓存层实现
### 目标
实现 Redis、Memcached、本地缓存并统一工厂出口。

### 目录范围
- internal/cache/redis
- internal/cache/memcached
- internal/cache/local
- internal/cache/factory.go

### 退出标准
- 三类缓存实现具备可调用能力。
- 缓存接口与工厂可切换。
- 门禁测试通过。

## M4 业务服务层开发
### 目标
构建用户、角色、RBAC、审计等服务层，完成事务管理能力。

### 目录范围
- internal/service/user
- internal/service/role
- internal/service/rbac
- internal/service/audit
- internal/service/common/transaction_manager.go

### 退出标准
- 主要业务服务具备最小闭环。
- 服务层单元测试达到计划覆盖目标。
- 门禁测试通过。

## M5 API 接口层开发
### 目标
提供 HTTP v1、中间件与 CLI 接口能力，形成对外可调用入口。

### 目录范围
- internal/handler/http/v1
- internal/handler/http/router.go
- internal/handler/http/response.go
- internal/handler/middleware
- internal/handler/cli

### 退出标准
- API 路由、统一响应、中间件链可工作。
- 关键接口具备基础测试。
- 门禁测试通过。

## M6 事件总线与异步处理
### 目标
实现事件发布订阅机制并打通核心领域事件。

### 目录范围
- internal/event/bus.go
- internal/event/publisher.go
- internal/event/subscriber.go
- internal/event/events

### 退出标准
- 事件机制具备可发布/订阅能力。
- 核心事件结构定义完成。
- 门禁测试通过。

## M7 集成测试与优化
### 目标
完成整体测试、性能基线、文档与部署配置收口。

### 目录范围
- tests/unit
- tests/integration
- tests/benchmark
- deploy/docker
- deploy/k8s
- deploy/compose
- docs/api、docs/development、docs/user、docs/architecture

### 退出标准
- 覆盖率达到计划目标（整体 >= 80%）。
- 性能基线报告可复现。
- Docker/K8s/Compose 配置可用。
- 文档完整并与实现一致。

## 阶段门禁（每个里程碑都执行）
```bash
go fmt ./...
go test ./...
go test -race ./...
```

## 里程碑推进记录模板
- 里程碑：M<编号>
- 完成项：
- 未完成项：
- 风险与阻塞：
- 已执行命令：
- 结果摘要：
- 下一里程碑进入条件：
