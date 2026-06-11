# 插件开发教程

## 1. 概述

Skoll 提供完整的插件系统，支持业务功能通过插件方式扩展。插件按 `ui_mode` 分为四种类型：

| 模式 | 常量 | 说明 | 示例 |
|------|------|------|------|
| 纯后端 | `backend_only` | 仅有后端服务逻辑，无前端 UI | `demo-system` |
| 纯前端 | `frontend_only` | 仅有前端静态页面 | `demo-frontend` |
| 单体 | `monolith` | 前后端在同一个插件目录 | `demo-monolith` |
| 分离 | `separated` | 前后端分属独立目录/工程 | `demo` |

插件目录：`plugins/`，已内置 6 个示例插件：
- `demo`（分离模式，完整前后端示例）
- `demo-backend`（纯后端）
- `demo-frontend`（纯前端）
- `demo-monolith`（单体模式）
- `demo-system`（系统级后端插件）
- `developer-portal`（开发者门户）

## 2. 插件项目结构

以 `demo`（separated 模式）为参考：

```
plugins/demo/
├── plugin.yaml          # 【必需】插件清单（元数据）
├── backend/             # 后端代码
│   ├── main.go          # 独立后端入口
│   ├── handler.go       # HTTP Handler
│   ├── service.go       # 业务逻辑
│   └── handler_test.go  # 测试
├── frontend/            # 前端工程（独立 Vite/Vue 项目）
│   ├── package.json
│   ├── vite.config.ts
│   ├── src/
│   └── dist/
└── migrations/          # 插件数据库迁移（可选）
    ├── 001_init.up.sql
    └── 001_init.down.sql
```

monolith 模式示例（`demo-monolith`）：

```
plugins/demo-monolith/
├── plugin.yaml
├── handler.go           # HTTP Handler（含业务逻辑）
├── main.go              # 入口
├── service.go           # 服务层
└── static/              # 前端静态资源（HTML/CSS/JS）
    ├── index.html
    ├── style.css
    └── app.js
```

## 3. manifest.yaml 编写

插件清单 `plugin.yaml` 是插件的核心元数据文件，Schema 定义见 `docs/schemas/plugin-manifest.schema.json`。

### 3.1 必填字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 插件唯一标识（如 `demo`、`crm-order`） |
| `name` | string | 插件显示名称 |
| `version` | string | 插件版本号（如 `0.2.0`） |

### 3.2 完整字段示例

```yaml
id: demo
name: "Demo Separated Plugin"
name_zh_cn: "示例分离插件"
name_en_us: "Demo Separated Plugin"
version: 0.2.0
api_version: v1
compatibility_skoll: ">=1.0.0 <2.0.0"
migration_version: v0.1.0
ui_mode: separated          # backend_only / frontend_only / monolith / separated
level: app                  # system / app
mount_policy: admin         # admin / user / mixed
ui_nav_position: sidebar    # none / sidebar / top_tab
ui_open_mode: integrated    # integrated / standalone
ui_tab_mode: fixed          # optional / fixed / disabled
app_id: demo                # 仅 level=app 时需要
ui_menu:                    # 可选，声明集成式插件的侧栏菜单
  label_zh_cn: "示例插件"
  label_en_us: "Demo Plugin"
  path: /skoll/plugins/demo
  icon: plugins
  order: 100
  required_permissions:
    - menu.read
config_schema:              # 可选，声明插件配置表单
  title_zh_cn: "示例插件配置"
  title_en_us: "Demo Plugin Config"
  fields:
    - key: demo.endpoint
      label_zh_cn: "服务地址"
      label_en_us: "Endpoint"
      type: string
      required: true
      min_length: 8
      max_length: 120
      pattern: "^https?://"
      placeholder: "https://api.example.com"
    - key: demo.mode
      label_zh_cn: "运行模式"
      label_en_us: "Mode"
      type: select
      default: safe
      options:
        - value: safe
          label_zh_cn: "安全"
          label_en_us: "Safe"
        - value: fast
          label_zh_cn: "快速"
          label_en_us: "Fast"
    - key: demo.enabled
      label_zh_cn: "启用"
      label_en_us: "Enabled"
      type: boolean
      default: true
i18n_locales:               # 非 backend_only 必须声明
  - zh-CN
  - en-US
permissions:
  - menu.read
  - route.read
dependencies:               # 可选，声明依赖其他插件
  - id: auth
    version: ">=1.0.0"
```

### 3.3 字段约束

