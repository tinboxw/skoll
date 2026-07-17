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

## Action Catalog

首版 action catalog 覆盖 M2 需要迁移的核心模块。新增 action 先进入本表，再接入 service/middleware 写入路径。

| 模块 | Action | EventType | Result | Risk | 触发场景 |
|---|---|---|---|---|---|
| user | `user.account.create` | `operation` | `success`/`failure` | `medium` | 创建用户账号 |
| user | `user.account.update` | `operation` | `success`/`failure` | `medium` | 更新用户资料、邮箱或状态 |
| user | `user.account.disable` | `operation` | `success`/`failure` | `high` | 禁用用户 |
| role | `role.role.create` | `operation` | `success`/`failure` | `medium` | 创建角色 |
| role | `role.role.update` | `operation` | `success`/`failure` | `medium` | 更新角色信息 |
| role | `role.role.delete` | `operation` | `success`/`failure` | `high` | 删除角色 |
| rbac | `rbac.role.grant` | `security` | `success`/`failure` | `high` | 给角色授予权限 |
| rbac | `rbac.role.revoke` | `security` | `success`/`failure` | `high` | 撤销角色权限 |
| rbac | `rbac.policy.update` | `security` | `success`/`failure` | `high` | 更新角色策略 |
| plugin | `plugin.lifecycle.install` | `plugin` | `success`/`failure` | `high` | 安装插件 |
| plugin | `plugin.lifecycle.enable` | `plugin` | `success`/`failure` | `high` | 启用插件 |
| plugin | `plugin.lifecycle.disable` | `plugin` | `success`/`failure` | `high` | 禁用插件 |
| plugin | `plugin.lifecycle.uninstall` | `plugin` | `success`/`failure` | `high` | 卸载插件 |
| plugin | `plugin.release.rollout` | `plugin` | `success`/`failure` | `critical` | 插件灰度发布 |
| plugin | `plugin.release.rollback` | `plugin` | `success`/`failure` | `critical` | 插件回滚 |
| menu | `menu.node.create` | `operation` | `success`/`failure` | `medium` | 新增菜单节点 |
| menu | `menu.node.update` | `operation` | `success`/`failure` | `medium` | 保存、排序、显隐菜单节点 |
| menu | `menu.node.delete` | `operation` | `success`/`failure` | `medium` | 删除菜单节点 |
| file | `file.object.upload` | `operation` | `success`/`failure` | `medium` | 上传文件对象 |
| file | `file.object.download` | `operation` | `success`/`failure` | `low` | 下载文件对象 |
| file | `file.object.delete` | `operation` | `success`/`failure` | `high` | 删除文件对象 |
| system | `system.setting.update` | `operation` | `success`/`failure` | `high` | 更新系统设置 |
| system | `system.dictionary.update` | `operation` | `success`/`failure` | `medium` | 更新字典项 |
| system | `system.security.deny` | `security` | `denied` | `high` | 权限拒绝、签名失败或风险配置拒绝 |

登录和错误事件使用独立模块：

| 模块 | Action | EventType | Result | Risk | 触发场景 |
|---|---|---|---|---|---|
| auth | `auth.session.login` | `login` | `success` | `low` | 登录成功 |
| auth | `auth.session.login_failed` | `login` | `failure` | `medium` | 登录失败 |
| error | `error.request.handled` | `error` | `failure` | `medium` | handler 返回可捕获错误 |
| error | `error.request.panic` | `error` | `failure` | `critical` | panic 或未处理错误 |

## 文件组织规范

- `entity.go` 保留当前 `Record` 实体，后续统一事件结构按独立文件补充。
- `event_type.go` 定义事件分类。
- `action.go` 定义 action 命名规则。
- 新字段或值对象必须同步补测试。
