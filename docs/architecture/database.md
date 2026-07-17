# 数据库设计文档

> 医药 OA SQL 持久化的当前 schema 与 repository 边界见 [医药 OA 持久化契约](pharma-oa-persistence.md)。

## 1. 概述

Skoll 支持三种存储模式，通过 `SKOLL_STORE_MODE` 或 `store.mode` 配置项切换：

| 模式 | 实现路径 | 适用场景 |
|------|---------|---------|
| `memory` | `internal/store/memory/` | 开发测试、快速原型 |
| `mysql` | `internal/store/sql/mysql/` + `gormrepo/` | 生产环境主推 |
| `postgres` | `internal/store/sql/postgres/` + `gormrepo/` | 生产备选，ClickHouse 审计 |

存储工厂 `internal/store/sql/factory.go` 的 `NewBundle()` 根据 mode 返回聚合了所有 repo 实现的 Bundle。SQL 持久化层统一使用 GORM，通过 `internal/store/sql/gormrepo/` 下各 `*_model.go` + `*_store.go` 实现 GORM Model → Domain 的双向转换。

## 2. 核心表结构

### 2.1 用户表（sk_users）

Model 定义：`internal/store/sql/gormrepo/user_model.go`，迁移脚本：`migrations/mysql/20240101_000001_create_users.sql`

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | BIGINT UNSIGNED | PK, AUTO_INCREMENT | 用户唯一标识 |
| `account` | VARCHAR(128) | UNIQUE, NOT NULL | 登录账号 |
| `name` | VARCHAR(128) | NOT NULL | 显示名称 |
| `email` | VARCHAR(191) | UNIQUE | 邮箱地址 |
| `password` | VARCHAR(256) | - | bcrypt 密码哈希 |
| `status` | VARCHAR(32) | - | 状态：`active` / `disabled` |
| `created_at` | TIMESTAMP | - | 创建时间 |
| `updated_at` | TIMESTAMP | - | 更新时间 |

> **Domain 实体**：`internal/domain/user/entity.go` → `User` 结构体，包含 `Account`、`Name`、`Email`（值对象）、`PasswordHash`（值对象）、`Status`、`AuditMeta`。

### 2.2 角色表（sk_roles）

Model 定义：`internal/store/sql/gormrepo/role_model.go`

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | BIGINT UNSIGNED | PK, AUTO_INCREMENT | 角色唯一标识 |
| `name` | VARCHAR(128) | NOT NULL | 角色名称 |
| `key` | VARCHAR(128) | UNIQUE | 角色键名（小写） |
| `description` | VARCHAR(512) | - | 角色描述 |
| `permissions` | TEXT | - | 权限列表（JSON 数组） |
| `built_in` | BOOL | - | 是否内置角色 |
| `created_at` | TIMESTAMP | - | 创建时间 |
| `updated_at` | TIMESTAMP | - | 更新时间 |

> **Domain 实体**：`internal/domain/role/entity.go` → `Role` 结构体，`Permissions` 为字符串数组。内置角色（`BuiltIn=true`）不可删除。

### 2.3 RBAC 绑定表（sk_rbac_bindings）

Model 定义：`internal/store/sql/gormrepo/rbac_model.go`

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | BIGINT UNSIGNED | PK, AUTO_INCREMENT | 绑定唯一标识 |
| `subject_type` | VARCHAR(32) | INDEX(idx_subject) | 主体类型：`user` / `role` |
| `subject_id` | BIGINT UNSIGNED | INDEX(idx_subject) | 主体 ID |
| `role_id` | BIGINT UNSIGNED | INDEX | 被绑定的角色 ID |
| `scope` | VARCHAR(32) | - | 数据范围：self/dept/dept_tree/all/custom |
| `created_at` | TIMESTAMP | - | 创建时间 |
| `updated_at` | TIMESTAMP | - | 更新时间 |

> **Domain 实体**：`internal/domain/rbac/entity.go` → `Binding`，`SubjectType` 为 `user` 或 `role`，`Scope` 为 `DataScope` 枚举。

### 2.4 RBAC 策略规则表（sk_rbac_policy_rules）

