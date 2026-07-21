# Docker Compose 部署

该示例启动 MySQL 8 和 Skoll，适用于本地或受控测试环境。业务数据、应用数据、日志和插件包分别保存在 `mysql_data`、`skoll_data`、`skoll_logs`、`skoll_plugins` 卷中。

## 启动

```powershell
$env:SKOLL_MYSQL_PASSWORD = '<database-user-password>'
$env:SKOLL_MYSQL_ROOT_PASSWORD = '<database-root-password>'
$env:SKOLL_SECURITY_JWT_SECRET = '<at-least-32-random-characters>'
docker compose -f deploy/compose/docker-compose.yaml config --quiet
docker compose -f deploy/compose/docker-compose.yaml up --build -d
Invoke-RestMethod http://127.0.0.1:8080/skoll/health
Invoke-RestMethod http://127.0.0.1:8080/skoll/ready
```

可通过 `SKOLL_HTTP_PORT` 修改宿主机端口，通过 `SKOLL_IMAGE` 固定镜像标签。不要把密码写入 Compose 文件或提交到仓库。

## 停止

```powershell
docker compose -f deploy/compose/docker-compose.yaml down
```

普通停止不会删除卷。只有在已经完成并验证备份、且确认要销毁环境时，才执行带 `--volumes` 的命令。

## 验证

```powershell
docker compose -f deploy/compose/docker-compose.yaml config --quiet
docker compose -f deploy/compose/docker-compose.yaml ps
docker compose -f deploy/compose/docker-compose.yaml logs skoll
```