- **api_version**：若声明则必须同时声明 `compatibility_skoll`
- **i18n_locales**：非 `backend_only` 插件必须声明至少一个 locale
- **ui_open_mode=standalone** 时：`ui_nav_position` 必须为 `none`，`ui_tab_mode` 必须为 `disabled`
- **ui_menu.path**：若声明必须以 `/` 开头；当前前端支持 `dashboard/users/roles/permissions/audit/plugins/settings` 图标名，未知图标会回退为 `plugins`
- **ui_menu.required_permissions / required_roles**：用于前端菜单可见性过滤，并会继承到集成式插件前端路由守卫；后端 API 仍必须独立做权限校验兜底
- **config_schema.fields**：用于插件配置面板的结构化表单渲染；当前支持 `string`、`textarea`、`number`、`boolean`、`select`
- **config_schema.fields[].key**：必须唯一，保存时作为配置 JSON 的字段名；`select` 类型必须提供至少一个 `options`
- **config_schema 字段校验**：支持 `required`、`min`、`max`、`min_length`、`max_length`、`pattern`；前端会实时校验并阻止保存无效配置
- **level=app** 时：必须提供 `app_id`，且 `app_id` 不能为 `skoll`
- **level=system** 时：不能设置 `app_id`

### 3.4 Manifest 校验

```bash
# 单插件校验（CLI 工具）
go run ./internal/handler/cli/plugin_cmd.go validate plugins/demo

# 批量校验（CLI 工具）
go run ./internal/handler/cli/plugin_cmd.go validate-all plugins

# 通过 Dev Portal API（需开启开发者门户）
POST /skoll/v1/plugins/dev/manifest/validate
{ "pluginsRoot": "plugins", "pluginId": "demo", "manifest": "<yaml_content>" }
```

## 4. 扩展点注册

插件系统支持三种扩展点（`internal/plugin/registry.go` → `MemoryRegistry`）：

### 4.1 路由扩展（Routes）

插件注册自定义 HTTP 路由，由 `router.go` 的 `registerPluginExtensionRoutes` 动态挂载到 API Mux。

```go
// 路由示例（支持标准 HTTP 方法）
route := plugin.Route{
    Method: "GET",
    Path:   "/demo-separated/overview",
}
```

路由通过 `plugin.yaml` 的扩展信息声明，由 RuntimeManager 在 `Enable` 时注册。

### 4.2 中间件扩展（Middleware）

插件可注册自定义中间件，插入到中间件链中。

### 4.3 事件扩展（Events）

插件可注册事件处理器，订阅事件总线消息（内存 / Redis）。

## 5. 插件生命周期

```
安装（Install） → 已安装（Installed）
                  ├── 启用（Enable） → 已启用（Enabled）
                  │                     ├── 禁用（Disable） → 已禁用（Disabled）
                  │                     └── 卸载（Uninstall） → 已卸载（Uninstalled）
                  └── 卸载（Uninstall） → 已卸载（Uninstalled）
```

系统内置插件（`system_builtin=true`）受保护，不可禁用或卸载。

### Manager 接口（`internal/plugin/manager.go`）

| 方法 | 说明 |
|------|------|
| `Install(path)` | 安装插件（从指定路径加载） |
| `Enable(pluginID)` | 启用插件（注册扩展点） |
| `Disable(pluginID)` | 禁用插件（卸载扩展点） |
| `Uninstall(pluginID)` | 卸载插件 |
| `List()` | 列出所有插件 |
| `Get(pluginID)` | 获取插件详情 |

## 6. 本地调试

### 6.1 后端调试

```bash
# 启动 Skoll 主进程（插件目录自动扫描）
go run ./cmd/skoll

# 独立调试分离模式后端
cd plugins/demo/backend
go run .
# 监听 :18081，可独立测试 http://127.0.0.1:18081/demo-separated/overview
```

### 6.2 前端调试

```bash
cd plugins/demo/frontend
npm install
npm run dev
# 前端开发服务器独立运行
```

### 6.3 插件日志

```bash
# 查看插件日志
go run ./internal/handler/cli/plugin_cmd.go logs demo

# 或通过 Dev Portal 日志 API
GET /skoll/v1/plugins/dev/release-tasks/{taskId}/logs
```

## 7. 打包发布流程

### 7.1 使用 Dev Portal

```bash
# 1. 开启开发者门户
SKOLL_DEV_PORTAL_ENABLED=true

# 2. 校验 manifest
POST /skoll/v1/plugins/dev/manifest/validate

# 3. 打包插件制品
POST /skoll/v1/plugins/dev/package
{ "pluginsRoot": "plugins", "pluginId": "demo", "outputDir": "plugins/_dist" }

# 4. 创建发布单
POST /skoll/v1/plugins/dev/release-orders
{ "pluginsRoot": "plugins", "pluginId": "demo", "releaseVersion": "1.0.0", "changelog": "..." }

# 5. 审批
POST /skoll/v1/plugins/dev/release-orders/{orderId}/approve

# 6. 执行发布
POST /skoll/v1/plugins/dev/release-orders/{orderId}/execute
{ "pluginsRoot": "plugins", "pluginId": "demo", "targetEnv": "staging", "artifactPath": "plugins/_dist/demo-1.0.0.zip" }
```