Model 定义：`internal/store/sql/gormrepo/rbac_model.go` → `PolicyRuleModel`

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | BIGINT UNSIGNED | PK, AUTO_INCREMENT | 规则唯一标识 |
| `role_id` | BIGINT UNSIGNED | INDEX | 所属角色 ID |
| `resource` | VARCHAR(256) | - | 资源标识（支持 `*` 通配符） |
| `action` | VARCHAR(128) | - | 操作标识（支持 `*` 通配符） |
| `effect` | VARCHAR(16) | - | 效果：`allow` / `deny` |
| `scope` | VARCHAR(32) | - | 数据范围（同 Binding.scope） |

> **Domain 实体**：`internal/domain/rbac/policy.go` → `PolicyRule`，支持通配符匹配，`Effect` 为 `allow` 或 `deny`。

### 2.5 审计记录表（sk_audit_records）

Model 定义：`internal/store/sql/gormrepo/audit_model.go`

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | VARCHAR(64) | PK | 审计记录唯一标识（字符串 ID） |
| `actor_id` | VARCHAR(64) | INDEX | 操作者 ID |
| `action` | VARCHAR(64) | INDEX | 操作名称 |
| `resource` | VARCHAR(64) | INDEX | 操作资源类型 |
| `resource_id` | VARCHAR(128) | - | 操作资源 ID |
| `detail_json` | LONGTEXT | - | 操作详情（JSON） |
| `occurred_at` | TIMESTAMP | INDEX | 操作发生时间 |

> **Domain 实体**：`internal/domain/audit/entity.go` → `Record`，`Detail` 字段为 `map[string]any`。

### 2.5.1 统一审计事件表（sk_audit_events）

M2 起新增统一审计事件表，迁移脚本：

- `migrations/mysql/20260619_000015_create_audit_events.sql`
- `migrations/postgres/20260619_000015_create_audit_events.sql`

`sk_audit_events` 是 operation/login/error/plugin/security 的 canonical 持久化目标。`sk_audit_records` 仍描述当前存量模型，后续 M2 store/API 任务会直接迁移到统一表，不新增旧格式兼容查询路径。

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | VARCHAR(64) | PK | 审计事件唯一标识 |
| `event_type` | VARCHAR(32) | INDEX(event_type, occurred_at) | `operation` / `login` / `error` / `plugin` / `security` |
| `action` | VARCHAR(192) | INDEX(action, occurred_at) | `module.resource.action` 格式 |
| `actor_type` | VARCHAR(64) | NOT NULL | 主体类型，如 user、role、system、anonymous |
| `actor_id` | VARCHAR(64) | INDEX(actor_id, occurred_at) | 主体 ID，可为空字符串 |
| `actor_name` | VARCHAR(128) | - | 主体显示名 |
| `resource_type` | VARCHAR(64) | INDEX(resource_type, resource_id, occurred_at) | 资源类型 |
| `resource_id` | VARCHAR(128) | INDEX(resource_type, resource_id, occurred_at) | 资源 ID |
| `resource_name` | VARCHAR(128) | - | 资源显示名 |
| `result` | VARCHAR(32) | INDEX(result, occurred_at) | `success` / `failure` / `denied` |
| `risk` | VARCHAR(32) | INDEX(risk, occurred_at) | `low` / `medium` / `high` / `critical` |
| `trace_id` | VARCHAR(128) | INDEX | trace id |
| `request_id` | VARCHAR(128) | INDEX | request/correlation id |
| `request_method` | VARCHAR(16) | - | HTTP method |
| `request_path` | VARCHAR(512) | - | HTTP path |
| `request_ip` | VARCHAR(64) | - | 客户端 IP |
| `user_agent` | VARCHAR(512) | - | User-Agent |
| `metadata_json` | LONGTEXT/TEXT | - | 通用 metadata |
| `source_json` | LONGTEXT/TEXT | - | 类型专属源数据，如 LoginLog/ErrorLog 原始字段 |
| `occurred_at` | DATETIME(3)/TIMESTAMPTZ | INDEX | 事件发生时间 |
| `created_at` | DATETIME(3)/TIMESTAMPTZ | - | 记录写入时间 |

### 2.6 系统设置表（sk_system_settings）

