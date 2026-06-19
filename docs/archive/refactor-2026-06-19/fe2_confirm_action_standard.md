# FE2 Confirm Action Standard

日期: 2026-06-19  
范围: FE2-05  
状态: Accepted baseline

## 目标

统一 Skoll 前端危险动作确认交互。删除、禁用、卸载、重置、回滚、发布、清空、覆盖保存等可能造成数据、权限、配置或运行状态变化的动作，必须通过 `confirmAction` 阻断确认，不允许页面直接调用 `ElMessageBox` 或浏览器原生 confirm。

## 当前基础

| 文件 | 当前职责 | 结论 |
|---|---|---|
| `web/src/composables/useConfirmAction.ts` | 封装 Element Plus `ElMessageBox.confirm`，返回 `Promise<boolean>` | 保留为唯一确认动作 helper。 |
| `web/src/views/Plugin/index.vue` | 插件 disable/uninstall、Dev Portal 发布/灰度/回滚等确认 | 作为高风险操作参考。 |
| `web/src/views/Setting/index.vue` | 系统设置 reset 确认 | 作为系统配置重置参考。 |
| `web/src/views/User/list.vue`、`Role/list.vue`、`Menu/index.vue` 等 | 删除、保存、可见性变更确认 | 作为 CRUD 危险操作参考。 |

## Helper Contract

`confirmAction(options)` 必须保持:

- 输入为明确的 `title`、`message`、`confirmText`、`cancelText`。
- 返回 `true` 表示用户确认，`false` 表示取消或关闭。
- `danger: true` 时使用危险确认按钮样式。
- `type` 未指定时按 `danger` 推导 warning/info。
- 不直接执行业务动作，只返回确认结果。
- 不吞业务动作的错误；业务错误仍由调用方处理。

## 必须确认的动作

| 动作类别 | 示例 | 规则 |
|---|---|---|
| 删除/卸载 | 删除用户、删除角色、卸载插件 | 必须 `danger: true`，message 包含对象名称或 ID。 |
| 禁用/停用 | 禁用插件、隐藏菜单、停用权限资源 | 必须说明影响范围。 |
| 重置/清空 | 重置系统设置、清空日志、覆盖配置 | 必须说明不可逆或会覆盖当前值。 |
| 发布/回滚 | Dev Portal 发布、灰度、回滚 | 必须说明目标环境、版本或比例。 |
| 高影响保存 | 菜单树保存、权限分配、批量变更 | 如果会影响多个用户或全局配置，必须确认。 |

## 页面调用规则

- 页面只 import `confirmAction`，不 import `ElMessageBox`。
- 业务函数在执行 API 前先 `await confirmAction(...)`；用户取消时直接 return。
- 确认通过后再设置 `operating/saving/loading`，避免取消动作触发加载态。
- message 必须包含操作对象、影响范围或环境信息，不使用泛泛的 “Are you sure?”。
- 确认文案使用当前页面 i18n 或已有公共文案，不在 helper 中硬编码业务语言。
- helper 不处理权限；按钮权限仍由 `useButtonAccess` 或 `v-permission` 控制。

## 禁止项

- 禁止页面直接调用 `ElMessageBox.confirm`、`window.confirm` 或自建确认弹窗。
- 禁止确认通过前发起 API 请求。
- 禁止把取消操作记录为失败错误。
- 禁止危险动作使用普通 info 类型且没有 `danger: true`。
- 禁止在 helper 中写插件、用户、角色、菜单等业务分支。

## 当前缺口清单

| 模块 | 缺口 | 后续归属 |
|---|---|---|
| Plugin Dev Portal | 发布/灰度/回滚确认已接入，但页面较大，后续拆分后需保持 helper 入口 | FE4 Plugin 页面拆分。 |
| Menu/Permission | 高影响保存确认已存在，后续需要补充更细的影响范围文案 | FE3 Menu/Permission 页面升级。 |
| Audit export/cleanup | 当前导出不属于危险动作；如后续加入清理/归档必须接入确认 | FE2-07 或 M2 audit 后续任务。 |
| Bulk actions | 后续批量用户/角色/权限变更必须在提交前使用 `confirmAction` | FE3 User/Role 页面升级。 |

## FE2-05 验收结论

- Helper boundary: Passed，`ElMessageBox` 只允许在 `useConfirmAction.ts` 内出现。
- Dangerous actions: Passed，删除、禁用、重置、发布、回滚等动作必须使用 `confirmAction`。
- Page usage: Passed，现有 Audit、Dictionary、Menu、Organization、Plugin、Permission、Setting、Role、User 页面均已有 helper 锚点。
- No duplicate modal: Passed，不新增页面私有确认弹窗或旧兼容路径。
