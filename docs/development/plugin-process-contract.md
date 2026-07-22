# 插件受管进程契约

> 默认语言：简体中文。English: [Managed Plugin Process Contract](plugin-process-contract.en.md)

本文定义带独立后端的插件在 Skoll 中唯一的当前运行方式。插件后端必须由安装包提供并由宿主生命周期启动，不支持预启动远程服务、任意命令或备用入口。

## 安装包入口

后端可执行文件使用固定路径：

| 平台 | 路径 |
| --- | --- |
| Windows | `backend/bin/<plugin-id>-server.exe` |
| Linux/macOS | `backend/bin/<plugin-id>-server` |

声明 `service_base_url` 的插件在打包时必须包含当前平台入口。ZIP 中入口标记为可执行文件，安装时 Unix 平台保留执行位。入口必须是插件安装目录内的普通文件，符号链接、目录、缺失文件和不可执行文件都会阻止启用。

## 网络边界

`service_base_url` 与 `service_health_url` 必须同时满足：

- 使用 `http`；
- 主机为 `localhost` 或回环 IP；
- scheme、host 和 port 完全一致；
- 不包含用户信息、query 或 fragment。

宿主把 `service_base_url` 的 host 和 port 注入 `SKOLL_PLUGIN_ADDRESS`。插件必须监听该地址，并在 `service_health_url` 路径返回 `2xx`。插件业务路由仍只能使用 Manifest 声明的 `/v1/plugins/<plugin-id>/api/*`。

## 进程环境

子进程不继承宿主的完整环境。当前只保留运行操作系统所需的 `PATH`、临时目录和用户目录变量，并显式注入：

| 变量 | 含义 |
| --- | --- |
| `SKOLL_PLUGIN_ID` | 已校验的 Manifest 插件 ID |
| `SKOLL_PLUGIN_ADDRESS` | 插件服务监听地址 |
| `SKOLL_PLUGIN_DIR` | 插件安装根目录 |
| `SKOLL_PLUGIN_DATA_DIR` | 按插件 ID 隔离、跨重启与升级稳定的宿主管理数据目录 |
| `SKOLL_PLUGIN_HOST_URL` | 仅回环访问的宿主服务 v1 地址 |
| `SKOLL_PLUGIN_HOST_TOKEN` | 仅当前受管进程生命周期有效的宿主凭证 |

数据库凭据、JWT 密钥、宿主密钥及其他 `SKOLL_*` 变量不会透传。独立插件需要的宿主能力由公开插件协议提供，不通过进程环境绕过。

插件安装目录只保存随版本发布的程序材料，升级时可以变化。插件自有文件必须仅写入 `SKOLL_PLUGIN_DATA_DIR`。受管插件必须在 `data` 段显式声明 `uninstall_policy` 和 `rollback_policy`，宿主不会选择隐式默认策略。

## 生命周期

1. `Install` 校验 Manifest、包校验和与入口文件，不启动进程。
2. `Enable` 先执行当前迁移，再启动后端并等待健康检查；预算内未就绪则启用失败。
3. 进入 Ready 后，宿主持续检查进程退出与健康状态。崩溃或探活失败会进入 Failed 并关闭业务流量。
4. `Disable` 先关闭业务路由和事件订阅，再停止进程。
5. `Uninstall` 确认进程已停止后，对迁移和受管数据目录执行同一声明策略：`retain` 保留活动目录，`archive` 移入 `.archive/<plugin-id>/`，`drop` 删除目录。
6. 宿主关闭时停止全部受管插件进程，不留下后台进程。

Windows 使用终止进程完成停止；Unix 先发送中断信号，超时后强制终止。状态转换通过现有插件服务审计记录稳定 code，不记录进程环境或底层敏感错误。

## 验证

```powershell
go test ./internal/plugin -run "Test(ManagedProcessLauncher|BuildPackageRequiresManagedBackend)" -count=1
$env:SKOLL_GENERATOR_PLUGIN_E2E='1'
go test ./internal/service/generator -run TestGeneratedPluginBuildPackageAndInstallWithoutSourceEdits -count=1
```

生成器的 `plugin.ps1 package` 与 `plugin.sh package` 会先构建约定入口，再使用同一安装包执行当前开发和安装流程。
