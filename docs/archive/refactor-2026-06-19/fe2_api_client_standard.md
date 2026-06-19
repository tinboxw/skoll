# FE2 API Client Standard

日期: 2026-06-19  
范围: FE2-01  
状态: Accepted baseline

## 目标

统一 Skoll 前端 API client 规范，避免页面散落拼接请求、重复 envelope 解析、错误处理不一致和导出行为分裂。后续 FE2/FE3 页面改造必须优先把业务请求收敛到 typed API client，再由页面或 store 调用。

## 当前基础

| 文件 | 现状 | 结论 |
|---|---|---|
| `web/src/utils/api.ts` | 提供 `apiGet/apiPost/apiPut/apiPatch/apiDelete`、`ApiResponse<T>`、`ApiError`、401 redirect。 | 保留为唯一底层 JSON request wrapper。 |
| `web/src/audit/api.ts` | typed audit list/detail/export client，含 query builder、normalize、Blob export。 | 作为新 typed client 参考。 |
| `web/src/navigation/api.ts` | typed menu tree/save/reorder/visibility client。 | 作为 registry/menu client 参考。 |
| `web/src/permissions/api.ts` | typed permission catalog list/detail/state/diff client。 | 作为 catalog client 参考。 |
| `web/src/views/*` | User/Role/Plugin/Setting/Dictionary 等页面仍有 page-local API 调用。 | 后续页面升级时逐步迁入 feature API client。 |
| `web/src/plugins/index.ts` | 远端 plugin page fetch 不属于 JSON API client。 | 可保留为 plugin asset/page loading 特例。 |

## 分层合同

| 层 | 允许做什么 | 不允许做什么 |
|---|---|---|
| `utils/api.ts` | 处理 base prefix、auth header、JSON envelope、401、network error、HTTP error。 | 放业务 endpoint 类型或页面状态。 |
| feature API client | 定义请求类型、响应类型、query builder、normalizer、export/download wrapper。 | 直接操作 Pinia 状态、DOM、router、message。 |
| Pinia store/composable | 组合 API client，管理 loading/error/data/retry/refresh。 | 拼接裸 endpoint 或重复 envelope 解析。 |
| Vue page | 调用 store/composable 或 typed client，处理 UI 状态和用户交互。 | 复制 API 类型、在多处散落相同 endpoint、吞掉错误。 |

## 命名与目录

- 领域级 client 使用 `web/src/<feature>/api.ts`，例如 `audit/api.ts`、`permissions/api.ts`、`navigation/api.ts`。
- 共享基础能力保留在 `web/src/utils/api.ts`。
- 插件/远端页面加载保留在 `web/src/plugins/index.ts`，但 JSON lifecycle/config/dev portal API 后续应迁入 `plugins/api.ts`。
- 类型靠近 client 定义；只有跨 feature 使用的类型才上移到共享目录。
- 请求方法命名使用业务动词: `listUsers`、`getUser`、`createUser`、`updateUser`、`deleteUser`、`exportAuditEvents`。

## 请求规范

- JSON 请求必须通过 `apiGet/apiPost/apiPut/apiPatch/apiDelete`。
- GET query 使用 `URLSearchParams` 和 helper 设置参数，不手写 `?a=${value}`。
- path 参数必须 `encodeURIComponent`。
- POST/PUT/PATCH body 使用明确 request type，不传页面 ref/raw form 对象。
- Empty string、undefined、null 的筛选语义必须在 client 中统一处理。

## 响应规范

- 所有 JSON API client 接收 `ApiResponse<T>` envelope。
- Client 对外返回业务 payload，而不是完整 envelope，除非页面确实需要 code/message。
- Client 内部做必要 normalizer，保证页面拿到稳定数组、字符串、数字和可选对象。
- Flexible metadata 使用 `Record<string, unknown>`，不使用 `any`。
- 分页响应统一表达 `items/offset/limit/total`，没有 `total` 时明确为 optional。

## 错误规范

- 底层 request 抛 `ApiError`；页面或 store 使用 `toErrorMessage` 展示。
- Client 不把失败转换成空数组或成功状态；空数组只表示后端成功返回空数据。
- 403/404 等特殊状态可以在页面或 store 通过 `instanceof ApiError && status` 判断。
- Network error 使用 `ApiError("network_error", 0, "network_error")`。
- Export/download 失败必须复用 ApiError 语义，不能只触发浏览器下载失败。

## 分页与筛选

- 列表 query type 必须包含 `offset`、`limit` 或明确的 page/pageSize 映射。
- 页面筛选状态转换为 query object，再交给 client 构建 URL。
- 导出 API 必须复用当前列表 query type 或一个可证明兼容的 export query type。
- 不允许 list 和 export 对同一筛选字段使用不同名称或默认值。

## 导出与下载

- Blob/CSV 下载可以使用 `fetch` 特例，但必须:
  - 复用 `API_BASE_PREFIX` 和 auth header。
  - 复用 query builder。
  - 失败时转换为 `ApiError`。
  - 对外返回 `Blob`，DOM 下载动作留在 page/composable。
- 后续可抽取 `apiDownload`，但 FE2-01 不引入新 wrapper，避免半成品兼容层。

## 页面迁移规则

后续 FE3 页面升级时按以下顺序迁移:

1. 在 feature 目录创建或补齐 `api.ts`。
2. 把 endpoint、request/response type、query builder、normalizer 移入 client。
3. 页面改为调用 typed client 或 Pinia store。
4. 保留 UI 状态在页面或 store，不塞入 API client。
5. 通过 typecheck/build 和页面状态验收。

## 当前缺口清单

| 页面/模块 | 缺口 | 后续归属 |
|---|---|---|
| User pages | 用户、角色、组织、绑定 API 仍散落在页面。 | FE3 User 页面升级或 FE2 store/client 子任务。 |
| Role pages | 角色列表、编辑、授权/撤销仍为 page-local API。 | FE3 Role 页面升级。 |
| Plugin page | lifecycle/config/dev portal API 大量集中在页面。 | FE4 Plugin/Dev Portal 升级前建立 `plugins/api.ts`。 |
| Setting/Dictionary/Organization | 系统配置、字典、组织 API 在页面内。 | FE3 Setting 或 M4 页面升级。 |
| Export/download | Audit 已有 Blob export 特例；后续导出需复用同模式。 | FE2-07 统一导出/下载交互。 |

## FE2-01 验收结论

- Request standard: Passed，明确 JSON 请求、query、path、body 规则。
- Response standard: Passed，明确 `ApiResponse<T>`、payload 返回、normalizer、unknown metadata、pagination。
- Error standard: Passed，明确 `ApiError`、`toErrorMessage`、特殊状态和 export 错误。
- Pagination/export: Passed，明确 list/export query 复用和 Blob 下载边界。
- Migration boundary: Passed，明确后续页面 API client 迁移顺序，不新增旧兼容 wrapper。
