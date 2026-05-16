# M15 质量门禁采样报告

> 日期: 2026-05-16
> 里程碑: M15-测试体系完善与性能核验

## 1. 执行命令

### 1.1 回归测试

```powershell
go test ./...
```

### 1.2 覆盖率统计

```powershell
$out = "d:/workspace/3rdsrc/tinbox/skoll/coverage.out"
go test ./... -coverprofile="$out"
go tool cover -func="$out"
```

### 1.3 基准采样

```powershell
go test ./tests/benchmark -run ^$ -bench BenchmarkLocalCacheSetGet -benchmem -count 3
```

## 2. 采样结果

### 2.1 全量测试

- 结论：通过。

### 2.2 覆盖率

- 总覆盖率：42.3%
- 判定：未达到阶段目标 80%，需要继续补测。

### 2.4 本轮增量（service/user）

- 本轮新增测试：`internal/service/user/service_impl_test.go`
- 当前包覆盖率：58.1%（`go test ./internal/service/user -cover`）
- 结论：关键业务包覆盖率有提升，但总覆盖率尚未出现显著变化，下一轮需继续补充 handler 层与 store/sql 层测试。

### 2.3 性能基线

- BenchmarkLocalCacheSetGet
  - 样本1：188.5 ns/op, 123 B/op, 4 allocs/op
  - 样本2：185.4 ns/op, 123 B/op, 4 allocs/op
  - 样本3：188.7 ns/op, 123 B/op, 4 allocs/op
  - 中位数：188.5 ns/op, 123 B/op, 4 allocs/op

## 3. 变化对比

- 基线值：N/A（首次在 M15 统一归档）
- 当前值：见上
- 变化百分比：N/A
- 判定：建立了可复现基线，后续可执行同命令进行回归比对。

## 4. 结论

- 允许进入下一阶段的条件（部分满足）：
  - 已有可复现测试与基准命令。
  - 已有第一版性能基线归档。
- 当前阻塞项：
  - 覆盖率仅 42.3%，与目标 80% 差距较大。
  - 基准点过少，尚不足以覆盖关键业务热路径。
