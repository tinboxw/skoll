# Skoll

Skoll 北欧・巨狼｜Go 高性能、高并发

## 项目定位

Skoll 当前完成 M4 发布准备，并已启动 M5 首批通用模块脚手架（用户/角色/菜单/审计日志）。

## 目录结构

```text
.
├── cmd/skoll/                 # 可执行入口
├── internal/app/              # HTTP 传输与启动编排
├── internal/domain/           # 核心领域模型
├── internal/service/          # 核心服务层
├── pkg/version/               # 公共版本信息
├── docs/planning/             # 规划文档
└── docs/milestones/           # 里程碑记录
```

## 快速开始

### 运行服务

```bash
go run ./cmd/skoll -addr :8080
# 可选：启用 go-admin 最小接入
go run ./cmd/skoll -addr :8080 -go-admin-enabled -go-admin-mode dev
# 可选：关停前先置 not_ready 并排空 3 秒
go run ./cmd/skoll -addr :8080 -drain-time 3s
# 可选：通过环境变量统一下发排空/关停超时默认值
SKOLL_DRAIN_TIME=3s SKOLL_SHUTDOWN_TIMEOUT=12s go run ./cmd/skoll -addr :8080
# 可选：为 admin 路由启用可替换鉴权骨架（当前实现: static-token）
SKOLL_ADMIN_AUTH_MODE=static-token SKOLL_ADMIN_AUTH_TOKEN=secret go run ./cmd/skoll -addr :8080 -go-admin-enabled
# 可选：启用 HMAC 签名鉴权模式
SKOLL_ADMIN_AUTH_MODE=hmac-sha256 SKOLL_ADMIN_AUTH_HMAC_SECRET=secret go run ./cmd/skoll -addr :8080 -go-admin-enabled
# 可选：若必须在 prod 模式下使用 static-token，需要显式放行（默认禁止）
SKOLL_ADMIN_AUTH_MODE=static-token SKOLL_ADMIN_AUTH_TOKEN=secret SKOLL_ADMIN_AUTH_ALLOW_STATIC_TOKEN_IN_PROD=true go run ./cmd/skoll -addr :8080 -go-admin-enabled -go-admin-mode prod
# 可选：在多实例场景将 nonce 防重放存储切换到 Redis 共享存储
SKOLL_ADMIN_AUTH_MODE=hmac-sha256 SKOLL_ADMIN_AUTH_HMAC_SECRET=secret SKOLL_ADMIN_AUTH_NONCE_STORE=redis SKOLL_ADMIN_AUTH_NONCE_REDIS_ADDR=127.0.0.1:6379 go run ./cmd/skoll -addr :8080 -go-admin-enabled
# 推荐：使用生产环境模板作为部署基线
# 模板文件：.env.production.example
# 参数约束：shutdown-timeout 必须 > 0，drain-time 必须在 [0s, 30s]
# 行为说明：第一次终止信号触发优雅下线，第二次终止信号可中断排空等待并立即进入 shutdown
# 行为说明：go-admin prod 模式下默认拒绝 static-token，除非显式设置 admin-auth-allow-static-token-in-prod
# 可观测性：启动时会输出 runtime config 快照日志
```

### 健康检查

```bash
curl http://localhost:8080/health
curl http://localhost:8080/ready
curl http://localhost:8080/metrics
# 若启用 admin 鉴权，/metrics 额外包含 skoll_admin_auth_verifications_total 与 skoll_admin_auth_failures_total
# 仅在启用 go-admin 最小接入时可用
curl http://localhost:8080/admin/ping
# 若启用 admin 鉴权骨架，需要传入头
curl -H "X-Admin-Token: secret" http://localhost:8080/admin/ping
# 若启用 hmac-sha256，需要传入 X-Admin-Timestamp(Unix 秒)、X-Admin-Nonce、X-Admin-Signature(HMAC-SHA256 hex)
# 可选：X-Admin-Body-SHA256（请求体 sha256 hex）
```

