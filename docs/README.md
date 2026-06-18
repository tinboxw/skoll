# Skoll 项目文档索引

> 最后更新: 2026-06-18

## 当前执行入口

Skoll 当前处于开源基础建设重构阶段。重构执行入口统一放在:

| 文档 | 说明 |
|---|---|
| [refactor/README.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\refactor\README.md>) | 当前重构执行入口、执行顺序、提交规则 |
| [refactor/architecture_and_execution_plan.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\refactor\architecture_and_execution_plan.md>) | 架构模型、治理原则、里程碑计划 |
| [refactor/task_board.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\refactor\task_board.md>) | 父任务表，含里程碑级任务 |
| [refactor/work_items.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\refactor\work_items.md>) | 细粒度 Work Item 表，作为最小执行和提交单元 |
| [refactor/frontend_experience_plan.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\refactor\frontend_experience_plan.md>) | 前端体验、视觉、编码、测试、性能专项计划 |
| [refactor/acceptance_log.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\refactor\acceptance_log.md>) | 任务验收记录模板 |
| [refactor/quality_baseline.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\refactor\quality_baseline.md>) | M0 质量基线命令结果、失败原因和重跑命令 |
| [refactor/gin-vue-admin_analysis.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\refactor\gin-vue-admin_analysis.md>) | gin-vue-admin 对标分析 |
| [refactor/source_map.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\refactor\source_map.md>) | 旧文档吸收记录与归档说明 |

重构硬规则:

- 不做旧接口、旧数据结构、旧插件格式、旧页面路径的兼容方案。
- 每个任务必须有验收要求。
- 验收失败必须返工。
- 每完成一个任务并通过验收后，提交一次代码。

## 文档目录

```text
docs/
├── README.md
├── configuration.md
├── development_skills.md
├── refactor/
│   ├── README.md
│   ├── architecture_and_execution_plan.md
│   ├── task_board.md
│   ├── work_items.md
│   ├── frontend_experience_plan.md
│   ├── acceptance_log.md
│   ├── quality_baseline.md
│   ├── gin-vue-admin_analysis.md
│   └── source_map.md
├── architecture/
│   ├── README.md
│   ├── database.md
│   └── rbac.md
├── api/
│   ├── README.md
│   └── openapi.yaml
├── development/
│   ├── README.md
│   ├── getting-started.md
│   ├── plugin-guide.md
│   └── plugin_dev_tools.md
├── user/
│   ├── README.md
│   └── deployment.md
├── schemas/
│   └── plugin-manifest.schema.json
├── migrations/
│   └── 003_plugin_signature_support.sql
└── archive/
    ├── legacy-plans/
    └── *.md
```

## 项目级文档

| 文档 | 说明 |
|---|---|
| [configuration.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\configuration.md>) | 配置项、默认值与环境变量映射 |
| [development_skills.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\development_skills.md>) | Codex 项目专用 skills 与协作规则 |

## 架构文档

| 文档 | 说明 |
|---|---|
| [architecture/README.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\architecture\README.md>) | 整体架构概览 |
| [architecture/database.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\architecture\database.md>) | 数据库设计文档 |
| [architecture/rbac.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\architecture\rbac.md>) | RBAC 权限模型说明 |

## API 文档

| 文档 | 说明 |
|---|---|
| [api/README.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\api\README.md>) | API 概述、认证、响应格式 |
| [api/openapi.yaml](<D:\workspace\3rdsrc\tinbox\skoll\docs\api\openapi.yaml>) | OpenAPI 规范 |

## 开发文档

| 文档 | 说明 |
|---|---|
| [development/README.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\development\README.md>) | 开发文档入口 |
| [development/getting-started.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\development\getting-started.md>) | 开发快速入门 |
| [development/plugin-guide.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\development\plugin-guide.md>) | 插件开发教程 |
| [development/plugin_dev_tools.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\development\plugin_dev_tools.md>) | 插件开发工具 |

## 用户文档

| 文档 | 说明 |
|---|---|
| [user/README.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\user\README.md>) | 用户使用说明 |
| [user/deployment.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\user\deployment.md>) | 部署运维文档 |

## Schema 与迁移

| 文档 | 说明 |
|---|---|
| [schemas/plugin-manifest.schema.json](<D:\workspace\3rdsrc\tinbox\skoll\docs\schemas\plugin-manifest.schema.json>) | 插件 manifest JSON Schema |
| [migrations/003_plugin_signature_support.sql](<D:\workspace\3rdsrc\tinbox\skoll\docs\migrations\003_plugin_signature_support.sql>) | 插件签名支持迁移脚本 |

## 历史归档

旧计划类文档已经移动到 [archive/legacy-plans](<D:\workspace\3rdsrc\tinbox\skoll\docs\archive\legacy-plans>)。它们只作为历史参考，不再作为任务执行入口。