### 7.2 灰度发布

支持三种灰度策略（`strategyType`）：

| 策略 | strategyType | 参数 | 说明 |
|------|-------------|------|------|
| 百分比灰度 | `percent` | `percent`（0-100） | 按百分比逐步放量，最常用策略 |
| 标签灰度 | `tag` | `tags`（字符串数组） | 按用户/环境标签定向灰度 |
| 金丝雀版本 | `canary` | `canaryVersion`（版本号） | 指定金丝雀版本号，先发小范围验证 |

```bash
# 百分比灰度
POST /skoll/v1/plugins/dev/rollout
{ "pluginId": "demo", "strategyType": "percent", "percent": 20, "targetEnv": "production" }

# 标签灰度
POST /skoll/v1/plugins/dev/rollout
{ "pluginId": "demo", "strategyType": "tag", "tags": ["beta-users", "internal"], "targetEnv": "staging" }

# 金丝雀灰度
POST /skoll/v1/plugins/dev/rollout
{ "pluginId": "demo", "strategyType": "canary", "canaryVersion": "2.0.0-rc1", "targetEnv": "production" }
```

#### 回滚与回滚点机制

每次灰度变更自动记录为回滚点（`devRolloutPoint`），持久化到 `.devportal/{pluginId}.rollout.history.json`。回滚点包含版本号（`v1, v2, ...`）、灰度百分比、策略类型和目标环境。

```bash
# 回滚到指定版本
POST /skoll/v1/plugins/dev/rollback
{ "pluginId": "demo", "toVersion": "v2" }

# 回滚到指定百分比
POST /skoll/v1/plugins/dev/rollback
{ "pluginId": "demo", "toPercent": 50 }

# 回滚到上一个版本（默认行为）
POST /skoll/v1/plugins/dev/rollback
{ "pluginId": "demo" }
```

**回滚执行流程**：
1. 从 `rollout.history.json` 加载回滚历史
2. 根据 `toVersion` 或 `toPercent` 定位目标状态
3. 调用 `DevRolloutExecutor.ApplyRollback(previousState, targetStrategy)` 执行回滚
4. 生成新的回滚点记录并持久化

#### 灰度任务追踪

```bash
# 列出灰度/回滚任务
GET /skoll/v1/plugins/dev/rollout-tasks?pluginId=demo&status=success

# 查看任务详情
GET /skoll/v1/plugins/dev/rollout-tasks/{taskId}

# 查看任务日志
GET /skoll/v1/plugins/dev/rollout-tasks/{taskId}/logs
```

## 8. 脚手架工具

通过 Dev Portal API 或 CLI 工具快速创建插件骨架：

```bash
# CLI
go run ./internal/handler/cli/plugin_cmd.go scaffold plugins my-plugin "My Plugin" my-app

# Dev Portal API
POST /skoll/v1/plugins/dev/scaffold
{
  "pluginsRoot": "plugins",
  "pluginId": "my-plugin",
  "pluginName": "My Plugin",
  "appId": "my-app",
  "mode": "workspace"
}
```

生成的标准骨架目录结构：

```
plugins/my-plugin/
├── plugin.yaml
├── static/
│   ├── index.html
│   ├── style.css
│   └── app.js
└── migrations/
```

## 9. 插件间通信

- **事件总线**：通过 `event.Bus`（内存/Redis）在插件间发布/订阅事件
- **依赖声明**：`plugin.yaml` 中的 `dependencies` 声明插件间依赖关系
- **依赖解析**：`RuntimeManager` 在启用阶段通过 `TopologicalResolver` 进行拓扑排序依赖解析

## 10. 安全与签名

插件支持 RSA-SHA256 签名校验（`internal/plugin/signature.go`）：

```yaml
# plugin.yaml 中声明签名
signature:
  algorithm: RSA-SHA256
  timestamp: "2026-01-01T00:00:00Z"
  value: "<base64_encoded_signature>"
  vendor_id: "vendor-org"
  public_key: "<public_key_pem>"
```

## 11. 常见问题

### 插件页面样式"裸奔"

- 确保通过前端路由访问插件页面（`http://127.0.0.1:5173/skoll/plugins/{id}`），而非后端原始 HTTP 接口
- 插件页面通过 iframe 渲染，样式隔离，必须自带 CSS
- 推荐使用类选择器限制作用域（如 `.actions button`），避免全局样式污染

### 多语言不生效

- 宿主通过 `window.__SKOLL_LOCALE` 和 `skoll:locale` 自定义事件注入 locale
- 插件应实现多来源 locale 检测链：`query.locale` → `window.__SKOLL_LOCALE` → `localStorage` → `document.documentElement.lang` → 默认 `zh-CN`

更多细节参考 `docs/development/plugin_dev_tools.md`。
