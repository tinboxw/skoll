# Skoll 开发文档

本目录面向参与 Skoll 开发、插件扩展和本地调试的开发者。

## 当前文档

| 文档 | 用途 |
| --- | --- |
| [getting-started.md](getting-started.md) | 本地开发快速入门 |
| [plugin-guide.md](plugin-guide.md) | 插件开发教程 |
| [plugin-business-document-ui.en.md](plugin-business-document-ui.en.md) | Public schema-driven business document UI package and plugin composition contract (English) |
| [plugin-api-contract.md](plugin-api-contract.md) | 插件 API、权限与聚合 OpenAPI 契约（默认中文） |
| [plugin-api-contract.en.md](plugin-api-contract.en.md) | Plugin API, permission, and aggregated OpenAPI contract (English) |
| [plugin-event-contract.md](plugin-event-contract.md) | 插件业务事件投递、幂等、重试与生命周期契约（默认中文） |
| [plugin-event-contract.en.md](plugin-event-contract.en.md) | Plugin business event delivery, idempotency, retry, and lifecycle contract (English) |
| [plugin_dev_tools.md](plugin_dev_tools.md) | 插件开发工具、调试和验证流程 |

## 维护规则

1. 开发文档只放可执行、可复用的当前流程。
2. 历史计划、阶段报告和一次性验收材料放入 `docs/archive/`。
3. 新增开发文档时，同步更新本文件和 `docs/README.md`。
4. 示例命令必须写明工作目录和预期结果。
