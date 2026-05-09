# 集成测试说明

本目录用于跨层验证系统行为，覆盖 bootstrap + 路由 + 中间件 + 服务协同。

## 当前用例
- `http_runner_integration_test.go`: 启动真实 runner，验证
	- `/health` 在匿名访问下返回 200
	- `/v1/users` 未授权访问返回 401

## 运行方式
```bash
go test ./tests/integration -v
```

