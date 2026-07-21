# Docker 镜像

`Dockerfile` 使用 Go 1.24 多阶段构建，运行阶段使用非 root 用户 `10001`，并内置 `/skoll/health` 容器健康检查。

## 构建

```powershell
docker build -f deploy/docker/Dockerfile -t skoll:local .
```

## 运行

```powershell
docker run --rm -p 8080:8080 `
  -e SKOLL_SERVER_ADDRESS=:8080 `
  -e SKOLL_API_BASE_PREFIX=/skoll `
  -e SKOLL_STORE_MODE=mysql `
  -e 'SKOLL_STORE_DSN=skoll:<password>@tcp(host.docker.internal:3306)/skoll?charset=utf8mb4&parseTime=True&loc=UTC' `
  -e SKOLL_SECURITY_JWT_SECRET='<at-least-32-random-characters>' `
  -v skoll-data:/app/data `
  -v skoll-logs:/app/log `
  skoll:local
```

生产环境应由密钥管理系统注入 DSN 和 JWT 密钥，并固定不可变镜像标签。文件对象和日志目录需要独立持久化策略。

