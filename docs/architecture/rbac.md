# RBAC 权限模型说明

## 1. 模型概述

Skoll 使用 **RBAC（Role-Based Access Control）** 权限模型，支持基于角色 + 策略规则的多层权限控制。

### 1.1 核心概念链

```
Subject（主体） → Binding（绑定） → Role（角色） → PolicyRule（策略规则）
                   │                                     │
                   └── DataScope（数据范围）              └── Effect（allow/deny）
```

层级关系：
- **Subject**：权限主体，可以是用户（`user`）或角色（`role`），实现角色继承
- **Binding**：绑定关系，将主体与角色关联，同时限定数据范围
- **Role**：角色定义，包含名称、键名和权限列表
- **PolicyRule**：策略规则，定义"谁对什么资源能做什么操作"以及效果（允许/拒绝）

## 2. 领域模型定义

### 2.1 Subject（主体）

定义于 `internal/domain/rbac/entity.go`：

```go
type SubjectType string

const (
    SubjectUser SubjectType = "user"  // 用户主体
    SubjectRole SubjectType = "role"  // 角色主体（支持角色嵌套）
)
```

### 2.2 Binding（绑定）

```go
type Binding struct {
    ID          shared.ID      // 绑定唯一标识
    SubjectType SubjectType    // 主体类型（user / role）
    SubjectID   shared.ID      // 主体 ID
    RoleID      shared.ID      // 被绑定的角色 ID
    Scope       DataScope      // 数据范围限制
    Meta        shared.AuditMeta
}
```

### 2.3 Role（角色）

定义于 `internal/domain/role/entity.go`：

```go
type Role struct {
    ID          shared.ID
    Name        string      // 角色显示名称
    Key         string      // 角色唯一键名（小写）
    Description string
    Permissions []string    // 权限标识列表（如 menu.read、user.create）
    BuiltIn     bool        // 是否内置角色（内置角色不可删除）
    Meta        shared.AuditMeta
}
```

**内置角色**：系统提供的内置角色（`BuiltIn=true`）受保护，不可被删除。典型内置角色包括：
- `super_admin`：超级管理员，拥有全部权限
- `admin`：普通管理员
- `user`：普通用户

### 2.4 PolicyRule（策略规则）

定义于 `internal/domain/rbac/policy.go`：

```go
type PolicyRule struct {
    Resource string      // 资源标识（支持通配符 *）
    Action   string      // 操作标识（支持通配符 *）
    Effect   Effect      // 效果：allow / deny
    Scope    DataScope   // 数据范围
}
```

**Resource/Action 通配符匹配**（`wildcardMatch`）：
- `*` 匹配任意资源/操作
- `user:*` 匹配 `user` 资源的所有操作
- `*:read` 匹配所有资源的 `read` 操作
- 精确匹配优先于通配符匹配

## 3. Effect（allow/deny）逻辑

定义于 `internal/domain/rbac/policy.go`：

```go
type Effect string

const (
    EffectAllow Effect = "allow"  // 允许
    EffectDeny  Effect = "deny"   // 拒绝
)
```

**决策规则**：
1. 检查主体的所有绑定，收集关联角色的全部 PolicyRule
2. `deny` 规则具有最高优先级 —— 任一条 `deny` 匹配即返回拒绝
3. 所有 `deny` 不匹配时，任一条 `allow` 匹配即返回允许
4. 无任何规则匹配时，默认拒绝

## 4. DataScope（数据范围）五种级别

定义于 `internal/domain/rbac/data_scope.go`：

| 级别 | 常量 | 说明 |
|------|------|------|
| 本人 | `DataScopeSelf` | 仅能操作自己的数据 |
| 本部门 | `DataScopeDept` | 能操作所在部门的数据 |
| 部门树 | `DataScopeDeptTree` | 能操作所在部门及子部门的数据 |
| 全部 | `DataScopeAll` | 能操作全部数据 |
| 自定义 | `DataScopeCustom` | 自定义数据范围 |

DataScope 同时在两个层面生效：
- **Binding.Scope**：绑定到角色时设置的数据范围约束
- **PolicyRule.Scope**：每条策略规则可以独立指定数据范围要求

## 5. 权限检查流程

`internal/service/rbac/service.go` 接口定义，`CheckPermission` 方法：

```
CheckPermission(ctx, CheckPermissionInput) → (bool, error)
```

**流程图**：

```
1. 查询用户的所有 Binding（ListBindingsByUser）
      │
2. 遍历每个 Binding，获取关联 Role
      │
3. 遍历 Role 的所有 PolicyRule（ListPolicyRulesByRoleID）
      │
4. 对每条 PolicyRule：
   ├── 检查 Effect == deny 且 Matches(resource, action) → 返回 false（拒绝）
   └── 检查 Effect == allow 且 Matches(resource, action) → 收集到 allow 候选
      │
5. 任意 allow 候选匹配 → 返回 true（允许）
6. 无匹配 → 返回 false（默认拒绝）
```

