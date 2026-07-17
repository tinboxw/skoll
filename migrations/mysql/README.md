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
- 20260619_000015_create_audit_events.sql
- 20260622_000016_create_file_objects.sql
- 20260622_000017_create_dictionary.sql
- 20260629_000018_create_organization.sql
- 20260718_000019_create_pharma_oa_master_data.sql

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

## 新增文件元数据表

### 20260622_000016_create_file_objects.sql

- 表名: `sk_file_objects`
- 用途: 持久化 file domain 中的文件对象元数据；对象内容仍由 object store adapter 管理。
- 唯一约束: `uk_file_objects_key(object_key)`
- 状态字段: `status`，取值由 domain 限定为 `pending`、`available`、`failed`、`deleted`
- 查询索引: owner、visibility+status、storage+status、source、status+updated_at、hash
- 对应 GORM model: `internal/store/sql/gormrepo.FileObjectModel`

## 新增字典表

### 20260622_000017_create_dictionary.sql

- 表名: `sk_dictionary_types`、`sk_dictionary_items`
- 用途: 持久化 system domain 中的字典类型和字典条目。
- 唯一约束: `uk_dictionary_types_code(code)`、`uk_dictionary_items_type_value(type_code, value)`
- 查询索引: type status+sort、item type+sort、item status+sort
- 外键约束: `sk_dictionary_items.type_code` 引用 `sk_dictionary_types.code`，类型删除时级联删除条目。
- 对应 GORM model: `internal/store/sql/gormrepo.DictionaryTypeModel`、`internal/store/sql/gormrepo.DictionaryItemModel`

## 回滚说明

当前项目未集成自动迁移工具，也未维护 down migration 文件。需要回滚时由维护者按依赖倒序手动处理：先处理 `sk_dictionary_items`、`sk_dictionary_types` 和 `sk_file_objects`，再处理 `sk_menu_nodes`、`sk_permission_resources`，并在执行前备份业务数据。

## 医药 OA 主数据表

### 20260718_000019_create_pharma_oa_master_data.sql

- 表名：`pharma_oa_employees`、`pharma_oa_products`、`pharma_oa_suppliers`、`pharma_oa_customers`、`pharma_oa_warehouses`。
- 用途：持久化员工、药品、供应商、客户和仓库/库区/库位聚合。
- JSON 字段：只承载当前无需独立查询的证照、联系人、资质附件、温控与库区快照。
- 卸载策略：固定 `retain`；正常插件卸载不删除业务数据，不提供自动 down。
- 对应 GORM model：`internal/store/sql/gormrepo/pharmaoa_master_model.go`。