Model 定义：`internal/store/sql/gormrepo/system_setting_model.go`

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | BIGINT UNSIGNED | PK, AUTO_INCREMENT | 设置唯一标识 |
| `key` | VARCHAR(128) | UNIQUE | 设置键名（小写） |
| `value` | TEXT | - | 设置值 |
| `encrypted` | BOOL | - | 是否加密存储 |
| `created_at` | TIMESTAMP | - | 创建时间 |
| `updated_at` | TIMESTAMP | - | 更新时间 |

> **Domain 实体**：`internal/domain/system/entity.go` → `Setting`，Key 格式限制为 `^[a-z][a-z0-9_.-]{1,127}$`。

### 2.7 插件表（sk_plugins）

Model 定义：`internal/store/sql/gormrepo/plugin_model.go`，迁移脚本：`migrations/mysql/20260510_000010_create_plugins.sql`

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | BIGINT UNSIGNED | PK, AUTO_INCREMENT | 内部主键 |
| `plugin_id` | VARCHAR(128) | UNIQUE | 插件标识 |
| `name` | VARCHAR(128) | - | 插件名称 |
| `version` | VARCHAR(64) | - | 插件版本 |
| `description` | VARCHAR(512) | - | 描述 |
| `state` | VARCHAR(32) | INDEX | 状态：installed/enabled/disabled |
| `source` | VARCHAR(512) | - | 安装来源路径 |
| `ui_mode` | VARCHAR(64) | - | UI 模式 |
| `plugin_level` | VARCHAR(32) | - | 插件级别：system/app |
| `app_id` | VARCHAR(128) | - | 应用 ID |
| `system_builtin` | BOOL | - | 是否系统内置 |
| `permissions_json` | TEXT | - | 权限列表 JSON |
| `dependencies_json` | TEXT | - | 依赖列表 JSON |
| `installed_at` | TIMESTAMP | - | 安装时间 |
| `enabled_at` | TIMESTAMP | NULLABLE | 最近启用时间 |

插件路由和发布相关的附属表：`sk_plugin_routes`、`sk_plugin_releases`（`migrations/mysql/20260510_000011_create_plugin_routes_releases.sql`）。

### 2.8 文件对象元数据表（sk_file_objects）

M3 起新增文件对象元数据表，迁移脚本：

- `migrations/mysql/20260622_000016_create_file_objects.sql`
- `migrations/postgres/20260622_000016_create_file_objects.sql`

`sk_file_objects` 只保存文件 metadata。对象内容由 object store adapter 管理，本表不保存二进制内容，也不引入旧上传路径兼容字段。

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | VARCHAR(64) | PK | 文件元数据记录 ID |
| `object_key` | VARCHAR(256) | UNIQUE, NOT NULL | 对象存储 key |
| `name` | VARCHAR(255) | NOT NULL | 原始文件名或展示名 |
| `size_bytes` | BIGINT | NOT NULL | 文件大小 |
| `mime` | VARCHAR(128) | NOT NULL | MIME 类型 |
| `hash` | VARCHAR(256) | INDEX | 内容哈希 |
| `owner_type` | VARCHAR(64) | INDEX(owner_type, owner_id) | 所属主体类型 |
| `owner_id` | VARCHAR(64) | INDEX(owner_type, owner_id) | 所属主体 ID |
| `visibility` | VARCHAR(32) | INDEX(visibility, status) | `private` / `public` / `plugin_asset` |
| `storage_driver` | VARCHAR(64) | INDEX(storage_driver, status) | 存储驱动，例如 `local` |
| `status` | VARCHAR(32) | INDEX | `pending` / `available` / `failed` / `deleted` |
| `source_module` | VARCHAR(64) | INDEX(source_module, source_plugin_id) | 来源模块 |
| `source_plugin_id` | VARCHAR(64) | INDEX(source_module, source_plugin_id) | 来源插件 ID，可为空字符串 |
| `metadata_json` | LONGTEXT/TEXT | - | 扩展 metadata |
| `created_at` | DATETIME(3)/TIMESTAMPTZ | - | 创建时间 |
| `updated_at` | DATETIME(3)/TIMESTAMPTZ | INDEX(status, updated_at) | 更新时间 |

> **Domain 实体**：`internal/domain/file/object.go` → `FileObject`，包含 key、name、size、mime、hash、owner、visibility、storage driver、status、source 与 `AuditMeta`。

