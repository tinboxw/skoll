# 插件签名与风险分级设计

> P3 阶段：为插件系统引入供应链安全控制，包括签名校验、风险分级与策略引擎。

## 背景

当前插件安装流程无供应链安全机制，导致：
- 无法验证插件真实来源与完整性
- 无法对高风险插件进行治理（如权限、来源、审计）
- 组织无法制定统一的安全策略

## 目标

1. **签名机制**：确保插件真实性与完整性
2. **风险分级**：按功能与权限将插件分为：低/中/高/受限
3. **策略引擎**：组织可声明接受/拒绝不同风险等级
4. **审计追踪**：记录每个插件的安装、启停、卸载决策

## 设计

### 1. 签名机制

#### 1.1 清单签名

在 `plugin.yaml` 中添加签名字段：

```yaml
id: "example-plugin"
name: "Example Plugin"
version: "1.0.0"
vendor: "example-corp"
vendor_url: "https://example.com"
sign_algo: "RSA-SHA256"           # 签名算法
sign_timestamp: "2025-05-18T10:30:00Z"  # 签名时间
sign_value: "base64-encoded-signature"  # 签名值（对清单 JSON 签名）
```

#### 1.2 验证流程

**步骤**：
1. 从清单中提取 `sign_algo`、`sign_timestamp`、`sign_value`
2. 移除签名字段，计算清单 JSON 的 hash
3. 用公钥验证签名：`sign_value` 是否等于 hash 的签名
4. 检查 `sign_timestamp` 与当前时间的偏差（防重放）

**公钥来源**（阶段规划）：
- P3a：支持内联公钥（清单中 `vendor_pubkey`）
- P3b：支持公钥服务器（`https://keys.example.com/vendor/{vendor_id}.pub`）
- P3c：支持 PKI 与证书链

#### 1.3 签名校验失败处理

| 场景 | 默认行为 | 组织策略覆盖 |
| --- | --- | --- |
| 签名验证失败 | WARN，继续安装 | DENY（高安全组织） |
| 无签名信息 | WARN，继续安装 | DENY（强制签名） |
| 签名时间过旧 | INFO，继续安装 | WARN、DENY（防供应链污染） |
| 签名不受信任的厂商 | WARN，继续安装 | DENY（白名单厂商） |

### 2. 风险分级

#### 2.1 分级模型

```go
type RiskLevel string

const (
    RiskLevelLow      RiskLevel = "low"
    RiskLevelMedium   RiskLevel = "medium"
    RiskLevelHigh     RiskLevel = "high"
    RiskLevelRestricted RiskLevel = "restricted"
)
```

#### 2.2 风险评分规则

| 因子 | 得分 | 说明 |
| --- | --- | --- |
| **来源** | | |
| 系统内置 | 0 | 最低风险 |
| 官方认证厂商 | 10 | 签名 + 厂商白名单 |
| 社区仓库 | 30 | 签名存在但非官方 |
| 本地文件（无签名） | 50 | 无供应链证明 |
| **权限** | | |
| 仅读权限 | 0 | 最低风险 |
| 应用级权限 | 20 | 限制在应用内 |
| 系统权限（auth、config） | 40 | 敏感权限 |
| **UI 模式** | | |
| 后端只 | 0 | 最低风险 |
| 前端嵌入 | 15 | 可访问浏览器 API |
| 独立页面 | 25 | 可能持久化用户数据 |
| **其他** | | |
| 无更新超过 6 个月 | +15 | 维护风险 |
| 依赖已过期库 | +25 | 传递性风险 |

#### 2.3 风险等级判定

- **Low** (0-30)：日常使用，最小审核
- **Medium** (31-70)：需要明确授权，审核权限清单
- **High** (71-100)：需要组织层级审批
- **Restricted** (>100)：禁止安装或仅允许管理员手动确认

#### 2.4 清单风险字段

```yaml
id: "example-plugin"
name: "Example Plugin"
version: "1.0.0"
vendor: "example-corp"
# 插件声明的风险等级与理由（自评）
risk:
  level: "medium"
  reason: "Requires authentication API access"
  # 依赖关系表示意外风险
  dependencies:
    - id: "auth"
      notes: "Uses JWT token validation"
```