**Service 接口完整方法**（`internal/service/rbac/service.go`）：

| 方法 | 说明 |
|------|------|
| `BindRole(ctx, BindRoleInput)` | 创建绑定（用户/角色 → 角色） |
| `UnbindBinding(ctx, bindingID)` | 删除绑定 |
| `ListBindings(ctx, subjectType, subjectID)` | 按主体查询绑定列表 |
| `SetRolePolicies(ctx, SetRolePoliciesInput)` | 设置角色策略规则（全量替换） |
| `CheckPermission(ctx, CheckPermissionInput)` | 权限检查 |
| `ListBindingsByUser(ctx, userID)` | 查询用户所有绑定 |

## 6. 仓储接口

`internal/repository/rbac/rbac_repo.go`：

| 方法 | 说明 |
|------|------|
| `CreateBinding(ctx, binding)` | 创建绑定 |
| `DeleteBinding(ctx, id)` | 删除绑定 |
| `ListBindingsBySubject(ctx, subjectType, subjectID)` | 按主体查询绑定 |
| `ListPolicyRulesByRoleID(ctx, roleID)` | 查询角色策略规则 |
| `ReplacePolicyRules(ctx, roleID, rules)` | 全量替换角色策略规则 |

## 7. 使用示例

### 7.1 创建角色并设置策略

```bash
# 创建角色
POST /skoll/v1/roles
{
  "name": "内容编辑",
  "key": "content_editor",
  "description": "可编辑所有内容",
  "permissions": ["content.read", "content.write"]
}

# 设置策略规则
PUT /skoll/v1/rbac/roles/{roleId}/policies
{
  "rules": [
    { "resource": "content", "action": "read",  "effect": "allow", "scope": "all" },
    { "resource": "content", "action": "write", "effect": "allow", "scope": "dept" },
    { "resource": "content", "action": "delete", "effect": "deny",  "scope": "all" }
  ]
}
```

### 7.2 绑定用户到角色

```bash
POST /skoll/v1/rbac/bindings
{
  "subject_type": "user",
  "subject_id": "<user_id>",
  "role_id": "<role_id>",
  "scope": "dept"
}
```

### 7.3 权限检查

```bash
POST /skoll/v1/rbac/check
{
  "user_id": "<user_id>",
  "resource": "content",
  "action": "write"
}
```

返回 `{ "code": "ok", "message": "ok", "data": true }` 或 `false`。

## 8. Permission 标识约定

`internal/domain/rbac/entity.go` 中 `Permission` 使用 `Resource:Action` 格式作为 Key：

```go
func (p Permission) Key() string {
    return p.Resource + ":" + p.Action
}
```

权限标识示例：
- `user:create` — 创建用户
- `content:read` — 读取内容
- `menu:read` — 读取菜单
- `*:*` — 全部权限（通配符）

## 9. M1 权限目录与菜单 Registry

M1 将“可授权的权限点”和“可展示的侧栏入口”拆成两个可注册目录：

- **Permission Catalog**：由 `internal/domain/permission` 建模，记录权限 key、类型、模块、来源、风险等级、元数据和启用状态。
- **Menu Registry**：由 `internal/domain/menu` 建模，记录菜单 key、父级、来源、路径、组件、图标、排序、显隐和访问要求。

两者的职责不同：Permission Catalog 是“能否授权”的目录，Menu Registry 是“能否出现在导航里”的目录。菜单节点通过 `RequiredRoles` 和 `RequiredPermissions` 引用角色或权限 key；真正的 API 访问仍要由后端 RBAC、handler 或 service 层校验。

### 9.1 权限命名

权限 key 由 `internal/domain/permission/rules.go` 校验，必须匹配：

```text
^[a-z][a-z0-9_:.\-]{1,127}$
```

命名约定：

| 场景 | 推荐格式 | 示例 |
|---|---|---|
| 框架级 API | `<module>.<action>` 或 `<module>:<action>` | `permission.manage`, `menu.read`, `menu.manage` |
| 插件 API | `<plugin-or-module>:<resource>:<action>` | `reports:invoice:read` |
| 菜单可见性 | 复用对应读取或管理权限 | `report.read` |
| 高风险操作 | 使用独立 action，不复用 read/list | `plugin.install`, `permission.manage` |

系统启动时由 `internal/bootstrap/permission_menu_seed.go` 注册框架内置权限：

