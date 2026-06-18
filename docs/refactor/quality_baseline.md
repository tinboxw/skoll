# Skoll 质量基线记录

> 更新时间: 2026-06-18  
> 适用范围: M0 质量基线 Work Items

## M0-03-01: Go 全量测试

- 命令: `go test ./...`
- 执行日期: 2026-06-18
- 执行人: Codex
- 结果: Failed

### 结果摘要

`go test ./...` 已执行，当前全量 Go 测试基线为失败。

通过的代表性包:

- `internal/cache`
- `internal/domain/rbac`
- `internal/domain/role`
- `internal/domain/shared`
- `internal/domain/system`
- `internal/domain/user`
- `internal/event`
- `internal/handler/cli`
- `internal/handler/http/v1/audit`
- `internal/handler/http/v1/plugin`
- `internal/handler/http/v1/system`
- `internal/handler/http/v1/user`
- `internal/handler/middleware`
- `internal/plugin`
- `internal/repository/*`
- `internal/service/audit`
- `internal/service/rbac`
- `internal/service/role`
- `internal/service/system`
- `pkg/*`
- `plugins/demo*`

失败包:

- `cmd/skoll`
- `internal/bootstrap`
- `internal/handler/http`
- `internal/service/user`
- `internal/store`
- `internal/store/sql`
- `internal/store/sql/gormrepo`
- `internal/store/sql/mysql`
- `internal/store/sql/postgres`
- `tests/integration`

### 失败原因

失败来自 cgo 编译器环境缺失:

```text
# runtime/cgo
cgo: C compiler "D:\Program Files\JetBrains\CLion 2024.1.1\bin\mingw\bin\gcc.exe" not found: exec: "D:\Program Files\JetBrains\CLion 2024.1.1\bin\mingw\bin\gcc.exe": file does not exist
```

环境核对:

```powershell
go env CGO_ENABLED CC
Test-Path 'D:\Program Files\JetBrains\CLion 2024.1.1\bin\mingw\bin\gcc.exe'
```

结果:

```text
CGO_ENABLED=1
CC="D:\Program Files\JetBrains\CLion 2024.1.1\bin\mingw\bin\gcc.exe"
Test-Path=False
```

### 重跑命令

修复本机 cgo 编译器路径后重跑:

```powershell
go test ./...
```
