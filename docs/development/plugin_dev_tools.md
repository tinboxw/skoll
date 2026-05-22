# 插件开发工具（M11）

## 概述
`internal/handler/cli/plugin_cmd.go` 提供插件开发阶段最小工具集，覆盖：
- `list`：查看当前插件清单和状态。
- `debug <pluginId>`：输出插件依赖、权限、来源等调试信息。
- `logs <pluginId>`：读取插件日志文件（默认目录 `log/`）。
- `validate <pluginPathOrManifest>`：校验 `plugin.yaml` 元数据格式。
- `validate-all <pluginsRootDir>`：批量校验插件目录下所有 `plugin.yaml`。
- `scaffold <pluginsRootDir> <pluginId> <pluginName> [appId]`：生成标准插件骨架。
- `migrate <pluginDir> <plan|apply|rollback> [steps]`：执行插件迁移生命周期（MVP）。

## 命令行为
### list
- 输入：`["list"]`
- 输出示例：

```text
plugins=2
- auth@1.0.0 state=enabled
- dashboard@1.0.0 state=installed
```

### debug
- 输入：`["debug", "auth"]`
- 输出字段：
  - `id`
  - `name`
  - `version`
  - `state`
  - `deps`
  - `perms`
  - `source`

### logs
- 输入：`["logs", "demo"]`
- 行为：读取 `log/demo.log`（或构造时传入的自定义日志目录）。

### validate
- 输入：`["validate", "plugins/demo"]` 或 `plugin.yaml` 的绝对路径。
- 输出示例：

```text
valid id=demo version=0.1.0 deps=1 perms=2
```

### validate-all
- 输入：`["validate-all", "plugins"]`
- 输出示例：

```text
validated=3
- ok plugins/demo id=demo version=0.2.0
- ok plugins/demo-frontend id=demo-frontend version=0.1.0
- ok plugins/demo-backend id=demo-backend version=0.1.0
```

兼容性校验：
- `validate` 与 `validate-all` 会校验 `compatibility_skoll`。
- 可通过环境变量 `SKOLL_CORE_VERSION` 指定当前 core 版本（例如 `1.0.0`）。

### scaffold
- 输入：`["scaffold", "plugins", "oa", "OA Suite", "oa"]`
- 行为：创建 `plugins/oa` 标准目录与基础 `plugin.yaml`。
- 基建默认（可按插件调整）：
  - `ui_nav_position: none`
  - `ui_open_mode: integrated`
  - `ui_tab_mode: optional`
  - `i18n_locales: [zh-CN, en-US]`

Manifest 新增约束：
- 非 `backend_only` 插件必须声明 `i18n_locales`（格式示例：`zh-CN`、`en-US`）。
- 当 `ui_open_mode=standalone` 时，必须同时满足：
  - `ui_nav_position=none`
  - `ui_tab_mode=disabled`

### migrate
- 输入：`["migrate", "plugins/oa", "plan"]`
- 输入：`["migrate", "plugins/oa", "apply", "1"]`
- 输入：`["migrate", "plugins/oa", "rollback", "1"]`
- 约定：
  - 迁移文件目录：`<pluginDir>/migrations`
  - 文件命名：`NNN_name.up.sql` / `NNN_name.down.sql`
  - 状态文件：`<pluginDir>/.skoll/migration-state.json`

## 示例插件
示例目录：`plugins/demo/`
- `plugin.yaml`：可通过 `validate` 直接校验。
- `main.go`、`handler.go`：最小示例骨架，便于后续演进为真实插件入口。

## 测试
已覆盖命令最小回归：
- `internal/handler/cli/plugin_cmd_test.go`
  - 列表与调试输出
  - 日志读取
  - 元数据校验
  - 参数错误路径

## Web Dev Portal（HTTP）

在后端启用 Dev Portal 开关后，可安装 `plugins/developer-portal` 插件作为独立开发者页面入口。

安装流程：
- `POST /skoll/v1/plugins/install`，请求体：`{ "path": "plugins/developer-portal" }`
- 在插件管理中启用并点击该插件「访问」，进入 Developer Portal 页面

Developer Portal 中当前已接入以下 API：

- `GET /skoll/v1/plugins/dev/config`
  - 仅 `super_admin` 可调用。
  - 返回 `data`：`{ enabled, defaultRoot, allowedRoots[] }`
- `GET /skoll/v1/plugins/dev/manifest?pluginsRoot=plugins&pluginId=crm-order`
  - 仅 `super_admin` 可调用。
  - 返回 `data`：`{ operation,status,pluginsRoot,pluginId,pluginDir,manifestPath,manifest,validation,error? }`
