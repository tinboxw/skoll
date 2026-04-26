# Skoll

Skoll 北欧・巨狼｜Go 高性能、高并发

## 项目定位

Skoll 当前完成 M4 发布准备，已启动 M5 首批通用模块脚手架（用户/角色/菜单/审计日志），并完成 M6、M7 基础能力、M8（文件/任务/生成器）、M9（插件生命周期/打包/生态文档）、M10（仪表盘聚合与 UI 启动契约）、M11（仪表盘鉴权会话能力）、M12（仪表盘 JWT 会话桥接能力）、M13-step3（JWT 声明来源溯源字段）、M14-step1（JWT 溯源审计导出字段）、M14-step2（JWT 溯源审计导出文档与 SIEM 映射）、M15-step1（JWT 溯源导出运维指标与告警提示）、M15-step2（JWT 溯源运维 Runbook 与告警分诊）、M16-step1（JWT 溯源 SLO 看板与错误预算策略）、M16-step2（JWT 溯源 SLO 告警规则模板与发布护栏）、M17-step1（JWT 溯源基线重校准工作流与评审节奏）、M17-step2（JWT 溯源重校准证据模板与审批清单）、M18-step1（JWT 溯源阈值变更日志与月度归档流程）、M18-step2（JWT 溯源月度归档样例与审查清单执行示例）、M19-step1（JWT 溯源月度审查自动化清单与轮值负责人指引）、M19-step2（JWT 溯源季度轮值样例与升级交接模板）、M20-step1（JWT 溯源例外治理矩阵与到期复核流程）、M20-step2（JWT 溯源例外样例记录与复核决策日志模板）以及 M21-step1（JWT 溯源例外治理指标与月度趋势看板字段）基线。

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
# M6: admin 模块 API（支持 user/role/menu/audit 的最小可用读写）
curl -X POST http://localhost:8080/admin/v1/users -H "Content-Type: application/json" -d '{"name":"alice","email":"alice@example.com"}'
curl http://localhost:8080/admin/v1/users
curl http://localhost:8080/admin/v1/users/1
curl -X POST http://localhost:8080/admin/v1/roles -H "Content-Type: application/json" -d '{"name":"ops","permissions":["user.read"]}'
curl -X PUT http://localhost:8080/admin/v1/roles/1/menus -H "Content-Type: application/json" -d '{"menu_ids":[1,2]}'
curl http://localhost:8080/admin/v1/roles/1/menus
curl -X PUT http://localhost:8080/admin/v1/roles/1/apis -H "Content-Type: application/json" -d '{"apis":["GET:/admin/v1/users","POST:/admin/v1/users"]}'
curl http://localhost:8080/admin/v1/roles/1/apis
curl http://localhost:8080/admin/v1/apis
curl -X POST http://localhost:8080/admin/v1/menus -H "Content-Type: application/json" -d '{"title":"Dashboard","path":"/dashboard","order":1}'
curl -X POST http://localhost:8080/admin/v1/audit-logs -H "Content-Type: application/json" -d '{"actor":"system","action":"create","target":"user"}'
curl http://localhost:8080/admin/v1/audit-logs?limit=20
curl "http://localhost:8080/admin/v1/audit-logs?page=1&size=20&actor=system&action=create&q=user"
curl -X POST http://localhost:8080/admin/v1/configs -H "Content-Type: application/json" -d '{"key":"system.theme","value":"aurora","description":"ui theme"}'
curl http://localhost:8080/admin/v1/configs
curl http://localhost:8080/admin/v1/configs/system.theme
curl -X POST http://localhost:8080/admin/v1/dictionaries -H "Content-Type: application/json" -d '{"type":"status","label":"Enabled","value":"1","sort":10}'
curl http://localhost:8080/admin/v1/dictionaries
curl http://localhost:8080/admin/v1/dictionaries?type=status
curl -X POST http://localhost:8080/admin/v1/files -F "file=@./README.md"
curl http://localhost:8080/admin/v1/files
curl http://localhost:8080/admin/v1/files/1
curl -L http://localhost:8080/admin/v1/files/1/download -o downloaded.bin
curl -X POST http://localhost:8080/admin/v1/jobs -H "Content-Type: application/json" -d '{"name":"daily-sync","schedule":"0 0 * * *"}'
curl http://localhost:8080/admin/v1/jobs
curl -X POST http://localhost:8080/admin/v1/jobs/1/run
curl http://localhost:8080/admin/v1/jobs/1/history?limit=20
curl -X POST http://localhost:8080/admin/v1/generator/modules -H "Content-Type: application/json" -d '{"module":"billing"}'
curl -X POST http://localhost:8080/admin/v1/plugins/manifests -H "Content-Type: application/json" -d '{"name":"audit-ext","version":"1.0.0","hooks":["on_boot"]}'
curl http://localhost:8080/admin/v1/plugins
curl http://localhost:8080/admin/v1/plugins/audit-ext
curl -X POST http://localhost:8080/admin/v1/plugins/packages/install -H "Content-Type: application/json" -d '{"name":"audit-ext","version":"1.1.0","package_url":"https://example.com/plugins/audit-ext-1.1.0.tgz","package_hash":"sha256:abcd","hooks":["on_boot"]}'
curl -X POST http://localhost:8080/admin/v1/plugins/audit-ext/disable
curl -X POST http://localhost:8080/admin/v1/plugins/audit-ext/enable
curl -X POST http://localhost:8080/admin/v1/plugins/audit-ext/version-check -H "Content-Type: application/json" -d '{"latest_version":"1.2.0"}'
curl http://localhost:8080/admin/v1/system/status
curl http://localhost:8080/admin/v1/system/runtime-metrics
curl http://localhost:8080/admin/v1/system/node-health
curl http://localhost:8080/admin/v1/system/dashboard
curl -H "X-Admin-Token: secret" -H "X-Admin-Role-ID: 1" http://localhost:8080/admin/v1/system/dashboard
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
- 对标功能规划：`docs/planning/FEATURE_PARITY_PLAN.md`
- 生产部署环境模板：`docs/planning/PRODUCTION_ENV_TEMPLATE.md`
- 基准工具链策略：`docs/planning/BENCHMARK_TOOLCHAIN_POLICY.md`
- Admin HMAC 签名规范：`docs/planning/ADMIN_AUTH_SIGNATURE_CONTRACT.md`
- Admin 鉴权安全运行手册：`docs/planning/ADMIN_AUTH_SECURITY_RUNBOOK.md`
- 开源发布说明（M4）：`docs/releases/M4_OPEN_SOURCE_RELEASE_NOTE.md`
- 版本策略：`docs/community/VERSIONING_POLICY.md`
- 变更日志流程：`docs/community/CHANGELOG_PROCESS.md`
- 扩展开发指南：`docs/community/EXTENSION_DEVELOPER_GUIDE.md`
- 扩展兼容性策略：`docs/community/EXTENSION_COMPATIBILITY_POLICY.md`
- 仪表盘 UI 启动契约：`docs/community/DASHBOARD_UI_BOOTSTRAP_CONTRACT.md`
- 仪表盘鉴权会话策略：`docs/community/DASHBOARD_AUTH_SESSION_POLICY.md`
- 仪表盘 JWT 中间件桥接合同：`docs/community/DASHBOARD_JWT_MIDDLEWARE_BRIDGE_CONTRACT.md`
- 仪表盘 JWT 声明归一化策略：`docs/community/DASHBOARD_JWT_CLAIMS_NORMALIZATION_POLICY.md`
- 仪表盘 JWT 溯源审计导出指南：`docs/community/DASHBOARD_JWT_PROVENANCE_AUDIT_EXPORT_GUIDE.md`
- 仪表盘 JWT 溯源运维 Runbook：`docs/community/DASHBOARD_JWT_PROVENANCE_OPS_RUNBOOK.md`
- 仪表盘 JWT 溯源 SLO 策略：`docs/community/DASHBOARD_JWT_PROVENANCE_SLO_POLICY.md`
- 仪表盘 JWT 溯源 SLO 告警规则：`docs/community/DASHBOARD_JWT_PROVENANCE_SLO_ALERT_RULES.md`
- 仪表盘 JWT 溯源基线重校准流程：`docs/community/DASHBOARD_JWT_PROVENANCE_BASELINE_RECALIBRATION.md`
- 仪表盘 JWT 溯源重校准证据模板：`docs/community/DASHBOARD_JWT_PROVENANCE_RECALIBRATION_EVIDENCE_TEMPLATE.md`
- 仪表盘 JWT 溯源阈值变更日志：`docs/community/DASHBOARD_JWT_PROVENANCE_THRESHOLD_CHANGE_LOG.md`
- 仪表盘 JWT 溯源月度审查自动化：`docs/community/DASHBOARD_JWT_PROVENANCE_REVIEW_AUTOMATION.md`
- 仪表盘 JWT 溯源例外治理矩阵：`docs/community/DASHBOARD_JWT_PROVENANCE_EXCEPTION_GOVERNANCE.md`
- 仪表盘 JWT 溯源例外治理指标：`docs/community/DASHBOARD_JWT_PROVENANCE_EXCEPTION_METRICS.md`
- 仪表盘 JWT 溯源例外样例记录：`docs/milestones/JWT_PROVENANCE_EXCEPTION_RECORDS_2026-05.md`
- 仪表盘 JWT 溯源月度归档样例：`docs/milestones/JWT_PROVENANCE_THRESHOLD_CHANGE_ARCHIVE_2026-04.md`
- 仪表盘 JWT 溯源季度轮值样例：`docs/milestones/JWT_PROVENANCE_REVIEW_ROTATION_2026-Q2.md`
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
- M6 admin 模块 API 基线记录：`docs/milestones/M6-admin-module-api-baseline.md`
- M6 角色绑定能力记录：`docs/milestones/M6-role-bindings.md`
- M6 API 注册表与权限分配记录：`docs/milestones/M6-api-registry-and-permission-assignment.md`
- M6 RBAC 端到端冒烟记录：`docs/milestones/M6-rbac-e2e-smoke.md`
- M7 存储适配器契约记录：`docs/milestones/M7-storage-adapter-contract.md`
- M7 配置中心与字典记录：`docs/milestones/M7-config-and-dictionary.md`
- M7 审计日志分页过滤记录：`docs/milestones/M7-durable-audit-log.md`
- M8 文件服务基线记录：`docs/milestones/M8-file-service-baseline.md`
- M8 调度任务基线记录：`docs/milestones/M8-job-scheduler-baseline.md`
- M8 模块生成器记录：`docs/milestones/M8-module-generator.md`
- M9 插件清单生命周期记录：`docs/milestones/M9-plugin-manifest-lifecycle.md`
- M9 扩展打包与版本检查记录：`docs/milestones/M9-extension-packaging.md`
- M9 生态文档与兼容性策略记录：`docs/milestones/M9-ecosystem-docs.md`
- M10 系统状态 API 记录：`docs/milestones/M10-system-status-api.md`
- M10 运行时指标快照 API 记录：`docs/milestones/M10-dashboard-runtime-metrics.md`
- M10 节点与依赖健康摘要 API 记录：`docs/milestones/M10-dashboard-node-health.md`
- M10 仪表盘聚合 API 记录：`docs/milestones/M10-dashboard-aggregation.md`
- M10 仪表盘 UI 启动契约记录：`docs/milestones/M10-dashboard-ui-bootstrap-contract.md`
- M11 仪表盘鉴权会话对齐记录：`docs/milestones/M11-dashboard-auth-session-alignment.md`
- M11 仪表盘鉴权会话策略文档记录：`docs/milestones/M11-dashboard-auth-session-policy-docs.md`
- M11 仪表盘鉴权会话可观测性记录：`docs/milestones/M11-dashboard-auth-session-observability.md`
- M11 仪表盘鉴权会话可执行指引记录：`docs/milestones/M11-dashboard-auth-session-actionability.md`
- M12 仪表盘 JWT 会话启动字段记录：`docs/milestones/M12-dashboard-jwt-session-bootstrap.md`
- M12 仪表盘 JWT 刷新/过期提示记录：`docs/milestones/M12-dashboard-jwt-session-refresh-hints.md`
- M12 仪表盘 JWT 校验态提示记录：`docs/milestones/M12-dashboard-jwt-session-verification-state.md`
- M12 仪表盘 JWT 中间件桥接字段记录：`docs/milestones/M12-dashboard-jwt-session-middleware-bridge.md`
- M12 仪表盘 JWT 中间件桥接合同文档记录：`docs/milestones/M12-dashboard-jwt-session-bridge-contract-docs.md`
- M13 仪表盘 JWT 已验证声明适配器对齐记录：`docs/milestones/M13-dashboard-jwt-session-verified-claims-adapter.md`
- M13 仪表盘 JWT 声明归一化策略记录：`docs/milestones/M13-dashboard-jwt-session-claims-normalization.md`
- M13 仪表盘 JWT 声明来源溯源记录：`docs/milestones/M13-dashboard-jwt-session-source-provenance.md`
- M14 仪表盘 JWT 溯源审计导出字段记录：`docs/milestones/M14-dashboard-jwt-session-provenance-audit-export.md`
- M14 仪表盘 JWT 溯源审计导出文档与 SIEM 映射记录：`docs/milestones/M14-dashboard-jwt-session-provenance-audit-docs.md`
- M15 仪表盘 JWT 溯源运维指标与告警提示记录：`docs/milestones/M15-dashboard-jwt-session-provenance-ops-metrics.md`
- M15 仪表盘 JWT 溯源运维 Runbook 与告警分诊记录：`docs/milestones/M15-dashboard-jwt-session-provenance-ops-runbook.md`
- M16 仪表盘 JWT 溯源 SLO 看板与错误预算策略记录：`docs/milestones/M16-dashboard-jwt-session-provenance-slo-dashboards.md`
- M16 仪表盘 JWT 溯源 SLO 告警规则模板与发布护栏记录：`docs/milestones/M16-dashboard-jwt-session-provenance-slo-alert-rules.md`
- M17 仪表盘 JWT 溯源基线重校准工作流与评审节奏记录：`docs/milestones/M17-dashboard-jwt-session-provenance-baseline-recalibration.md`
- M17 仪表盘 JWT 溯源重校准证据模板与审批清单记录：`docs/milestones/M17-dashboard-jwt-session-provenance-recalibration-evidence-template.md`
- M18 仪表盘 JWT 溯源阈值变更日志与月度归档流程记录：`docs/milestones/M18-dashboard-jwt-session-provenance-threshold-change-log.md`
- M18 仪表盘 JWT 溯源月度归档样例与审查清单执行示例记录：`docs/milestones/M18-dashboard-jwt-session-provenance-archive-sample.md`
- M19 仪表盘 JWT 溯源月度审查自动化清单与轮值负责人指引记录：`docs/milestones/M19-dashboard-jwt-session-provenance-review-automation.md`
- M19 仪表盘 JWT 溯源季度轮值样例与升级交接模板记录：`docs/milestones/M19-dashboard-jwt-session-provenance-rotation-roster-sample.md`
- M20 仪表盘 JWT 溯源例外治理矩阵与到期复核流程记录：`docs/milestones/M20-dashboard-jwt-session-provenance-exception-governance.md`
- M20 仪表盘 JWT 溯源例外样例记录与复核决策日志模板记录：`docs/milestones/M20-dashboard-jwt-session-provenance-exception-sample-log.md`
- M21 仪表盘 JWT 溯源例外治理指标与月度趋势看板字段记录：`docs/milestones/M21-dashboard-jwt-session-provenance-exception-metrics.md`
- 里程碑模板：`docs/milestones/MILESTONE_LOG_TEMPLATE.md`

## 参与贡献

1. Fork 本仓库
2. 新建特性分支
3. 提交代码与验证结果
4. 发起 Pull Request
