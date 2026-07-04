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

## M0-03-02: Go 覆盖率统计

- 命令: `go test ./... -coverprofile=coverage.out`
- 后续命令: `go tool cover -func=coverage.out`
- 执行日期: 2026-06-18
- 执行人: Codex
- 结果: Failed

### 结果摘要

`go test ./... -coverprofile=coverage.out` 已执行，当前覆盖率统计基线为失败。

失败原因与 M0-03-01 相同: 本机 cgo 编译器路径不存在，导致依赖 cgo 的包构建失败。

```text
# runtime/cgo
cgo: C compiler "D:\Program Files\JetBrains\CLion 2024.1.1\bin\mingw\bin\gcc.exe" not found: exec: "D:\Program Files\JetBrains\CLion 2024.1.1\bin\mingw\bin\gcc.exe": file does not exist
```

### 覆盖率文件状态

`coverage.out` 未生成，因此 `go tool cover -func=coverage.out` 无法计算完整总覆盖率。

```powershell
if (Test-Path coverage.out) { go tool cover -func=coverage.out } else { Write-Output 'coverage.out missing' }
```

结果:

```text
coverage.out missing
```

### 已观察到的部分包覆盖率

由于全量命令失败，下列覆盖率只来自命令输出中的已通过包，不能作为完整项目总覆盖率:

| 包 | 覆盖率 |
|---|---:|
| `internal/cache` | 93.3% |
| `internal/domain/rbac` | 45.9% |
| `internal/domain/role` | 37.2% |
| `internal/domain/shared` | 100.0% |
| `internal/domain/system` | 100.0% |
| `internal/domain/user` | 38.2% |
| `internal/event` | 76.8% |
| `internal/handler/cli` | 71.1% |
| `internal/handler/http/v1/audit` | 51.5% |
| `internal/handler/http/v1/plugin` | 62.3% |
| `internal/handler/http/v1/system` | 62.5% |
| `internal/handler/http/v1/user` | 64.1% |
| `internal/handler/middleware` | 55.0% |
| `internal/plugin` | 65.9% |
| `internal/service/audit` | 100.0% |
| `internal/service/rbac` | 83.9% |
| `internal/service/role` | 73.5% |
| `internal/service/system` | 85.4% |
| `pkg/config` | 62.9% |
| `pkg/logging` | 70.8% |
| `pkg/metrics` | 90.5% |
| `pkg/security` | 75.6% |
| `pkg/utils` | 46.7% |
| `pkg/validator` | 71.0% |

### 重跑命令

修复本机 cgo 编译器路径后重跑:

```powershell
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

## M0-03-03: 前端构建

- 命令: `cd web && npm run build`
- 执行日期: 2026-06-18
- 执行人: Codex
- 结果: Passed

### 结果摘要

前端生产构建通过。

```text
vite v5.4.21 building for production...
3485 modules transformed.
built in 9.01s
```

主要产物尺寸观察:

| 产物 | 大小 | gzip |
|---|---:|---:|
| `assets/xlsx-DLNWaC59.js` | 332.45 kB | 113.83 kB |
| `assets/el-alert-BQWP3KR4.js` | 164.74 kB | 57.50 kB |
| `assets/index-Cap6jvEn.js` | 115.62 kB | 33.75 kB |
| `assets/vue-DNuytgzo.js` | 109.78 kB | 42.86 kB |
| `assets/index-Cv9E1btV.js` | 95.56 kB | 28.94 kB |
| `assets/el-table-column-BvYiDMwf.js` | 91.21 kB | 31.42 kB |

### Warning 记录

- Dart Sass legacy JS API deprecation warning: legacy JS API will be removed in Dart Sass 2.0.0.
- Rollup removed two `/* #__PURE__ */` comments from `node_modules/@vueuse/core/dist/index.js` because the annotation position could not be interpreted.
- npm notice: local npm has a newer minor version available.

### 重跑命令

```powershell
cd web
npm run build
```

## M0-03-04: 前端 typecheck

- 命令: `cd web && npm run typecheck`
- 脚本: `typecheck = vue-tsc --noEmit`
- 执行日期: 2026-06-18
- 执行人: Codex
- 结果: Passed

### 结果摘要

`web/package.json` 存在 `typecheck` 脚本，且 `npm run typecheck` 执行通过。

```text
> skoll-admin-web@0.1.0 typecheck
> vue-tsc --noEmit
```

未观察到 TypeScript 或 Vue 类型错误输出。

### 重跑命令

```powershell
cd web
npm run typecheck
```

## M0-03-05: 测试副作用记录

- 命令: `git status --short`
- 执行日期: 2026-06-18
- 执行人: Codex
- 结果: Passed

### 门禁后状态

执行 M0-03-01 至 M0-03-04 后，`git status --short` 输出:

```text
?? coverage
```

副作用说明:

- `coverage` 是 `go test ./... -coverprofile=coverage.out` 失败后留下的未跟踪覆盖率文件。
- 文件路径: `D:\workspace\3rdsrc\tinbox\skoll\coverage`
- 文件大小: 302781 bytes

### 修复动作

确认路径位于仓库根目录且为本轮命令生成的未跟踪文件后，已删除该副作用文件。

清理后再次执行 `git status --short`，输出为空。

### 重跑命令

```powershell
git status --short
```
