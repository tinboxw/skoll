# FE1 Skoll Admin Page Layout Standard

日期: 2026-06-19  
范围: FE1-02  
状态: Accepted baseline

## 目标

本标准把 FE1-01 的视觉原则落实为后台页面布局合同。后续 Dashboard、User、Role、Permission、Menu、Plugin、Audit、Setting 和插件宿主页面，必须优先使用同一页面骨架组织标题、工具栏、筛选、表格、详情和抽屉。

## 标准页面骨架

所有常规后台页面默认使用以下顺序:

1. Page shell: 页面根节点使用 `*-page` 类，内部使用 grid/flex gap 组织，不在页面根节点外再套卡片。
2. Page header: 左侧为页面标题和状态摘要，右侧为页面级主操作。
3. Status alerts: 错误、成功、权限限制等页面级反馈紧贴 header 下方。
4. Filter panel: 查询条件、视图切换、快速筛选、批量范围选择集中在同一面板。
5. Data panel: 表格、列表、树表或主要数据视图占据页面主体。
6. Detail surface: 详情使用 drawer、side inspector 或页面内详情区，避免弹窗承载长文本。
7. Dialog surface: 只用于短流程确认、小表单或不可离开上下文的阻断操作。

## 区块规则

| 区块 | 标准结构 | 验收要求 |
|---|---|---|
| Page header | `page-header` + title block + `header-actions` | 标题、说明、刷新、新建、导出等页面级动作位置稳定；窄屏可换行。 |
| Toolbar | 操作按钮、视图切换、批量动作、状态 chips | 主操作靠右或贴近数据上下文；危险动作不与普通刷新混淆。 |
| Filter panel | `el-form` top labels + responsive grid + `filter-actions` | 查询、重置、清空局部条件、快速标签都在筛选区内；提交按钮可预测。 |
| Data panel | `el-table` / tree table / list + pager | 主数据区域只承载数据和局部状态；表格空态、加载、无权限不能脱离数据区。 |
| Detail drawer | `el-drawer` + descriptions + sections + pre/code blocks | 审计 trace、插件日志、配置 diff 等长内容进入抽屉或 side inspector。 |
| Dialog | `el-dialog` + concise form/confirmation | 删除、禁用、回滚、发布等危险操作必须确认；长流程转 drawer 或页面。 |
| Plugin host | shared shell + plugin content area | 插件页面保留 Skoll 导航、顶部状态、页签和权限上下文。 |

## Page Header

页面 header 必须回答三个问题: 当前在哪、当前状态是什么、现在能做什么。

- 标题使用 `h2` 或同级工具标题，不使用 hero 级字号。
- 标题下方只放业务上下文摘要，例如同步状态、筛选范围、当前对象范围。
- `header-actions` 只放页面级动作: refresh、create、export、import、sync、open dev action。
- 行内动作不放到 header；表格行操作必须留在行内或详情区。
- Header 在窄屏下变为单列，操作按钮可换行但不能覆盖标题。

## Toolbar

Toolbar 用于承载“当前视图”的动作，而不是解释功能。

- 标签页或 segmented 控件用于同一数据集的分类视图，例如 Audit type tabs。
- 批量操作只在有选择状态时出现，或以 disabled 形式表达不可用原因。
- 危险动作使用 `type="danger"` 或 warning 语义，并经 `confirmAction` 保护。
- 使用图标时优先选择现有 Element Plus icon 或 lucide icon，避免手写图形。

## Filter Panel

筛选区必须紧凑、可扫、可重置。

- 默认使用 top label，三列或两列响应式 grid；宽字段可以跨列。
- 日期范围、风险等级、状态、来源、角色、插件 ID 等筛选项要靠近同类条件。
- 搜索、刷新、清除局部条件放在 `filter-actions`，不要散落到表格上方多个角落。
- 快速标签放在筛选面板底部，使用 wrap 布局，不因长文本撑破容器。
- 筛选提交必须同步到当前数据请求或 URL query；不得展示假筛选。

## Data Panel

数据面板是运营后台的视觉重心。

- 表格必须在同一面板内处理 loading、empty、error、no-permission。
- 分页紧贴表格下方，保持 `prev/pager/next/sizes/total` 的稳定顺序。
- 树表、权限矩阵、插件列表等重数据页面可以使用 side inspector 分摊详情。
- 行操作使用 compact buttons 或 dropdown；超过三项优先折叠菜单。
- 面板不能嵌套卡片；需要分组时使用 tabs、collapse、border 或标题分区。

## Details And Drawers

详情布局优先选择非阻断式 drawer 或 inspector。

- `el-drawer` 适用于审计详情、插件配置、风险报告、发布任务详情、日志。
- 抽屉宽度默认使用 `min(560px, 92vw)` 或百分比，并保证窄屏不溢出。
- 结构化数据用 `el-descriptions`；长 JSON、trace、日志使用 `pre` 且允许换行。
- Drawer 内可有 tabs，但每个 tab 要对应明确任务: detail、steps、logs、config。
- 保存中、加载中、失败状态必须在 drawer 内可见。

## Dialogs

Dialog 只用于短而明确的交互。

- 删除、禁用、回滚、发布、清空日志等危险动作必须使用统一确认。
- 小型创建或编辑表单可以使用 dialog；超过一屏、需要长说明或多区块时改 drawer。
- Dialog 按钮顺序保持取消在前、确认在后；危险确认使用 danger/warning。
- Dialog 关闭后必须明确刷新或保留当前列表状态。

## Responsive Layout

- 常规页面在 960px 以下 header、actions、filters 变为单列或两列。
- 行操作和 toolbar 必须 `flex-wrap`，按钮文本不得遮挡相邻控件。
- 详情 drawer 使用 viewport 宽度约束，长 code/log 使用内部滚动或换行。
- 插件宿主页面保持高度约束，远端 iframe 不撑破 Skoll shell。

## 当前页面锚点

- Audit 页面已经具备较好的标准骨架: `page-header`、`header-actions`、`panel`、`filters`、`filter-actions`、`el-table`、`el-drawer`。
- Plugin 页面具备复杂后台布局锚点: `summary-grid`、`content-grid`、`devportal-grid`、`inspector-card`、DevPortal task drawer。
- Layout shell 当前由 `Sidebar`、`HeaderBar`、`PinnedTabs`、`MainContent` 组成，是系统页和插件页的共同外壳。

## FE1-02 验收结论

- Page header: Passed，定义了标题、摘要、页面级动作和窄屏规则。
- Toolbar: Passed，定义了视图切换、批量动作、危险动作和图标规则。
- Filter panel: Passed，定义了 top label、grid、filter-actions、快速标签和真实查询要求。
- Data panel: Passed，定义了表格、分页、状态、行操作和无嵌套卡片要求。
- Detail drawer: Passed，定义了抽屉、inspector、结构化详情和长文本规则。
- Dialog: Passed，定义了短流程边界、危险确认和关闭后的刷新关系。
