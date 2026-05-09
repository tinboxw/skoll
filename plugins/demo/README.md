# plugins/demo

## 功能说明
第三方插件最小示例目录，用于演示安装、校验与调试流程。

## 文件说明
- plugin.yaml：插件元数据。
- main.go：插件入口示例。
- handler.go：插件处理器示例。
- static/：静态资源目录。

## 快速验证
- 使用 `validate plugins/demo` 校验元数据。
- 使用 `debug demo` 查看插件状态与依赖信息。
- 使用 `logs demo` 查看插件日志输出。

