# Skoll API 文档

## 1. 基地址

| 环境 | 基地址 |
|------|--------|
| 本地开发 | `http://localhost:8080/skoll` |
| 可配置 | 通过 `SKOLL_API_BASE_PREFIX` 环境变量或 `server.api_prefix` 配置项修改前缀 |

默认 API 前缀为 `/skoll`（由 `pkg/config/loader.go` 中 `DefaultAPIBasePrefix` 常量定义）。

## 2. 认证方式

**JWT Bearer Token**（`internal/handler/middleware/auth.go`）

所有 API 请求（除白名单外）必须在 HTTP Header 中携带 JWT Token：

```
Authorization: Bearer <jwt_token>
```

**白名单路径**（无需认证）：

| 路径 | 说明 |
|------|------|
| `/skoll/health` | 健康检查 |
| `/skoll/ready` | 就绪检查 |
| `/skoll/v1/plugins` | 插件列表（公开） |
| `/skoll/v1/auth` | 认证相关（登录/登出） |

**获取 Token**：通过 `POST /skoll/v1/auth/login` 登录接口获取。

JWT 密钥通过 `SKOLL_JWT_SECRET` 环境变量或 `security.jwt_secret` 配置项设置，默认值为 `dev-secret-change-me`（仅限开发环境）。

## 3. 通用响应格式

所有 API 返回统一 JSON 格式（`internal/handler/http/response.go`）：

```json
{
  "code": "ok",
  "message": "ok",
  "data": {}
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `code` | string | 状态码：`"ok"` 表示成功，`"error"` 表示失败 |
| `message` | string | 描述信息 |
| `data` | any | 业务数据（成功时返回，失败时省略） |

**HTTP 状态码约定**：
- `200`：成功
- `400`：参数错误
- `401`：未认证（Token 缺失/无效）
- `403`：无权限
- `404`：资源不存在
- `409`：资源冲突
- `500`：服务器内部错误

## 4. 端点分类列表

路由注册见 `internal/handler/http/router.go`，所有 v1 端点前缀为 `/skoll/v1`。

### 4.1 用户管理（User）

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/skoll/v1/users` | 创建用户 |
| `POST` | `/skoll/v1/users/batch` | 批量创建用户 |
| `GET` | `/skoll/v1/users/{id}` | 获取用户详情 |
| `GET` | `/skoll/v1/users` | 用户列表 |
| `PUT` | `/skoll/v1/users/{id}` | 更新用户信息 |
| `PUT` | `/skoll/v1/users/{id}/email` | 更新用户邮箱 |
| `PUT` | `/skoll/v1/users/{id}/disable` | 禁用用户 |
| `DELETE` | `/skoll/v1/users/{id}` | 删除用户 |
| `PUT` | `/skoll/v1/users/{id}/roles` | 分配用户角色 |

### 4.2 角色管理（Role）

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/skoll/v1/roles` | 创建角色 |
| `GET` | `/skoll/v1/roles/{id}` | 获取角色详情 |
| `GET` | `/skoll/v1/roles` | 角色列表 |
| `PUT` | `/skoll/v1/roles/{id}` | 更新角色 |
| `DELETE` | `/skoll/v1/roles/{id}` | 删除角色 |
| `PUT` | `/skoll/v1/roles/{id}/permissions/grant` | 授予权限 |
| `PUT` | `/skoll/v1/roles/{id}/permissions/revoke` | 撤销权限 |
| `GET` | `/skoll/v1/roles/{id}/users` | 查询拥有该角色的用户列表 |

### 4.3 RBAC 权限管理

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/skoll/v1/rbac/bindings` | 创建绑定（用户/角色 → 角色） |
| `DELETE` | `/skoll/v1/rbac/bindings/{id}` | 删除绑定 |
| `GET` | `/skoll/v1/rbac/bindings` | 查询绑定列表 |
| `PUT` | `/skoll/v1/rbac/roles/{id}/policies` | 设置角色策略规则 |
| `POST` | `/skoll/v1/rbac/check` | 权限检查 |
| `GET` | `/skoll/v1/rbac/users/{id}/bindings` | 查询用户所有绑定 |

