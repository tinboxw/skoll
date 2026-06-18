# internal/domain/audit

## 功能说明

审计领域模型定义可查询的审计记录、事件分类和 action 命名规则。M2 的目标是把操作日志、登录日志、错误日志、插件生命周期日志和安全事件统一到一套审计事件模型中。

## EventType

`EventType` 只接受五类值：

| 类型 | 说明 |
|---|---|
| `operation` | 用户或管理员的业务操作 |
| `login` | 登录成功、登录失败、会话事件 |
| `error` | 已处理或未处理错误，包含 trace/request 上下文 |
| `plugin` | 插件安装、启用、禁用、发布、灰度、回滚 |
| `security` | 权限拒绝、风险配置、签名失败等安全事件 |

使用 `ParseEventType` 解析外部输入；使用 `EventType.Validate` 校验领域值。

## AuditAction

新审计 action 必须使用 `module.resource.action` 格式：

```text
^[a-z][a-z0-9_-]{0,63}\.[a-z][a-z0-9_-]{0,63}\.[a-z][a-z0-9_-]{0,63}$
```

示例：

| Action | 说明 |
|---|---|
| `user.account.create` | 创建用户账号 |
| `rbac.role.grant` | 给角色授权 |
| `plugin.release.rollout` | 插件灰度发布 |
| `file.object.upload` | 上传文件对象 |
| `menu.node.update` | 更新菜单节点 |

旧式 action（例如 `login_failed`、`plugin_catalog_import`、`update_email`）不再作为新增审计 action 使用。后续 M2 任务会把分散写入点迁移到统一命名。

## 文件组织规范

- `entity.go` 保留当前 `Record` 实体，后续统一事件结构按独立文件补充。
- `event_type.go` 定义事件分类。
- `action.go` 定义 action 命名规则。
- 新字段或值对象必须同步补测试。
