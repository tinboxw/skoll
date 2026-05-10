# internal/store/sql

## 功能说明
SQL 存储抽象层，放置公共 SQL 逻辑与事务实现。

## 文件组织规范
- 采用小写与下划线命名，按职责拆分文件。
- 接口定义与实现分离，避免单文件过大。
- 变更时同步补充测试与文档。

## 注意事项
- `internal/store/sql/gormrepo/model` 必须保持“每个表一个模型文件”的组织方式，禁止回退为单文件堆叠。
- SQL 共享模型新增时，必须同步补充 `model/all_models.go` 以保证迁移清单完整。
- `internal/store/sql/gormrepo/store` 必须按领域拆分仓储文件（`user/role/system/rbac`），禁止回退为单一 `stores.go` 承载全部实现。

## 当前实现
- `common.go`：通用 CRUD/List/NotFound 辅助函数。
- `transaction.go`：支持真实 GORM 事务边界的 `UnitOfWork`。
- `gormrepo/model`：跨 MySQL/PostgreSQL 的共享模型定义与映射（每表独立文件）。
- `gormrepo/store`：跨 MySQL/PostgreSQL 的共享仓储实现（按 user/role/system/rbac 分文件）。
- `gormrepo/stores.go`：根包导出门面，保持外部调用稳定。

