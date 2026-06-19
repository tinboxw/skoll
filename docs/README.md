# Skoll 文档索引

> 最后更新: 2026-06-19

Skoll 当前处于开源基础建设阶段。文档按读者和用途分层，主入口只保留当前有效文档；历史计划、阶段过程记录和细碎验收材料统一放入 `docs/archive/`。

## 快速入口

| 入口 | 用途 |
| --- | --- |
| [refactor/README.md](refactor/README.md) | 当前重构执行入口、任务规则、收口状态 |
| [refactor/architecture_and_execution_plan.md](refactor/architecture_and_execution_plan.md) | 架构模型、治理原则、里程碑计划 |
| [refactor/task_board.md](refactor/task_board.md) | 父任务表和里程碑级状态 |
| [refactor/work_items.md](refactor/work_items.md) | 最小 Work Item 表，作为执行、验收、提交单元 |
| [refactor/next_work_items.md](refactor/next_work_items.md) | 下一批 M3-M7 候选最小任务池 |
| [refactor/acceptance_log.md](refactor/acceptance_log.md) | 已完成任务的验收记录 |
| [refactor/frontend-foundation.md](refactor/frontend-foundation.md) | 前端体验、视觉系统、架构和核心页面规范 |
| [refactor/frontend-quality-performance.md](refactor/frontend-quality-performance.md) | 前端测试、浏览器验收、响应式、性能基线 |
| [refactor/milestone-closeout.md](refactor/milestone-closeout.md) | M2、FE0-FE6、尾盘优化和发布前收口摘要 |

## 项目级文档

| 文档 | 用途 |
| --- | --- |
| [configuration.md](configuration.md) | 配置项、默认值、环境变量映射 |
| [development_skills.md](development_skills.md) | Codex 项目专用 skills 与协作规则 |
| [collaboration.md](collaboration.md) | 团队职能、沟通节奏、任务流转和验收协作机制 |

## 架构文档

| 文档 | 用途 |
| --- | --- |
| [architecture/README.md](architecture/README.md) | 整体架构概览 |
| [architecture/database.md](architecture/database.md) | 数据库设计 |
| [architecture/rbac.md](architecture/rbac.md) | RBAC 权限模型 |

## API 文档

| 文档 | 用途 |
| --- | --- |
| [api/README.md](api/README.md) | API 概览、认证、响应格式 |
| [api/openapi.yaml](api/openapi.yaml) | OpenAPI 规范 |

## 开发者文档

| 文档 | 用途 |
| --- | --- |
| [development/README.md](development/README.md) | 开发文档入口 |
| [development/getting-started.md](development/getting-started.md) | 开发快速入门 |
| [development/plugin-guide.md](development/plugin-guide.md) | 插件开发教程 |
| [development/plugin_dev_tools.md](development/plugin_dev_tools.md) | 插件开发工具 |

## 用户文档

| 文档 | 用途 |
| --- | --- |
| [user/README.md](user/README.md) | 用户使用说明 |
| [user/deployment.md](user/deployment.md) | 部署与运维 |

## Schema 与迁移

| 文档 | 用途 |
| --- | --- |
| [schemas/plugin-manifest.schema.json](schemas/plugin-manifest.schema.json) | 插件 manifest JSON Schema |
| [migrations/003_plugin_signature_support.sql](migrations/003_plugin_signature_support.sql) | 插件签名支持迁移脚本 |

## 归档

| 目录 | 用途 |
| --- | --- |
| [archive/legacy-plans](archive/legacy-plans) | 旧计划，仅作历史参考 |
| [archive/refactor-2026-06-19](archive/refactor-2026-06-19) | 本轮重构拆分出的阶段过程文档和细节证据 |

## 文档维护规则

1. 当前执行入口只放在 `docs/refactor/README.md`。
2. 不做旧接口、旧数据结构、旧插件格式、旧页面路径的兼容方案。
3. 新增顶层文档时必须同步更新本文档。
4. 阶段过程记录优先归档，主目录只保留可持续维护的规范、索引、任务和验收文档。
5. 任务通过验收后应同步 `work_items.md`、`acceptance_log.md` 和提交记录。
