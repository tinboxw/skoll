# Skoll 任务微调建议清单

> 日期: 2026-06-19
> 用途: 独立承载巡检后的任务追加、插队、修正建议，供后续开发人员合并到 `task_board.md` 与 `work_items.md`。
> 并发写入规则: 本文档先生成 `.tmp.md`，内容完整后再重命名为正式文件，避免开发人员读取到半成品。
> 约束: Skoll 当前处于开源基础建设阶段，不做旧接口、旧数据结构、旧插件格式、旧页面路径的兼容方案。

## 1. 进度总览

| 项目 | 当前值 | 说明 |
| --- | ---: | --- |
| Work Items 总数 | 170 | 以当前 `docs/refactor/work_items.md` 为准 |
| 已完成 | 170 | 所有已拆分 Work Items 均为 Done |
| 剩余待办 | 0 | 无剩余待办 |
| 阈值判断 | 已完成 | 尾盘优化流程已完成 |

## 2. 问题与风险

| 风险 | 等级 | 说明 | 建议处置 |
| --- | --- | --- | --- |
| 完成态后继续扩散范围 | High | 当前 170 项已全部完成，继续新增功能性任务会破坏收口 | 冻结本轮重构范围，只处理质量门禁失败和文档索引缺口 |
| 文档入口滞后 | Medium | `docs/README.md`、`docs/refactor/README.md` 有并行修改，需要确认是否指向最新完成态 | 发布前执行文档索引校准 |
| 验收记录与提交整理风险 | Medium | acceptance_log 与 git log 已覆盖尾盘项，但工作区仍有未提交文档变更 | 按变更归属拆分提交，不混入其他开发者改动 |

## 3. 剩余任务微调清单

| 建议 ID | 动作 | 建议位置 | 优先级 | 依赖项 | Skill | 预期交付物 | 验收标准 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| CLOSEOUT-20260619-01 | 不新增 | 发布前 | P0 | 所有 Work Items Done | `skoll-refactor-governance` | 完成态冻结说明 | 不再追加新功能性 Work Item，只允许修正文档索引、验收记录和质量门禁失败项 |
| CLOSEOUT-20260619-02 | 校准 | 发布前文档整理 | P0 | FE6 完成 | `skoll-docs-writer` | 文档入口校准记录 | `docs/README.md`、`docs/refactor/README.md` 指向最新计划、巡检、验收和 FE6 性能文档 |
| CLOSEOUT-20260619-03 | 执行 | 发布前质量门禁 | P0 | 文档索引完成 | `skoll-quality-gate` | 最终质量门禁记录 | `go test ./...`、`cd web && npm run typecheck`、`cd web && npm run build` 结果被记录，失败则返工对应项 |

## 4. 后续里程碑细化表

### 里程碑 R1: 文档索引与验收记录冻结

| 原子任务 | Skill | 优先级 | 依赖项 | 预期交付物 | 验收标准 |
| --- | --- | --- | --- | --- | --- |
| CLOSEOUT-20260619-01: 完成态冻结说明 | `skoll-refactor-governance` | P0 | 170/170/0 快照 | 冻结说明 | 不再扩散新范围，不引入旧兼容方案 |
| CLOSEOUT-20260619-02: 文档入口校准 | `skoll-docs-writer` | P0 | 完成态冻结 | 文档入口更新 | README、refactor README、task_board、work_items、acceptance_log 指向一致 |

里程碑验收: 文档入口能让后续开发者直接定位当前计划、任务表、验收日志和 FE6 性能基线。

### 里程碑 R2: 发布前质量门禁

| 原子任务 | Skill | 优先级 | 依赖项 | 预期交付物 | 验收标准 |
| --- | --- | --- | --- | --- | --- |
| CLOSEOUT-20260619-03: 最终质量门禁 | `skoll-quality-gate` | P0 | 文档索引完成 | 门禁执行记录 | `go test ./...`、`cd web && npm run typecheck`、`cd web && npm run build` 结果被记录 |
| 提交整理 | `skoll-refactor-governance` | P0 | 门禁通过 | 清晰提交 | 只提交本轮巡检/收口相关文件，不混入并行开发者无关改动 |

里程碑验收: 质量门禁通过且提交历史能追溯本轮巡检、尾盘完成和文档收口。
