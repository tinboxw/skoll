# FE2 SchemaForm Standard

日期: 2026-06-19  
范围: FE2-04  
状态: Accepted baseline

## 目标

统一 Skoll `SchemaForm` 的使用边界，让插件配置、系统设置和后续生成器表单复用同一套 schema-driven 表单能力。`SchemaForm` 只负责字段渲染、基础校验和模型事件，不吞业务语义、不直接请求 API、不处理权限、不发起保存。

## 当前基础

| 文件 | 当前职责 | 结论 |
|---|---|---|
| `web/src/components/Common/SchemaForm.vue` | 渲染 `PluginConfigSchema.fields`，支持 string、textarea、number、boolean、select，暴露 `update:modelValue`、`update:valid`、`update:errors` | 保留为 schema-driven 表单唯一通用组件。 |
| `web/src/plugins/types.ts` | 定义 `PluginConfigSchema`、`PluginConfigField`、`PluginConfigOption` | 作为 FE2 期间通用 schema 类型源。 |
| `web/src/views/Plugin/index.vue` | 插件配置读取 schema/config，使用 SchemaForm 或 JSON fallback 保存 | 作为插件配置场景参考。 |
| `web/src/views/Setting/index.vue` | 系统配置 schema、fallback schema、批量保存 settings | 作为系统设置场景参考。 |
| 生成器页面 | 尚未实现通用 schema 表单 | 后续生成器任务复用同一 schema 语义，不新增第二套表单引擎。 |

## 组件职责

`SchemaForm` 允许:

- 根据 `schema.fields` 渲染字段。
- 根据 `field.type` 做基础输入控件选择。
- 根据 `required`、`min`、`max`、`minLength`、`maxLength`、`pattern` 做同步校验。
- 根据 `locale` 选择 label 和 option label。
- 通过 `update:modelValue` 输出完整浅层 model。
- 通过 `update:valid` 和 `update:errors` 输出校验状态。
- 根据 `disabled` 禁用内部控件。

`SchemaForm` 禁止:

- 直接调用 API client、store、router、ElMessage 或 confirm helper。
- 自行保存、重置、导出、加密、脱敏或弹出业务确认。
- 读取当前用户、权限、插件状态或系统配置状态。
- 存储页面级 loading/error/success。
- 渲染嵌套业务布局、详情抽屉、表格或危险操作按钮。

## Schema 字段契约

| 字段 | 规则 |
|---|---|
| `key` | 必填，宿主保存时作为配置 key 或 payload field；不得为空。 |
| `type` | 支持 `string`、`textarea`、`number`、`boolean`、`select`；未知类型按 string 处理。 |
| `default` | 可选；组件按字段类型做基础 coercion，宿主可以在加载后再次 apply defaults。 |
| `label/labelZhCN/labelEnUS` | 至少提供一个；缺失时 fallback 到 key。 |
| `placeholder/help` | 只作为辅助文案，不参与业务逻辑。 |
| `required/min/max/minLength/maxLength/pattern` | 仅做前端同步校验；后端仍必须执行最终校验。 |
| `options` | 仅用于 select；option value 为 string，label 按 locale fallback。 |

## 宿主页面职责

- 获取 schema 和初始 model。
- 合并 defaults，决定 schema 缺失时是否显示 JSON fallback 或禁用保存。
- 管理 loading、saving、error、success、dirty state。
- 根据权限决定入口、保存按钮和危险操作是否可用。
- 将 model 转换为 API request payload。
- 处理敏感字段、加密标记、审计提示、确认弹窗和后端错误。
- 在保存成功后刷新页面数据或 store。
- 将 `update:valid` 绑定到保存按钮禁用状态。

## 场景规范

### 插件配置

- 插件配置优先使用后端返回的 `configSchema`，其次使用 manifest `configSchema`。
- 有有效 schema 时使用 `SchemaForm`；无 schema 时可以保留 JSON editor fallback。
- `saveConfig` 负责决定提交 `configForm` 或 JSON parse 结果。
- 插件配置保存必须受 `plugin.manage` 保护；只读查看受 `plugin.read` 保护。
- 插件配置 schema 只描述配置字段，不描述插件生命周期动作。

### 系统设置

- 系统设置优先使用 `/v1/system/settings/schema` 返回 schema，失败时允许使用明确的 local fallback schema。
- 保存时由宿主页面逐项转换为 settings API payload，并保留 encrypted 标记。
- 搜索、分组、raw editor 与 SchemaForm 是同页不同区域，不放入 SchemaForm 组件内部。
- 敏感项展示、加密状态和审计提示由 Setting 页面处理。

### 生成器表单

- 生成器后续如需 schema-driven 表单，应复用 `PluginConfigSchema` 语义或先显式扩展类型。
- 不允许在生成器任务里新增与 `SchemaForm` 并行的临时表单生成器。
- 生成器 schema 可以扩展 field metadata，但渲染组件必须先支持并验收。

## 校验与保存规则

- 前端校验只用于即时反馈和保存按钮状态，不替代后端校验。
- `update:errors` 用于宿主页面展示摘要、日志或调试信息；当前宿主可只消费 `update:valid`。
- 保存按钮必须同时考虑 `valid`、`saving/loading` 和权限。
- 批量保存应记录失败反馈，不能把局部失败展示为全量成功。
- 后端返回字段级错误时，后续可扩展字段错误注入，但 FE2-04 不引入半成品兼容 API。

## 当前缺口清单

| 模块 | 缺口 | 后续归属 |
|---|---|---|
| `SchemaForm.vue` | 校验错误中文文案存在编码显示风险，需要后续统一 i18n 文案来源 | FE5 i18n/accessibility 或 FE3 页面升级时修复。 |
| `Plugin/index.vue` | 插件配置 API 和 schema/default 逻辑仍在大页面中 | FE4 Plugin 页面拆分时迁入 API/store/composable。 |
| `Setting/index.vue` | 系统设置 schema 保存为逐项串行请求，缺少字段级失败摘要 | FE3 Setting 页面体验升级。 |
| 生成器表单 | 尚无实际页面锚点 | FE4/Generator 任务实现时复用本标准。 |
| Schema 类型 | 目前复用 plugin types，未来 generator metadata 可能需要扩展 | 扩展前必须更新本标准和类型定义。 |

## FE2-04 验收结论

- Component boundary: Passed，SchemaForm 组件职责与禁止事项明确。
- Field contract: Passed，字段 key/type/default/label/validation/options 规则明确。
- Plugin config: Passed，schema 优先级、JSON fallback、权限和保存边界明确。
- System settings: Passed，remote schema、fallback schema、encrypted/save 边界明确。
- Generator reuse: Passed，后续生成器不得新增第二套临时表单引擎。
