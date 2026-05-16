# 基准测试说明

本目录用于阶段性能基线记录与回归对比。

## 当前基线
- `cache_benchmark_test.go`
	- `BenchmarkLocalCacheSetGet`: 本地缓存写入与读取热路径。

## 运行方式
```bash
go test ./tests/benchmark -run ^$ -bench . -benchmem -count 3
```

建议记录中位数作为阶段基线，避免单次抖动误判。

## M15 当前基线（2026-05-16）

采样命令：

```bash
go test ./tests/benchmark -run ^$ -bench BenchmarkLocalCacheSetGet -benchmem -count 3
```

采样结果：

1. 188.5 ns/op, 123 B/op, 4 allocs/op
2. 185.4 ns/op, 123 B/op, 4 allocs/op
3. 188.7 ns/op, 123 B/op, 4 allocs/op

中位数：

- 188.5 ns/op
- 123 B/op
- 4 allocs/op

说明：当前仓库仅有一个稳定微基准项，后续可在 RBAC 检查与插件路由热路径补充基准点。

