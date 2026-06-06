# 部署运维文档

## 1. 概述

Skoll 是一个 Go 单体应用，编译为单个可执行文件，支持以下部署方式：

| 方式 | 适用场景 | 资源文件 |
|------|---------|---------|
| 直接运行 | 裸机 / VM | `go build -o skoll ./cmd/skoll` |
| Docker | 容器化单节点 | `deploy/docker/Dockerfile` |
| Docker Compose | 开发/测试环境 | `deploy/compose/docker-compose.yaml` |
| Kubernetes | 生产集群 | `deploy/k8s/deployment.yaml` + `service.yaml` |

## 2. Docker 部署

### 2.1 Dockerfile

`deploy/docker/Dockerfile` 使用多阶段构建：

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/skoll ./cmd/skoll

FROM alpine:3.20
RUN adduser -D -u 10001 app
USER app
WORKDIR /app
COPY --from=builder /out/skoll /app/skoll
EXPOSE 8080
ENTRYPOINT ["/app/skoll"]
```

### 2.2 构建与运行

```bash
# 构建镜像
docker build -f deploy/docker/Dockerfile -t skoll:latest .

# 运行容器（内存模式）
docker run -d -p 8080:8080 \
  -e SKOLL_STORE_MODE=memory \
  -e SKOLL_JWT_SECRET=my-production-secret \
  --name skoll skoll:latest

# 运行容器（MySQL 模式）
docker run -d -p 8080:8080 \
  -e SKOLL_STORE_MODE=mysql \
  -e SKOLL_STORE_DSN="user:pass@tcp(host.docker.internal:3306)/skoll?charset=utf8mb4&parseTime=True&loc=Local" \
  -e SKOLL_JWT_SECRET=my-production-secret \
  --name skoll skoll:latest
```

## 3. Docker Compose 部署

`deploy/compose/docker-compose.yaml`：

```yaml
version: "3.9"
services:
  skoll:
    build:
      context: ../..
      dockerfile: deploy/docker/Dockerfile
    image: skoll:local
    container_name: skoll
    ports:
      - "8080:8080"
    environment:
      SKOLL_SERVER_ADDRESS: ":8080"
      SKOLL_STORE_MODE: "memory"
      SKOLL_JWT_SECRET: "compose-dev-secret"
    restart: unless-stopped
```

```bash
# 启动
docker-compose -f deploy/compose/docker-compose.yaml up -d

# 停止
docker-compose -f deploy/compose/docker-compose.yaml down
```

## 4. Kubernetes 部署

### 4.1 Deployment

`deploy/k8s/deployment.yaml` 定义单副本 Deployment，包含健康检查探针：

```yaml
spec:
  replicas: 1
  template:
    spec:
      containers:
        - name: skoll
          image: skoll:latest
          ports:
            - containerPort: 8080
          env:
            - name: SKOLL_SERVER_ADDRESS
              value: ":8080"
            - name: SKOLL_STORE_MODE
              value: "memory"
            - name: SKOLL_JWT_SECRET
              value: "k8s-dev-secret"
          readinessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 2
            periodSeconds: 5
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
```

### 4.2 Service

`deploy/k8s/service.yaml` 定义 ClusterIP 类型服务：

```yaml
spec:
  type: ClusterIP
  ports:
    - name: http
      port: 80
      targetPort: 8080
