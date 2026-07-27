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
- 20260722_000023_create_plugin_migration_ledger.sql
- 20260722_000024_create_workflow_persistence.sql
- 20260722_000025_create_notification_persistence.sql
- 20260722_000026_create_job_persistence.sql
- 20260722_000027_create_document_number_persistence.sql
- 20260722_000028_create_document_workflow_persistence.sql
- 20260722_000029_create_document_collaboration_persistence.sql

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

## 插件迁移账本

### 20260722_000023_create_plugin_migration_ledger.sql

- 表名：`sk_plugin_migrations`。
- 用途：记录每个插件已成功提交的 migration 版本、名称、SHA-256 和执行时间。
- 唯一约束：`idx_plugin_migration_version(plugin_id, version)`，用于保证生命周期重试幂等。
- 事务边界：插件 SQL 与账本写入由同一个 GORM transaction 提交；失败不会生成成功账本。

## 工作流持久化

### 20260722_000024_create_workflow_persistence.sql

- 表名：工作流定义、节点、节点审批人、流转、实例、任务和动作历史共 7 张表。
- 聚合边界：定义和实例分别作为事务写入单元，子记录写入失败时整笔回滚。
- 查询索引：业务对象、实例状态、发起人、任务审批人、动作参与人和时间线顺序。
- 对应实现：`internal/store/sql/gormrepo/workflow_model.go`、`workflow_store.go`。

## 通知持久化

### 20260722_000025_create_notification_persistence.sql

- 表名：`sk_notification_items`、`sk_notification_reminder_rules`、`sk_notification_delivery_attempts`。
- 用途：持久化通知收件箱、已读/完成状态、提醒规则及不可变投递尝试。
- 幂等：通知 ID 唯一；投递按通知、渠道和幂等键唯一；成功渠道不再创建重复尝试。
- 重试：失败尝试保留错误，新幂等键产生下一次尝试并递增 attempt。

## 持久化任务调度

### 20260722_000026_create_job_persistence.sql

- 表名：`sk_jobs`。
- 用途：统一保存宿主和插件的定时任务、执行载荷、结果、租约、重试与死信状态。
- 并发：领取任务时原子写入 worker、租约令牌和到期时间；只有当前令牌可提交结果。
- 恢复：进程重启不丢任务，过期租约可重新领取；达到最大尝试次数后进入 `dead_letter`。
- 幂等：`namespace + idempotency_key` 唯一，未提供幂等键的任务不受空值冲突影响。

## 业务单据编号持久化

### 20260722_000027_create_document_number_persistence.sql

- 表名：`sk_document_number_sequences`、`sk_document_number_issues`。
- 隔离：序列按插件、租户、单据类型和周期独立；同一命名空间内的序号唯一。
- 幂等：发号记录按插件、租户、单据类型和幂等键唯一。
- 规则：序列表保存规则指纹，周期内规则不可变；序列增量和发号记录在业务事务中原子提交。

## 医药 OA 主数据表

### 20260718_000019_create_pharma_oa_master_data.sql

- 表名：`pharma_oa_employees`、`pharma_oa_products`、`pharma_oa_suppliers`、`pharma_oa_customers`、`pharma_oa_warehouses`。
- 用途：持久化员工、药品、供应商、客户和仓库/库区/库位聚合。
- JSON 字段：只承载当前无需独立查询的证照、联系人、资质附件、温控与库区快照。
- 卸载策略：固定 `retain`；正常插件卸载不删除业务数据，不提供自动 down。
- 对应 GORM model：`internal/store/sql/gormrepo/pharmaoa_master_model.go`。

## 医药 OA 库存与业务单据表

### 20260718_000020_create_pharma_oa_inventory_orders.sql

- 表名：批次、余额、不可变流水、库存锁、采购申请/订单/入库、销售订单/出库、盘点和调拨共 11 张表。
- 事务：余额使用行锁与 version 条件更新；batch、balance、ledger 在同一事务提交，多行入出库失败时整体回滚。
- 幂等：流水、采购入库、销售出库和调拨均有唯一 idempotency key；流水只允许追加。
- 金额：最终金额列使用 `DECIMAL(18,2)`，聚合行 JSON 保留当前领域数量与单价语义。
- 卸载策略：固定 `retain`；不提供 destructive down 或旧结构兼容路径。

## 医药 OA 工作流关联业务记录

### 20260718_000021_create_pharma_oa_workflow_records.sql

- 表名：合同、质量投诉、召回、客户跟进、销售机会、付款计划、发票、付款提醒、库存告警与报表导出共 11 张表。
- 聚合：可查询列保存状态、范围、到期、参与人和幂等信息；`payload_json` 保存附件、任务、回款、日志等完整快照。
- 金额：合同使用 `DECIMAL(18,2)`；机会、付款和发票金额使用 `BIGINT` 分值。
- 幂等：库存告警与报表导出作业使用唯一 idempotency key，库存告警按类型与余额位置唯一。
- 卸载策略：固定 `retain`；不提供 destructive down 或旧结构兼容路径。

## 医药 OA Schema 补全

### 20260718_000022_complete_pharma_oa_schema.sql

- 表名：`pharma_oa_announcements`、`pharma_oa_cold_chain_records`。
- 用途：补齐当前公告聚合和不可变冷链采集记录的建表与查询索引。
- 资质归属：员工、供应商和客户资质继续保存在所属主数据聚合中，不创建独立候选表或双写路径。
- 卸载策略：固定 `retain`；不提供 destructive down。


### 20260722_000028_create_document_workflow_persistence.sql

- 表名：`sk_document_workflow_bindings`、`sk_document_workflow_actions`。
- 用途：在同一宿主事务内持久化业务单据、审批实例绑定和动作幂等结果。
- 隔离：单据主键包含 `plugin_id` 与 `tenant_id`；审批实例在插件命名空间内唯一。
- 查询：保存单号、标题和创建/更新人投影，并提供更新时间、创建时间、单号、类型与状态的稳定游标索引。
- 一致性：动作记录通过复合外键绑定单据，删除绑定时级联删除幂等历史。

### 20260722_000029_create_document_collaboration_persistence.sql

- 表名：`sk_document_attachments`、`sk_document_comments`、`sk_document_timeline_events`。
- 用途：保存附件快照与软移除归属、不可编辑评论和单据内严格递增的只增时间线。
- 隔离：所有主键与索引包含插件、租户和单据边界；关联记录级联绑定当前单据。
- 当前规则：附件移除不删除文件或历史，评论与时间线没有更新/删除路径。

### 20260727_000030_create_plugin_event_outbox.sql

- 表名：`sk_plugin_event_outbox`。
- 用途：在宿主业务事务内持久化插件发布的当前事件信封，提交后才允许后续调度器读取。
- 幂等：`publisher + idempotency_key` 唯一；相同身份和内容返回原事件，不同内容直接冲突。
- 调度：保存尝试次数、下次运行时间、租约、错误和终态时间，并按状态、运行时间和发布者建立竞争领取索引。
- 当前规则：发布入口只写 Outbox；提交后的后台 worker 才能派发，成功确认、失败退避、死信和受控重放都以本表为事实源。

### 20260727_000031_create_plugin_event_inbox.sql

- 表名：`sk_plugin_event_inbox`。
- 用途：在调用插件订阅处理器前，按当前事件、订阅插件和处理器持久化唯一投递身份。
- 幂等：成功投递永久抑制重复副作用；失败或过期租约允许同一投递身份重新抢占。
- 隔离：记录发布者、订阅者、事件版本和租户作用域；只有当前启用且精确声明的订阅可以创建记录。
