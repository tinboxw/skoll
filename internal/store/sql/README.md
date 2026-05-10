# internal/store/sql

## 功能说明
SQL 存储抽象层，放置公共 SQL 逻辑与事务实现。

## 文件组织规范
- 采用小写与下划线命名，按职责拆分文件。
- 接口定义与实现分离，避免单文件过大。
- 变更时同步补充测试与文档。

## 当前实现
- `common.go`：通用 CRUD/List/NotFound 辅助函数。
- `transaction.go`：支持真实 GORM 事务边界的 `UnitOfWork`。
- `gormrepo/`：跨 MySQL/PostgreSQL 的共享仓储实现（user/role/rbac/system）。

