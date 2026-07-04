# web/src/plugins

## 功能说明
前端插件注册与扩展。

## 文件组织规范
- 采用小写与下划线命名，按职责拆分文件。
- 接口定义与实现分离，避免单文件过大。
- 变更时同步补充测试与文档。

## 当前规划文件
- index.ts
- host-sdk.ts

## Host SDK
远程插件页面加载时，宿主会注入 `window.__SKOLL_HOST__`。插件应通过该对象访问当前稳定宿主能力，不直接拼接旧接口或读取旧插件上下文。

```ts
const host = window.__SKOLL_HOST__;

if (host) {
	const me = await host.auth.me();
	const dictionaries = await host.dictionary.list();
	const departments = await host.organization.departments();
	const config = await host.config.get();

	await host.audit.list({ action: "plugin.enabled" });
	await host.config.update({ ...config, compact: true });
}
```

当前能力覆盖 `auth`、`user`、`organization`、`dictionary`、`file`、`audit`、`config`、`permission`，请求会自动使用宿主 API 前缀与当前登录 token。

## 后续待补充实现
- [ ] 按目录职责补齐核心实现代码。
- [ ] 补充单元测试与必要的集成测试。
- [ ] 完善示例、边界条件与错误处理说明。

