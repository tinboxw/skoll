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
- .skoll/signature-assets.json: 示例签名覆盖清单，记录 manifest、backend entry 和 frontend dist 资产 hash。
- backend/: 后端示例代码。
	- main.go
	- handler.go
	- service.go
- frontend/: 前端独立工程示例。
	- package.json
	- src/
	- dist/

## 当前目录约定
- 前端资源统一放在 `frontend/` 目录。

## 快速验证
- 使用 validate plugins/demo 校验元数据。
- 使用 manifest/schema review 确认 ui_menu、config_schema、permissions、risk 和签名资产覆盖清单完整。
- 使用 debug demo 查看插件状态与扩展信息。
- 启动独立后端演示服务：`go run ./plugins/demo/backend`
- 示例调用：
	- `curl "http://127.0.0.1:18081/demo-separated/overview?plugin_id=demo"`
	- `curl "http://127.0.0.1:18081/demo-separated/recommendations?active_users=80&error_count=2&rpm=210"`

## Manifest 能力覆盖

| 能力 | 字段 |
|---|---|
| 菜单注册 | `ui_menu.key=plugin.demo`, `ui_menu.path=/skoll/plugins/demo`, `required_permissions=demo.menu.read` |
| 配置 schema | `demo.endpoint`, `demo.mode`, `demo.enabled` |
| 权限目录 | `demo.menu.read`, `demo.route.read`, `demo.report.export` |
| 风险字段 | `demo.report.export` 标记为 `medium`，其余最小读取权限为 `low` |
| 签名覆盖 | `.skoll/signature-assets.json` 覆盖 `plugin.yaml`、`backend/main.go` 和 `frontend/dist/*` |

## 生命周期验收

见 [lifecycle_acceptance.md](lifecycle_acceptance.md)，记录 install/enable/disable/upgrade/rollback、审计测试覆盖和前端 build 门禁。