| Key | 类型 | 模块 | 来源 | 风险 | 覆盖范围 |
|---|---|---|---|---|---|
| `permission.manage` | `api` | `permission` | `system` | `high` | Permission Catalog 读写与 diff |
| `menu.read` | `api` | `menu` | `system` | `low` | 读取菜单 registry tree |
| `menu.manage` | `api` | `menu` | `system` | `medium` | 保存、排序、显隐菜单 registry |

新增权限时必须同步：

1. 在 Permission Catalog 注册资源，或在插件 `plugin.yaml` 的 `permissions` 中声明。
2. 在角色授权、菜单 `required_permissions`、按钮权限指令中复用同一个 key。
3. 在测试或 smoke 中覆盖授权、撤销和拒绝路径。

### 9.2 Catalog 字段与来源

`PermissionResource` 的核心字段：

| 字段 | 说明 |
|---|---|
| `Key` | 权限唯一标识，保存前会 trim 并转小写 |
| `Type` | `api`、`menu`、`button`、`data_scope`、`plugin` |
| `Module` | 业务模块，必须匹配小写模块名规则 |
| `Source` | 来源，系统权限使用 `system`，插件权限使用 `plugin.<plugin_id>` |
| `Name` | 面向管理员的显示名 |
| `Risk` | `low`、`medium`、`high`、`critical` |
| `Metadata` | 附加信息，例如关联 routes |
| `Enabled` | 是否可授权；禁用后仍可保留历史记录 |

插件启用时，`RuntimeManager` 调用 catalog registry 导入插件 manifest 中的权限声明；插件禁用时，已导入权限会被标记为 disabled，而不是删除；卸载时再移除对应来源的权限与菜单。

### 9.3 菜单 Registry 字段与来源

菜单 key 由 `internal/domain/menu/rules.go` 校验，必须匹配：

```text
^[a-z][a-z0-9_.\-]{1,127}$
```

`MenuNode` 的核心字段：

| 字段 | 说明 |
|---|---|
| `Key` | 菜单唯一标识，建议使用层级命名，如 `system.users`、`plugin.reports` |
| `ParentKey` | 父菜单 key，空值表示根节点 |
| `Source` | 来源，系统菜单使用 `system`，插件菜单使用 `plugin.<plugin_id>` |
| `Path` | 前端路由路径，必须以 `/` 开头 |
| `Component` | 前端组件或插件页面标识 |
| `Icon` | 前端图标名 |
| `Sort` | 同级排序，数值越小越靠前 |
| `Visible` | 是否进入可见菜单树 |
| `RequiredRoles` | 访问该菜单所需角色 |
| `RequiredPermissions` | 访问该菜单所需权限 key |

菜单 registry 的 canonical HTTP API 是：

| 方法 | 路径 | 用途 |
|---|---|---|
| `GET` | `/v1/menus/tree` | 按 `source`、`parentKey`、`visible`、`roles`、`permissions` 查询菜单树 |
| `PUT` | `/v1/menus` | 合并保存菜单节点 |
| `POST` | `/v1/menus/reorder` | 调整同级菜单顺序 |
| `PATCH` | `/v1/menus/visibility` | 切换菜单显隐 |

前端侧栏、菜单管理页和 M1 smoke 均以 `web/src/navigation/api.ts`、`web/src/stores/navigation.ts` 为 registry 入口。新增侧栏能力时，应写入 Menu Registry，而不是在页面、路由守卫或本地常量中复制一份菜单树。

### 9.4 插件导入规则

插件 manifest 的 `permissions` 会映射为 Permission Catalog：

- 字符串写法会使用该字符串作为 key，并补齐默认类型与模块。
- 对象写法可以显式声明 `key`、`type`、`module`、`name`、`risk`、`metadata`。
- 导入后的 source 统一为 `plugin.<plugin_id>`。

插件 manifest 的 `ui_menu` 会映射为 Menu Registry：

- 未声明 `key` 时，默认使用 `plugin.<plugin_id>`。
- 未声明 `path` 时，使用插件前端入口解析结果。
- `required_roles` 和 `required_permissions` 会原样进入菜单访问过滤。
- 插件禁用时，导入的菜单会被标记为不可见；插件卸载时移除。

### 9.5 前端访问规则

M1 前端统一使用三个入口：

| 场景 | 入口 |
|---|---|
| 路由访问 | `web/src/permissions/route.ts` 的 `canAccessRoute`、`isPublicRoute` |
| 按钮访问 | `web/src/permissions/button.ts` 的 `canUseButton` |
| 侧栏菜单 | `web/src/stores/navigation.ts` 加载 `/v1/menus/tree` 后生成 |

页面代码不应直接拼权限判断表达式，也不应继续依赖本地默认 permission catalog 作为权限矩阵来源。权限矩阵以后端 catalog 为准，按钮和菜单只消费同一批权限 key。
