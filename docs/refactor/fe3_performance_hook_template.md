# FE3 Performance Hook Template

日期: 2026-06-19  
范围: ADJ-FE-20260619-04  
状态: Accepted baseline

## 目标

把 FE6 性能意识提前挂到 FE3 页面升级中。每个核心页面改造都必须记录构建影响、路由懒加载、重表格风险、请求数量和重面板加载策略，避免页面体验升级后留下性能债。

## 页面性能记录模板

每个 FE3 页面验收时追加以下记录:

| 字段 | 记录要求 |
|---|---|
| Page/route | 页面名称和路由，例如 `/skoll/user`。 |
| Build command | `cd web && npm run build` 结果；如未运行必须说明原因。 |
| Typecheck command | `cd web && npm run typecheck` 结果。 |
| Bundle impact | 是否新增重依赖、chunk warning、明显体积变化。 |
| Route lazy loading | 页面是否通过 router dynamic import 加载。 |
| Heavy table risk | 表格是否分页、是否存在大列表客户端全量渲染。 |
| Request behavior | 首屏请求数量、重复刷新、stale write 风险。 |
| Loading behavior | loading 是否有意义，是否掩盖慢请求或重复请求。 |
| Deferred panels | 日志、详情、配置、Dev Portal 等重面板是否按需加载。 |
| Narrow viewport | 窄屏下表格、drawer、actions 是否导致横向溢出或布局跳动。 |

## 页面风险提示

| 页面 | 性能关注点 | FE3 记录要求 |
|---|---|---|
| Dashboard | 多卡片、多状态聚合、插件状态 | 记录首屏请求数量和空/错误聚合方式。 |
| User | 用户表、组织/岗位/角色选项、批量导入 | 记录分页策略和辅助数据缓存/加载时机。 |
| Role | 权限矩阵、关联用户 | 记录矩阵渲染规模和用户列表分页。 |
| Permission | 权限资源、diff、矩阵保存 | 记录资源列表规模、保存 diff 和重复请求保护。 |
| Menu | 树表、拖拽/排序、显隐保存 | 记录树节点规模、重排写入和 layout shift 风险。 |
| Plugin | 插件列表、日志、配置、Dev Portal | 记录重面板按需加载、日志请求和任务轮询风险。 |
| Audit | 审计列表、详情 sourceData、CSV export | 记录筛选请求、详情 drawer 大 JSON 渲染和导出反馈。 |
| Setting | SchemaForm、raw editor、批量保存 | 记录 schema 加载、批量请求和大文本编辑风险。 |

## 验收命令建议

```powershell
cd web
npm run typecheck
npm run build
rg -n "component:\\s*\\(\\)\\s*=>" web/src/router/index.ts
rg -n "el-table|v-loading|StateBlock|SchemaForm|downloadBlob" web/src/views/<Page>
```

## 记录格式

```markdown
### Performance Hook

- Page/route:
- Typecheck:
- Build:
- Bundle impact:
- Route lazy loading:
- Heavy table risk:
- Request behavior:
- Loading behavior:
- Deferred panels:
- Narrow viewport:
- Follow-up:
```

## FE3 使用规则

- 页面改造提交前必须补充 Performance Hook。
- 如果 build 出现 chunk warning，必须记录当前 warning 和是否由本页引入。
- 如果页面新增重依赖，必须说明原因和替代方案。
- 大表格必须优先服务端分页；临时 client-side 大列表必须记录后续 FE6 跟进。
- 插件/Dev Portal 重面板不得阻塞首屏。

## 验收结论

- Build impact template: Passed。
- Route lazy loading hook: Passed。
- Heavy table risk hook: Passed。
- Request/loading behavior hook: Passed。
- Deferred panel hook: Passed。