- `POST /skoll/v1/plugins/dev/manifest/validate`
  - 仅 `super_admin` 可调用。
  - 请求体：`{ "pluginsRoot": "plugins", "pluginId": "crm-order", "manifest": "...yaml..." }`
  - 返回 `data`：`{ operation,status,pluginsRoot,pluginId,pluginDir,manifestPath,manifest,validation }`
- `PUT /skoll/v1/plugins/dev/manifest`
  - 仅 `super_admin` 可调用。
  - 请求体：`{ "pluginsRoot": "plugins", "pluginId": "crm-order", "manifest": "...yaml..." }`
  - 返回 `data`：`{ operation,status,pluginsRoot,pluginId,pluginDir,manifestPath,manifest,validation }`
- `POST /skoll/v1/plugins/dev/projects`
  - 仅 `super_admin` 可调用。
  - 请求体：`{ "pluginsRoot": "plugins" }`
  - 返回 `data`：`{ operation, status, pluginsRoot, projects:[{ pluginId,name,version,path,mode,installed,enabled,previewUrl,buildHint,packageHint,publishHint,lastUpdatedAt }] }`
- `POST /skoll/v1/plugins/dev/remove`
  - 仅 `super_admin` 可调用。
  - 请求体：`{ "pluginsRoot": "plugins", "pluginId": "crm-order", "removeFiles": true }`
  - 返回 `data`：`{ operation, status, pluginsRoot, pluginId, pluginDir, uninstalled, filesRemoved }`
- `POST /skoll/v1/plugins/dev/package`
  - 仅 `super_admin` 可调用。
  - 请求体：`{ "pluginsRoot": "plugins", "pluginId": "crm-order", "outputDir": "plugins/_dist" }`
  - 返回 `data`：`{ operation, status, pluginsRoot, pluginId, pluginDir, artifactPath }`
- `POST /skoll/v1/plugins/dev/pipeline`
  - 仅 `super_admin` 可调用。
  - 请求体：`{ "pluginsRoot": "plugins", "pluginId": "crm-order", "outputDir": "plugins/_dist" }`
  - 返回 `data`：`{ operation,status,pluginsRoot,pluginId,pluginDir,startedAt,finishedAt,steps:[{name,status,message?,artifactPath?,durationMs}] }`
- `POST /skoll/v1/plugins/dev/rollout`
  - 仅 `super_admin` 可调用。
  - 请求体：`{ "pluginId": "crm-order", "rolloutPercent": 20 }`
  - 返回 `data`：`{ operation,status,pluginId,rolloutPercent,persisted,message?,task:{ taskId,pluginId,action,requestedRolloutPercent,previousRolloutPercent,rolloutPercent,taskStatus,createdBy,createdAt,startedAt?,finishedAt?,failureReason?,steps?,logs? } }`
  - 说明：运行时必须提供配置写入能力；不再支持“仅内存保留历史”的降级路径。
- `POST /skoll/v1/plugins/dev/rollback`
  - 仅 `super_admin` 可调用。
  - 请求体：`{ "pluginId": "crm-order" }`
  - 返回 `data`：`{ operation,status,pluginId,rolloutPercent,persisted,message?,task:{...} }`
- `GET /skoll/v1/plugins/dev/rollout-tasks?pluginId=crm-order&status=success`
  - 仅 `super_admin` 可调用。
  - 返回 `data`：`{ operation,status,pluginId,taskStatus,tasks:[...] }`
- `GET /skoll/v1/plugins/dev/rollout-tasks/{taskId}`
  - 仅 `super_admin` 可调用。
  - 返回 `data`：`{ operation,status,task:{...} }`
- `GET /skoll/v1/plugins/dev/rollout-tasks/{taskId}/logs`
  - 仅 `super_admin` 可调用。
  - 返回 `data`：`{ operation,status,taskId,logs:[{timestamp,step?,level,message}] }`
- `POST /skoll/v1/plugins/dev/release-orders`
  - 仅 `super_admin` 可调用。
  - 请求体：`{ "pluginsRoot": "plugins", "pluginId": "crm-order", "releaseVersion": "1.2.0", "changelog": "..." }`
  - 返回 `data`：`{ operation,status,order:{ orderId,pluginId,releaseVersion,orderStatus,changelog?,createdBy,createdAt,approvedBy?,approvedAt?,rejectedBy?,rejectedAt?,reviewComment? } }`
