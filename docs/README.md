# Skoll 项目文档索引

> 最后更新: 2026-06-06

---

## 目录结构

```
docs/
├── README.md                              # 本文档 — 文档总索引
├── configuration.md                       # 配置参考文档
├── feature_implementation_status.md       # 功能实现状态报告 (v2.0)
├── iteration_plan.md                      # 项目迭代计划 (v2.1)
├── plugin_platform_baseline_v1.md         # 插件平台基建蓝图 V1
├── PLUGIN_SECURITY_DESIGN.md              # 插件签名与风险分级设计
│
├── architecture/
│   ├── README.md                          # 架构概览 (v2.0)
│   ├── database.md                        # 数据库设计文档
│   └── rbac.md                            # RBAC 权限模型说明
│
├── api/
│   ├── README.md                          # API 文档概述
│   └── openapi.yaml                       # OpenAPI 规范
│
├── development/
│   ├── README.md                          # 开发文档索引
│   ├── getting-started.md                 # 开发快速入门指南
│   ├── plugin-guide.md                    # 插件开发教程
│   └── plugin_dev_tools.md                # 插件开发工具 (M11)
│
├── user/
│   ├── README.md                          # 用户使用说明 (M7)
│   └── deployment.md                      # 部署运维文档
│
├── schemas/
│   └── plugin-manifest.schema.json        # 插件清单 JSON Schema
│
├── migrations/
│   └── 003_plugin_signature_support.sql   # 插件签名支持迁移脚本
│
└── archive/                               # 历史过程文档归档 (14 份)
```

---

## 一、项目级文档

| 文档 | 说明 |
|------|------|
| [configuration.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\configuration.md>) | 所有配置项的键名、类型、默认值和说明表格，含环境变量映射规则 |
| [feature_implementation_status.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\feature_implementation_status.md>) | 项目功能实现状态报告，标注各模块完成度 |
| [iteration_plan.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\iteration_plan.md>) | 项目迭代计划，M12-M16 里程碑状态 |
| [plugin_platform_baseline_v1.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\plugin_platform_baseline_v1.md>) | 插件平台基建蓝图，描述插件系统整体设计 |
| [PLUGIN_SECURITY_DESIGN.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\PLUGIN_SECURITY_DESIGN.md>) | 插件签名校验、风险分级与安全策略引擎设计 |

## 二、架构文档

| 文档 | 说明 |
|------|------|
| [architecture/README.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\architecture\README.md>) | 整体架构概览：分层设计、依赖关系、技术栈说明 |
| [architecture/database.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\architecture\database.md>) | 7 张核心表结构、GORM Model ↔ Domain 转换、存储模式选择 |
| [architecture/rbac.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\architecture\rbac.md>) | Subject→Binding→Role→PolicyRule 权限模型、DataScope 五级、权限检查流程 |

## 三、API 文档

| 文档 | 说明 |
|------|------|
| [api/README.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\api\README.md>) | API 基地址、JWT 认证方式、通用响应格式、端点分类列表、Swagger 地址 |
| [api/openapi.yaml](<D:\workspace\3rdsrc\tinbox\skoll\docs\api\openapi.yaml>) | OpenAPI 3.0 规范文件 |

## 四、开发文档

| 文档 | 说明 |
|------|------|
| [development/README.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\development\README.md>) | 开发文档入口索引 |
| [development/getting-started.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\development\getting-started.md>) | 环境搭建、本地启动（后端+前端）、健康检查验证、常用命令 |
| [development/plugin-guide.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\development\plugin-guide.md>) | 插件项目结构、manifest.yaml 编写、扩展点注册、打包发布流程 |
| [development/plugin_dev_tools.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\development\plugin_dev_tools.md>) | 插件脚手架工具使用说明 (M11) |

## 五、用户文档

| 文档 | 说明 |
|------|------|
| [user/README.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\user\README.md>) | 用户使用说明：本地启动方式 |
| [user/deployment.md](<D:\workspace\3rdsrc\tinbox\skoll\docs\user\deployment.md>) | Docker/Docker Compose/K8s 部署步骤、环境变量完整参考、健康检查 |

## 六、Schema 与迁移

| 文件 | 说明 |
|------|------|
| [schemas/plugin-manifest.schema.json](<D:\workspace\3rdsrc\tinbox\skoll\docs\schemas\plugin-manifest.schema.json>) | 插件清单 (manifest.yaml) 的 JSON Schema 校验定义 |
| [migrations/003_plugin_signature_support.sql](<D:\workspace\3rdsrc\tinbox\skoll\docs\migrations\003_plugin_signature_support.sql>) | 数据库迁移脚本：为插件表添加签名相关字段 |

---

## 附录：历史文档归档 (archive/)

以下 14 份文档已完成历史使命，移至 `archive/` 目录保存备查：

| 文档 | 类型 |
|------|------|
| audit-e2e-checklist.md | 审计端到端验证清单 |
| audit_logging_enhancement_plan.md | 审计日志增强计划 |
| database_schema_refactor_plan.md | 数据库 Schema 重构计划 |
| m15_quality_gate_report.md | M15 质量门禁报告 |
| P2_P3a_COMPLETION_REPORT.md | P2/P3a 阶段完成报告 |
| plugin_capabilities_plan.md | 插件能力规划 |
| PLUGIN_DB_MIGRATION.md | 插件数据库迁移执行记录 |
| plugin_db_migration_plan.md | 插件数据库迁移计划 |
| refactor.md | 项目重构总计划 |
| p12_integration_report.md | P12 集成测试报告 |
| p12_regression_checklist.md | P12 回归测试清单 |
| plugin_platform_m12_m14_roadmap.md | 插件平台 M12-M14 路线图 |
| refactor_milestone_plan.md | 重构里程碑计划 |
| refactor_milestone_plan_detailed.md | 重构里程碑计划（详细版） |
| refactor_stage_review_log.md | 重构阶段审查日志 |
