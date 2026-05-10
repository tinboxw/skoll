# plugins/demo

## 功能说明
该目录已升级为前后端分离示例，对应重构文档中的推荐方案。

## 核心功能模块
1. 运行总览模块（backend）
- 接口：`GET /demo-separated/overview`
- 功能：输出插件运行状态、关键高亮信息，适合在插件中心首页展示。

2. 智能建议模块（backend）
- 接口：`GET /demo-separated/recommendations`
- 参数：
	- `active_users`：当前活跃用户数
	- `error_count`：错误请求数
	- `rpm`：每分钟请求数
- 功能：根据上下文自动生成高/中/低优先级运维建议。

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

## 兼容目录说明
- static/ 已退役，不再作为前端资源目录。
- 新增或修改前端资源时，仅使用 frontend/ 目录。

## 快速验证
- 使用 validate plugins/demo 校验元数据。
- 使用 debug demo 查看插件状态与扩展信息。
- 启动独立后端演示服务：`go run ./plugins/demo/backend`
- 示例调用：
	- `curl "http://127.0.0.1:18081/demo-separated/overview?plugin_id=demo"`
	- `curl "http://127.0.0.1:18081/demo-separated/recommendations?active_users=80&error_count=2&rpm=210"`

