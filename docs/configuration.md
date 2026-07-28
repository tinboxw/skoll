# 配置参考文档

## 1. 配置文件格式

Skoll 使用 YAML 格式配置文件，通过 Viper 加载。配置优先级（由高到低）：

```
SKOLL_* 环境变量 > 配置文件 > 内置默认值
```

### 1.1 配置文件搜索路径

启动时自动按以下顺序查找配置文件（`pkg/config/loader.go` → `defaultConfigCandidates`）：

```
skoll.yaml
skoll.yml
config/skoll.yaml
config/skoll.yml
configs/skoll.yaml
configs/skoll.yml
```

也可通过 `SKOLL_CONFIG_FILE` 环境变量指定自定义配置文件路径。

### 1.2 环境变量映射规则

- 前缀：`SKOLL_`
- 分隔符：`.` 替换为 `_`
- 示例：`SKOLL_SERVER_ADDRESS` → `server.address`
- API 前缀变量：`SKOLL_API_BASE_PREFIX` 映射到 `api.base_prefix`

## 2. 完整配置项参考

### 2.1 server（服务器）

| 键名 | 类型 | 默认值 | 环境变量 | 说明 |
|------|------|--------|---------|------|
| `server.address` | string | `:8080` | `SKOLL_SERVER_ADDRESS` | HTTP 监听地址，格式 `:8080` 或 `127.0.0.1:8080` |
| `server.port` | string | - | `SKOLL_SERVER_PORT` | 监听端口，设置后覆盖 `address` 中的端口部分 |
| `api.base_prefix` | string | `/skoll` | `SKOLL_API_BASE_PREFIX` | API 路径前缀，所有接口均挂载在此前缀之下 |
| `server.shutdown_timeout` | string | `10s` | `SKOLL_SERVER_SHUTDOWN_TIMEOUT` | 优雅关闭超时时间（Go Duration 格式，如 `30s`、`1m`） |

### 2.2 store（存储）

| 键名 | 类型 | 默认值 | 环境变量 | 说明 |
|------|------|--------|---------|------|
| `store.mode` | string | `mysql` | `SKOLL_STORE_MODE` | 存储模式。可选值：`memory` / `mysql` / `postgres` |
| `store.dsn` | string | `root:root@tcp(127.0.0.1:3306)/skoll?charset=utf8mb4&parseTime=True&loc=Local` | `SKOLL_STORE_DSN` | 数据库连接串（GORM DSN 格式） |

**存储模式行为**：

| 模式 | 数据持久化 | 适用场景 |
|------|----------|---------|
| `memory` | 否（进程内存储） | 开发测试 |
| `mysql` | 是（GORM + MySQL） | 生产主推 |
| `postgres` | 是（GORM + PostgreSQL + ClickHouse 审计） | 生产备选 |

### 2.3 cache（缓存）

| 键名 | 类型 | 默认值 | 环境变量 | 说明 |
|------|------|--------|---------|------|
| `cache.mode` | string | `memory` | `SKOLL_CACHE_MODE` | 缓存模式。可选值：`memory` / `redis` / `memcached` |
| `cache.redis_addr` | string | `127.0.0.1:6379` | `SKOLL_CACHE_REDIS_ADDR` | Redis 服务器地址（仅 redis 模式生效） |
| `cache.memcached_addr` | string | `127.0.0.1:11211` | `SKOLL_CACHE_MEMCACHED_ADDR` | Memcached 服务器地址（仅 memcached 模式生效） |
| `cache.local_size` | int | `4096` | `SKOLL_CACHE_LOCAL_SIZE` | 本地 LRU 缓存最大条目数（仅 memory 模式生效） |

### 2.4 event（事件总线）

| 键名 | 类型 | 默认值 | 环境变量 | 说明 |
|------|------|--------|---------|------|
| `event.mode` | string | `memory` | `SKOLL_EVENT_MODE` | 事件总线模式。可选值：`memory` / `redis` |
| `event.redis_addr` | string | - | `SKOLL_EVENT_REDIS_ADDR` | Redis 事件总线地址（仅 redis 模式生效） |
| `event.channel_prefix` | string | `skoll.events` | `SKOLL_EVENT_CHANNEL_PREFIX` | Redis Pub/Sub 频道前缀 |

**事件总线模式行为**：

| 模式 | 实现文件 | 说明 |
|------|---------|------|
| `memory` | `internal/event/inmemory_bus.go` | 进程内发布/订阅，单实例 |
| `redis` | `internal/event/redis_bus.go` | Redis Pub/Sub，支持多实例广播 |

### 2.5 plugin.quota（插件资源治理）

每项配置都是当前运行时的强制边界。速率、突发量、并发数、容量、字节数和超时必须为正值；任一值无效时，Skoll 启动失败，不会使用兼容默认值或关闭治理。

下表中的资源名可取 `request`、`host_call`、`query`、`mutation`、`event`、`job`、`export`、`storage`、`process`。每类资源均使用三个键：

