# FE3 Page-By-Page Acceptance Map

日期: 2026-06-19  
范围: ADJ-FE-20260619-03  
状态: Accepted baseline

## 目标

为 FE3 核心页面体验升级建立逐页验收地图。每个页面任务必须引用 FE1 视觉/组件标准、FE2 架构标准、FE0 手工验收模板和本页状态清单，不能只以 build 通过作为完成标准。

## 通用验收门禁

每个 FE3 页面至少记录:

- Route: 页面路径和入口菜单。
- Data path: API client、store 或 page-local state 来源。
- Permission path: route meta、sidebar、button 或 `v-permission`。
- States: normal、loading、empty、error、no-permission、saving/operating、success/failure。
- Interactions: filters、table/list、form、drawer/dialog、dangerous action、export/import if present。
- Responsive: desktop 和窄屏不重叠、不溢出。
- Commands: `npm run typecheck`；涉及页面实现时运行 `npm run build`。
- Browser/manual: 至少记录一个人工或浏览器 smoke 路径。

## 页面验收地图

| 页面 | 路由 | 核心状态 | 关键交互 | 必跑命令 |
|---|---|---|---|---|
| Dashboard | `/skoll/dashboard` | normal、loading、empty/system unavailable、error | 系统状态、插件状态、快捷入口、风险提示 | `cd web && npm run build` |
| User | `/skoll/user`、add/edit/batch | normal、loading、empty、error、no-permission、saving、batch partial failure | 筛选、分页、创建、编辑、删除、角色分配、组织归属、批量导入 | `cd web && npm run build` |
| Role | `/skoll/role`、edit | normal、loading、empty、error、no-permission、saving | 角色列表、创建、删除、授权、关联用户、危险确认 | `cd web && npm run build` |
| Permission | `/skoll/permission` | normal、loading、empty、error、no-permission、saving | 权限矩阵、角色授权、资源启停、diff、保存反馈 | `cd web && npm run build` |
| Menu | `/skoll/menu` | normal、loading、empty、error、no-permission、saving | 树表、排序、显隐、权限字段、保存确认、retry | `cd web && npm run build` |
| Plugin | `/skoll/plugin` | normal、loading、empty、error、no-permission、operating、task failed | 插件列表、启停、卸载、配置 SchemaForm、日志/调试、Dev Portal 任务 | `cd web && npm run build` |
| Audit | `/skoll/audit` | normal、loading、empty、error、no-permission、export success/failure | tabs、筛选、quick tags、详情 drawer、CSV 导出、清理确认 | `cd web && npm run build` |
| Setting | `/skoll/setting` | normal、loading、empty、error、no-permission、saving、reset success/failure | SchemaForm、raw editor、敏感项、批量保存、reset confirmation | `cd web && npm run build` |

## 状态验收模板

每页验收记录使用以下字段:

| 字段 | 记录要求 |
|---|---|
| Route opened | 记录路径、入口和角色。 |
| Normal | 有数据时主流程可扫、操作可达。 |
| Loading | 刷新/保存/导出/详情加载时 loading 可见且重复动作禁用。 |
| Empty | 空结果不被误判为错误，提供 refresh/reset/create 等真实操作。 |
| Error | 后端错误通过 `toErrorMessage` 或 store `lastError` 可见，并有 retry/refresh 路径。 |
| No permission | 受限用户看到权限提示或禁用/隐藏动作，数据不静默消失。 |
| Success/failure | 成功只在真实成功后出现，失败保留原因。 |
| Narrow viewport | 375px 到 768px 之间主要文字、按钮、表格、drawer 不重叠。 |

## 页面级附加命令

```powershell
rg -n "apiGet|apiPost|apiPut|apiDelete|fetch\\(" web/src/views/<Page> web/src/<feature>
rg -n "BUTTON_ACCESS|useButtonAccess|v-permission|canAccess" web/src/views/<Page> web/src/permissions
rg -n "StateBlock|el-empty|el-result|loading|error|retry|refresh" web/src/views/<Page>
cd web
npm run typecheck
npm run build
```

使用 SchemaForm 的页面补充:

```powershell
rg -n "SchemaForm|update:valid|update:errors|PluginConfigSchema" web/src/views/<Page> web/src/components/Common/SchemaForm.vue
```

使用导出/下载的页面补充:

```powershell
rg -n "downloadBlob|export.*\\(|toErrorMessage" web/src/views/<Page> web/src
```

## FE3 验收结论

- Dashboard/User/Role/Permission/Menu/Plugin/Audit/Setting acceptance map: Passed。
- State coverage: Passed，normal/loading/empty/error/no-permission/success/failure/narrow viewport 均纳入。
- Command coverage: Passed，typecheck/build 和页面级 rg 命令已定义。
- FE3 readiness: Passed，FE3 页面任务可以逐项进入实现和验收。
