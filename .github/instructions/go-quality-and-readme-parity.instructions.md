---
description: "Use when: 编写或修改 Go 代码、测试、README 文档。约束 Go 代码风格、测试门禁、README 中英同步规则。关键词: go style, test gate, readme parity, bilingual docs"
name: "Go Quality And README Parity"
applyTo: ["**/*.go", "README.md", "README.en.md"]
---
# Go Quality and README Parity Rules

## Go 代码风格
- 优先使用标准库与标准 Go 工具链，保持实现清晰、可维护。
- 提交前必须执行并通过 `go fmt ./...`。
- 文件规模治理：
  - 单个 Go 源文件超过约 800 行时，应优先按职责拆分（例如 `*_handlers.go`、`*_types.go`、`*_service.go`），避免继续堆叠实现。
  - 单个测试文件超过约 1200 行时，应按场景拆分为多个 `*_test.go` 文件。
  - 发生拆分时保持包边界与公共 API 不变，避免与功能改动耦合提交。
- 包边界要稳定：
  - `cmd/` 仅放可执行入口。
  - `internal/` 放私有实现。
  - `pkg/` 放可复用公共能力。
- Admin 模块子包化与对象化调用（新增强制规则）：
  - **包名与目录名必须一致**：`internal/app/admin/<domain>/` 下的包名必须声明为 `package <domain>`，不得加 `admin` 前缀（例如目录 `rbac/` → `package rbac`，而非 `package adminrbac`）。导入时如需区分同名包可使用别名（`import adminrbac "...admin/rbac"`），但源文件 `package` 声明本身必须等于目录名。
  - 当 `internal/app` 中某一业务域出现 `handler + dto + 路由装配` 三类逻辑时，必须优先拆分为子包（示例：`internal/app/admin/<domain>/`）。
  - 子包必须提供对象化入口，不再新增同域的散落函数式装配。推荐固定形态：
    - `type Handler struct { ... }`
    - `func NewHandler(...) *Handler`
    - `func (h *Handler) Register(mux *http.ServeMux, wrapper func(http.Handler) http.Handler, apis APIRegistry)`
  - 路由注册必须在域内 `Register` 完成；`internal/app` 根包仅保留 façade/组装职责，不继续扩展跨域超大 `Mount...Routes` 细节。
  - 新增/迁移子包时，必须保持外部 API 路径、请求/响应 JSON 字段与语义兼容，避免结构重构与协议变更耦合。
  - 域内优先定义最小端口接口（small interface）；禁止通过单一超大接口向所有子包暴露无关能力。
  - 通用能力（如 path 解析、分页、统一响应）可放入共享位置，但不得把业务 DTO 无差别下沉到全局共享包。
  - 每完成一个域迁移，必须执行并记录最小门禁：`go fmt ./...`、`go test ./internal/app`、`go test ./...`；涉及并发路径时追加 `go test -race ./...`。
- 并发相关实现必须显式说明取消、超时和错误传播路径，避免 goroutine 泄漏。
- 新增性能敏感路径时，优先补基准测试并记录 `-benchmem` 结果。

## 测试门禁
- 基线门禁：`go test ./...` 必须通过。
- 涉及并发改动时，额外门禁：`go test -race ./...` 必须通过。
- 修复 bug 或引入复杂分支时，应补最小回归测试，优先覆盖边界与失败路径。
- 如果因为环境限制无法执行测试，必须在变更说明中明确风险与未验证范围。

## README 中英同步规则
- 影响使用方式、目录结构、构建/测试命令时，必须同步更新 `README.md` 与 `README.en.md`。
- 若无法同次完成双语同步，必须在变更说明中标注语言差异与补齐计划。
- 命令示例在中英文 README 中保持一致（仅文字说明可本地化）。

## 变更输出要求
- 在提交说明或阶段报告中至少包含：
  - 里程碑标识（固定 `M0/M1/M2/M3-简短内容` 格式）。
  - 已执行验证命令。
  - 性能对比数据（基线 vs 当前，含变化百分比与采样命令）。
  - 关键测试结论。
  - README 同步状态（已同步/存在差异）。
