# Skoll 里程碑与收口摘要

> 合并范围: M2 验收、FE0-FE6 巡检、任务微调、尾盘阈值和发布前收口建议。  
> 当前快照: Work Items `170/170/0`。

## 当前状态

| 项目 | 状态 |
| --- | --- |
| M0-M2 | Done |
| FE0-FE6 | Done |
| 调整项 | Done |
| 剩余待办 | 0 |
| 尾盘优化 | 已触发并完成 |

## 已完成里程碑

| 里程碑 | 交付 |
| --- | --- |
| M0 | 架构决策、质量基线、任务验收模板、文档入口 |
| M1 | 权限资源目录、菜单注册中心、权限前后端联动 |
| M2 | 操作、登录、错误、插件、安全审计事件可查询 |
| FE0 | 前端体验、构建、类型、浏览器验收基线 |
| FE1 | Skoll Admin 视觉、布局、表格、表单、状态、响应式规范 |
| FE2 | API client、Pinia、Router、权限、SchemaForm、错误与导出规范 |
| FE3 | Dashboard/User/Role/Permission/Menu/Plugin/Audit/Setting 核心页面升级 |
| FE4 | 插件与开发者门户体验升级 |
| FE5 | 前端测试与验收体系 |
| FE6 | 前端性能与可观测基线 |

## 尾盘优化结论

尾盘触发时剩余 5 项，执行顺序为:

```text
FE6-04 -> FE6-05 -> FE6-06 -> ADJ-FE-20260619-08 -> ADJ-TAIL-20260619-04
```

收口结果:

| 项目 | 结论 |
| --- | --- |
| FE6-04 | 共享数据缓存策略完成 |
| FE6-05 | 插件/Dev Portal 性能检查完成 |
| FE6-06 | 前端性能验收模板完成 |
| ADJ-FE-20260619-08 | 性能回归清单完成 |
| ADJ-TAIL-20260619-04 | 尾盘阈值检查完成 |

## 发布前剩余动作

当前不再新增功能性 Work Item。后续只允许处理以下收口事项:

| 建议 ID | 动作 | Skill | 验收标准 |
| --- | --- | --- | --- |
| CLOSEOUT-20260619-01 | 完成态冻结 | `skoll-refactor-governance` | 不再扩散新范围，不引入旧兼容方案 |
| CLOSEOUT-20260619-02 | 文档入口校准 | `skoll-docs-writer` | `docs/README.md`、`docs/refactor/README.md` 指向当前主文档 |
| CLOSEOUT-20260619-03 | 最终质量门禁 | `skoll-quality-gate` | `go test ./...`、`cd web && npm run typecheck`、`cd web && npm run build` 结果被记录 |

## 发布前检查清单

1. `task_board.md`、`work_items.md`、`acceptance_log.md`、`git log` 对已完成任务一致。
2. 文档入口不再指向已归档的细碎过程文档。
3. 归档文档只作为历史证据，不作为执行入口。
4. 质量门禁通过后再做发布或对外说明。
5. 若门禁失败，新增修复任务只能围绕失败项，不得借机扩大重构范围。
