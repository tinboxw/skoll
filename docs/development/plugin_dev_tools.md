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
