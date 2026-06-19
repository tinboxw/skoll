# FE3 Core Page State Acceptance

日期: 2026-06-19
范围: FE3-09
状态: Accepted with browser smoke blocked

## 验收范围

Dashboard `/skoll/dashboard`、User `/skoll/user`、Role `/skoll/role` 与编辑页、Permission `/skoll/permission`、Menu `/skoll/menu`、Plugin `/skoll/plugin`、Audit `/skoll/audit`、Setting `/skoll/setting`。

## Browser smoke

- Result: Blocked
- Reason: 当前线程未暴露可调用的 in-app browser 工具；此前 bundled Playwright 运行时缺少 `playwright-core`，无法完成真实点击、截图或窄屏浏览器 smoke。
- Replacement gate: 静态状态锚点检查、路由/权限检查、typecheck、build、人工代码审阅。
- Follow-up: FE5 smoke 工具链可用后，补跑默认态、空态、错误态、无权限态和窄屏路径。

## Page matrix

| Page | Route | 正常态 | Loading/操作态 | 空态 | 错误态 | 无权限态 | 窄屏态 | 命令/危险操作 |
|---|---|---|---|---|---|---|---|---|
| Dashboard | `/skoll/dashboard` | system/plugin summary、quick links、risk items 可扫 | stores loading 保留 | quick links/risk clear 有空提示 | store error 通过页面状态反馈 | quick link 按权限过滤 | 960px/640px summary 与 lists 折列 | N/A |
| User | `/skoll/user` | summary-grid、filters、table、bulk assign 可扫 | list/assign operating 禁用 | filtered table empty | error/opSuccess alert | `StateBlock` forbidden | filters/table 响应式 | assign/deactivate 等操作保留确认与反馈 |
| Role | `/skoll/role` | list、edit、summary、user/permission panels 可扫 | save/grant/revoke loading | role/user/permission empty | save/grant/revoke error | `StateBlock` forbidden | 编辑区窄屏折列 | grant/revoke/save 使用确认或明确反馈 |
| Permission | `/skoll/permission` | matrix、summary、filters 可扫 | save/grant/revoke loading | matrix empty | save/grant/revoke error | `StateBlock` forbidden | matrix filters 单列化 | confirmAction 覆盖授权、回收、保存 |
| Menu | `/skoll/menu` | tree table、summary、filters、visibility 可扫 | save/delete loading | filtered tree empty | invalid warning/error | `StateBlock` forbidden | editor/table 窄屏折列 | save/delete 使用 confirmAction |
| Plugin | `/skoll/plugin` | system/app table、summary、risk/access/config/log/debug panels 可扫 | lifecycle/config loading | plugin filtered empty | lifecycle/config error | `StateBlock` forbidden | table/panels 窄屏折列 | enable/disable/install/uninstall 使用 confirmAction |
| Audit | `/skoll/audit` | summary、filters、detail tabs、risk tags 可扫 | list/detail/export/clear loading | list empty | list/detail/export error | `StateBlock` forbidden | filters/table/detail 窄屏折列 | export/clear 保留确认与反馈 |
| Setting | `/skoll/setting` | SchemaForm、raw list、summary、audit hint 可扫 | schema/settings/save/reset loading | schema/list filtered empty | schema/settings/save/reset error | `StateBlock` forbidden | form/list 窄屏单列化 | reset 使用 confirmAction，保存有审计提示 |

## State coverage

- Normal: 页面主内容、摘要、筛选、表格或表单入口均存在静态锚点。
- Empty: 列表、矩阵、筛选结果或 schema 缺失均有 `StateBlock`/empty 文案。
- Error: API、保存、导出、生命周期等失败路径均转为页面反馈或 toast。
- No permission: 核心后台页统一通过权限 guard 或 `StateBlock` 呈现。
- Narrow viewport: FE3 页面均保留 960px/820px/640px/560px 级别的响应式折列规则。

## 验证命令

```powershell
rg -n "StateBlock|forbidden|empty|loading|error|success|summary-grid|filters|reset|confirmAction|downloadBlob|SchemaForm|canRead|canManage|quickLinks|riskItems" web/src/views/Dashboard/index.vue web/src/views/User/list.vue web/src/views/Role/list.vue web/src/views/Role/edit.vue web/src/views/Permission/index.vue web/src/views/Menu/index.vue web/src/views/Plugin/index.vue web/src/views/Audit/index.vue web/src/views/Setting/index.vue
rg -n "const .*Page = \(\) => import|/skoll/dashboard|/skoll/user|/skoll/role|/skoll/permission|/skoll/menu|/skoll/plugin|/skoll/audit|/skoll/setting|permissions" web/src/router/index.ts web/src/navigation/menu.ts
cd web
npm run typecheck
npm run build
```
