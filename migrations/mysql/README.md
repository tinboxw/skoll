# migrations/mysql

## 功能说明
MySQL 迁移脚本目录。

## 文件组织规范
- 采用小写与下划线命名，按职责拆分文件。
- 接口定义与实现分离，避免单文件过大。
- 变更时同步补充测试与文档。

## 当前规划文件
- 20240101_000001_create_users.sql
- 20240102_000002_create_roles.sql
- 20260510_000010_create_plugins.sql
- 20260510_000011_create_plugin_routes_releases.sql
- 20260616_000012_add_user_organization.sql
- 20260618_000013_create_permission_resources.sql
- 20260618_000014_create_menu_nodes.sql

## 执行顺序

按文件名前缀从小到大执行。`20260618_000013_create_permission_resources.sql` 必须先于 `20260618_000014_create_menu_nodes.sql` 执行，二者分别建立权限资源目录和菜单节点目录，不包含 seed 数据。

## 新增权限与菜单表

### 20260618_000013_create_permission_resources.sql

- 表名: `sk_permission_resources`
- 用途: 持久化 permission domain 中的权限资源目录。
- 唯一约束: `uk_permission_resources_key(permission_key)`
- 查询索引: resource type、module+type、source+type、risk、enabled
- 对应 GORM model: `internal/store/sql/gormrepo.PermissionResourceModel`

### 20260618_000014_create_menu_nodes.sql

- 表名: `sk_menu_nodes`
- 用途: 持久化 menu domain 中的菜单节点目录。
- 唯一约束: `uk_menu_nodes_key(menu_key)`
- 查询索引: parent+sort、source、path、sort、visible
- 对应 GORM model: `internal/store/sql/gormrepo.MenuNodeModel`

## 回滚说明

当前项目未集成自动迁移工具，也未维护 down migration 文件。需要回滚时由维护者按依赖倒序手动处理：先处理 `sk_menu_nodes`，再处理 `sk_permission_resources`，并在执行前备份业务数据。

