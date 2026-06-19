# FE2 Route Permission Standard

日期: 2026-06-19  
范围: FE2-03  
状态: Accepted baseline

## 目标

统一 Skoll 前端 route guard、菜单过滤、按钮权限和插件入口权限规则，确保所有可见入口与实际路由访问使用同一套 `AccessRule` 语义。后续 FE3 页面升级时，页面只能消费 `permissions` 模块提供的能力，不在页面中复制角色/权限判断。

## 当前基础

| 文件 | 当前职责 | 结论 |
|---|---|---|
| `web/src/permissions/access.ts` | `canAccess` 和 `useAccess`，处理 roles、permissions、all/any、super_admin bypass、implied permission | 保留为唯一基础权限判定入口。 |
| `web/src/permissions/route.ts` | `isPublicRoute`、`canAccessRoute`、`withRouteAccessMeta` | 保留为 route meta 与 plugin route access 的标准入口。 |
| `web/src/router/index.ts` | 登录态、插件 route bootstrap、default home 校验、route access guard | 保留为唯一全局 route guard。 |
| `web/src/navigation/menu.ts` | 系统菜单和插件菜单合并、权限过滤、排序 | 菜单显隐必须复用 `canAccess`。 |
| `web/src/permissions/button.ts` | `BUTTON_ACCESS`、`canUseButton`、`useButtonAccess` | 按钮权限 key 必须集中在这里或来自后端权限目录。 |
| `web/src/permissions/directive.ts` | `v-permission` 指令 | 模板级显隐使用 `canUseButton`，不重复权限逻辑。 |
| `web/src/permissions/catalog.ts` | 内置权限 catalog 和角色默认权限 | 后续新增 key 需同步后端权限目录和审计/文档规则。 |
| `web/src/plugins/index.ts` | 插件 route 注册并通过 `withRouteAccessMeta` 绑定 manifest 权限 | 插件入口必须声明 route/menu 权限，不能只依赖菜单隐藏。 |

## 单一判定入口

- 所有角色、权限和 implied permission 判断必须经过 `canAccess`。
- `super_admin` bypass 只允许在 `canAccess` 中定义，页面、router、store 不得重复写 bypass。
- `mode` 默认为 `all`；需要任一权限满足时显式使用 `mode: "any"`。
- 空 rule 表示允许访问；空字符串、空数组在 normalize 后不产生权限要求。
- `hasPermissionValue` 负责 implied permission，例如 `plugin.manage` implied `plugin.read`。

## Route Guard 规范

- 静态后台页面必须在 `web/src/router/index.ts` 的 route meta 中声明 `permissions` 或 `roles`，公开页面只允许 `meta.public = true`。
- 插件 route 必须通过 `withRouteAccessMeta` 注入 manifest 中的 `requiredRoles` 和 `requiredPermissions`。
- `router.beforeEach` 的顺序保持为:
  1. 未登录访问非公开页面时跳转 `/skoll/login`。
  2. 已登录访问登录页时使用安全 redirect 或 default home。
  3. 插件 home/path 先等待 bootstrap 并验证目标 path。
  4. 最后使用 `canAccessRoute` 做访问判定。
- 禁止页面级 `router.push("/skoll/login")` 处理鉴权失效；401 继续由 API wrapper 处理，route access 继续由 global guard 处理。
- 禁止新增旧路由别名或兼容 redirect，新增页面只能使用 canonical `/skoll/...` path。

## 菜单权限规范

- 系统菜单来源为后端 `SystemMenuRecord` 或 `SYSTEM_MENU` fallback；两者都必须携带 `requiredRoles/requiredPermissions`。
- 插件菜单来源为 manifest `uiMenu.requiredRoles/requiredPermissions`。
- 菜单显隐只表示入口可见性，不作为安全边界；route meta 必须同步声明同等或更严格权限。
- 系统菜单 path 必须与 router canonical path 对齐，插件 path 必须由 `resolvePluginMenuPath` 规范化。
- 菜单排序和 label 处理不得改变权限判断结果。

## 按钮与指令规范

- 新增按钮权限优先补充 `BUTTON_ACCESS` key，并使用 `useButtonAccess` 或 `v-permission`。
- 页面可以使用 computed `canXxx` 控制禁用、提示和空态，但 computed 内部必须调用 `buttonAccess.can(...)`。
- `v-permission` 只负责显隐，不负责禁用、确认弹窗或操作状态。
- 批量操作、危险操作、导出操作必须同时满足按钮显隐和提交前权限/状态校验。
- 禁止在模板中直接写 `userStore.profile?.role === ...` 或 `permissions.includes(...)`。

## 插件权限规范

- 插件 sidebar 入口权限来自 `uiMenu.requiredRoles/requiredPermissions`。
- 插件 route 权限来自 `requiredRoles/requiredPermissions`，通过 `withRouteAccessMeta` 写入 meta。
- 插件页面内部按钮权限继续使用 host 的 `BUTTON_ACCESS` 或后续插件注册的 button permission key。
- backend-only、standalone 或 disabled 插件不进入 sidebar；route 可达性仍由插件注册和 route guard 决定。

## 当前缺口清单

| 模块 | 缺口 | 后续归属 |
|---|---|---|
| `BUTTON_ACCESS` | 仅覆盖 User/Role/Plugin/Permission，Dictionary、Organization、Audit、Setting 按钮 key 仍不完整 | FE3 页面升级时补齐。 |
| `permissions/catalog.ts` | 前端内置 catalog 与后端权限目录存在重复维护风险 | M1/M2 权限目录任务继续收敛。 |
| `navigation/menu.ts` | 后端菜单与静态 route meta 的权限一致性依赖人工对齐 | FE2 architecture gate 和 FE3 页面验收中逐页检查。 |
| `router/index.ts` | forbidden fallback 目前回 default home，没有专用 403 页面 | FE3 no-permission state 统一时决定是否新增。 |
| Plugin dev portal | 页面内操作权限较集中在 Plugin page，后续拆分 API/store 时需同步 button key | FE4 plugin/dev portal 升级。 |

## FE2-03 验收结论

- Route guard: Passed，登录态、插件 bootstrap、安全 default home 与 `canAccessRoute` 顺序明确。
- Menu permission: Passed，系统菜单和插件菜单统一通过 `canAccess` 过滤。
- Button permission: Passed，`BUTTON_ACCESS`、`useButtonAccess`、`v-permission` 的边界明确。
- Plugin route permission: Passed，插件 route 使用 `withRouteAccessMeta` 注入权限。
- Current gaps: Passed，记录 button key、catalog 同步、后端菜单一致性和 no-permission state 缺口。