### 2.9 字典表（sk_dictionary_types / sk_dictionary_items）

M4 起新增系统字典类型与字典条目表，迁移脚本：

- `migrations/mysql/20260622_000017_create_dictionary.sql`
- `migrations/postgres/20260622_000017_create_dictionary.sql`

`sk_dictionary_types` 保存字典分类；`sk_dictionary_items` 保存分类下可排序、可启停的条目。条目通过 `type_code` 引用类型 `code`，删除类型时级联删除条目。

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `sk_dictionary_types.id` | VARCHAR(64) | PK | 字典类型 ID |
| `sk_dictionary_types.code` | VARCHAR(128) | UNIQUE, NOT NULL | 类型编码，按 domain 规则归一化 |
| `sk_dictionary_types.name` | VARCHAR(255) | NOT NULL | 类型名称 |
| `sk_dictionary_types.description` | VARCHAR(512) | - | 类型说明 |
| `sk_dictionary_types.status` | VARCHAR(32) | INDEX(status, sort) | `enabled` / `disabled` |
| `sk_dictionary_types.sort` | INT/INTEGER | INDEX(status, sort) | 类型排序 |
| `sk_dictionary_types.builtin` | BOOL/BOOLEAN | - | 系统内置标记 |
| `sk_dictionary_items.id` | VARCHAR(64) | PK | 字典条目 ID |
| `sk_dictionary_items.type_code` | VARCHAR(128) | UNIQUE(type_code, value), FK | 所属类型编码 |
| `sk_dictionary_items.label` | VARCHAR(255) | NOT NULL | 条目展示名 |
| `sk_dictionary_items.value` | VARCHAR(255) | UNIQUE(type_code, value), NOT NULL | 条目值 |
| `sk_dictionary_items.status` | VARCHAR(32) | INDEX(status, sort) | `enabled` / `disabled` |
| `sk_dictionary_items.sort` | INT/INTEGER | INDEX(type_code, sort) | 条目排序 |
| `sk_dictionary_items.builtin` | BOOL/BOOLEAN | - | 系统内置标记 |

> **Domain 实体**：`internal/domain/system/dictionary.go` → `DictionaryType`、`DictionaryItem`。

### 2.10 组织表（sk_departments / sk_positions / sk_user_organization_assignments）

M4 新增一等组织模型表，迁移脚本：
- `migrations/mysql/20260629_000018_create_organization.sql`
- `migrations/postgres/20260629_000018_create_organization.sql`

`sk_departments` 保存部门树节点；`sk_positions` 保存岗位；`sk_user_organization_assignments` 保存用户和部门/岗位归属。旧 `sk_users.department_id`、`sk_users.position_id` 字段保留给用户列表和兼容读取，新的组织归属以 assignment 表为准。

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `sk_departments.id` | VARCHAR(64) | PK | 部门 ID |
| `sk_departments.parent_id` | VARCHAR(64) | INDEX(parent_id, sort) | 父部门 ID，根节点为空字符串 |
| `sk_departments.code` | VARCHAR(128) | UNIQUE, NOT NULL | 部门编码，按 domain 规则归一化 |
| `sk_departments.name` | VARCHAR(255) | NOT NULL | 部门名称 |
| `sk_departments.leader_user_id` | VARCHAR(64) | INDEX | 部门负责人用户 ID |
| `sk_departments.status` | VARCHAR(32) | INDEX(status, sort) | `enabled` / `disabled` |
| `sk_departments.sort` | INT/INTEGER | INDEX(parent_id, sort) | 同级排序 |
| `sk_positions.id` | VARCHAR(64) | PK | 岗位 ID |
| `sk_positions.code` | VARCHAR(128) | UNIQUE, NOT NULL | 岗位编码 |
| `sk_positions.name` | VARCHAR(255) | NOT NULL | 岗位名称 |
| `sk_positions.description` | VARCHAR(512) | - | 岗位说明 |
| `sk_positions.status` | VARCHAR(32) | INDEX(status, sort) | `enabled` / `disabled` |
| `sk_positions.sort` | INT/INTEGER | INDEX(status, sort) | 岗位排序 |
| `sk_user_organization_assignments.user_id` | VARCHAR(64) | PK | 用户 ID |
| `sk_user_organization_assignments.department_id` | VARCHAR(64) | INDEX | 归属部门 ID |
| `sk_user_organization_assignments.position_id` | VARCHAR(64) | INDEX | 归属岗位 ID，可为空字符串 |
| `sk_user_organization_assignments.primary_assignment` | BOOL/BOOLEAN | - | 是否主归属 |

