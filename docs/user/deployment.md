# 部署、升级与恢复

Skoll 是 Go 单体应用。默认 API 前缀为 `/skoll`，默认界面语言为中文；生产环境应使用 MySQL 或 PostgreSQL、强随机 JWT 密钥和外部密钥管理。

## 部署方式

| 方式 | 适用场景 | 入口 |
| --- | --- | --- |
| 二进制 | 裸机或 VM | `go build -trimpath -o skoll.exe ./cmd/skoll` |
| Docker | 单容器 | `deploy/docker/Dockerfile` |
| Docker Compose | 本地或受控测试 | `deploy/compose/docker-compose.yaml` |
| Kubernetes | 集群 | `deploy/k8s/kustomization.yaml` |

关键配置：

| 环境变量 | 要求 |
| --- | --- |
| `SKOLL_SERVER_ADDRESS` | 默认 `:8080` |
| `SKOLL_API_BASE_PREFIX` | 默认 `/skoll` |
| `SKOLL_STORE_MODE` | 持久环境使用 `mysql` 或 `postgres` |
| `SKOLL_STORE_DSN` | 从密钥管理系统注入，不写入镜像或仓库 |
| `SKOLL_SECURITY_JWT_SECRET` | 必须替换开发默认值，建议至少 32 个随机字符 |
| `SKOLL_LOG_DIR` / `SKOLL_LOG_FILE` | 确保目录可写或交由标准输出采集 |

## Docker Compose

```powershell
$env:SKOLL_MYSQL_PASSWORD = '<database-user-password>'
$env:SKOLL_MYSQL_ROOT_PASSWORD = '<database-root-password>'
$env:SKOLL_SECURITY_JWT_SECRET = '<at-least-32-random-characters>'
docker compose -f deploy/compose/docker-compose.yaml config --quiet
docker compose -f deploy/compose/docker-compose.yaml up --build -d
Invoke-RestMethod http://127.0.0.1:8080/skoll/health
Invoke-RestMethod http://127.0.0.1:8080/skoll/ready
```

Compose 的 MySQL、应用数据、日志和插件包卷相互独立。普通 `down` 不删除卷；不要在未验证备份时使用 `down --volumes`。

## Kubernetes

```powershell
kubectl create secret generic skoll-runtime `
  --from-literal=SKOLL_STORE_DSN='skoll:<password>@tcp(mysql.example:3306)/skoll?charset=utf8mb4&parseTime=True&loc=UTC' `
  --from-literal=SKOLL_SECURITY_JWT_SECRET='<at-least-32-random-characters>'
kubectl kustomize deploy/k8s | Out-Null
kubectl apply -k deploy/k8s
kubectl rollout status deployment/skoll --timeout=120s
```

基础清单使用非 root 用户、只读根文件系统，并为应用数据和插件包提供独立 PVC。生产 overlay 还必须固定镜像 digest、StorageClass、资源配额、网络策略、Ingress 和 TLS。

## 健康与就绪

| 端点 | 用途 | 认证 |
| --- | --- | --- |
| `GET /skoll/health` | 进程存活和 startup/liveness | 无 |
| `GET /skoll/ready` | 接收流量前的 readiness | 无 |

探针端点只报告运行状态，不执行权限或审计动作。业务 API 仍按 JWT、RBAC 和审计契约执行。

## MySQL 备份与恢复

密码仅通过环境变量传递。备份脚本同时生成 `<backup>.metadata.json`，包含大小和 SHA-256；恢复时会自动校验。

```powershell
$env:SKOLL_MYSQL_PASSWORD = '<database-password>'
./scripts/skoll-mysql-backup.ps1 `
  -Database skoll `
  -OutputPath ./backup/skoll-20260721.sql `
  -HostName 127.0.0.1 -Port 3306 -User skoll

./scripts/skoll-mysql-restore.ps1 `
  -Database skoll_restore_check `
  -InputPath ./backup/skoll-20260721.sql `
  -HostName 127.0.0.1 -Port 3306 -User skoll `
  -AllowRecreate
```

`-AllowRecreate` 表示确认删除并重建指定目标库。脚本拒绝系统数据库名；首次恢复应使用隔离库，核对记录数、关键业务数据和应用探针后再制定切换窗口。文件对象、插件包和外部对象存储需要按各自策略备份。

## 前向升级

`migrations/mysql/` 是已有版本基线上的增量迁移集合，不是空库安装器。空库安装由当前应用模型创建 schema；升级必须先确认当前版本、做恢复点备份，再执行明确版本区间。

```powershell
$env:SKOLL_MYSQL_PASSWORD = '<database-password>'
./scripts/skoll-mysql-upgrade.ps1 `
  -Database skoll `
  -FromExclusive 20260718000021 `
  -ToInclusive 20260718000022 `
  -BackupPath ./backup/skoll-before-22.sql `
  -ReportPath ./backup/skoll-upgrade-22.json `
  -HostName 127.0.0.1 -Port 3306 -User skoll `
  -AllowUpgrade
```

升级脚本先备份，再按文件名前缀顺序执行选定 migration。成功后应启动新版本，验证 `/skoll/health`、`/skoll/ready` 和关键业务读写。

## 回滚

当前不维护 down migration，也不保留旧接口、旧数据结构、旧插件格式或旧页面路径兼容方案。数据库回滚采用恢复点策略：

1. 停止写流量并保留失败升级日志。
2. 使用升级前备份恢复到新的隔离数据库。
3. 用旧版本应用验证健康、就绪和关键业务数据。
4. 完成流量切换；不要在原库上手工逆向删除字段。

## 可重复演练

以下命令会创建并删除随机命名的 `skoll_h6_*` 隔离数据库，必须显式确认：

```powershell
$env:SKOLL_H6_MYSQL_PASSWORD = '<local-test-password>'
./scripts/h6-deployment-recovery-smoke.ps1 -AllowDatabaseLifecycle
```

演练覆盖 Compose 清单、Kustomize、二进制构建、MySQL 干净部署、health/readiness、备份校验、隔离恢复、前向升级和恢复点回滚。Docker daemon 未启动时仍会校验 Compose 配置，并在证据中如实记录环境限制。

## English Summary

Skoll supports binary, Docker Compose, and Kubernetes deployments. Use `/skoll/health` for liveness and `/skoll/ready` for readiness. Keep DSNs and `SKOLL_SECURITY_JWT_SECRET` in secret management. MySQL upgrades require a verified restore-point backup and an explicit migration range; rollback restores that backup into an isolated database before traffic is switched.