- `GET /skoll/v1/plugins/dev/release-orders?pluginsRoot=plugins&pluginId=crm-order`
  - 仅 `super_admin` 可调用。
  - 返回 `data`：`{ operation,status,pluginsRoot,pluginId,orders:[...] }`
- `POST /skoll/v1/plugins/dev/release-orders/{orderId}/approve`
  - 仅 `super_admin` 可调用。
  - 请求体：`{ "pluginsRoot": "plugins", "pluginId": "crm-order", "comment": "approved" }`
  - 返回 `data`：`{ operation,status,order:{...orderStatus=approved...} }`
- `POST /skoll/v1/plugins/dev/release-orders/{orderId}/reject`
  - 仅 `super_admin` 可调用。
  - 请求体：`{ "pluginsRoot": "plugins", "pluginId": "crm-order", "comment": "need more checks" }`
  - 返回 `data`：`{ operation,status,order:{...orderStatus=rejected...} }`
- `POST /skoll/v1/plugins/dev/release-orders/{orderId}/execute`
  - 仅 `super_admin` 可调用。
  - 请求体：`{ "pluginsRoot": "plugins", "pluginId": "crm-order", "targetEnv": "staging", "artifactPath": "plugins/_dist/crm-order-1.2.0.zip" }`
  - 返回 `data`：`{ operation,status,task:{ taskId,orderId,pluginId,releaseVersion,targetEnv,artifactPath,taskStatus,createdBy,createdAt } }`
  - 说明：
    - 仅允许执行 `approved` 状态发布单。
    - 必须显式提供 `artifactPath`，不会自动降级为“仅记录不执行”。
    - 同一 `pluginId + targetEnv` 若已有 `running` 任务，返回 `409 task_conflict`。
- `GET /skoll/v1/plugins/dev/release-tasks?pluginsRoot=plugins&pluginId=crm-order&orderId=ro-xxx&status=running`
  - 仅 `super_admin` 可调用。
  - 返回 `data`：`{ operation,status,pluginsRoot,pluginId,orderId,taskStatus,tasks:[...] }`
- `GET /skoll/v1/plugins/dev/release-tasks/{taskId}?pluginsRoot=plugins`
  - 仅 `super_admin` 可调用。
  - 返回 `data`：`{ operation,status,task:{...,steps:[{name,status,message?,startedAt?,finishedAt?,durationMs}],failureStep?,failureReason?} }`
- `GET /skoll/v1/plugins/dev/release-tasks/{taskId}/logs?pluginsRoot=plugins`
  - 仅 `super_admin` 可调用。
  - 返回 `data`：`{ operation,status,taskId,logs:[{timestamp,step?,level,message}] }`

- `POST /skoll/v1/plugins/dev/scaffold`
  - 仅 `super_admin` 可调用。
  - 请求体：`{ "pluginsRoot": "plugins", "pluginId": "crm-order", "pluginName": "CRM Order", "appId": "crm", "mode": "workspace|repository" }`
  - 返回 `data`：`{ operation, status, pluginsRoot, pluginId, pluginName, pluginDir, mode }`
- `POST /skoll/v1/plugins/dev/validate-all`
  - 仅 `super_admin` 可调用。
  - 请求体：`{ "pluginsRoot": "plugins" }`
  - 返回 `data`：`{ operation, status, pluginsRoot, summary:{total,valid,invalid}, results:[...] }`

- API Tools 模块：
  - 支持带当前登录 token 的任意 API 请求调试（GET/POST/PUT/DELETE）。
  - 内置快捷按钮：`GET /skoll/health`、`GET /skoll/v1/auth/me`。

- 开发中插件模块：
  - 按 allowlist 根目录扫描插件工程（支持独立仓库根目录）。
  - 支持“预览 / 打包 / 删除并清理目录”。
  - 列表项提供 `buildHint/packageHint/publishHint`，用于 CI/CD 对接。

- Manifest 编辑器模块：
  - 加载并编辑 `plugin.yaml`，保存前后执行校验。
  - 提供“只校验不落盘”的校验接口，避免误写入。

- 一键流水线模块：
  - 先执行 manifest 校验，再执行制品打包。
  - 返回步骤耗时与产物路径，用于发布前人工确认。

- 灰度/回滚模块：
  - 支持按百分比设置 rollout 值（0-100）。
  - 支持回滚到上一次 rollout 值。
  - 严格基建模式下，配置写入器缺失会直接失败，不做兼容/兜底。
  - 所有灰度与回滚动作都会生成任务记录，可通过 rollout-task 接口追踪步骤与日志。

