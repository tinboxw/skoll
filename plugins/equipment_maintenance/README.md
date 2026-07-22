# 设备维保证明插件

`equipment_maintenance` 是用于验证 Skoll 独立业务插件能力的完整业务样本。它不是核心模块，也不允许通过修改核心路由、依赖注入或内部服务来获得能力。

## 业务边界

| 领域 | 负责内容 | 不负责内容 |
| --- | --- | --- |
| 设备资产 | 设备台账、位置、状态、保养日期、退役 | 组织与用户主数据 |
| 维修工单 | 报修、派工、执行、审批、完工、关闭 | 通用审批引擎实现 |
| 保养点检 | 预防性计划、点检执行、发现项、证据附件 | 通用文件存储实现 |
| 备件库存 | 备件目录、余额、出入库与工单领用流水 | 通用仓储或采购系统 |
| 运营看板 | 到期、故障、工单效率、低库存指标 | 平台级监控与审计查询 |

所有业务记录必须包含 `tenant_id`、`organization_id` 和 `owner_id`，查询范围只能由宿主可信上下文收窄，不能采信请求体中的租户或组织字段。

## 运行边界

- 后端是安装包内的独立受管进程，使用 `service_base_url` 暴露 Manifest 中声明的 API 和固定事件端点。
- 前端是独立 Vue 3 + TypeScript + Element Plus 应用，通过 `PluginRuntime` 集成，消费宿主主题、语言、认证和权限上下文。
- 插件业务代码只能依赖 Go 标准库、公开第三方库以及 Skoll 的公开插件客户端，不得导入 `github.com/tinboxw/skoll/internal/`。
- 插件通过公开宿主服务协议使用可信数据范围、文件、审计、配置、密钥、工作流和持久任务；缺失能力时失败关闭。
- 插件自有表、迁移和数据目录属于插件生命周期；卸载执行 Manifest 中唯一的 `drop` 策略。
- 当前项目只实现现行协议，不提供旧清单、旧路由、双路径或降级实现。

## 工作流与任务

`work_order_cost_approval` 在工单预计费用达到配置阈值时启动。插件保存宿主返回的工作流实例 ID，并通过 `approval-completed` 的稳定 `deliveryId` 幂等推进工单；未经批准的工单不能关闭。

`maintenance.due_scan` 与 `spare.low_stock_scan` 使用宿主持久任务，分别生成保养到期工作项和备件低库存告警。调度成功不等于业务完成，插件必须处理租约、重试和死信状态。

## 当前依赖

本目录先冻结业务与验收契约。完整实现开始前，平台必须依次完成：

1. 从安装包启动、监督和停止外部插件进程，并注入最小运行上下文。
2. 为独立进程提供插件身份绑定、失败关闭的宿主服务 HTTP 协议和公开客户端。
3. 证明插件数据目录、迁移、升级与卸载策略由生命周期统一控制。

这些能力属于平台基础设施，不允许由证明插件通过核心代码特判获得。

## 验收来源

机器可读映射见 [`contract/acceptance-map.json`](contract/acceptance-map.json)。Manifest、权限、路由、数据表、事件、宿主服务与端到端场景必须保持一一对应，契约测试不允许未声明入口或内部依赖。

## English Summary

This independent proof plugin covers assets, work orders, preventive inspections, spare parts, and operational dashboards. It must run from an installable package, consume only public current Skoll contracts, and require no plugin-specific core registration or source change.
