# Skoll 前端质量与性能基线

> 合并范围: FE5 前端测试与验收体系、FE6 前端性能与可观测基线。  
> 原始材料: 已归档到 `docs/archive/refactor-2026-06-19/`。

## 质量门禁

前端任务不能只以 build 通过作为验收。每个重要页面或流程至少需要覆盖命令门禁、状态验收、权限验收、响应式验收和必要的浏览器证据。

| 门禁 | 命令或证据 | 通过标准 |
| --- | --- | --- |
| Typecheck | `cd web && npm run typecheck` | 类型检查通过，或明确记录缺口 |
| Build | `cd web && npm run build` | 构建通过，warning 被记录并评估 |
| Core smoke | 页面验收清单 | Dashboard/User/Role/Permission/Menu/Plugin/Audit/Setting 有步骤 |
| Permission state | admin 与 restricted role 检查 | 路由、菜单、按钮、API denial、审计信号可验证 |
| Responsive | 桌面、窄屏、最低宽度检查 | 无重叠、无按钮溢出、表格有降级策略 |
| Browser smoke | 浏览器执行记录 | 登录、权限拒绝、插件操作、审计导出、窄屏导航有 pass/fail/blocked |

## Browser smoke 最小集

| 场景 | 验收点 |
| --- | --- |
| Login | 登录成功、失败反馈、跳转路径 |
| Permission denied | 受限账号无法进入无权限页面或触发无权限动作 |
| Plugin operation | 插件列表、详情、风险或配置动作可见 |
| Audit export | 当前筛选条件参与导出，失败可见 |
| Narrow navigation | 窄屏导航、筛选、表格和操作区不重叠 |

若 Playwright 或浏览器工具暂不可用，记录为 `Blocked`，并写明重试命令、依赖和恢复条件。

## 性能基线

| 领域 | 基线要求 |
| --- | --- |
| Bundle | 记录 build chunk、最大 JS/CSS 资产、warning、明显依赖风险 |
| Route lazy loading | 主要业务页面按需加载，重页面不进入首屏主包 |
| Heavy tables | 服务端分页优先，大列表有虚拟化或降级触发条件 |
| Shared cache | 菜单、权限、字典等共享数据有缓存、失效、刷新和错误处理规则 |
| Plugin/Dev Portal | 重面板按需加载，请求数量可解释，首屏不被日志、风险、发布历史阻塞 |
| Regression checklist | bundle、懒加载、表格、请求数量和关键交互纳入回归检查 |

## 共享数据缓存策略

共享数据缓存只覆盖高频、低变化、跨页面复用的数据，例如菜单、权限、字典、系统设置摘要。不要把它扩展成通用缓存平台。

最低要求:

1. 缓存 key 稳定，包含租户、用户、权限上下文等必要维度。
2. 失效规则明确，权限、菜单、字典变更后能刷新。
3. 错误不污染缓存，失败状态可见。
4. 页面可主动 refresh。
5. 关键缓存命中与刷新路径能被手动验收。

## 性能验收模板

每次性能相关任务记录:

| 字段 | 说明 |
| --- | --- |
| Affected routes/pages | 受影响页面 |
| Build result | build 命令与结果 |
| Bundle change | 主要 chunk 或依赖变化 |
| Lazy loading | 是否影响路由懒加载 |
| Request behavior | 关键请求数量与触发时机 |
| Heavy table risk | 大表格、虚拟化、分页风险 |
| Manual observation | 页面卡顿、首屏阻塞、重面板加载观察 |
| Failure policy | 失败时回滚或返工动作 |

## 发布前命令

```powershell
go test ./...
cd web
npm run typecheck
npm run build
```

命令失败时，不允许把任务标为 Done。需要记录失败原因、修复动作和重新验收结果。
