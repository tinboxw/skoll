# FE2 Download Export Standard

日期: 2026-06-19  
范围: FE2-07  
状态: Accepted baseline

## 目标

统一 Skoll 前端导出/下载交互，确保导出使用当前筛选条件，失败可见，下载 DOM 动作集中在 helper 中。API client 负责返回 `Blob` 和错误语义，页面负责 query、权限、loading、成功/失败反馈和文件名。

## 当前基础

| 文件 | 当前职责 | 结论 |
|---|---|---|
| `web/src/audit/api.ts` | `exportAuditEvents(query)` 复用 `AuditEventListQuery`，返回 `Blob`，失败转换为 `ApiError` | 作为 typed export client 参考。 |
| `web/src/views/Audit/index.vue` | 构造当前筛选 query，调用导出，展示成功/失败 | 作为页面导出交互参考。 |
| `web/src/composables/useDownloadBlob.ts` | 集中处理 `URL.createObjectURL`、anchor click、revoke URL | 作为浏览器下载 helper。 |
| `docs/refactor/m2_export_list_filter_parity.md` | 审计 list/export 过滤一致性记录 | 作为后端过滤一致性参考。 |

## 分层职责

| 层 | 允许做什么 | 不允许做什么 |
|---|---|---|
| API client | 构造 export endpoint、复用 list query type、返回 `Blob`、把失败转为 `ApiError` | 创建 DOM anchor、展示消息、读取页面状态。 |
| Download helper | 接收 `Blob` 和 filename，触发浏览器下载，释放 object URL | 拼接 API URL、吞掉业务错误、决定成功提示。 |
| Page/store | 构造当前筛选 query、控制 loading/disabled、调用 API client 和 helper、展示错误/成功 | 复制 Blob 下载 DOM 代码、把失败显示为成功。 |

## 导出规则

- 导出 query 必须复用当前列表 query type，或使用可证明字段兼容的 export query type。
- 导出必须携带当前筛选条件，不得只导出默认全量数据，除非 UI 明确选择全量。
- 导出按钮 loading 使用独立 `operating/exporting` 状态，避免重复点击。
- 成功提示只能在 API 返回 Blob 且 helper 已触发下载后设置。
- 失败必须进入页面错误状态，并通过 `toErrorMessage` 或同等路径展示。
- 403 必须进入无权限态或错误态，不能触发空文件下载。

## 文件名规则

- 文件名由页面决定，必须稳定、可读、带扩展名。
- 默认 filename 为空时 helper fallback 为 `download`。
- 后续若后端提供 `Content-Disposition` filename，需要在 API client 中解析并作为返回 metadata 显式建模；FE2-07 不引入半成品兼容解析。

## 当前缺口清单

| 模块 | 缺口 | 后续归属 |
|---|---|---|
| Audit export | 文件名仍为固定 `audit_events.csv`，未包含日期或筛选摘要 | FE3 Audit 页面体验升级。 |
| Other pages | User/Role/Permission/Plugin 等页面尚无统一导出入口 | 后续页面新增导出时复用本标准。 |
| Content-Disposition | 当前 helper 不解析后端文件名 | 需要后端契约后再扩展 typed client。 |
| Browser feedback | 当前成功提示为下载已开始，未检测浏览器下载失败 | 浏览器限制下可接受；真实 API 失败必须可见。 |

## FE2-07 验收结论

- Query parity: Passed，Audit 导出复用 `buildAuditEventQuery()` 和 `AuditEventListQuery`。
- Blob boundary: Passed，API client 返回 `Blob`，页面使用 `downloadBlob` 触发下载。
- Error visibility: Passed，导出失败进入 `error.value = toErrorMessage(e)`。
- No duplicate DOM download: Passed，页面不再直接创建 anchor/object URL。
