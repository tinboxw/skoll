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

## 当前实现状态（M8）
- 已实现插件生命周期管理（install/enable/disable/uninstall/list/get）。
- 已实现基于 `plugin.yaml` 的元数据加载。
- 已实现依赖解析与循环依赖检测。
- 已实现基础权限校验器与内存扩展点注册器。
- 已补充单元测试：`manager_test.go`、`loader_test.go`、`resolver_test.go`。

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

当前卸载策略为 `retain`、`archive`、`drop`；回滚策略为 `manual`、`automatic`、`none`。`drop` 会被安装预检标记为高风险/破坏性策略。

## API Contract
业务插件如果暴露后端 API，必须在 `api.routes` 中声明当前契约。路径必须位于 `/v1/plugins/{pluginId}/api/` 下，每条 route 必须绑定权限；审计动作使用 `module.resource.action` 格式。安装预检会展示 route、permission、audit action 和 OpenAPI path 预览。

```yaml
api:
  routes:
    - method: GET
      path: /v1/plugins/pharma-oa/api/products
      summary: List pharma products
      permission: pharma-oa.products.read
      audit_action: pharma_oa.products.read
```

## 后续待补充实现
- [ ] 完善插件包校验（签名/哈希）与版本约束匹配。
- [ ] 对接 CLI 与 HTTP 管理接口。
- [ ] 增加内置插件示例的扩展点注册实现。

