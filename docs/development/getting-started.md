# 开发快速入门指南

## 1. 环境要求

| 依赖 | 版本要求 | 说明 |
|------|---------|------|
| Go | >= 1.24 | 后端运行时（`go.mod` 声明 `go 1.24` / `toolchain go1.24.1`） |
| Node.js | >= 16.x | 前端构建工具链（Vite 2.9 + Vue 3.4） |
| MySQL | 8.0+（生产）/ 可选（开发） | 生产存储后端；开发可使用内存模式跳过 |
| Git | 任意最新版 | 版本管理 |

## 2. 克隆项目

```bash
git clone <repo-url> skoll
cd skoll
```

## 3. 安装依赖

```bash
# 后端依赖
go mod download

# 前端依赖
cd web
npm install
cd ..
```

## 4. 项目目录结构（核心）

```
skoll/
├── cmd/skoll/          # 程序入口（main.go → newServerRunner → bootstrap.Run）
├── internal/
│   ├── bootstrap/      # 依赖注入组装（di.go）与Runner（run.go）
│   ├── domain/         # 领域实体（user/role/rbac/audit/system/shared）
│   ├── service/        # 业务服务层（接口+实现）
│   ├── repository/     # 仓储接口（纯契约）
│   ├── store/          # 存储实现（memory/mysql/postgres + gormrepo）
│   ├── handler/http/   # HTTP Handler（v1 路由 + 中间件）
│   ├── plugin/         # 插件运行时核心（Manager/Registry/Loader/Signature）
│   ├── event/          # 事件总线（内存/Redis）
│   └── cache/          # 缓存层（内存/Redis/Memcached）
├── pkg/                # 公共基础设施（config/errors/logging/security）
├── web/                # 前端项目（Vue 3 + Vite + Pinia）
├── plugins/            # 插件目录（demo / demo-backend / demo-frontend / ...）
├── configs/            # 配置文件（skoll.yaml）
├── migrations/         # 数据库迁移脚本（mysql/ / postgres/）
└── deploy/             # 部署资源（docker/ / compose/ / k8s/）
```

## 5. 本地启动

### 5.1 后端启动

**方式一：内存模式（无需数据库，开箱即用）**

```bash
# 设置环境变量使用内存存储
$env:SKOLL_STORE_MODE="memory"
$env:SKOLL_JWT_SECRET="dev-secret-local"

# 启动后端
go run ./cmd/skoll
```

服务默认监听 `:8080`，API 前缀为 `/skoll`。

**方式二：MySQL 模式（完整功能）**

```bash
# 先创建数据库
mysql -u root -p -e "CREATE DATABASE IF NOT EXISTS skoll CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"

# 确保 configs/skoll.yaml 中 DSN 正确，或设置环境变量
$env:SKOLL_STORE_MODE="mysql"
$env:SKOLL_STORE_DSN="root:root@tcp(127.0.0.1:3306)/skoll?charset=utf8mb4&parseTime=True&loc=Local"

go run ./cmd/skoll
```

### 5.2 前端启动

```bash
cd web
npm run dev
```

前端开发服务器默认运行在 `http://localhost:5173`，API 代理配置需指向后端 `http://localhost:8080`。

## 6. 健康检查验证

```bash
# 健康检查端点（无需认证）
curl http://localhost:8080/skoll/health

# Swagger UI（API 文档浏览器）
# 浏览器访问 http://localhost:8080/skoll/docs/swagger

# 认证测试（先通过 POST /skoll/v1/auth/login 获取 JWT Token）
curl -H "Authorization: Bearer <token>" http://localhost:8080/skoll/v1/auth/me
```

## 7. 常用命令速查

| 命令 | 说明 |
|------|------|
| `go run ./cmd/skoll` | 启动后端服务 |
| `go build -o skoll.exe ./cmd/skoll` | 编译可执行文件 |
| `go test ./...` | 运行全部测试 |
| `go vet ./...` | 静态代码检查 |
| `cd web && npm install` | 安装前端依赖 |
| `cd web && npm run dev` | 启动前端开发服务器 |
| `cd web && npm run build` | 构建前端生产包 |
| `cd web && npm run smoke:all` | 运行冒烟测试套件 |

## 8. 环境变量速查

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `SKOLL_SERVER_ADDRESS` | `:8080` | 监听地址 |
| `SKOLL_STORE_MODE` | `mysql` | 存储模式（memory/mysql/postgres） |
| `SKOLL_STORE_DSN` | 见配置文件 | 数据库连接串 |
| `SKOLL_JWT_SECRET` | `dev-secret-change-me` | JWT 签名密钥（**生产必须修改**） |
| `SKOLL_LOG_LEVEL` | `info` | 日志级别（debug/info/warn/error） |
| `SKOLL_DEV_PORTAL_ENABLED` | `false` | 开发者门户开关 |
| `SKOLL_EVENT_MODE` | `memory` | 事件总线模式（memory/redis） |

## 9. 下一步

- **API 文档**：访问 `http://localhost:8080/skoll/docs/swagger` 查看 Swagger UI
- **架构概览**：阅读 `docs/architecture/README.md`
- **配置参考**：阅读 `docs/configuration.md`
- **插件开发**：阅读 `docs/development/plugin-guide.md`