### 4.4 审计日志（Audit）

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/skoll/v1/audit/records` | 追加审计记录 |
| `GET` | `/skoll/v1/audit/records/{id}` | 获取审计记录详情 |
| `GET` | `/skoll/v1/audit/records` | 审计记录列表（支持时间范围） |
| `GET` | `/skoll/v1/audit/records/actor/{id}` | 按操作者查询审计 |
| `DELETE` | `/skoll/v1/audit/records` | 按时间范围清理审计 |
| `GET` | `/skoll/v1/audit/records/export` | 导出审计记录 |

### 4.5 系统设置（System）

| 方法 | 路径 | 说明 |
|------|------|------|
| `PUT` | `/skoll/v1/system/settings/{key}` | 创建或更新单个设置 |
| `GET` | `/skoll/v1/system/settings/{key}` | 获取单个设置 |
| `GET` | `/skoll/v1/system/settings` | 设置列表 |
| `PUT` | `/skoll/v1/system/settings` | 批量更新设置 |
| `POST` | `/skoll/v1/system/settings/reset` | 重置设置 |

### 4.6 插件管理（Plugin）

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/skoll/v1/plugins/install` | 安装插件 |
| `POST` | `/skoll/v1/plugins/{id}/enable` | 启用插件 |
| `POST` | `/skoll/v1/plugins/{id}/disable` | 禁用插件 |
| `DELETE` | `/skoll/v1/plugins/{id}` | 卸载插件 |
| `GET` | `/skoll/v1/plugins` | 插件列表 |
| `GET` | `/skoll/v1/plugins/{id}` | 获取插件详情 |
| `GET` | `/skoll/v1/plugins/{id}/config` | 获取插件配置 |
| `PUT` | `/skoll/v1/plugins/{id}/config` | 更新插件配置 |
| `POST` | `/skoll/v1/plugins/{id}/validate` | 校验插件 |

### 4.7 插件开发者门户（Dev Portal）

需通过 `SKOLL_DEV_PORTAL_ENABLED=true` 开启，仅 `super_admin` 角色可调用。

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/skoll/v1/plugins/dev/config` | 获取 Dev Portal 配置 |
| `GET` | `/skoll/v1/plugins/dev/manifest` | 获取插件 manifest |
| `POST` | `/skoll/v1/plugins/dev/manifest/validate` | 校验 manifest |
| `PUT` | `/skoll/v1/plugins/dev/manifest` | 更新 manifest |
| `POST` | `/skoll/v1/plugins/dev/projects` | 列出开发中插件工程 |
| `POST` | `/skoll/v1/plugins/dev/remove` | 删除开发中插件目录 |
| `POST` | `/skoll/v1/plugins/dev/package` | 打包插件制品 |
| `POST` | `/skoll/v1/plugins/dev/pipeline` | 一键流水线（校验+打包） |
| `POST` | `/skoll/v1/plugins/dev/scaffold` | 生成插件骨架 |
| `POST` | `/skoll/v1/plugins/dev/validate-all` | 批量校验插件 |
| `POST` | `/skoll/v1/plugins/dev/rollout` | 灰度发布 |
| `POST` | `/skoll/v1/plugins/dev/rollback` | 回滚灰度 |
| `GET` | `/skoll/v1/plugins/dev/rollout-tasks` | 灰度任务列表 |
| `POST` | `/skoll/v1/plugins/dev/release-orders` | 创建发布单 |
| `GET` | `/skoll/v1/plugins/dev/release-orders` | 发布单列表 |
| `POST` | `.../release-orders/{id}/approve` | 审批通过 |
| `POST` | `.../release-orders/{id}/reject` | 审批驳回 |
| `POST` | `.../release-orders/{id}/execute` | 执行发布 |
| `GET` | `/skoll/v1/plugins/dev/release-tasks` | 发布任务列表 |

## 5. Swagger UI

**访问地址**：`http://localhost:8080/skoll/docs/swagger`

内嵌的 OpenAPI 3.0 文档由 `internal/handler/http/openapi.yaml` 提供（由 router 动态注入 API 前缀），覆盖所有 v1 端点的完整 Schema 定义。

OpenAPI YAML 原始文件也可直接获取：`GET /skoll/docs/openapi.yaml`
