# internal/domain/permission

## 功能说明
权限目录领域包，承载 API、菜单、按钮、数据范围与插件声明权限的统一领域模型。

## 权限资源类型
- `api`: HTTP/API 访问权限。
- `menu`: 菜单节点可见性权限。
- `button`: 页面按钮或动作权限。
- `data_scope`: 数据范围权限。
- `plugin`: 插件声明的安装、启停、配置或生命周期权限。

## 命名规则
- `key` 使用小写字母开头，可包含小写字母、数字、`_`、`:`、`.`、`-`，示例: `user.read`、`plugin:install`。
- `module` 使用小写模块名，示例: `user`、`plugin`。
- `source` 使用权限来源，示例: `system`、`plugin_demo`。
- `name` 为面向管理员的可读名称，不能为空。
- `metadata` key 使用小写字母开头，可包含小写字母、数字、`_`、`.`、`-`。

## 风险等级
- `low`: 普通读取或低影响操作。
- `medium`: 会改变普通配置或业务数据的操作。
- `high`: 影响权限、插件、审计、文件等高风险能力的操作。
- `critical`: 影响系统安全边界、插件安装执行、密钥或数据破坏的操作。

## No-compat 规则
- 不添加旧权限别名。
- 不保留兼容授权路径。
- 不维护新旧权限双轨来源。
- 插件权限必须通过 manifest 进入统一 catalog，不能直接污染核心权限表或页面常量。

## 文件组织规范
- 采用小写与下划线命名，按职责拆分文件。
- domain 包只放领域模型和值对象，不依赖 store、service、handler 或 GORM model。
- 不添加旧权限别名、兼容授权路径或新旧权限双轨实现。
- 变更时同步补充测试与文档。

## 当前规划文件
- doc.go
- metadata.go
- resource.go
- rules.go

## 后续待补充实现
- [ ] 接入 repository、store 与 service 层。
