# internal/domain/menu

## 功能说明
菜单注册中心领域包，承载系统菜单、插件菜单和生成模块菜单的统一领域模型。

## 文件组织规范
- 采用小写与下划线命名，按职责拆分文件。
- domain 包只放领域模型和值对象，不依赖 store、service、handler、web 或 GORM model。
- 不添加旧菜单来源、兼容路由或新旧菜单双轨实现。
- 变更时同步补充测试与文档。

## 当前规划文件
- doc.go
- node.go
- rules.go
- tree.go

## 菜单节点

`MenuNode` 是菜单注册中心的领域值对象，用于描述系统菜单、插件菜单和生成模块菜单的统一结构。

- `Identity.Key`: 菜单唯一键，使用小写字母开头，可包含小写字母、数字、下划线、点和短横线。
- `Identity.ParentKey`: 父菜单键，允许为空；非空时必须满足菜单键命名规则。
- `Identity.Source`: 菜单来源，例如 `system`、插件标识或生成模块标识。
- `View.Name`: 菜单展示名称，不能为空，最长 128 个字符。
- `View.Path`: 前端路由路径，必须以 `/` 开头，不能包含空白字符。
- `View.Component`: 前端组件标识，只在 domain 内作为字符串值对象保存，不绑定具体 web 实现。
- `View.Icon`: 图标标识，只保存语义值，不依赖前端图标库。
- `Sort`: 同级菜单排序值，必须为非负整数。
- `Visible`: 是否展示，`NewNode` 默认设为 `true`。

`NewNode` 会统一规范化 identity 和 view 字段，并在返回领域对象前完成校验。

## 菜单树规则

- `SortSiblings` 对同级节点按 `Sort` 升序稳定排序，相同排序值保留原输入顺序。
- `FilterVisible` 过滤 `Visible=false` 的隐藏节点。
- `FilterAuthorized` 根据 `AccessContext` 过滤缺少角色或权限的节点。
- 排序和过滤函数都返回新切片，不修改调用方传入的节点列表。

## 权限字段

- `RequiredRoles`: 节点所需角色列表，空列表表示不要求角色。
- `RequiredPermissions`: 节点所需权限列表，空列表表示不要求权限。
- `AccessContext` 会对传入角色和权限做 trim/lower 规范化。
- 节点同时配置角色和权限时，访问上下文必须同时满足全部要求。

## 无兼容规则

- 不保留旧菜单来源或旧路由映射。
- 不在 domain 包内引入前端路由、store、service、handler、GORM model 或插件加载实现。
- 后续 service/store 层只能依赖本包公开的领域类型与规则，不反向污染 domain 边界。
