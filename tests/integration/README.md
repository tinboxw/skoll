# 集成测试说明

本目录用于跨层验证系统行为，覆盖 bootstrap + 路由 + 中间件 + 服务协同。

## 当前用例
- `http_runner_integration_test.go`: 启动真实 runner，验证
	- `/health` 在匿名访问下返回 200
	- `/v1/users` 未授权访问返回 401
- `pharma_inventory_smoke_test.go`: 验证医药 OA 库存业务闭环
	- 采购申请审批、采购入库、销售订单和销售出库
	- 盘点审批、跨仓调拨、批次余额和不可变库存流水
	- 近效期/低库存告警、通知中心和可操作目标链接
- `pharma_oa_acceptance_smoke_test.go`: 验证医药 OA 样板端到端链路
	- 员工入职、采购申请审批、采购入库和销售出库
	- 资质预警、客户跟进、库存证据和演示数据幂等性

## 运行方式
```bash
go test ./tests/integration -v
```

库存闭环可通过独立验收脚本重复执行：

```powershell
.\scripts\smoke-pharma-inventory.ps1
```

完整样板验收可通过独立命令执行：

```powershell
.\scripts\smoke-pharma-oa-e2e.ps1
```

脚本默认输出中文，也可用 `-Locale en-US` 切换英文输出。

