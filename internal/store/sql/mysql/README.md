# internal/store/sql/mysql

## 功能说明
MySQL 存储方言入口与适配器。

## 文件组织规范
- 采用小写与下划线命名，按职责拆分文件。
- 接口定义与实现分离，避免单文件过大。
- 变更时同步补充测试与文档。

## 当前实现文件
- adapter.go
- normalize.go

> 说明：具体仓储实现已统一沉淀到 `internal/store/sql/gormrepo`，本目录仅保留 MySQL 方言相关逻辑。

