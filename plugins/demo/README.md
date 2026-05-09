# plugins/demo

## 功能说明
该目录已升级为前后端分离示例，对应重构文档中的推荐方案。

## 目录结构
- plugin.yaml: 插件元数据，声明 ui_mode=separated。
- backend/: 后端示例代码。
	- main.go
	- handler.go
	- service.go
- frontend/: 前端独立工程示例。
	- package.json
	- src/
	- dist/

## 快速验证
- 使用 validate plugins/demo 校验元数据。
- 使用 debug demo 查看插件状态与扩展信息。

