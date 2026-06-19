# FE1 Skoll Admin State Component Standard

日期: 2026-06-19  
范围: FE1-05  
状态: Accepted baseline

## 目标

Skoll Admin 的状态反馈必须一致、可见、可恢复。每个核心页面、插件页面、drawer、dialog 和表格数据区都要明确 normal、loading、empty、error、no-permission、success、failed 状态，不能只依赖 build 通过或默认组件文案。

## 状态矩阵

| 状态 | 推荐组件 | 出现位置 | 验收要求 |
|---|---|---|---|
| Loading | `v-loading`、`el-skeleton`、button loading | 数据区、详情 drawer、提交按钮 | 重复动作被禁用；旧数据不会被误认为新数据。 |
| Empty | `el-empty`、table `empty-text` | 表格、列表、inspector、drawer tab | 空态说明当前筛选或当前对象无数据，不像错误。 |
| Error | `el-alert type="error"`、drawer 内错误块 | 页面 header 下、数据面板内、drawer 内 | 错误摘要可见，重试路径仍可用。 |
| No permission | `el-result icon="warning"`、disabled controls、permission hint | 页面主体、数据面板、行操作 | 用户知道是权限限制，而不是数据消失。 |
| Success | `el-alert type="success"`、success tag、message | 页面 header 下、表单提交后、任务提交后 | 成功说明真实结果，不能和保存中混淆。 |
| Failed | danger tag、error alert、timeline danger item | 任务表、Dev Portal、插件生命周期、审计结果 | 失败原因可见，并保留重试、查看日志或详情入口。 |

## Loading

- 页面刷新和表格查询使用 `v-loading` 覆盖数据区。
- 详情、日志、trace、配置等二级内容使用 `el-skeleton` 或局部 loading，不阻塞整个页面。
- 保存、导出、安装、发布、回滚等操作按钮使用 loading state，并禁用同一资源重复提交。
- Loading 结束后必须进入 normal、empty、error 或 no-permission 之一。

## Empty

- 空态只说明当前范围无数据，不加入营销式说明或功能介绍。
- 表格空态优先使用 table `empty-text`；整块无数据可以使用 `el-empty`。
- 筛选导致空结果时保留筛选条件和重置入口。
- Inspector 或 drawer 的空态要说明用户需要先选择对象，不能显示空白面板。

## Error

- 页面级错误放在 header 下方，使用 `el-alert type="error"`。
- 数据区错误应保留页面外壳、筛选和操作上下文。
- Drawer 内加载详情失败时，错误必须显示在 drawer 内或页面错误区，并保留关闭能力。
- 后端错误使用 `toErrorMessage` 或同等路径展示摘要，不吞掉真实错误。

## No Permission

- 无权限页面或数据区使用 `el-result icon="warning"` 或明确权限提示。
- 无权限按钮可以隐藏或 disabled，但页面不能静默消失。
- 受限角色验收必须覆盖页面访问、行操作、危险动作和导出/保存。
- 插件页面无权限要保留 Skoll shell、同步状态和可返回导航。

## Success And Failed

- Success 用于已经完成的真实操作，例如保存成功、导出生成、任务提交、同步完成。
- Failed 用于已经结束且失败的流程，例如发布失败、回滚失败、插件校验失败。
- Pending/running 不应显示为 success；需要使用 warning/info/primary 或 loading。
- Dev Portal timeline、插件生命周期、审计结果要区分 success、failed、running、pending。

## 状态放置规则

- 页面级状态: header 下方 alerts。
- 数据区状态: table/list/panel 内部。
- 表单状态: 表单顶部或字段级错误。
- Drawer 状态: drawer 内容顶部或对应 tab 内。
- 行级状态: tag 或 timeline item，不用长段文案撑高行。
- 全局 shell 状态: Header sync chips 或 plugin count chips，不覆盖页面状态。

## 当前代码锚点

- Audit: `el-alert`、`el-result` forbidden、`el-empty`、`v-loading`、`el-skeleton`。
- Plugin: page alerts、forbidden alert、DevPortal timeline danger/success、task drawer empty states。
- Dictionary/Organization/Permission/Role/Setting/User: error/success alerts、table loading、empty text。
- FE0 manual template: normal、loading、empty、backend error、no permission、dangerous action、save success/failure、detail/drawer/dialog 状态验收。

## FE1-05 验收结论

- Loading: Passed，定义了数据区、详情、按钮和状态收敛规则。
- Empty: Passed，定义了表格、整块、筛选结果和 inspector/drawer 空态。
- Error: Passed，定义了页面级、数据区、drawer 和后端错误展示。
- No permission: Passed，定义了受限页面、按钮、行操作、插件页权限态。
- Success/failed: Passed，定义了真实成功、失败结束、running/pending 区分。
