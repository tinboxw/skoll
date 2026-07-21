# H6-01 部署、升级与恢复演练报告

> 日期：2026-07-21
> 默认语言：`zh-CN`
> Work Item：`H6-01`
> 结论：通过

## English Summary

The H6-01 rehearsal passed clean MySQL deployment, liveness/readiness probes, checksummed backup, isolated restore, migration 21 to 22, post-upgrade startup, and restore-point rollback. Docker Compose and Kubernetes manifests were validated locally; the Docker daemon was unavailable, so no container runtime launch is claimed.

## 演练环境

| 项目 | 值 |
| --- | --- |
| 主机 | Windows 10.0.26200 |
| Go | 1.24.1 |
| MySQL Server | 5.7.26，本地隔离数据库 |
| MySQL Client | 8.0.31 |
| Docker Compose | 配置解析通过，daemon 未启动 |
| Kubernetes | `kubectl kustomize` 客户端渲染通过，未连接集群 |

## 验收结果

| 阶段 | 结果 | 证据 |
| --- | --- | --- |
| 清单校验 | Pass | Compose 配置可解析；Kustomize 可渲染 Deployment、Service、ConfigMap 及应用数据/插件包 PVC |
| 干净部署 | Pass | 随机 MySQL 库由当前应用创建 47 张表，`/skoll/health` 与 `/skoll/ready` 均通过 |
| 备份 | Pass | `mysqldump` 单事务备份成功，生成 91,705 B SQL 和 SHA-256 元数据 |
| 恢复 | Pass | 备份恢复到独立随机库，标记值从备份中恢复为 `clean-v1` |
| 前向升级 | Pass | 从 21 号基线执行 `20260718_000022_complete_pharma_oa_schema.sql`，目标表由 0 增至 2，升级后服务探针通过 |
| 回滚 | Pass | 升级前恢复点还原到独立库，两张目标表均不存在，业务标记保持 `source-v1` |
| 清理 | Pass | 演练结束自动删除四个随机命名的 `skoll_h6_*` 数据库和临时文件 |

机器证据：`docs/refactor/current/evidence/h6-01/deployment-recovery-smoke.json`。

## 操作约束

- 空库安装由当前应用模型创建 schema；`migrations/mysql/` 只用于明确版本区间的前向增量升级。
- 恢复必须显式传入 `-AllowRecreate`，升级必须显式传入 `-AllowUpgrade`，完整演练必须显式传入 `-AllowDatabaseLifecycle`。
- 密码只从指定环境变量读取，不出现在命令行参数、备份或报告中。
- 当前没有 down migration；回滚采用升级前备份恢复到隔离库并切换流量。
- 不提供旧接口、旧数据结构、旧插件格式或旧页面路径兼容方案。

## 重试记录

| 次数 | 失败原因 | 修复与复验 |
| --- | --- | --- |
| 1 | MySQL 8 客户端向 MySQL 5.7 查询 `COLUMN_STATISTICS` 失败 | 检测客户端能力并加入 `--skip-column-statistics` |
| 2 | migration 文件名解析未组合日期与序号，未选中脚本 | 使用 `YYYYMMDD_NNNNNN` 组合成 14 位版本号 |
| 3 | 本地 MySQL 默认 MyISAM，历史建表索引超过限制 | 每次迁移输入首行设置会话默认引擎为 InnoDB |
| 4 | Windows `Start-Process` 拆分带空格的 init 参数 | 改为临时 UTF-8 SQL 输入，执行后无条件清理 |
| 5 | 历史迁移与当前表名不能从空库连续回放 | 按真实升级语义，从已安装基线恢复后只执行目标增量 |
| 6 | Docker daemon 不可用的探测被 PowerShell 提升为异常 | 使用进程退出码记录 `daemon-unavailable`，不伪报容器运行 |
| 7 | 全链路通过 | 最终机器报告用 54.95 秒完成配置、部署、备份、恢复、升级和回滚 |

## 复现命令

```powershell
$env:SKOLL_H6_MYSQL_PASSWORD = '<local-test-password>'
./scripts/h6-deployment-recovery-smoke.ps1 -AllowDatabaseLifecycle
docker compose -f deploy/compose/docker-compose.yaml config --quiet
kubectl kustomize deploy/k8s | Out-Null
go test ./...
go vet ./...
```

## 影响面

- API/OpenAPI：新增公开 `GET /ready`，与 `/health` 分离并同步两份 OpenAPI。
- 权限：探针属于明确公开边界，不授予业务权限。
- 审计：探针不产生业务审计；备份、恢复和升级报告由运维侧留存。
- migration/seed：未修改产品 migration 或 seed；新增的是受确认开关保护的升级执行工具。
