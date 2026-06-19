# FE1 Style Variable Inventory

日期: 2026-06-19  
范围: FE1-06  
状态: Accepted baseline

## 目标

抽查 `web/src/styles`、核心 views 和共享 components 的颜色、间距、字号、圆角和状态色来源，确认当前样式基础是否能支撑 FE1 设计系统。此项不做大面积样式重构，只建立变量来源和后续收敛清单。

## 变量来源

`web/src/styles/variables.scss` 当前集中定义:

- Background/surface: `--color-bg`、`--color-bg-accent`、`--color-surface`、`--color-surface-soft`
- Border: `--color-border`、`--color-border-strong`
- Text: `--color-text`、`--color-text-muted`
- Primary: `--color-primary`、`--color-primary-strong`、`--color-on-primary`
- Tags/status: `--color-tag-bg`、`--color-tag-text`、`--color-warning-soft`、`--color-warning-text`、`--color-danger`、`--color-success`
- Shell: `--color-sidebar-bg`、`--color-sidebar-link`、`--color-sidebar-hover`、`--color-sidebar-active`
- Elevation/radius: `--color-shadow`、`--radius-md`、`--radius-lg`

## 抽查结果

| 项目 | 结果 | 说明 |
|---|---|---|
| 颜色来源 | Partial | 主色、文字、边框、surface、状态色大多来自变量。 |
| 硬编码颜色 | Needs follow-up | 除变量定义外仍有 `#fcfdfd`、`#ffffff`、`#182230`、`#f6f8fb` 等少量页面/组件硬编码。 |
| 渐变 | Needs follow-up | Header、PinnedTabs、remote plugin frame 使用 surface 渐变；App shell 使用 radial background；后续需确认不形成营销式背景。 |
| 间距 | Partial | 8/10/12/14/16px 分布广泛，但尚未抽象为 spacing token。 |
| 字号 | Partial | 页面标题常见 1.35rem，Header 1.2rem，辅助信息 0.75-0.95rem；仍缺少 font-size token。 |
| 圆角 | Partial | `--radius-md`、`--radius-lg` 已存在，但仍有 6px、8px、10px、999px 硬编码。 |
| 状态色 | Partial | danger/success/warning 基础变量存在，但 info/critical/risk 等状态没有完整 token。 |

## 需要收敛的锚点

| 文件 | 锚点 | 建议 |
|---|---|---|
| `web/src/styles/global.scss` | `#fcfdfd` auth plugin permissions 背景 | 替换为 surface/soft 变量或新增 subtle surface token。 |
| `web/src/components/Layout/PinnedTabs.vue` | `#ffffff` active tab 背景 | 使用 `--color-surface`。 |
| `web/src/components/Layout/Sidebar.vue` | `#182230` sidebar gradient stop | 提取为 sidebar strong/dark token。 |
| `web/src/views/Plugin/index.vue` | `#f6f8fb` code block 背景 | 使用 `--color-surface-soft` 或 code block token。 |
| 多个 views/components | `gap/padding/font-size/border-radius` 硬编码 | FE1 后续可引入 spacing/font/radius token 或共享 class。 |

## 不需要立即调整的锚点

- `variables.scss` 中的 hex 色值是变量定义源，允许保留。
- `rgba(27, 51, 78, 0.24)` 作为 `--color-shadow` 允许保留。
- `rgba(255, 255, 255, ...)` loading shimmer 可以保留，但应视为组件状态效果，不扩散到页面装饰。
- `999px` 用于 pill、avatar、chip 是合理形状，但后续可以通过 token 命名。

## 后续规则

- 新页面禁止新增页面级硬编码品牌色、状态色、风险色。
- 新页面优先使用 `var(--color-*)`、`var(--radius-*)` 和现有 Element Plus 语义色。
- 如果需要新增颜色，应先进入 `variables.scss`，并说明语义。
- 页面级 spacing/font-size 可以暂按当前模式使用，但 FE1/FE3 后续应逐步收敛到共享 token 或 utility class。
- 不引入单页专属主题，不为插件页面单独创建一套颜色系统。

## 验证记录

首次执行原 Work Item 命令失败:

```powershell
rg "#[0-9A-Fa-f]{3,6}\|var\\(" web/src/styles web/src/views
```

失败原因: `\|` 和 `var\\(` 组合在 ripgrep 正则中导致 unclosed group，不能作为可复制验证命令。

已重试并改为以下可复制命令:

```powershell
rg -n "#[0-9A-Fa-f]{3,8}" web/src/styles web/src/views web/src/components -g "*.scss" -g "*.vue"
rg -n "var\(" web/src/styles web/src/views web/src/components -g "*.scss" -g "*.vue"
rg -n "linear-gradient|radial-gradient|rgba?\(" web/src/styles web/src/views web/src/components -g "*.scss" -g "*.vue"
```

重试结果: Passed。变量来源、硬编码颜色、渐变和状态色来源均已记录。

## FE1-06 验收结论

- Color source: Passed，变量定义和页面引用来源清楚。
- Spacing source: Passed，当前 spacing 分布和 token 缺口已记录。
- Font source: Passed，页面标题、辅助文本和 compact UI 字号分布已记录。
- State colors: Passed，success/warning/danger 变量与 info/risk 缺口已记录。
- No one-page theme: Passed，未发现需要保留的单页独立视觉系统；少量硬编码列入后续收敛。
