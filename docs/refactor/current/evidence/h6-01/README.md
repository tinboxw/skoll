# H6-01 演练证据

本目录保存部署与恢复演练的机器可读结果。默认语言为中文，报告包含英文摘要。

| 文件 | 内容 |
| --- | --- |
| `deployment-recovery-smoke.json` | 环境、干净部署、探针、备份恢复、前向升级、回滚和清单校验结果 |

复现：

```powershell
$env:SKOLL_H6_MYSQL_PASSWORD = '<local-test-password>'
./scripts/h6-deployment-recovery-smoke.ps1 -AllowDatabaseLifecycle
```

脚本仅创建并清理随机命名的 `skoll_h6_*` 隔离数据库。Docker daemon 不可用时只校验 Compose 配置，并在 JSON 中记录限制。
