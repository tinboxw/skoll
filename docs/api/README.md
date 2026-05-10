# API 文档（M7）

## Swagger/OpenAPI
- OpenAPI 规范文件：`docs/api/openapi.yaml`
- 在线查看（Swagger UI）：`GET /docs/swagger`
- 原始规范访问：`GET /docs/openapi.yaml`

## 基础信息
- Base URL: `http://127.0.0.1:8080`
- 响应格式: JSON，统一字段 `code/message/data`
- 非健康检查接口默认要求 `Authorization` 请求头。

## 健康检查
- `GET /health`
- 返回示例:
```json
{
	"code": "ok",
	"message": "ok"
}
```

## 用户接口
- `POST /v1/users`
- `GET /v1/users?offset=0&limit=10`
- `GET /v1/users/{id}`
- `PATCH /v1/users/{id}/email`
- `POST /v1/users/{id}/disable`

`POST /v1/users` 请求示例:
```json
{
	"account": "alice",
	"name": "Alice",
	"email": "alice@example.com",
	"passwordHash": "1234567890abcdef",
	"actorID": "admin-1"
}
```

## 角色接口
- `POST /v1/roles`
- `GET /v1/roles?offset=0&limit=10`
- `POST /v1/roles/{id}/grant`
- `POST /v1/roles/{id}/revoke`

## RBAC 接口
- `POST /v1/rbac/bind`
- `PUT /v1/rbac/policies/{roleId}`
- `POST /v1/rbac/check`

`POST /v1/rbac/check` 请求示例:
```json
{
	"subjectType": "user",
	"subjectId": "u-1",
	"resource": "user:profile",
	"action": "read"
}
```

