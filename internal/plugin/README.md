# internal/plugin

## 功能说明
插件管理核心层。

## 文件组织规范
- 采用小写与下划线命名，按职责拆分文件。
- 接口定义与实现分离，避免单文件过大。
- 变更时同步补充测试与文档。

## 当前规划文件
- manager.go
- registry.go
- loader.go
- resolver.go
- permission.go
- types.go

## 当前实现状态
- 已实现插件生命周期管理（install/enable/disable/uninstall/list/get）。
- 已实现基于 `plugin.yaml` 的元数据加载。
- 已实现依赖解析与循环依赖检测。
- 已实现 manifest API 权限解析、参数化路由匹配与统一插件 API 分发。
- 已实现进程内后端惰性工厂、外部服务代理、健康检查和生命周期释放。
- 已实现事务迁移、业务事件投递与服务监督。
- 已发布进程内 Go 插件事务与可信数据范围端口，使用规范见 `docs/development/plugin-host-services.md`。

## Data Manifest
业务插件如果需要自有数据表，必须在 `plugin.yaml` 中声明 `data:` 段。表名必须位于插件 namespace 下，不能使用 `sk_` 系统前缀；安装预检会展示表、索引、migration 与卸载策略风险。

```yaml
data:
  namespace: pharma_oa
  migration_version: v1.0.0
  migration_directory: migrations
  uninstall_policy: retain
  rollback_policy: manual
  tables:
    - name: pharma_oa_products
      description: Pharma product master data
      primary_key: id
      columns: id, code, name, status
      indexes: unique idx_pharma_oa_products_code(code); idx_pharma_oa_products_status(status)
```

迁移文件必须成对命名为 `{version}_{name}.up.sql` 和 `{version}_{name}.down.sql`。插件启用与升级会在服务启动、路由发布之前执行全部 pending migration，并在主数据库的 `sk_plugin_migrations` 账本中记录版本与 SHA-256；SQL 或账本任一步失败都会回滚本批次并阻止插件启用，重复执行不会再次运行已入账版本。已执行的 up 文件禁止修改，checksum 漂移会直接阻断生命周期。

当前卸载策略为 `retain`、`archive`、`drop`：`retain` 与 `archive` 保留插件数据和迁移账本，`drop` 按版本倒序执行全部 down migration 并删除账本记录。回滚策略为 `manual`、`automatic`、`none`，自动 downgrade 仅允许 `automatic`。`drop` 会被安装预检标记为高风险/破坏性策略。真实迁移只通过已配置主数据库的插件生命周期执行；CLI `plan` 仅做静态预览，不提供脱离事务账本的 apply/rollback 路径。

## API Contract
业务插件如果暴露后端 API，必须在 `api.routes` 中声明当前契约。路径必须位于 `/v1/plugins/{pluginId}/api/` 下，每条 route 必须绑定权限；审计动作使用 `module.resource.action` 格式。安装预检会展示 route、permission、audit action 和 OpenAPI path 预览。

```yaml
api:
  routes:
    - method: GET
      path: /v1/plugins/pharma_oa/api/products
      summary: List pharma products
      permission: pharma_oa.products.read
      audit_action: pharma_oa.products.read
```

运行时只接受 manifest 已声明的 method/path。仓库内置 Go 插件通过 `RegisterInProcessBackend` 注册惰性 `http.Handler` 工厂；外部插件通过 `service_base_url` 接入。Disable、Uninstall、Reload 和运行时关闭会释放进程内实例，禁用状态不会继续提供 API。业务插件不能在主应用 router 或依赖容器中注册专用业务路由。

仓库内置 Vue 插件的集成路由集中定义在 `web/src/plugins/integrated-routes.ts`，并由后端插件同步结果动态挂载。插件禁用、卸载或同步失败时会删除对应菜单状态和路由。

## Business Events
业务插件如果需要订阅平台业务事件，必须在 `events.subscriptions` 中声明事件名、处理器和重试策略。当前事件名使用小写字母、数字、点、下划线或短横线；内置医药 OA 场景至少覆盖 `approval-completed`、`inbound-completed` 和 `qualification-expiring`。安装预检会展示订阅清单和 retry policy，启用插件时会把订阅导入扩展注册表。

```yaml
events:
  subscriptions:
    - name: approval-completed
      handler: onApprovalCompleted
      retry_policy: standard
    - name: qualification-expiring
      handler: onQualificationExpiring
      retry_policy: standard
```

## 后续待补充实现
- [ ] 完善插件包校验（签名/哈希）与版本约束匹配。
- [ ] 对接 CLI 与 HTTP 管理接口。

