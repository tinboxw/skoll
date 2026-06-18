# FE1 Responsive Minimum Standard

日期: 2026-06-19  
范围: FE1-07  
状态: Accepted baseline

## 目标

Skoll Admin 的响应式标准不是把重后台页面变成移动端应用，而是保证窄屏检查时没有重叠、没有按钮溢出、没有抽屉和表格撑破 shell，且关键操作仍可理解和恢复。

## 最低视口

| 视口 | 用途 | 最低要求 |
|---|---|---|
| 390px width | 窄屏人工验收下限 | 标题、按钮、筛选、表格容器、drawer/dialog 不重叠。 |
| 860px width | Sidebar 折叠/隐藏断点 | Shell 切换为单列；顶部状态不遮挡内容。 |
| 960px width | 常规表单/筛选断点 | Header、actions、filters 从多列降为单列或两列。 |
| Desktop width | 高频使用主视口 | 表格、筛选、详情区保持高密度可扫。 |

## 通用规则

- 页面 header、toolbar、filter actions、row actions 必须支持 `flex-wrap` 或单列降级。
- grid 布局使用 `minmax(0, 1fr)`，避免内容撑破列。
- 长 ID、路径、邮箱、token、trace、plugin ID 使用 tooltip、ellipsis、word-break 或详情区。
- 操作按钮文本不得覆盖相邻按钮；按钮组在窄屏下可以换行。
- 任何响应式策略都不能隐藏错误、无权限、保存中和危险确认状态。

## 表格降级

- 表格可以横向滚动，但页面主体不能整体出现不可控横向溢出。
- 关键列保持稳定: identity/name/status/actions 优先保留。
- 宽字段使用 `min-width`、`show-overflow-tooltip`、drawer 详情或 inspector，不在 cell 内展示长 JSON。
- 权限矩阵、菜单树表、插件任务表等复杂表格在窄屏下至少保证可滚动、可查看状态、可取消危险动作。
- 表格分页和批量动作在窄屏下换行，不遮挡数据行。

## Drawer 和 Dialog 降级

- Drawer 宽度使用 `min(560px, 92vw)` 或等价 viewport 约束；百分比 drawer 需要窄屏检查。
- Drawer 内 long code/log/pre 必须 `white-space: pre-wrap`、`word-break: break-word` 或局部滚动。
- Dialog 宽度不超过 viewport；超过一屏的内容转 drawer 或独立页面。
- Drawer/dialog 关闭后保留列表上下文和筛选状态。

## Shell 和插件宿主

- 860px 以下 Sidebar 隐藏或转为非遮挡导航，内容区使用单列。
- PinnedTabs 使用 overflow menu 或横向滚动，不挤压 Header。
- Plugin host route 保持高度约束，远端 iframe 不能撑破 content shell。
- 插件详情错误、加载、无权限状态必须留在 Skoll shell 内。

## 当前代码锚点

- `App.vue`: 860px 以下 app shell 单列；plugin content area 使用固定高度网格。
- `Sidebar.vue`: 860px 以下隐藏 sidebar。
- `PinnedTabs.vue`: 支持 overflow menu、横向滚动和 860px 单列。
- `Audit/index.vue`: 960px 以下 header/actions/filters 降级，drawer 使用 `min(560px, 92vw)`。
- `Dictionary/index.vue`、`Permission/index.vue`、`Plugin/index.vue`: 使用 `minmax` grid 和断点降级复杂布局。
- `Common/Table.vue`: legacy table shell 使用 overflow auto，可作为旧表格横向滚动锚点，但后续不扩展旧 wrapper。

## 浏览器验收要求

后续可视化页面任务必须记录:

- Desktop viewport 和 390px viewport。
- Header/actions/filter/table/drawer/dialog 是否重叠。
- 表格是否有可控滚动或降级策略。
- 长文本是否截断、换行或进入详情区。
- 危险确认、错误、无权限、保存中状态是否仍可见。

## FE1-07 验收结论

- Narrow viewport: Passed，定义 390px 下无重叠和无按钮溢出要求。
- No overlap: Passed，定义 header、toolbar、filter、row actions、drawer/dialog 的换行和约束规则。
- Table fallback: Passed，定义横向滚动、关键列保留、长字段详情化和分页换行策略。
- Drawer/dialog fit: Passed，定义 viewport 宽度、长文本和关闭上下文要求。
- Plugin shell: Passed，定义 Sidebar、PinnedTabs、Plugin host 和 iframe 的降级要求。
