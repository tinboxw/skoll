# 基准测试说明

本目录用于阶段性能基线记录与回归对比。

## 当前基线
- `cache_benchmark_test.go`
	- `BenchmarkLocalCacheSetGet`: 本地缓存写入与读取热路径。
- `pharma_oa_benchmark_test.go`
	- `BenchmarkPharmaOAEmployeePagedQuery`: 1000 条员工下的关键字、状态和分页组合查询。
	- `BenchmarkPharmaOACustomerScopedQuery`: 1000 条客户下的负责人、区域和分页组合查询。

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

## F12-04 医药 OA 基线（2026-07-17）

采样命令：

```bash
go test ./tests/benchmark -run ^$ -bench BenchmarkPharmaOA -benchtime 100x -count 1
```

| 场景 | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| 员工分页关键字/状态查询 | 1,301,641 | 232,453 | 1,003 |
| 客户负责人/区域范围查询 | 1,094,226 | 172,257 | 1,504 |

该基线用于进程内实现回归，不代表生产数据库容量上限。

