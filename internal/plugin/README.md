# internal/plugin

## 功能说明
插件管理核心层。

## 文件组织规范
- 采用小写与下划线命名，按职责拆分文件。
- 接口定义与实现分离，避免单文件过大。
- 变更时同步补充测试与文档。

## 当前规划文件
- manager.go
- registry.go
- loader.go
- resolver.go
- permission.go
- types.go

## 当前实现状态（M8）
- 已实现插件生命周期管理（install/enable/disable/uninstall/list/get）。
- 已实现基于 `plugin.yaml` 的元数据加载。
- 已实现依赖解析与循环依赖检测。
- 已实现基础权限校验器与内存扩展点注册器。
- 已补充单元测试：`manager_test.go`、`loader_test.go`、`resolver_test.go`。

## 后续待补充实现
- [ ] 完善插件包校验（签名/哈希）与版本约束匹配。
- [ ] 对接 CLI 与 HTTP 管理接口。
- [ ] 增加内置插件示例的扩展点注册实现。