> **Domain 实体**：`internal/domain/organization/organization.go` → `Department`、`Position`、`UserAssignment`。

## 3. GORM Model 与 Domain 转换

所有 SQL 持久化遵循统一转换模式：

```
持久化（写）：Domain 实体 ──[ModelFromDomain]──→ GORM Model ──[GORM Save]──→ 数据库
查询（读）：  数据库 ──[GORM Find]──→ GORM Model ──[ToDomain]──→ Domain 实体
```

| Domain 实体 | GORM Model | 转换文件 |
|------------|-----------|---------|
| `domain/user.User` | `UserModel` (sk_users) | `user_model.go` |
| `domain/role.Role` | `RoleModel` (sk_roles) | `role_model.go` |
| `domain/rbac.Binding` | `BindingModel` (sk_rbac_bindings) | `rbac_model.go` |
| `domain/rbac.PolicyRule` | `PolicyRuleModel` (sk_rbac_policy_rules) | `rbac_model.go` |
| `domain/audit.Record` | `AuditRecordModel` (sk_audit_records) | `audit_model.go` |
| `domain/system.Setting` | `SystemSettingModel` (sk_system_settings) | `system_setting_model.go` |
| `plugin.Info` | `PluginModel` (sk_plugins) | `plugin_model.go` |
| `domain/file.FileObject` | `FileObjectModel` (sk_file_objects) | `file_model.go` |
| `domain/system.DictionaryType` | `DictionaryTypeModel` (sk_dictionary_types) | `dictionary_model.go` |
| `domain/system.DictionaryItem` | `DictionaryItemModel` (sk_dictionary_items) | `dictionary_model.go` |

**ID 转换**：Domain 使用 `shared.ID`（字符串类型）。用户、角色、系统设置等早期 GORM Model 使用 `uint64`，通过 `parseUintID` / `formatUintID` 辅助函数双向转换；审计事件、文件对象、字典类型与字典条目直接使用字符串 ID。

**JSON 序列化**：`Permissions`、`Dependencies`、`Detail`、`Signature` 等复杂字段在 GORM Model 中以 JSON 字符串存储，通过 `json.Marshal` / `json.Unmarshal` 与 Domain 实体转换。

## 4. 存储模式选择

| 场景 | 推荐模式 | 说明 |
|------|---------|------|
| 本地开发/快速原型 | `memory` | 零依赖启动，数据不持久化 |
| 中小规模生产 | `mysql` | 成熟稳定，生态丰富 |
| 大规模 + 审计分析 | `postgres` | PG 主库 + ClickHouse 审计时序存储 |

**切换方式**：

```bash
# 环境变量
SKOLL_STORE_MODE=memory

# 配置文件
store:
  mode: mysql
  dsn: "user:password@tcp(host:port)/skoll?charset=utf8mb4&parseTime=True&loc=Local"
```

## 5. 迁移脚本

迁移脚本位于 `migrations/` 目录，按数据库类型分 `mysql/` 和 `postgres/` 子目录，以时间戳版本号命名：

```
migrations/
├── mysql/
│   ├── 20240101_000001_create_users.sql
│   ├── 20240102_000002_create_roles.sql
│   ├── 20260510_000010_create_plugins.sql
│   ├── 20260510_000011_create_plugin_routes_releases.sql
│   ├── 20260718_000019_create_pharma_oa_master_data.sql
│   └── 20260718_000020_create_pharma_oa_inventory_orders.sql
└── postgres/
    ├── 20240101_000001_create_users.sql
    ├── 20260510_000010_create_plugins.sql
    ├── 20260510_000011_create_plugin_routes_releases.sql
    ├── 20260718_000019_create_pharma_oa_master_data.sql
    └── 20260718_000020_create_pharma_oa_inventory_orders.sql
```

> 注意：迁移脚本由 GORM AutoMigrate 或手动执行，当前项目未集成自动迁移工具。
