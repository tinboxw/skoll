# FE2 State Block Standard

日期: 2026-06-19  
范围: FE2-08  
状态: Accepted baseline

## 目标

统一 Skoll 前端空态、错误态和无权限态的组件边界。状态组件服务数据区、drawer 和局部面板，不替代页面结构，不承载营销式说明，也不吞掉真实错误和权限信息。

## 当前基础

| 文件 | 当前职责 | 结论 |
|---|---|---|
| `web/src/components/Common/StateBlock.vue` | 统一 `empty`、`error`、`forbidden` 状态块，提供 actions slot | 作为 FE3 页面迁移的状态组件入口。 |
| `web/src/views/Audit/index.vue` | 接入 `StateBlock` 的空态和无权限态 | 作为页面迁移参考。 |
| `docs/refactor/fe1_state_component_standard.md` | 定义 normal/loading/empty/error/no-permission/success/failed 状态矩阵 | 作为体验规则来源。 |
| `docs/refactor/fe1_component_acceptance_examples.md` | 定义列表页、详情抽屉、空态、错误态、无权限态验收示例 | 作为 FE3 逐页验收参考。 |

## 组件职责

`StateBlock` 允许:

- 渲染 `empty`、`error`、`forbidden` 三种局部状态。
- 接收 `title` 和 `description`。
- 通过 `actions` slot 承载 refresh、retry、reset filter、create 等真实操作。
- 用于页面数据区、drawer tab、inspector 或局部面板。

`StateBlock` 禁止:

- 直接调用 API、store、router 或权限 helper。
- 自行决定 retry/refresh 逻辑。
- 隐藏真实错误、覆盖页面级 alert 或显示假成功。
- 包装整页 shell、header、filter、table 或业务表单。
- 写入长篇说明、功能介绍或营销式空态文案。

## 使用规则

- Loading 结束后必须进入 normal、empty、error 或 forbidden 之一。
- Empty 说明当前范围无数据，必须保留筛选上下文；建议提供 refresh、reset filter 或 create 动作。
- Error 说明请求或操作失败，必须提供 retry/refresh 或保留页面级错误入口。
- Forbidden 说明权限限制，不能让数据静默消失。
- 表格自身无数据可继续使用 `empty-text`；整块数据区无结果时使用 `StateBlock`。
- Drawer 内无详情、无日志、无步骤时可以使用 `StateBlock`，但不阻断关闭 drawer。

## FE3 迁移顺序

1. 先保留页面现有 loading/error/success 数据流。
2. 将整块 `el-empty` 或 `el-result` 替换为 `StateBlock`。
3. 将 retry/refresh/reset/create 等真实操作放入 `actions` slot。
4. 验证窄屏下 actions 不溢出、不遮挡后续内容。
5. 保留页面级 `el-alert` 用于后端错误摘要，除非错误态只影响局部数据区。

## 当前缺口清单

| 模块 | 缺口 | 后续归属 |
|---|---|---|
| Audit | 已接入 empty/forbidden，错误仍使用页面级 alert | FE3 Audit 页面升级时复查局部 error block。 |
| Menu | 已有 error alert + retry 和 empty block，尚未迁入 `StateBlock` | FE3 Menu 页面升级。 |
| Plugin | 多个 inspector/task drawer empty state 尚未迁入 `StateBlock` | FE4 Plugin 页面拆分。 |
| User/Role/Permission/Setting | 仍以 table `empty-text` 和 page alert 为主 | FE3 页面逐页迁移。 |

## FE2-08 验收结论

- Component boundary: Passed，新增 `StateBlock`，职责限定为局部 empty/error/forbidden。
- Audit reference: Passed，Audit 数据区空态和无权限态已接入 `StateBlock`。
- Action slot: Passed，空态提供 refresh action，后续页面可放 retry/reset/create。
- No fallback UI path: Passed，不新增旧页面路径或第二套状态系统。
