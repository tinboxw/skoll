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