- 发布审批流模块：
  - 支持创建发布单（pending），并按 `orderId` 审批通过或驳回。
  - 发布单内含审计字段：`createdBy/createdAt/approvedBy/approvedAt/rejectedBy/rejectedAt/reviewComment`。
- 审批通过后可执行发布任务，并通过任务列表/详情/日志跟踪执行状态（pending/running/success/failed/cancelled）。

安全边界：
- 通过 `SKOLL_DEV_PORTAL_ENABLED=true` 显式开启。
- `SKOLL_DEV_PLUGINS_ROOT` 支持多目录白名单（使用 `;` 或 `,` 分隔）。
- `pluginsRoot` 必须命中白名单中的某个目录，否则返回 400。
- 路由仅在开关开启时注册；关闭时访问 `/v1/plugins/dev/*` 为 404。
- 运行时接口仅允许 `super_admin` 调用，非 `super_admin` 返回 403。

## 常见问题：插件页面“裸奔”

现象：页面能打开，但字体、按钮、输入框看起来像浏览器默认样式，缺少管理台风格。

已验证的常见原因：
- 访问了后端原始页面接口，而不是前端插件路由。
  - 错误示例：`http://127.0.0.1:8080/skoll/v1/plugins/{id}/page`
  - 正确示例：`http://127.0.0.1:5173/skoll/plugins/{id}`（或从插件管理点击“访问”）
- 误以为宿主后台样式会自动作用到插件页面。
  - 实际上插件页面通过 iframe 渲染，样式是隔离的；插件必须自带自己的 CSS。
- 浏览器缓存命中了旧资源，导致 `style.css`/`app.js` 未更新。
  - 处理方式：`Ctrl+F5` 强刷，并为静态资源追加版本参数（如 `style.css?v=20260521`）。
- 插件样式写法过于全局，覆盖了不该覆盖的控件（例如给所有 `button` 固定高度）。
  - 建议优先使用类选择器限制作用域（如 `.actions button`），避免全局重置导致排版错位。

开发建议（建议加入插件模板检查项）：
- 插件静态页必须包含基础样式兜底（critical CSS），避免外链样式加载失败时出现原始控件。
- 模板验收时至少检查：模块卡片、多行按钮、输入框、移动端断点（`<=860px`）布局是否正常。

## 常见问题：插件页面多语言不生效（一直英文）

现象：宿主管理台已经切到中文，但插件页面仍显示英文。

根因总结（已在开发者门户插件中复现并修复）：
- 宿主与插件之间的语言状态未可靠传递。
  - 仅依赖一次性注入时，可能被时序、独立页打开方式影响。
- 插件页面通过 blob iframe 渲染时，静态资源基路径可能失效。
  - 资源脚本未执行时，插件内部 i18n 逻辑不会生效。
- 浏览器缓存命中旧静态资源，导致新 i18n 脚本没有加载。

推荐修复方案（插件基线）：
- 宿主侧：同时提供两条语言通道。
  - 注入插件上下文：`window.__SKOLL_LOCALE`、`window.__SKOLL_LOCALES`。
  - 运行时广播：`postMessage` + 自定义事件（`skoll:locale`）。
- 插件侧：实现“多来源 locale 检测链路”。
  - 建议优先级：`query.locale` -> `window.__SKOLL_LOCALE` -> `localStorage(skoll.ui.locale)` -> `document.documentElement.lang` -> 默认值（建议 `zh-CN`）。
- 资源加载：避免依赖相对路径。
  - 在 blob iframe 场景，建议使用绝对 URL（带 `window.top.location.origin`）加载 `style.css` 与 `app.js`。
  - 可以先探测可用基路径，再动态挂载脚本与样式。
- 缓存控制：升级资源版本参数。
  - 每次修复 i18n 逻辑后，更新 `style.css?v=...`、`app.js?v=...`，并要求 `Ctrl+F5` 强刷验证。

排查清单（建议模板内置）：
- 检查插件诊断信息：
  - `host locale`
  - `localStorage locale`
  - `active locale`
  - `last source`
- 若诊断字段全部为空或 `-`，优先判断脚本是否实际执行。
- 若 `active locale` 与宿主不一致，检查 locale 优先级逻辑和消息监听是否完整。

实现注意事项：
- 独立页（standalone）与集成页（integrated）都应支持 locale 同步，不能只覆盖一种打开方式。
- 默认回退语言建议与平台默认一致（当前建议 `zh-CN`），避免“注入失败就回英文”。
- 不要把多语言能力绑定到单个插件 ID；应作为插件模板能力复用。
