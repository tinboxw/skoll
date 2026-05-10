# 数据库结构重建方案（无兼容层）

更新时间：2026-05-10

## 1. 目标与原则（强制）

1. 项目处于初始阶段，禁止保留老旧结构兼容层。
2. 禁止新增任何用于兼容旧数据的映射表、双轨字段、回退逻辑。
3. 发现老旧数据或老旧表结构时，直接清理并按新规范重建。
4. 用户模型统一采用“账号 + 名称”模式，禁止 `displayname` / `display_name`。

## 2. 新数据规范

### 2.1 主键规范

- 业务主键统一：`BIGINT UNSIGNED`。
- 主键生成策略：优先数据库自增；若后续分布式扩展，再统一切雪花算法。
- 外键字段必须与被引用主键同类型、同符号位。

### 2.2 存储引擎与字符集

- 引擎统一 `InnoDB`。
- 字符集统一 `utf8mb4`，排序规则统一 `utf8mb4_unicode_ci`。

### 2.3 用户表规范

推荐结构：

- `id BIGINT UNSIGNED PRIMARY KEY`
- `account VARCHAR(64) NOT NULL UNIQUE`
- `name VARCHAR(128) NOT NULL`
- `email VARCHAR(191) NOT NULL UNIQUE`
- `status TINYINT UNSIGNED NOT NULL DEFAULT 1`
- `password_hash VARCHAR(255) NOT NULL`
- `created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)`
- `updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)`

禁止字段：

- `displayname`
- `display_name`

## 3. 立即执行项

1. 删除所有兼容映射表：
   - `sk_id_map_users`
   - `sk_id_map_roles`
   - `sk_id_map_bindings`
   - `sk_id_map_settings`
2. 清理所有老旧业务数据（按全新结构重建，不做迁移兼容）。
3. 用户表字段规范化：
   - `display_name` 统一改为 `name`。
4. 停止所有双轨字段扩展策略，不再新增兼容字段。

## 4. 实施阶段

### 阶段 A：数据库重置

1. 备份当前库（仅用于审计，不用于兼容回放）。
2. 删除兼容层对象与老旧数据。
3. 按新规范重建表结构与索引。

### 阶段 B：后端切换

1. 领域模型、仓储模型、DTO 全量切换到新字段（`account`、`name`、整型 ID）。
2. 删除所有旧字段读写代码与兼容分支。
3. 所有接口仅暴露新结构字段。

### 阶段 C：前端切换

1. 前端请求/响应模型统一使用新字段。
2. 所有页面与表单移除 `displayName` 相关逻辑与文案。
3. 仅保留新结构渲染与提交。

### 阶段 D：校验与发布

1. 回归登录、用户管理、角色与权限全链路。
2. 校验数据库中不存在禁用字段与兼容对象。
3. 形成最终字段字典与开发约束。

## 5. 严格校验清单

- 不存在 `displayname` / `display_name` 字段。
- 不存在 `sk_id_map_*` 映射表。
- 用户相关主键与关联字段均为 `BIGINT UNSIGNED`。
- 前后端代码中不存在旧结构兼容逻辑。
- 全量测试通过。

## 6. 本次执行结果（2026-05-10）

### 6.1 最终结构检查结论

- `sk_users.id` 为 `bigint(20) unsigned`，`auto_increment`。
- `sk_roles.id` 为 `bigint(20) unsigned`，`auto_increment`。
- `sk_rbac_bindings.id` 为 `bigint(20) unsigned`，`auto_increment`。
- `sk_rbac_bindings.subject_id` 为 `bigint(20) unsigned`。
- `sk_rbac_bindings.role_id` 为 `bigint(20) unsigned`。
- `sk_rbac_policy_rules.id` 为 `bigint(20) unsigned`，`auto_increment`。
- `sk_rbac_policy_rules.role_id` 为 `bigint(20) unsigned`。
- `sk_system_settings.id` 为 `bigint(20) unsigned`，`auto_increment`。

### 6.2 存储规范检查

- 所有业务表引擎均为 `InnoDB`。
- 所有业务表排序规则均为 `utf8mb4_unicode_ci`。

### 6.3 残留兼容字段检查

- 未发现 `id_int`、`role_id_int`、`subject_id_int` 等双轨兼容字段残留。
- 未发现 `sk_id_map_*` 映射表残留。

### 6.4 执行注意事项

- 当前 MySQL 版本不支持 `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` 语法，需改为显式 `ADD COLUMN` 或先查询 `information_schema` 后条件执行。
- 查询 `key` 字段时需使用反引号（如 ``SELECT `key` ...``）或采用 ORM map 条件，避免关键字语法错误。