## 质量门禁与阶段要求

1. 里程碑标签固定为 `M0/M1/M2/M3-简短内容`（示例：`M1-鉴权RBAC`）。
2. PR 必须包含性能对比数据（基线、当前、变化百分比、采样命令）。
3. 基线验证命令：`go test ./...`。
4. 并发相关改动额外执行：`go test -race ./...`。
5. 性能快照建议执行：`go test -bench=. -benchmem ./...`。
6. CI 位于 `.github/workflows/ci.yml`，在模块初始化后自动执行门禁检查。

## 本地开发命令

```bash
go fmt ./...
go test ./...
go test -race ./...
go test -bench=. -benchmem ./...
```

## 文档

- 规划路线：`docs/planning/IMPLEMENTATION_ROADMAP.md`
- 生产部署环境模板：`docs/planning/PRODUCTION_ENV_TEMPLATE.md`
- 基准工具链策略：`docs/planning/BENCHMARK_TOOLCHAIN_POLICY.md`
- Admin HMAC 签名规范：`docs/planning/ADMIN_AUTH_SIGNATURE_CONTRACT.md`
- Admin 鉴权安全运行手册：`docs/planning/ADMIN_AUTH_SECURITY_RUNBOOK.md`
- 开源发布说明（M4）：`docs/releases/M4_OPEN_SOURCE_RELEASE_NOTE.md`
- 版本策略：`docs/community/VERSIONING_POLICY.md`
- 变更日志流程：`docs/community/CHANGELOG_PROCESS.md`
- 贡献指南：`CONTRIBUTING.md`
- M0 基线记录：`docs/milestones/M0-project-baseline.md`
- M1 核心域记录：`docs/milestones/M1-core-domain.md`
- M2 并发与性能记录：`docs/milestones/M2-concurrency-and-performance.md`
- M3 可观测性与加固记录：`docs/milestones/M3-observability-and-hardening.md`
- M3 go-admin 最小接入记录：`docs/milestones/M3-go-admin-minimal-integration.md`
- M3 go-admin 探针接入记录：`docs/milestones/M3-go-admin-probe-hook.md`
- M3 优雅下线排空记录：`docs/milestones/M3-graceful-drain-readiness.md`
- M3 排空参数守卫记录：`docs/milestones/M3-drain-config-guard.md`
- M3 运行时环境变量覆盖记录：`docs/milestones/M3-runtime-env-overrides.md`
- M3 信号感知排空与关停时序记录：`docs/milestones/M3-shutdown-sequence-and-signal-aware-drain.md`
- M3 metrics 路由模板记录：`docs/milestones/M3-metrics-route-template.md`
- M4 admin 鉴权骨架记录：`docs/milestones/M4-admin-auth-skeleton.md`
- M4 admin 可替换鉴权接口记录：`docs/milestones/M4-admin-auth-pluggable-interface.md`
- M4 admin HMAC 鉴权记录：`docs/milestones/M4-admin-auth-hmac-sha256.md`
- M4 admin nonce 存储接口记录：`docs/milestones/M4-admin-auth-nonce-store-interface.md`
- M4 admin 共享 nonce 存储接线记录：`docs/milestones/M4-admin-auth-shared-nonce-store-wiring.md`
- M4 admin 鉴权可观测性记录：`docs/milestones/M4-admin-auth-observability.md`
- M4 admin 鉴权安全运行手册记录：`docs/milestones/M4-admin-auth-security-runbook.md`
- M4 admin prod static-token 守卫记录：`docs/milestones/M4-admin-auth-prod-static-token-guard.md`
- M4 发布清单记录：`docs/milestones/M4-release-checklist.md`
- M5 首批模块脚手架记录：`docs/milestones/M5-initial-module-scaffolds.md`
- 里程碑模板：`docs/milestones/MILESTONE_LOG_TEMPLATE.md`

## 参与贡献

1. Fork 本仓库
2. 新建特性分支
3. 提交代码与验证结果
4. 发起 Pull Request
