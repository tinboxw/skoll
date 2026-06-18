# Skoll 质量基线报告

> 日期：2026-06-09
> 目标：为短期主线 S0 建立真实测试、覆盖率、前端构建和类型检查基线。

## 1. 执行结论

| 门禁 | 命令 | 结果 | 备注 |
| --- | --- | --- | --- |
| Go 全量测试 | `go test ./...` | 通过 | 所有包测试通过 |
| Go 覆盖率 | `go test ./... -coverprofile .\coverage.out` | 通过 | 总覆盖率 57.5% |
| 覆盖率汇总 | `go tool cover -func=.\coverage.out` | 通过 | 已生成覆盖率明细 |
| 前端类型检查 | `cd web && npm run typecheck` | 通过 | 新增 `vue-tsc --noEmit` |
| 前端构建 | `cd web && npm run build` | 通过 | 仍有既有 Sass/VueUse 警告 |

## 2. 本轮新增质量门禁

- 新增前端开发依赖：`vue-tsc`。
- 新增 npm 脚本：`typecheck`，执行 `vue-tsc --noEmit`。
- `web/tsconfig.json` 增加 `skipLibCheck`，避免第三方声明文件噪音影响项目类型检查。
- 修复项目内类型问题：
  - `web/src/plugins/index.ts` 的 iframe `ref` 回调类型。
  - `web/src/views/Permission/index.vue` 的 `permissions` 字段归一化类型。

## 3. 覆盖率现状

总覆盖率：**57.5%**。

当前高覆盖模块：

| 模块 | 覆盖率 |
| --- | ---: |
| `internal/cache` | 93.3% |
| `internal/domain/shared` | 100.0% |
| `internal/domain/system` | 100.0% |
| `internal/service/audit` | 100.0% |
| `internal/service/rbac` | 86.4% |
| `internal/service/system` | 85.4% |
| `pkg/metrics` | 90.5% |

当前主要缺口：

| 模块 | 覆盖率 | 后续动作 |
| --- | ---: | --- |
| `internal/handler/http/v1/rbac` | 0.0% | 补权限绑定、策略保存、权限检查失败场景 |
| `internal/handler/http/v1/role` | 0.0% | 补角色 CRUD、授权、撤权、角色用户查询 |
| `internal/store/memory` | 0.0% | 补内存存储边界与并发基础用例 |
| `internal/store/clickhouse` | 0.0% | 补审计/指标写入和查询 adapter 用例 |
| `internal/store/sql/mysql` | 0.0% | 补 DSN/build/open 配置分支用例 |
| `internal/store/sql/postgres` | 0.0% | 补 DSN/build/open 配置分支用例 |
| `internal/domain/user` | 39.7% | 补用户状态、邮箱、密码规则边界 |
| `internal/domain/role` | 37.2% | 补角色 key、权限集合规则 |
| `internal/domain/rbac` | 45.9% | 补策略匹配、数据范围、拒绝优先级 |
| `pkg/utils` | 46.7% | 补通用工具边界用例 |

## 4. 构建与依赖风险

前端构建通过，但仍存在非阻塞警告：

- Dart Sass legacy JS API deprecation。
- VueUse `/* #__PURE__ */` 注释位置警告。

`npm install -D vue-tsc` 后 npm audit 报告：

- 3 个漏洞：2 个 moderate，1 个 high。
- 暂不执行 `npm audit fix --force`，因为可能引入破坏性升级。
- 下一步应单独评估 `xlsx` 替代或隔离方案。

## 5. 测试副作用

运行 Go 测试会触碰以下 DevPortal fixture 文件：

- `internal/handler/http/v1/plugin/plugins/dev-edit/.devportal/dev-edit.rollout.history.json`
- `internal/handler/http/v1/plugin/plugins/dev-edit/.devportal/dev-edit.rollout.tasks.json`

后续应将相关测试改为临时目录夹具，避免全量测试污染工作区。

## 6. 下一步短期任务

| 顺序 | 任务 | 验收标准 |
| --- | --- | --- |
| 1 | 修复 DevPortal 测试夹具副作用 | `go test ./...` 不再修改 tracked fixture |
| 2 | 补 `handler/http/v1/rbac` 测试 | 覆盖率从 0% 提升到 60%+ |
| 3 | 补 `handler/http/v1/role` 测试 | 覆盖率从 0% 提升到 60%+ |
| 4 | 评估前端依赖漏洞 | 给出 `xlsx` 处理方案，不盲目 force fix |
| 5 | 修复 docs 编码 | README、iteration、status 文档统一 UTF-8 |
| 6 | 将门禁接入 CI | Go test、coverage、web typecheck、web build 全部进入流水线 |