| 键名规则 | 类型 | 环境变量规则 | 说明 |
|------|------|---------|------|
| `plugin.quota.<resource>_rate` | float | `SKOLL_PLUGIN_QUOTA_<RESOURCE>_RATE` | 每插件每秒补充令牌数 |
| `plugin.quota.<resource>_burst` | int | `SKOLL_PLUGIN_QUOTA_<RESOURCE>_BURST` | 每插件令牌桶容量 |
| `plugin.quota.<resource>_concurrency` | int | `SKOLL_PLUGIN_QUOTA_<RESOURCE>_CONCURRENCY` | 每插件最大活动并发 |

| 资源 | 默认速率 | 默认突发量 | 默认并发 |
|------|---------:|-----------:|---------:|
| `request` | 50 | 100 | 32 |
| `host_call` | 100 | 200 | 64 |
| `query` | 40 | 80 | 16 |
| `mutation` | 20 | 40 | 8 |
| `event` | 20 | 40 | 8 |
| `job` | 10 | 20 | 4 |
| `export` | 2 | 4 | 2 |
| `storage` | 10 | 20 | 4 |
| `process` | 1 | 2 | 1 |

| 键名 | 类型 | 默认值 | 环境变量 | 说明 |
|------|------|--------|---------|------|
| `plugin.quota.max_pending_events` | int | `100` | `SKOLL_PLUGIN_QUOTA_MAX_PENDING_EVENTS` | 每插件最大待投递事件数 |
| `plugin.quota.max_pending_jobs` | int | `100` | `SKOLL_PLUGIN_QUOTA_MAX_PENDING_JOBS` | 每插件最大待执行任务数 |
| `plugin.quota.max_file_bytes` | int64 | `16777216` | `SKOLL_PLUGIN_QUOTA_MAX_FILE_BYTES` | 单个插件文件最大字节数 |
| `plugin.quota.max_storage_bytes` | int64 | `536870912` | `SKOLL_PLUGIN_QUOTA_MAX_STORAGE_BYTES` | 每插件文件总存储最大字节数 |
| `plugin.quota.max_request_bytes` | int64 | `8388608` | `SKOLL_PLUGIN_QUOTA_MAX_REQUEST_BYTES` | 插件业务请求体最大字节数 |
| `plugin.quota.max_response_bytes` | int64 | `16777216` | `SKOLL_PLUGIN_QUOTA_MAX_RESPONSE_BYTES` | 插件业务响应体最大字节数 |
| `plugin.quota.request_timeout` | duration | `30s` | `SKOLL_PLUGIN_QUOTA_REQUEST_TIMEOUT` | 插件业务请求处理上限 |
| `plugin.quota.process_memory_bytes` | int64 | `536870912` | `SKOLL_PLUGIN_QUOTA_PROCESS_MEMORY_BYTES` | 每插件托管进程内存上限 |
| `plugin.quota.process_max_procs` | int | `2` | `SKOLL_PLUGIN_QUOTA_PROCESS_MAX_PROCS` | 每插件进程数与 Go 调度并行度上限 |

### 2.6 security（安全）

| 键名 | 类型 | 默认值 | 环境变量 | 说明 |
|------|------|--------|---------|------|
| `security.jwt_secret` | string | `dev-secret-change-me` | `SKOLL_SECURITY_JWT_SECRET` | JWT HMAC 签名密钥。**生产环境必须修改为强随机字符串** |

认证中间件还读取以下安全环境变量：

| 环境变量 | 类型 | 默认值 | 说明 |
|---------|------|--------|------|
| `SKOLL_AUTH_ENABLED` | bool | `true` | 是否为普通受保护 API 启用 JWT 认证 |
| `SKOLL_AUTH_SKIP_PATHS` | string | - | 额外免认证路径，多个路径使用逗号分隔 |

H3 数据范围浏览器矩阵可在隔离验收环境设置 `SKOLL_SCOPE_MATRIX_FIXTURES=true`。该开关默认关闭；启用后会创建 `scope_self`、`scope_department`、`scope_tree`、`scope_all`、`scope_denied` 五个测试账号、对应组织树/角色/策略和四条客户数据，统一测试密码为 `Scope@123456`。生产环境不得启用该开关。

插件公共面仅限 `GET /<api-prefix>/v1/plugins/{id}/page` 与 `GET /<api-prefix>/v1/plugins/{id}/assets/*`。插件业务 API `/<api-prefix>/v1/plugins/{id}/api/*` 始终要求 JWT，并按照已启用插件 manifest 的 `api_contract.routes[].permission` 执行 RBAC；`SKOLL_AUTH_ENABLED=false` 和 `SKOLL_AUTH_SKIP_PATHS` 都不能跳过该边界。路由缺少权限声明或权限解析器不可用时，系统默认拒绝并写入安全审计。

### 2.7 log（日志）