```

### 4.3 部署命令

```bash
kubectl apply -f deploy/k8s/deployment.yaml
kubectl apply -f deploy/k8s/service.yaml
```

## 5. 环境变量完整参考表

所有配置项均支持 `SKOLL_` 前缀的环境变量覆盖（由 `pkg/config/loader.go` 中 Viper 处理）。环境变量名遵循：`SKOLL_<SECTION>_<KEY>`（全大写，下划线分隔）。

### 5.1 服务器配置

| 环境变量 | 配置键 | 默认值 | 说明 |
|---------|--------|--------|------|
| `SKOLL_SERVER_ADDRESS` | `server.address` | `:8080` | HTTP 监听地址 |
| `SKOLL_SERVER_PORT` | `server.port` | - | 监听端口（优先级高于 address） |
| `SKOLL_API_BASE_PREFIX` | `api.base_prefix` | `/skoll` | API 路径前缀 |
| `SKOLL_SERVER_API_PREFIX` | `server.api_prefix` | - | API 前缀（兼容旧版） |
| `SKOLL_SERVER_SHUTDOWN_TIMEOUT` | `server.shutdown_timeout` | `10s` | 优雅关闭超时时间 |

### 5.2 存储配置

| 环境变量 | 配置键 | 默认值 | 说明 |
|---------|--------|--------|------|
| `SKOLL_STORE_MODE` | `store.mode` | `mysql` | 存储模式：memory/mysql/postgres |
| `SKOLL_STORE_DSN` | `store.dsn` | `root:root@tcp(127.0.0.1:3306)/skoll?...` | 数据库连接串 |

### 5.3 缓存配置

| 环境变量 | 配置键 | 默认值 | 说明 |
|---------|--------|--------|------|
| `SKOLL_CACHE_MODE` | `cache.mode` | `memory` | 缓存模式：memory/redis/memcached |
| `SKOLL_CACHE_REDIS_ADDR` | `cache.redis_addr` | `127.0.0.1:6379` | Redis 地址 |
| `SKOLL_CACHE_MEMCACHED_ADDR` | `cache.memcached_addr` | `127.0.0.1:11211` | Memcached 地址 |
| `SKOLL_CACHE_LOCAL_SIZE` | `cache.local_size` | `4096` | 本地缓存容量（条目数） |

### 5.4 事件总线配置

| 环境变量 | 配置键 | 默认值 | 说明 |
|---------|--------|--------|------|
| `SKOLL_EVENT_MODE` | `event.mode` | `memory` | 事件总线模式：memory/redis |
| `SKOLL_EVENT_REDIS_ADDR` | `event.redis_addr` | - | Redis 事件总线地址 |
| `SKOLL_EVENT_CHANNEL_PREFIX` | `event.channel_prefix` | `skoll.events` | 事件频道前缀 |

### 5.5 安全配置

| 环境变量 | 配置键 | 默认值 | 说明 |
|---------|--------|--------|------|
| `SKOLL_SECURITY_JWT_SECRET` | `security.jwt_secret` | `dev-secret-change-me` | JWT 签名密钥（**生产必须修改**） |

### 5.6 日志配置

| 环境变量 | 配置键 | 默认值 | 说明 |
|---------|--------|--------|------|
| `SKOLL_LOG_LEVEL` | `log.level` | `info` | 日志级别：debug/info/warn/error |
| `SKOLL_LOG_DIR` | `log.dir` | `log` | 日志文件目录 |
| `SKOLL_LOG_FILE` | `log.file` | - | 统一日志文件名 |
| `SKOLL_LOG_PLUGIN_PER_FILE` | `log.plugin_per_file` | `false` | 是否按插件分日志文件 |

### 5.7 开发者配置

| 环境变量 | 配置键 | 默认值 | 说明 |
|---------|--------|--------|------|
| `SKOLL_DEV_PORTAL_ENABLED` | `dev.portal_enabled` | `false` | 开启开发者门户 |
| `SKOLL_DEV_PLUGINS_ROOT` | `dev.plugins_root` | `plugins` | 插件根目录（支持 `;` / `,` 分隔多目录） |

## 6. 存储模式选择

| 环境 | 推荐 | 理由 |
|------|------|------|
| 本地开发 | `memory` | 零依赖，快速迭代 |
| CI/CD 流水线 | `memory` | 环境纯净，无需外部服务 |
| 测试/预发布 | `mysql` | 与生产一致的数据行为 |
| 生产环境 | `mysql` / `postgres` | 数据持久化 + 备份恢复 |

## 7. 健康检查端点

| 端点 | 方法 | 认证 | 说明 |
|------|------|------|------|
| `/skoll/health` | GET | 否 | 服务健康状态（liveness/readiness probe） |
| `/skoll/ready` | GET | 否 | 服务就绪状态 |

健康检查端点位于认证中间件白名单中，无需 JWT Token 即可访问。

### K8s 探针配置示例

```yaml
readinessProbe:
  httpGet:
    path: /skoll/health
    port: 8080
  initialDelaySeconds: 2
  periodSeconds: 5
livenessProbe:
  httpGet:
    path: /skoll/health
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10
```

## 8. 启动流程

`cmd/skoll/main.go` → `newServerRunner()` → `bootstrap.NewRunnerFromEnv()`：

1. 加载配置（`pkg/config.Load()`：环境变量 > 配置文件 > 默认值）
2. 初始化 Logger（Zap）
3. 创建 Store Bundle（根据 `store.mode` 选择 memory/mysql/postgres）
4. 构建 5 个 Service（user/role/rbac/audit/system）
5. 创建 PluginManager（扫描 `plugins/` 目录 + 内置插件）
6. 构建 HTTP Router（注册 v1 路由 + 中间件链）
7. 启动 HTTP Server（优雅关闭）
