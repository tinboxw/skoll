# 插件 API 与 OpenAPI 契约

> 默认语言：简体中文。English: [plugin-api-contract.en.md](plugin-api-contract.en.md)

## 适用范围

本文说明当前业务插件如何声明后端 API，以及 Skoll 如何把已启用插件的路由聚合到 `/skoll/docs/openapi.yaml`。仅使用当前 `plugin.yaml` 格式，不提供旧格式或旧路径兼容。

## Manifest 声明

每条业务路由必须位于 `/v1/plugins/{pluginId}/api/`，并声明可授权的权限键。需要审计成功或失败结果时，同时声明三段式 `audit_action`。

```yaml
id: pharma_oa
version: 1.0.0
service_base_url: http://127.0.0.1:18090
service_health_url: http://127.0.0.1:18090/health
api:
  routes:
    - method: GET
      path: /v1/plugins/pharma_oa/api/employees
      summary: List Pharma OA employees
      permission: pharma_oa.employee.read
      audit_action: pharma_oa.employee.read
```

安装预检会校验 method、path、permission、audit action 和来源。无效声明不会进入路由权限注册表或聚合 OpenAPI。

`service_base_url` 是插件后端的 HTTP(S) 基地址，必须与 `service_health_url` 成对声明。Skoll 仅执行 `api.routes` 中明确声明的 method/path，并将声明路径追加到该基地址。例如，上述请求会发送到 `http://127.0.0.1:18090/v1/plugins/pharma_oa/api/employees`。如果基地址带路径前缀，该前缀会保留。

## 运行时行为

| 插件状态 | 业务路由执行 | 聚合 OpenAPI |
| --- | --- | --- |
| Installed | 不可执行 | 不展示 |
| Enabled | 按 JWT 和声明权限执行 | 展示 |
| Disabled | 拒绝执行 | 移除 |
| Uninstalled | 不可执行 | 移除 |

插件页面和静态资源使用独立的公开边界；`/v1/plugins/{pluginId}/api/*` 始终属于受保护业务 API。缺少路由解析器、权限声明或权限检查器时，服务端默认拒绝请求。

通过认证与授权后，Skoll 会把请求 method、声明路径、query、body 和普通请求头发送到插件后端，并把后端 status、响应头和 body 返回给调用方。后端不可达或缺少执行配置时不会返回占位成功。

启用、禁用、卸载和元数据重载与路由执行共享同一个生命周期闸门。启用操作返回时，声明路由、扩展快照和权限快照已经同时生效，无需重启主机；禁用或卸载会先等待在途请求完成，返回后新的业务请求、扩展查询和权限解析均不可再访问该插件。再次启用沿用同一 Router 实例重新发布当前声明。

声明外部服务的插件同时受服务监督器管理。启用会在 5 秒预算内完成首次 readiness，失败时插件保持关闭；进入 Ready 后每 5 秒复用健康契约监测服务，异常退出或探针失败进入 Failed。禁用、卸载和主机关机先关闭业务流量，再在 5 秒内优雅停止；超时会记录稳定失败 code 并执行强制释放。所有状态变化写入插件服务审计，报告不包含服务 URL 或底层网络错误。未声明 `service_base_url` 的插件状态为 `not_applicable`，不会创建服务句柄。

Skoll 使用无用户凭据的 GET 请求探测 `service_health_url`，2xx 表示健康，重定向、非 2xx、超时和连接错误均表示不健康。不健康插件不能接收业务流量。业务请求复用最长 5 秒的健康结果；`GET /ready` 与 `GET /v1/plugins/{pluginId}/health` 强制刷新探针。报告只包含插件 ID、稳定状态码、时间、延迟和 HTTP 状态，不包含 URL、token 或底层错误文本。

| HTTP | Code | 含义 |
| --- | --- | --- |
| 502 | `plugin_route_unavailable` | 主机没有可用的插件路由执行器 |
| 502 | `plugin_backend_unavailable` | 插件后端连接或执行失败 |
| 503 | `plugin_backend_not_configured` | 已声明业务路由但未配置有效后端地址 |
| 503 | `plugin_not_enabled` | 插件当前未启用 |
| 503 | `plugin_unhealthy` | 插件健康探针未通过，业务流量已阻断 |

## 聚合元数据

已启用插件的每个 OpenAPI operation 包含：

| 字段 | 含义 |
| --- | --- |
| `security: [{ bearerAuth: [] }]` | 请求必须携带 Skoll JWT |
| `x-skoll-plugin-id` | 插件 ID |
| `x-skoll-plugin-source` | 权限与路由来源 |
| `x-skoll-permission` | 服务端实际检查的权限键 |
| `x-skoll-audit-action` | 路由声明的审计动作，可选 |

基线规范的 `components.securitySchemes.bearerAuth` 定义为 HTTP Bearer JWT。运行时聚合只增加已启用插件的 paths，不修改磁盘上的两份基线规范。

## 查看与验证

```powershell
# 启动 Skoll 后查看当前启用插件的聚合规范
Invoke-WebRequest http://127.0.0.1:8080/skoll/docs/openapi.yaml | Select-Object -ExpandProperty Content

# 验证嵌入规范、公开规范和聚合安全元数据
go test ./internal/handler/http -run 'Test(RouterOpenAPI|OpenAPIContractFilesStayInSync|EmbeddedOpenAPIReferencesResolve)' -count=1
```

调用插件业务 API 时使用登录接口签发的 access token，并为当前用户或角色授予 operation 中 `x-skoll-permission` 指定的权限。菜单可见性不能替代后端授权。

## 变更影响

新增、删除或修改 `api.routes` 时，必须同步权限目录、审计 action、相关测试和开发文档。本契约不新增 migration、seed 或前端页面路径。
