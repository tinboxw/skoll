# 插件测试契约

Skoll 为独立插件提供两套当前测试入口：

- Go：`github.com/tinboxw/skoll/pkg/plugintest`
- 前端：`@skoll/plugin-test`

插件测试不得导入 `internal/`，不得复制宿主测试桩，也不得保留旧契约、兼容分支或降级路径。

## Go 宿主与进程测试

`pkg/plugintest` 提供确定性时钟、JWT 身份、租户/组织/所有者范围、事务、数据存储、事件、单号、文件、审计、配置、密钥、工作流、任务、故障注入、插件包和受管进程生命周期。

```go
clock := plugintest.NewClock(time.Date(2026, 7, 27, 8, 0, 0, 0, time.UTC))
services, err := plugintest.NewServices(plugintest.ServicesOptions{
    Clock: clock,
    Identity: plugintest.Identity{
        Subject: "tester",
        TenantID: "tenant-1",
        OrganizationID: "org-1",
    },
})
if err != nil {
    t.Fatal(err)
}
host, err := services.Host("my-plugin")
if err != nil {
    t.Fatal(err)
}
```

用 `Failures.FailNext` 验证一次性故障后的回滚和重试；用 `Failures.Set` 验证持续不可用；验收结束后必须清除持续故障。

```go
services.Failures.FailNext(
    plugintest.OperationJobLeaseDue,
    errors.New("temporary unavailable"),
)
```

`RunConcurrent` 使用同步起跑栅栏制造真实争用，并按 worker 索引稳定收集结果。`GeneratePropertyCases` 使用固定 seed 生成属性样例；失败信息包含 seed 和 case 编号，可直接重放。

```go
report, err := plugintest.RunConcurrent(ctx, 32, func(ctx context.Context, worker int) (pluginsdk.Job, error) {
    return host.Jobs.Schedule(ctx, input)
})
if err != nil {
    t.Fatal(err)
}
if err := report.RequireSuccess(); err != nil {
    t.Fatal(err)
}

cases, err := plugintest.GeneratePropertyCases(0x5c011, 100, generateInput)
if err != nil {
    t.Fatal(err)
}
if err := plugintest.CheckProperty(cases, checkInvariant); err != nil {
    t.Fatal(err)
}
```

`TransactionalValue` 在 mutation 返回错误或上下文取消时丢弃工作副本，用于验证插件补偿和事务回滚不变量。`Eventually` 在明确的 context 期限内轮询对账条件，用于验证重试、重启或异步任务后的最终一致性；它不会吞掉检查错误，也没有无限等待的降级行为。

包和真实进程验收使用 `PackageRunner` 与 `Runtime`，至少覆盖 `Install -> Enable -> Restart -> Disable -> Enable -> Uninstall`。每个测试使用独立插件根目录、数据根目录和回环端口。

## 前端状态与性能测试

`@skoll/plugin-test` 提供六种标准 UI 状态：`loading`、`empty`、`error`、`forbidden`、`conflict`、`success`。插件可以扩展业务状态，但不能重命名或跳过适用的标准状态。

```ts
await runPluginStateMatrix(
  PLUGIN_UI_STATES,
  async (state) => renderFixture(state),
  async (state) => assertState(page, state)
);
```

浏览器验收至少覆盖：

| 维度 | 验收要求 |
| --- | --- |
| 权限 | 未授权时不请求插件页面或业务 API |
| 失败 | 超时、超限和服务异常显示受控状态，宿主仍可操作 |
| 恢复 | 重试、刷新和进程重启后恢复当前状态 |
| 视口 | 390px 移动端和桌面无横向溢出、遮挡或不可达操作 |
| 主题与语言 | 明暗主题、中文和英文均使用宿主当前令牌 |
| 性能 | 路由、交互、长任务、内存和包体积均通过显式预算 |

使用 `installPluginPerformanceObserver` 和 `readAndResetPluginLongTasks` 收集长任务；使用 `measurePluginAction` 测量从操作开始到可观察完成；最后调用 `assertPluginPerformanceBudget` 一次性判定所有预算。`createPluginVisualMatrix` 生成稳定截图名，视觉基线不得混用不同 viewport、theme 或 locale。

## 验收命令

```powershell
go test -race ./pkg/plugintest -count=1
& .\web\node_modules\.bin\tsc.cmd -p packages/skoll-plugin-test/tsconfig.json
node packages/skoll-plugin-test/test/self-test.mjs
npm --prefix web run test:plugin-runtime
```

Playwright 配置只声明 `baseURL`。执行浏览器验收前，必须在 `127.0.0.1:5174` 启动当前 Web 开发服务；验收完成后停止该进程。
