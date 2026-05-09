# P12 联调报告（批次3）

> 阶段：P12 联调收口与发布评审
> 执行日期：2026-05-09
> 执行人：Copilot 协助执行（待用户复核）

## 1. 联调目标

- 验证后端插件同步接口可被前端联通访问。
- 验证接口返回来源为运行时插件（非默认回退）。
- 验证前端页面可展示插件注册信息。

## 2. 环境与启动

- Workspace: `d:/workspace/3rdsrc/tinbox/skoll`
- Backend: `go run ./cmd/skoll`
- Frontend: `cd web && npm run dev`
- Frontend URL: `http://127.0.0.1:5173/`
- API Base: `http://127.0.0.1:8080`

## 3. 用例与结果

| 用例 | 请求 | 预期 | 实测 | 结果 |
| --- | --- | --- | --- | --- |
| 健康检查 | `GET /health` | HTTP 200，服务正常 | `200`，`{"code":"ok","message":"ok"}` | Pass |
| 插件列表 | `GET /v1/plugins` | HTTP 200，返回运行时插件 | `200`，`data` 包含 `demo@0.1.0(enabled=true)` | Pass |
| 启用插件过滤 | `GET /v1/plugins?enabled=true` | HTTP 200，仅返回启用插件 | `200`，`data` 仅包含 `demo@0.1.0(enabled=true)` | Pass |
| 前端插件宿主加载 | 浏览器访问 `/` | 页面加载，展示插件清单 | 标题为 `Skoll Frontend Plugin System`，`Loaded Plugins: 2`，列表含 `Builtin Auth`、`Demo Plugin` | Pass |

## 4. 关键响应证据

### 4.1 `GET /health`

```json
{"code":"ok","message":"ok"}
```

### 4.2 `GET /v1/plugins`

```json
{"code":"ok","message":"ok","data":[{"id":"demo","name":"Demo Plugin","version":"0.1.0","enabled":true}]}
```

### 4.3 `GET /v1/plugins?enabled=true`

```json
{"code":"ok","message":"ok","data":[{"id":"demo","name":"Demo Plugin","version":"0.1.0","enabled":true}]}
```

## 5. 页面联调证据

- 浏览器可见主标题：`Skoll Frontend Plugin System`。
- 页面显示：`Loaded Plugins: 2`。
- Plugin Registry 显示：`Builtin Auth(builtin-auth@1.0.0)` 与 `Demo Plugin(demo@0.1.0)`。
- Plugin View 区域显示宿主状态文案：`Frontend plugin host is ready.`

## 6. 已知现象（非阻塞）

- Vue 控制台存在 feature flag 警告（开发态常见，不影响当前联调结论）。
- 直接访问未注册路由（例如 `/plugins/auth`）会提示 `No match found`，当前不影响插件列表同步与宿主渲染。

## 7. 结论

- P12 批次3联调用例全部通过。
- 前后端插件同步链路已打通，运行时插件数据可达并可视化。
- 可进入发布前评审与签字确认流程。
