# web

## 功能说明
前端后台管理工程根目录。

## 文件组织规范
- 采用小写与下划线命名，按职责拆分文件。
- 接口定义与实现分离，避免单文件过大。
- 变更时同步补充测试与文档。

## 当前规划文件
- index.html
- package.json
- vite.config.ts
- tsconfig.json

## 后续待补充实现
- [ ] 按目录职责补齐核心实现代码。
- [ ] 补充单元测试与必要的集成测试。
- [ ] 完善示例、边界条件与错误处理说明。

## 联调代理与脚本

### Vite 代理环境变量
- `SKOLL_API_PROXY_TARGET`：后端代理目标地址（默认 `http://127.0.0.1:8080`）。
- `SKOLL_API_PROXY_TIMEOUT_MS`：代理超时毫秒数（默认 `10000`）。
- `SKOLL_API_BASE_PREFIX`：统一 API 基础前缀（默认 `/skoll`，用于前端构建注入与 smoke 脚本）。
- `SKOLL_WEB_BASE_PATH`：前端页面基础路径（默认 `/skoll`，例如页面入口为 `http://127.0.0.1:5173/skoll`）。
- 代理前缀使用 `SKOLL_API_BASE_PREFIX`，后端接口统一挂载到 `{API_BASE_PREFIX}/v1/*`。

### 页面访问建议
- 当前默认：admin 页面入口是 `http://127.0.0.1:5173/skoll`。
- 多应用阶段：可按应用设置不同 `SKOLL_WEB_BASE_PATH`，或将某个应用设为 `/` 作为主入口。

### Smoke 脚本
- `npm run smoke:health`：检查 `/skoll/health`。
- `npm run smoke:auth`：检查登录与受保护接口。
- `npm run smoke:plugins`：检查插件列表、详情与 404 回退行为。
- `npm run smoke:auth-actor`：检查 actorId 回退链路。
- `npm run smoke:all`：顺序执行 health + auth + plugins + auth-actor。

### 常用联调命令
```powershell
npm run build
npm run smoke:all
```


