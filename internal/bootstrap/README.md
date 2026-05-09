# internal/bootstrap

## 功能说明
应用启动与运行时编排层，负责配置加载、依赖注入与进程生命周期管理。

## 文件组织规范
- 采用小写与下划线命名，按职责拆分文件。
- 接口定义与实现分离，避免单文件过大。
- 变更时同步补充测试与文档。

## 当前规划文件
- auth_policy.go
- config.go
- di.go
- middleware.go
- run.go
- run_test.go
- shutdown.go
- validator.go

## 后续待补充实现
- [ ] 按目录职责补齐核心实现代码。
- [ ] 补充单元测试与必要的集成测试。
- [ ] 完善示例、边界条件与错误处理说明。