### 3. 策略引擎

#### 3.1 组织策略定义

```go
type PluginSecurityPolicy struct {
    // 签名要求
    SignatureRequired  bool
    TrustedVendors    []string  // 白名单厂商
    
    // 风险策略
    AllowedRiskLevels []RiskLevel  // [low, medium] → 拒绝 high/restricted
    RequireApproval   RiskLevel    // RiskLevelHigh → 需要手动审批
    
    // 权限策略
    ForbiddenPermissions []string  // ["admin:user:write", "config:delete"]
    
    // 来源限制
    AllowedSources []string  // ["builtin", "system", "https://..."]
}
```

#### 3.2 策略评估流程

```
安装请求 → 提取清单 → 计算风险评分 → 与策略对比 → 决策
    ↓
    风险评分(计算)
    ↓
策略层判定:
  - 签名验证？
  - 厂商白名单？
  - 风险等级允许？
  - 需要审批？
  - 权限冲突？
    ↓
    决策: ALLOW / WARN / DENY
    ↓
    审计日志 + 通知
```

### 4. 审计与决策记录

#### 4.1 审计日志结构

```go
type PluginInstallDecision struct {
    PluginID        string    // 插件 ID
    Version         string    // 版本
    Timestamp       time.Time // 决策时间
    Actor           string    // 操作人
    Action          string    // install / enable / disable / uninstall
    
    // 风险评估结果
    ComputedRisk    RiskLevel
    RiskScore       int
    RiskFactors     []string  // 风险因子列表
    
    // 策略评估结果
    PolicyStatus    string    // ALLOW / WARN / DENY
    PolicyReasons   []string  // 具体原因
    
    // 决策结果
    Decision        string    // ALLOW / DENY / MANUAL_APPROVAL_PENDING
    DecisionReason  string
    ApprovedBy      string    // 审批人（若需要）
    
    // 审计
    IPAddress       string
    UserAgent       string
}
```

### 5. 实现阶段

#### P3a：基础签名验证（当前）

- [x] 清单解析支持签名字段
- [ ] 实现 RSA-SHA256 签名验证
- [ ] 添加 sign_algo / sign_timestamp / sign_value 校验
- [ ] 验证失败日志与 WARN 提示
- [ ] 内联公钥支持（简单场景）
- [ ] 单元测试

#### P3b：风险分级评估（后续）

- [ ] 定义风险评分规则
- [ ] 实现风险计算引擎
- [ ] 清单中风险字段解析
- [ ] 权限与依赖的风险传递
- [ ] 风险可视化 API（返回评分 + 因子）

#### P3c：策略引擎与决策（后续）

- [ ] 组织策略定义与存储
- [ ] 策略评估与决策链
- [ ] 手动审批流程（UI + API）
- [ ] 审计日志与合规报告

## 代码结构

```
internal/plugin/
├── signature/
│   ├── verifier.go          # 签名验证核心
│   ├── rsa.go               # RSA-SHA256 实现
│   └── verifier_test.go
├── risk/
│   ├── calculator.go        # 风险评分引擎
│   ├── scorer.go            # 得分规则
│   └── calculator_test.go
├── policy/
│   ├── policy.go            # 策略定义与存储
│   ├── evaluator.go         # 策略评估
│   └── evaluator_test.go
├── audit/
│   ├── logger.go            # 审计日志
│   └── logger_test.go
└── security.go              # 统一安全入口
```

## 迁移策略

### 过渡期（P3a 期间）

- 新清单无签名时：打印 WARN 但继续（不中断用户）
- 现有插件无签名时：补充风险评估（可选 WARN）
- 签名验证失败时：可通过 `PLUGIN_SECURITY_MODE=warn` 环境变量降级为警告

### 策略生效（P3c 期间）

- 管理员可配置 `plugin_security_policy.yaml`
- 默认策略为宽松（兼容旧行为）
- 可逐步升级到严格（如：`AllowedRiskLevels: [low, medium]`）

## 参考

- [Plugin Manifest Contract](refactor.md#104-插件清单契约兼容演进)
- [Architecture Three Planes](refactor.md#102-三层架构)
- [Phased Roadmap](refactor.md#107-分阶段落地开始重构推进)