| 键名 | 类型 | 默认值 | 环境变量 | 说明 |
|------|------|--------|---------|------|
| `log.level` | string | `info` | `SKOLL_LOG_LEVEL` | 日志级别。可选值：`debug` / `info` / `warn` / `error` |
| `log.dir` | string | `log` | `SKOLL_LOG_DIR` | 日志文件输出目录 |
| `log.file` | string | - | `SKOLL_LOG_FILE` | 统一日志文件名。设置后所有日志（含插件日志）写入此文件。为空时输出到 stdout |
| `log.plugin_per_file` | bool | `false` | `SKOLL_LOG_PLUGIN_PER_FILE` | 是否按插件分日志文件。为 `true` 且 `log.file` 为空时，插件日志写入 `log/<pluginId>.log` |

**日志行为矩阵**：

| `log.file` | `log.plugin_per_file` | 主进程日志 | 插件日志 |
|-----------|----------------------|----------|---------|
| 设置 | 任意 | 写入该文件 | 写入该文件 |
| 空 | `false` | stdout | stdout（带 `plugin_id` 字段） |
| 空 | `true` | stdout | `log/<pluginId>.log` |

### 2.8 dev（开发者工具）

| 键名 | 类型 | 默认值 | 环境变量 | 说明 |
|------|------|--------|---------|------|
| `dev.portal_enabled` | bool | `false` | `SKOLL_DEV_PORTAL_ENABLED` | 开启开发者门户（Dev Portal）。开启后 `/v1/plugins/dev/*` 路由可用 |
| `dev.plugins_root` | string | `plugins` | `SKOLL_DEV_PLUGINS_ROOT` | 插件根目录。支持 `;` 或 `,` 分隔多个目录（如 `plugins;extensions`） |

**开发者门户安全约束**：
- 仅 `super_admin` 角色可调用 Dev Portal API
- `pluginsRoot` 参数必须在白名单目录中（`dev.plugins_root` 配置的目录列表）
- 关闭时 `/v1/plugins/dev/*` 返回 404

## 3. 配置示例

### 3.1 最小开发配置（内存模式）

```yaml
# configs/skoll.yaml
server:
  address: ":8080"

store:
  mode: "memory"

security:
  jwt_secret: "dev-secret-local"

log:
  level: "debug"
```

### 3.2 生产配置（MySQL + Redis）

```yaml
server:
  address: ":8080"
  shutdown_timeout: "30s"

store:
  mode: "mysql"
  dsn: "skoll_user:StrongPassword@tcp(db-primary.internal:3306)/skoll?charset=utf8mb4&parseTime=True&loc=Local"

cache:
  mode: "redis"
  redis_addr: "redis-cluster.internal:6379"
  local_size: 8192

event:
  mode: "redis"
  redis_addr: "redis-cluster.internal:6379"
  channel_prefix: "skoll.events"

security:
  jwt_secret: "<set-through-secret-manager>"

log:
  level: "info"
  dir: "/var/log/skoll"
  file: "skoll.log"

dev:
  portal_enabled: false
```

### 3.3 环境变量覆盖示例

```bash
# 完全通过环境变量配置（无需配置文件）
export SKOLL_SERVER_ADDRESS=":9090"
export SKOLL_STORE_MODE="postgres"
export SKOLL_STORE_DSN="host=db.internal user=skoll password=xxx dbname=skoll sslmode=require"
export SKOLL_CACHE_MODE="redis"
export SKOLL_CACHE_REDIS_ADDR="cache.internal:6379"
export SKOLL_EVENT_MODE="redis"
export SKOLL_EVENT_REDIS_ADDR="cache.internal:6379"
export SKOLL_SECURITY_JWT_SECRET="$(openssl rand -hex 32)"
export SKOLL_LOG_LEVEL="warn"
export SKOLL_LOG_DIR="/var/log/skoll"
export SKOLL_DEV_PORTAL_ENABLED="false"

go run ./cmd/skoll
```

## 4. 配置加载流程

`pkg/config/loader.go` → `Load()` 函数：

1. 创建 Viper 实例，设置环境变量前缀 `SKOLL_`、分隔符替换规则
2. 注册所有默认值（`v.SetDefault(...)`）
3. 搜索并加载配置文件（按 `defaultConfigCandidates` 顺序，或 `SKOLL_CONFIG_FILE` 指定）
4. 绑定特定环境变量映射（如 `SKOLL_API_BASE_PREFIX` → `api.base_prefix`）
5. 解析特殊字段：`shutdown_timeout`（Go Duration）、`server.port`（覆盖 address）
6. 标准化字段：`store.mode`、`cache.mode`、`event.mode` 统一为小写
7. 返回 `AppConfig` 结构体

## 5. 运行时配置值获取

```go
import "github.com/tinboxw/skoll/pkg/config"

cfg, err := config.Load()
if err != nil {
    log.Fatal(err)
}
// cfg.Server.Address       → ":8080"
// cfg.Store.Mode           → "mysql"
// cfg.Security.JWTSecret   → "dev-secret-change-me"
```
