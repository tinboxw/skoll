# 医药 OA 数据库生命周期验收报告

> Work Item: `H2-05`
> Date: `2026-07-18`
> Status: `Done`
> Default language: 简体中文

## 结论

H2-05 的数据库生命周期严格门禁已通过。SQLite、MySQL 8.0.31 和 PostgreSQL 16.14 均实际运行；MySQL/PostgreSQL `000019` 至 `000022` migration 在专用空库中各执行两遍，29 张业务表、列和命名索引与 `SchemaBaseline()` 一致。主数据、订单/库存、工作流业务记录 repository contract 全部通过。

文件 SQLite 的 fresh install、既有主数据 schema 升级、重复 seed、进程重启、备份恢复也全部通过。严格命令整体退出码为 0，没有数据库用例被跳过。

## 验收环境

| Item | Value |
| --- | --- |
| OS | Windows NT 10.0.26200.0, amd64 |
| Go | go1.24.1 |
| GORM drivers | sqlite/mysql/postgres v1.6.0 |
| SQLite | 文件数据库，实际执行完整生命周期与 repository contract |
| MySQL | MySQL Community Server 8.0.31，隔离数据目录，`127.0.0.1:13306` |
| PostgreSQL | EDB PostgreSQL 16.14 便携二进制，隔离数据目录，`127.0.0.1:15432` |
| Test database | 两个实例均使用专用 `skoll_acceptance`，migration 测试拒绝重置非 acceptance 命名数据库 |
| Go cache/temp | `D:\skoll-h2-05-db\gocache`、`D:\skoll-h2-05-db\gotmp` |

本机原有 3306 MySQL 实例未修改、未重置账号、未执行验收数据写入。

## 交付物

| Deliverable | Result | Location |
| --- | --- | --- |
| 数据库验收脚本 | Pass | `scripts/smoke-pharma-oa-database.ps1` |
| seed 重启幂等恢复 | Pass | `internal/service/pharmaoa/demo_seed_service.go` |
| 生命周期集成测试 | Pass | `internal/service/pharmaoa/database_acceptance_test.go` |
| schema/migration 验收 | Pass | `internal/store/sql/gormrepo/pharmaoa_schema_acceptance_test.go` |
| 公告与冷链 schema 补全 | Pass | `internal/store/sql/gormrepo/pharmaoa_collaboration_model.go`、双数据库 `000022` migration |
| 支持数据库矩阵 | Pass | SQLite、MySQL 8.0.31、PostgreSQL 16.14 均实际运行 |

## Schema 与 migration

最终 `SchemaBaseline()` 包含 29 张真实业务表，并由 `AllModels()` 与 MySQL/PostgreSQL `000019` 至 `000022` migration 完整覆盖。员工、供应商、客户资质随所属主数据聚合以 JSON 持久化；没有引入无领域仓储与服务的独立 `pharma_oa_qualifications` 候选表，也没有旧表或双写兼容层。

`000022` migration 新增：

- `pharma_oa_announcements`
- `pharma_oa_cold_chain_records`

全部生产 migration 为 retain-only，不包含 `DROP` 或 `TRUNCATE`。验收测试中的重置逻辑仅允许作用于 `skoll_acceptance` 或 `_acceptance` 后缀的专用测试数据库。

真实 PostgreSQL migration 首轮验收发现客户代码使用 UNIQUE constraint，而 GORM model 声明命名 unique index，导致 migration 后执行 `AutoMigrate` 失败。迁移已统一为显式 `uk_pharma_customers_code` unique index，并通过 migration -> AutoMigrate -> repository 回归。

## 验收结果

| Gate | Result | Evidence |
| --- | --- | --- |
| Fresh install | Pass | 空 SQLite 使用 `AllModels()` 建库；MySQL/PostgreSQL 专用库由真实 migration 从空库创建。 |
| Migration idempotency | Pass | MySQL/PostgreSQL `000019` 至 `000022` 各连续执行两遍，29 张表、列和索引完整。 |
| Upgrade | Pass | 只含主数据模型和既有记录的 SQLite schema 升级到完整模型，原记录保留。 |
| Repeated seed | Pass | 同进程与关闭数据库、重建 repository/service 后重复 seed，均复用相同实体 ID。 |
| Restart | Pass | 重启后采购、销售、出入库、客户跟进、余额和不可变流水保持完整关联。 |
| Backup/restore fixture | Pass | 恢复备份后额外测试记录消失，原 seed 快照仍可复用。 |
| SQLite contracts | Pass | 主数据、订单/库存、工作流记录契约均实际运行。 |
| MySQL contracts | Pass | MySQL 8.0.31 三套 repository contract 均实际运行。 |
| PostgreSQL contracts | Pass | PostgreSQL 16.14 三套 repository contract 均实际运行。 |
| Full Go regression | Pass | `go test ./... -count=1` 在两种外部 DSN 均启用时通过。 |
| Static gate | Pass | `go vet ./...` 通过。 |
| Strict fail-closed | Pass | 缺少任一外部 DSN 时 `-RequireExternalDatabases` 立即失败。 |

## 重试摘要

验收过程按 `Failed -> Doing` 重试并保留详细日志，主要问题包括：migration 覆盖缺口、Windows PowerShell 5 编码、Docker engine HTTP 500、命令超时、系统盘空间不足、PostgreSQL 安装器授权取消、ZIP 解压超时，以及 PostgreSQL UNIQUE constraint/index 命名不一致。所有代码或可恢复环境问题均已修复；最终门禁不再依赖 Docker。

Go 编译缓存和临时目录迁移到 `D:`，避免系统盘空间影响链接结果。PostgreSQL 使用 EDB 官方无安装器 ZIP，以用户态 `initdb`/`pg_ctl` 启动。

## 严格验收命令

```powershell
$env:GOCACHE = 'D:\skoll-h2-05-db\gocache'
$env:GOTMPDIR = 'D:\skoll-h2-05-db\gotmp'
$env:SKOLL_TEST_MYSQL_DSN = 'skoll:skoll@tcp(127.0.0.1:13306)/skoll_acceptance?charset=utf8mb4&parseTime=true&loc=UTC'
$env:SKOLL_TEST_POSTGRES_DSN = 'host=127.0.0.1 user=postgres dbname=skoll_acceptance port=15432 sslmode=disable TimeZone=UTC'
.\scripts\smoke-pharma-oa-database.ps1 -RequireExternalDatabases -Full
```

最终运行耗时约 313 秒，整体退出码为 0，输出明确为 `all gates passed, including MySQL and PostgreSQL`。

## 影响同步

- API/OpenAPI：未新增或修改 HTTP endpoint 与 schema，无需同步。
- 权限：未新增 permission key，现有权限链保持。
- 审计：未新增 audit action，现有业务审计保持。
- Migration/seed：补齐双数据库 schema；seed 增加进程重启后的持久化快照恢复。
- i18n：无新增前端或用户可见运行时文案；验收和架构说明默认使用中文，脚本保持 ASCII 以兼容 Windows PowerShell 5。
- 兼容性：未实现旧接口、旧数据结构、旧表或双写兼容方案。

## 完成状态

变更后的定向 race、build、diff、文档链接和 CodeGraph 复核均通过。H2-05 按 `Review -> Done` 完成，提交格式为 `H2-05: validate pharma database lifecycle`；后续按正式 Work Item 顺序领取 H3-01。
