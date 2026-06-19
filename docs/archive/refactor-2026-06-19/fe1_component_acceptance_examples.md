# FE1 Component Acceptance Examples

日期: 2026-06-19  
范围: ADJ-FE-20260619-01  
状态: Accepted baseline

## 目标

本文件把 FE1 视觉、布局、表格、表单、状态和响应式规范转换为后续 FE3/FE4 页面升级可直接引用的组件验收示例。示例不是新 UI 设计稿，也不引入新组件库；它们是页面改造时的最低验收合同。

## 示例矩阵

| 示例 | 适用页面 | 必须引用的 FE1 规范 | 验收重点 |
|---|---|---|---|
| 列表页 | User、Role、Plugin、Audit、Setting | FE1-02、FE1-03、FE1-05、FE1-07 | header、filter、table、pagination、loading/empty/error/no-permission、窄屏不重叠 |
| 编辑页 | User add/edit、Role edit、Setting edit | FE1-02、FE1-04、FE1-05、FE1-07 | top-label form、校验、保存中、后端错误、成功反馈、返回上下文 |
| 详情抽屉 | Audit detail、Plugin config/logs、Dev Portal task | FE1-02、FE1-04、FE1-05、FE1-07 | drawer 宽度、skeleton、descriptions、long JSON/log、关闭保留列表状态 |
| 空态 | Menu、Dictionary、Plugin inspector、Audit filters | FE1-05、FE1-07 | 空态在数据区内，保留筛选/选择上下文，不写营销说明 |
| 错误态 | 所有 API 页面 | FE1-04、FE1-05 | `el-alert` 可见，错误摘要保留，重试路径可用，不显示假成功 |
| 无权限态 | Permission、Plugin、Audit、Role/User actions | FE1-03、FE1-05 | 页面或控件明确权限限制，不能静默消失或点击后无反馈 |

## 列表页示例

适用: User list、Role list、Plugin list、Audit list、Setting list。

最低结构:

1. `page-header`: 左侧标题与摘要，右侧 refresh/create/export 等页面级动作。
2. `filter panel`: top-label form，筛选、重置、快速标签和批量范围在同一区块。
3. `data panel`: `el-table`、分页、loading、empty、error、no-permission 留在数据区域。
4. `row-actions`: 最多三项直接显示，更多进入 dropdown，危险动作确认。
5. `responsive`: 390px 下 header/actions/filter/table/pager 不重叠。

验收清单:

- 正常数据: 表格显示真实数据，identity/name/status/actions 位置稳定。
- Loading: 刷新时数据区 loading，重复刷新/导出禁用。
- Empty: 当前筛选无结果时显示空态，筛选条件仍可修改。
- Error: 后端失败显示错误 alert，保留重试按钮或刷新入口。
- No permission: 页面主体或行操作显示权限限制，不静默消失。
- Narrow: 390px 下按钮换行，表格可控横向滚动或关键列可访问。

## 编辑页示例

适用: User add/edit、Role edit、Setting edit、Plugin config。

最低结构:

1. 页面或 drawer 内使用 top-label form。
2. 字段按 identity、basic、status、permissions、advanced 顺序组织。
3. 必填、长度、范围、pattern、enum 在提交前可见。
4. 保存按钮绑定 `saving`/`operating`，无效、权限不足或保存中禁用。
5. 成功和失败状态必须在当前上下文可见。

验收清单:

- 校验: 空必填、非法格式、超出范围均可见。
- Saving: 保存中按钮 loading，重复提交被禁用。
- Error: 后端错误显示摘要，用户输入不丢失。
- Success: 成功说明真实结果，并明确刷新、关闭或保留上下文。
- Dangerous action: 删除、禁用、回滚、清空等经过 `confirmAction`。

## 详情抽屉示例

适用: Audit detail、Plugin config/logs/debug、Dev Portal task detail。

最低结构:

1. `el-drawer` 使用 viewport 约束，例如 `min(560px, 92vw)` 或经窄屏验收的百分比。
2. 结构化字段用 `el-descriptions`。
3. trace、metadata、sourceData、diff、logs 使用 code block/pre，允许换行或局部滚动。
4. drawer 内有 loading、empty、error、no-permission 状态。
5. 关闭 drawer 后保留列表筛选、页码和当前 shell。

验收清单:

- Loading: 打开详情先显示 skeleton 或局部 loading。
- Detail: 关键字段可读，长文本不撑破 drawer。
- Error: 详情加载失败不清空列表。
- Narrow: 390px 下 drawer 不超过 viewport，关闭按钮可访问。
- Context: 关闭后返回原列表位置和筛选上下文。

## 空态示例

适用: Audit no rows、Plugin inspector no selection、Menu empty tree、Dictionary empty items。

最低结构:

- 空态必须位于原数据区域。
- 空态描述当前范围，不写产品介绍。
- 筛选空态保留筛选条件和重置入口。
- 选择型空态说明需要先选对象，例如 inspector/drawer tab。

验收清单:

- Empty state 不被误认为错误。
- 不显示陈旧数据。
- 用户有下一步: reset filter、refresh、create、select item。

## 错误态示例

适用: 所有 API 页面和 drawer。

最低结构:

- 页面级错误放在 header 下方 `el-alert type="error"`。
- 局部详情错误放在 drawer 或对应面板内。
- 错误来自 `toErrorMessage` 或同等统一路径。
- 错误不转换为 success，不只写 console。

验收清单:

- 错误摘要可见。
- 重试或返回路径可见。
- 表单输入和筛选上下文保留。
- 后续 acceptance log 记录失败原因和返工动作。

## 无权限态示例

适用: Audit forbidden、Plugin read/manage、Role/User protected actions、Permission matrix。

最低结构:

- 页面无权限使用 `el-result icon="warning"` 或明确权限提示。
- 按钮无权限可隐藏或 disabled，但不能显示为可点击后无反馈。
- 行级权限限制需要保留当前行上下文。
- 插件无权限必须保留 Skoll shell 和导航。

验收清单:

- 受限角色访问页面时有明确反馈。
- 受限按钮不会执行操作。
- 危险动作无法绕过权限确认。
- 无权限状态不破坏 layout 或窄屏布局。

## FE3/FE4 引用方式

后续每个页面升级任务至少选择对应示例:

- Dashboard: 列表页状态、空态、响应式示例。
- User/Role: 列表页、编辑页、无权限、危险动作示例。
- Permission/Menu: 列表/矩阵、编辑、无权限、响应式示例。
- Plugin: 列表页、详情抽屉、错误态、无权限、Dev Portal 任务示例。
- Audit: 列表页、详情抽屉、空态、错误态、无权限、导出动作示例。
- Setting: 列表页、编辑页、SchemaForm、错误态和敏感项保存示例。

## ADJ-FE-20260619-01 验收结论

- List page example: Passed，覆盖 header、filter、table、pagination 和状态。
- Edit page example: Passed，覆盖表单、校验、保存中、错误、成功和危险确认。
- Detail drawer example: Passed，覆盖 drawer、skeleton、descriptions、long JSON/log 和上下文保留。
- Empty state example: Passed，覆盖数据区空态、筛选上下文和下一步。
- Error state example: Passed，覆盖 alert、统一错误摘要、重试和返工记录。
- No-permission example: Passed，覆盖页面、按钮、行级限制和插件 shell。
