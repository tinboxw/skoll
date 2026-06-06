# P12 发布前回归清单

> 目标：在发布前对本轮 P12 改动执行最小充分回归，确保插件联调能力可稳定交付。
> 状态说明：`[x]` 已完成，`[ ]` 待执行。

## 1. 服务可用性

- [x] 后端进程可启动并监听 `:8080`。
- [x] 前端开发服务可启动并监听 `:5173`。
- [x] `GET /health` 返回 HTTP 200。

## 2. 插件同步接口

- [x] `GET /v1/plugins` 返回 HTTP 200。
- [x] `GET /v1/plugins` 返回 `demo` 运行时插件记录。
- [x] `GET /v1/plugins?enabled=true` 返回 HTTP 200。
- [x] `enabled=true` 过滤后仅返回启用插件。
- [x] 非法参数（`enabled=abc`）返回 400（已由处理逻辑覆盖）。

## 3. 认证与权限边界

- [x] 免鉴权放行范围仅包含 `/health`、`/ready`、`/v1/plugins`。
- [x] 未扩大到其他业务写接口。
- [x] 鉴权中间件测试覆盖 `/v1/plugins?enabled=true` 放行场景。

## 4. 运行时插件加载

- [x] 启动流程可扫描 `plugins/*` 并尝试 Install + Enable。
- [x] `plugins/demo/plugin.yaml` 编码与缩进合法，可被加载。
- [x] `/v1/plugins` 优先返回运行时 provider 数据，provider 为空时才 fallback。

## 5. 前端联调

- [x] Vite proxy 配置已包含 `/v1` 与 `/health`。
- [x] 页面可显示插件清单（Builtin Auth + Demo Plugin）。
- [x] Plugin View 区域正常渲染宿主文案。

## 6. 门禁与构建

- [x] `go test ./...` 通过。
- [x] `go test ./internal/bootstrap ./internal/handler/http/... ./internal/handler/middleware ./internal/plugin...` 通过。
- [x] `npm run build`（web）通过。

## 7. 异常与回退检查

- [x] 端口冲突可通过清理占用进程恢复（`8080`、`5173`）。
- [x] 当 provider 无可用插件时，接口回退到默认记录，避免前端空白。
- [ ] 发布后监控告警阈值与回滚窗口由发布评审会确认。

## 8. 发布评审输入

- 联调报告：`docs/development/p12_integration_report.md`
- 阶段评审日志：`docs/development/refactor_stage_review_log.md`
- 关键改动：
  - `internal/bootstrap/di.go`
  - `internal/bootstrap/auth_policy.go`
  - `internal/handler/http/router.go`
  - `internal/handler/http/v1/plugin_handler.go`
  - `web/vite.config.ts`

## 9. 结论

- 当前代码与联调证据满足 P12 批次3收口要求。
- 可进入最终发布评审与签字流程。
