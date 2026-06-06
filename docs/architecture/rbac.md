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
