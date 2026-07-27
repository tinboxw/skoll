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
远程插件只通过 `@skoll/plugin-sdk` 解析当前宿主。SDK 会校验契约版本、插件身份和声明能力；校验失败时返回结构化错误并停止调用。业务代码不得直接读取宿主全局对象。

```ts
import { getPluginHost } from "@skoll/plugin-sdk";

const host = getPluginHost({
	pluginId: "example_plugin",
	requiredCapabilities: ["auth", "dictionary", "organization", "config"]
});
const me = await host.auth.me();
const dictionaries = await host.dictionary.list();
const departments = await host.organization.departments();
const config = await host.config.get();

await host.config.update({ ...config, compact: true });
```

当前能力覆盖请求、导航、命令、权限、语言、主题、生命周期、认证、用户、组织、字典、文件、审计和插件配置。宿主请求会自动应用当前 API 前缀与登录凭据。

