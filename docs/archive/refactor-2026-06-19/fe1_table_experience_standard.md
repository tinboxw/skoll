# FE1 Skoll Admin Table Experience Standard

日期: 2026-06-19  
范围: FE1-03  
状态: Accepted baseline

## 目标

表格是 Skoll Admin 的主要工作界面。所有核心页面和插件管理页面的表格必须支持快速扫描、稳定比较、低成本操作和明确状态反馈。本标准承接 FE1-01 的 Scannable 原则和 FE1-02 的 Data panel 规则。

## 表格基本合同

| 项目 | 标准 | 验收要求 |
|---|---|---|
| Stable columns | 关键列顺序稳定，ID/名称/状态/时间/操作有固定位置。 | 用户在不同页面切换时不需要重新学习列组织。 |
| Row key | 有唯一 ID 的表格必须设置 `row-key`。 | 刷新、展开、选择和详情跳转不会错位。 |
| Overflow | 长 ID、路径、邮箱、trace、plugin ID 使用 `show-overflow-tooltip` 或明确换行。 | 文本不挤压状态列和操作列。 |
| Status tags | 状态、风险、权限、加密、启停使用 `el-tag`，语义色统一。 | 用户能一眼区分 enabled、disabled、warning、danger、success。 |
| Actions | 行操作靠右，三项以上使用 dropdown 或紧凑分组。 | 操作列不撑爆表格，危险动作需要确认。 |
| Bulk actions | 批量动作只在支持选择时出现，并与选择状态绑定。 | 禁止展示不会生效的批量按钮。 |
| Empty/loading/error | 数据区内处理 loading、empty、error、no-permission。 | 用户不因状态变化丢失页面上下文。 |

## 标准列顺序

常规资源表推荐列顺序:

1. Identity: ID、key、code、path、plugin ID。
2. Name: 名称、标题、显示名。
3. Classification: 类型、来源、模块、所属组织、角色。
4. Status: enabled/disabled、risk、result、encrypted、visible。
5. Metrics: 用户数、权限数、插件数、任务数、排序值。
6. Time: createdAt、updatedAt、occurredAt、lastSyncedAt。
7. Actions: view/edit/config/enable/disable/delete/export 等固定右侧操作。

如果页面有矩阵、树表或审批任务，可以调整列顺序，但必须保持 identity、status、actions 可快速定位。

## 状态标签

- `success`: enabled、active、synced、approved、completed。
- `info`: disabled、inactive、idle、draft、no data。
- `warning`: pending、syncing、medium risk、sensitive、encrypted、needs review。
- `danger`: failed、critical risk、forbidden、blocked、delete/uninstall impact。
- `primary`: selected、current、default、system-owned。

状态标签必须表达真实状态，不用作装饰。风险标签在 Permission、Plugin、Audit、Dev Portal 中使用同一语义。

## 行操作

- 常用低风险操作可以直接显示: view、edit、config、visit。
- 启用/禁用、删除、卸载、回滚、发布、清空等危险或高影响操作必须使用确认。
- 行操作超过三项时进入 dropdown，保留一个最常用主操作在外部。
- 操作列默认固定右侧；宽度要足够容纳当前页面最常见按钮组合。
- 权限不足的行操作可以禁用或隐藏，但不能显示为可点击后静默失败。

## 批量动作

- 支持批量时使用 selection 列，并在 toolbar 中显示已选数量。
- 批量启停、删除、授权、导出必须确认影响范围。
- 批量动作执行中要锁定相关按钮，结束后刷新当前筛选结果。
- 不支持批量的页面不要展示“批量”占位按钮。

## 分页与数据量

- 服务端分页优先；大列表不得默认一次拉取大量数据。
- 分页位于表格下方，和 FE1-02 的 pager 规则一致。
- 表格应保留稳定高度或清晰的滚动容器，避免 loading/empty 导致页面大幅跳动。
- Plugin/Dev Portal 日志、任务、项目列表可以使用 `max-height` 控制局部滚动。

## 详情入口

- 行点击打开详情时，必须有 hover/current-row 或明确 affordance。
- 详情内容进入 drawer 或 side inspector，表格只保留可扫描摘要。
- 长 JSON、sourceData、diff、日志不放在表格 cell 内。
- 详情加载失败要在页面或 drawer 内给出错误，不清空用户当前列表上下文。

## 当前页面锚点

- Audit: `row-key="id"`、`show-overflow-tooltip`、action tag、分页、详情 drawer。
- Plugin: 多个 `el-table`、状态 tag、右侧 fixed actions、DevPortal 任务详情 drawer。
- Menu/Dictionary/Organization/Role/User: 树表、固定右侧操作列、状态标签和 row-key 使用广泛。
- Common `Table.vue`: 仍是 legacy wrapper，后续 FE2/FE3 应优先使用 Element Plus table 合同而不是扩展旧 table shell。

## FE1-03 验收结论

- Stable columns: Passed，定义了 identity、name、classification、status、metrics、time、actions 列顺序。
- Status tags: Passed，定义了 success/info/warning/danger/primary 的后台语义。
- Bulk actions: Passed，定义了 selection、已选数量、确认、锁定和刷新规则。
- Compact row actions: Passed，定义了右侧固定操作列、三项以上 dropdown、危险确认。
- Data states: Passed，定义了 loading、empty、error、no-permission 必须留在数据区内。
