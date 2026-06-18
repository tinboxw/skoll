# FE2 Error Display Standard

日期: 2026-06-19  
范围: FE2-06  
状态: Accepted baseline

## 目标

统一 Skoll 前端错误展示和传播规则，确保后端错误不会被吞掉，不会把失败操作显示为成功，也不会在页面中散落不一致的错误文案。JSON API 错误由 `ApiError` 表达，页面和 store 使用 `toErrorMessage` 转换为用户可见信息。

## 当前基础

| 文件 | 当前职责 | 结论 |
|---|---|---|
| `web/src/utils/api.ts` | 将 network、HTTP 和 401 场景转换为 `ApiError`，401 触发登录跳转 | 保留为底层错误来源。 |
| `web/src/utils/common.ts` | `toErrorMessage` 将 `ApiError`、JS Error 和 fetch error 转换为 i18n 文案 | 保留为页面错误文案入口。 |
| `web/src/audit/api.ts` | Blob/export 失败转换为 `ApiError` | 作为导出错误参考。 |
| `web/src/stores/permissions.ts`、`navigation.ts` | store 设置 `lastError` 并重抛错误 | 作为 shared async state 错误参考。 |
| `web/src/views/*` | 页面将 catch 结果写入 `error.value`、`profileError`、`bindError` 等 | 后续页面升级继续收敛为同一展示模式。 |

## Error Contract

- JSON API 失败必须抛出 `ApiError` 或能被 `toErrorMessage` 识别的 Error。
- 页面 catch 后必须展示错误、写入局部错误状态、写入 store `lastError`，或显式重抛。
- Store catch 后如果是共享异步状态，必须设置 `lastError` 并向调用方重抛，除非这是明确的 optional fallback。
- 操作成功提示只允许在 API 成功返回后设置，不能在 `finally` 中设置成功态。
- 401、403、404、500、network error 必须保留可区分语义。
- Forbidden/no-permission 可以用 `ApiError.status === 403` 或权限 helper 判断，并进入无权限态。

## 允许静默的场景

以下场景允许空 catch，但必须是局部容错，不得代表业务操作成功:

- 读取 localStorage、sessionStorage、历史 tab 等本地持久化失败。
- 可选 schema、可选插件页面、可选远程资源加载失败后回退到明确 fallback。
- JSON parse 或 RegExp parse 用于校验规则探测，失败后返回字段错误。
- 后台 bootstrap 日志已经通过 console 或 store 状态记录，且不影响当前页面继续使用。

不允许静默的场景:

- 用户点击后的保存、删除、禁用、发布、回滚、导出。
- 页面主数据加载失败。
- 权限、角色、菜单、配置、插件生命周期操作失败。
- 导入、批量创建、批量绑定等部分失败。

## 页面展示规则

- 页面级错误使用 `el-alert`、`el-result` 或 FE2-08 后续统一状态组件展示。
- 数据区错误必须保留 retry/refresh 入口。
- Drawer/dialog 内错误留在当前 drawer/dialog，不覆盖主页面错误，除非影响主页面数据一致性。
- 批量操作错误需要展示失败数量和失败原因，不只显示第一条成功提示。
- 表单校验错误与后端保存错误分开: 前者阻止提交，后者在提交失败后展示。

## Store 错误规则

- `syncStatus = "loading"` 前清空 `lastError`。
- catch 时设置 `syncStatus = "error"` 和 `lastError`。
- catch 后重抛错误，让页面决定 alert、toast、drawer error 或 retry。
- `retry/refresh` 必须复用 `lastQuery`，不能清空用户上下文。
- Store 不直接调用 Element Plus message。

## 成功态规则

- 成功提示、`success.value`、`info.value` 只能在 API 成功后设置。
- 如果后续 refresh 失败，不能保留刚才操作的全量成功文案；应展示“操作已提交但刷新失败”或直接显示刷新错误。
- finally 只用于关闭 loading/operating/saving，不设置成功或错误文案。
- JSON parse、权限检查、前置校验失败不进入 API 成功态。

## 当前缺口清单

| 模块 | 缺口 | 后续归属 |
|---|---|---|
| Page-local errors | 多数页面已有 `error.value = toErrorMessage(e)`，但布局和 retry 入口不完全统一 | FE2-08 状态组件与 FE3 页面升级。 |
| Store lastError | `navigation.ts` 使用 raw `error.message`，后续可改为 `toErrorMessage` 或统一 store helper | FE2 store 后续或 FE3 Menu 页面升级。 |
| Optional fallback | `Setting` schema fallback、插件远程页面 fallback 有空 catch，需要在页面体验中保持可见 fallback 说明 | FE3 Setting/FE4 Plugin。 |
| Batch operations | User batch 已记录行级失败，后续需要统一失败摘要组件 | FE3 User 页面升级。 |
| Console-only bootstrap | `main.ts` bootstrap failure 目前 console 记录，后续可接入 shell 状态提示 | FE5 shell/polish。 |

## FE2-06 验收结论

- ApiError source: Passed，底层 request 和 audit export 均有 `ApiError` 锚点。
- Message helper: Passed，页面错误文案通过 `toErrorMessage` 收敛。
- No fake success: Passed，标准要求成功态只在 API 成功后设置，finally 只关闭 loading。
- Store propagation: Passed，shared async store 必须设置 `lastError` 并重抛。
- Silent catch boundary: Passed，明确可静默和不可静默场景。
