# 架构概览（M7）

## 分层结构
- `cmd/skoll`: 进程入口、信号处理。
- `internal/bootstrap`: 配置加载、依赖装配、HTTP 服务器生命周期。
- `internal/domain`: 领域实体与规则。
- `internal/repository`: 仓储契约。
- `internal/store`: memory/mysql/postgres/clickhouse 实现。
- `internal/cache`: local/redis/memcached 缓存实现与工厂。
- `internal/service`: 业务服务编排与事务管理。
- `internal/handler`: HTTP v1、CLI、中间件。
- `internal/event`: 事件总线、发布订阅、事件模型。

## 请求链路
1. `cmd/skoll/main.go` 调用 bootstrap runner 启动。
2. `internal/bootstrap/di.go` 装配 store/service/handler。
3. `internal/handler/http/router.go` 注册 v1 路由。
4. `internal/handler/middleware/*` 执行认证、日志、限流。
5. 服务层调用仓储与缓存，必要时触发事件总线。

## 事件机制
- 事件总线采用 `internal/event/inmemory_bus.go`。
- 发布接口：`event.Publisher`。
- 订阅接口：`event.Subscriber`。
- 当前核心事件模型位于 `internal/event/events/user_events.go`。

## 可部署形态
- Docker: `deploy/docker/Dockerfile`
- Kubernetes: `deploy/k8s/deployment.yaml`, `deploy/k8s/service.yaml`
- Compose: `deploy/compose/docker-compose.yaml`

