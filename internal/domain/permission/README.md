# internal/domain/permission

## 功能说明
权限目录领域包，承载 API、菜单、按钮、数据范围与插件声明权限的统一领域模型。

## 文件组织规范
- 采用小写与下划线命名，按职责拆分文件。
- domain 包只放领域模型和值对象，不依赖 store、service、handler 或 GORM model。
- 不添加旧权限别名、兼容授权路径或新旧权限双轨实现。
- 变更时同步补充测试与文档。

## 当前规划文件
- doc.go

## 后续待补充实现
- [ ] 定义 `PermissionResource` 类型和值对象。
- [ ] 实现权限 key、类型、来源、风险与 metadata 校验规则。
- [ ] 补充单元测试与序列化边界说明。
