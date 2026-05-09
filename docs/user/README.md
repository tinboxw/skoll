# 用户使用说明（M7）

## 本地启动
```bash
go run ./cmd/skoll
```

默认监听 `:8080`，可通过环境变量覆盖：
- `SKOLL_SERVER_ADDRESS`
- `SKOLL_STORE_MODE` (`memory`/`mysql`/`postgres`)
- `SKOLL_STORE_DSN`
- `SKOLL_JWT_SECRET`

## 健康检查
```bash
curl http://127.0.0.1:8080/health
```

## 创建用户示例
```bash
curl -X POST http://127.0.0.1:8080/v1/users \
	-H "Content-Type: application/json" \
	-H "Authorization: Bearer dev-token" \
	-d '{
		"username":"alice",
		"displayName":"Alice",
		"email":"alice@example.com",
		"passwordHash":"1234567890abcdef",
		"actorID":"admin-1"
	}'
```

## 运行测试
```bash
go test ./...
go test -race ./...
```

## 运行容器
```bash
docker compose -f deploy/compose/docker-compose.yaml up --build
```

