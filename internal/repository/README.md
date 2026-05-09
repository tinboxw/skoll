# internal/repository

## 功能说明
仓储契约层，定义数据访问接口与事务边界。

## 文件组织规范
- 采用小写与下划线命名，按职责拆分文件。
- 接口定义与实现分离，避免单文件过大。
- 变更时同步补充测试与文档。

## 当前规划文件
- user_repo.go
- role_repo.go
- rbac_repo.go
- audit_repo.go
- system_repo.go
- transaction.go

## 后续待补充实现
- [ ] 按目录职责补齐核心实现代码。
- [ ] 补充单元测试与必要的集成测试。
- [ ] 完善示例、边界条件与错误处理说明。

