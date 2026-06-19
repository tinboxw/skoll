# FE2 Pinia Store Standard

日期: 2026-06-19  
范围: FE2-02  
状态: Accepted baseline

## 目标

统一 Skoll 前端 Pinia store 的状态边界，确保 shared async state 使用一致的 loading、error、data、retry、refresh 模式。Store 负责组合 typed API client 和共享状态，不负责页面布局、DOM 下载、Element Plus message 或 router 跳转。

## 当前基础

| Store | 当前职责 | 结论 |
|---|---|---|
| `permissions.ts` | permission catalog items、`syncStatus`、`lastError`、`lastQuery`、`load/retry/refresh`。 | 作为 async list store 参考。 |
| `navigation.ts` | system menu tree、customized、sync status、save/reorder/visibility。 | 需要补齐统一 retry/refresh 语义。 |
| `plugins.ts` | plugin inventory、backend records、sync status、default home helpers。 | store 与 helper 混合，后续 plugin API 迁移时拆清 async 边界。 |
| `user.ts` | session/profile/permissions persistence 与 hydrate。 | 认证态 store，可保留 session 特例但应记录 loading/error 缺口。 |
| `tabs.ts` | pinned/recent tabs 持久化。 | 本地 UI state store，不需要 async 合同。 |
| `app.ts` | sidebar collapsed 等 UI shell state。 | 本地 UI state store，不需要 async 合同。 |

## Store 分类

| 类型 | 示例 | 必备字段 | 必备动作 |
|---|---|---|---|
| Async list/detail store | permissions、navigation、future users/roles/plugins | `items` 或 `data`、`syncStatus`、`lastError`、`lastLoadedAt`、`lastQuery` | `load`、`refresh`、`retry`、`clear` |
| Session/auth store | user | `profile`、`permissions`、session persistence | `hydrateProfile`、`setSession`、`logout` |
| Local UI store | app、tabs | local UI state | local mutations only |
| Domain workflow store | future Dev Portal/release tasks | `items`、`selectedId`、`syncStatus`、`lastError`、operation state | `load`、`runAction`、`refresh`、`retry` |

## Async Store Contract

推荐状态字段:

```ts
type AsyncStatus = "idle" | "loading" | "success" | "error";

type StoreState<TItem, TQuery> = {
	items: TItem[];
	syncStatus: AsyncStatus;
	lastError: string | null;
	lastLoadedAt: string | null;
	lastQuery: TQuery;
};
```

推荐 getter:

- `isLoading`: `syncStatus === "loading"`
- `hasError`: `syncStatus === "error"`
- `isReady`: `syncStatus === "success"`
- 领域查询 getter，例如 `byKey`、`enabledItems`、`bySource`

推荐 action:

- `load(query, options?: { force?: boolean })`
- `refresh()`: 使用 `lastQuery` 强制重载
- `retry()`: 使用 `lastQuery` 重试失败请求
- `clear()`: 清空数据并回到 `idle`
- 局部 mutation，例如 `patchEnabled`、`patchMenuVisibility`

## Loading/Error 规则

- `load` 开始时设置 `syncStatus = "loading"`，清空 `lastError`。
- 成功时更新 data、`syncStatus = "success"`、`lastLoadedAt`。
- 失败时设置 `syncStatus = "error"`、`lastError`，并向调用方重新抛出错误，页面才能显示上下文反馈。
- Store 不直接调用 `ElMessage`；页面或 composable 决定如何展示。
- Store 不把错误转换为空数组；空数组只表示成功返回空数据。

## Query/Cache 规则

- Store 负责保存 `lastQuery`，页面负责把 filter state 转换成 query。
- `load` 可在 query 未变且已有成功数据时跳过请求。
- `refresh/retry` 必须使用 `{ force: true }` 或等价逻辑。
- Query normalization 应靠近 store 或 API client，避免页面重复整理 offset/limit/status。
- 缓存失效必须显式，例如角色权限变更后刷新 permission/menu/user session。

## Data Boundary

- Store 存业务数据和共享异步状态，不存临时 form draft，除非 draft 被多个页面共享。
- 页面局部状态保留在 page: dialog open、drawer open、current tab、form dirty、local validation。
- 业务 API 类型优先从 feature API client 导入，不在 store 内复制。
- Flexible metadata 仍使用 `Record<string, unknown>`，不使用 `any`。

## Retry/Refresh UX Contract

- 页面显示 error 时必须能调用 `retry` 或 `refresh`。
- Refresh 不清空用户筛选上下文。
- Retry 不改变 `lastQuery`。
- Refresh 成功后页面保留当前 route/shell，除非认证态失效触发底层 401 redirect。
- 需要重新拉多个 store 时，由 page/composable 组合，不让 store 互相隐式调用形成循环。

## 当前缺口清单

| Store/Page | 缺口 | 后续归属 |
|---|---|---|
| `navigation.ts` | 有 load/save/reorder/visibility，但缺少统一 `retry/refresh/clear`。 | FE2-03/FE3 Menu 页面升级。 |
| `plugins.ts` | sync 状态存在，但 plugin lifecycle/dev portal API 仍在页面。 | FE4 plugin API/store 拆分。 |
| `user.ts` | hydrate profile 没有显式 loading/error。 | FE2 route/session 规范或后续 profile 页面升级。 |
| User/Role/Setting/Dictionary/Organization pages | 大量 page-local loading/error/data。 | FE3 页面升级时按页面迁入 store 或 composable。 |
| Audit page | typed API 已有，但 list/detail state 仍 page-local。 | FE3 Audit 页面升级时决定是否抽 store。 |

## FE2-02 验收结论

- Loading/error/data: Passed，定义 async store 必备状态和失败传播规则。
- Retry/refresh: Passed，定义 `lastQuery`、`refresh`、`retry`、`force` 和用户上下文保留。
- Store boundary: Passed，明确 store 不处理 DOM、message、router 和局部 form draft。
- Cache/query: Passed，明确 query normalization、缓存跳过和失效规则。
- Current gaps: Passed，记录 navigation/plugins/user/page-local 状态迁移缺口。
