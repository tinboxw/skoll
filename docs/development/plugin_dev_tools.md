# 插件开发工具（M11）

## 概述
`internal/handler/cli/plugin_cmd.go` 提供插件开发阶段最小工具集，覆盖：
- `list`：查看当前插件清单和状态。
- `debug <pluginId>`：输出插件依赖、权限、来源等调试信息。
- `logs <pluginId>`：读取插件日志文件（默认目录 `plugins/logs/`）。
- `validate <pluginPathOrManifest>`：校验 `plugin.yaml` 元数据格式。

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
- 行为：读取 `plugins/logs/demo.log`（或构造时传入的自定义日志目录）。

### validate
- 输入：`["validate", "plugins/demo"]` 或 `plugin.yaml` 的绝对路径。
- 输出示例：

```text
valid id=demo version=0.1.0 deps=1 perms=2
```

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
