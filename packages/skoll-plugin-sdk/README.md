# @skoll/plugin-sdk

Skoll 集成式前端插件的版本化宿主契约。业务插件通过该包访问宿主能力，不直接读取宿主运行时全局对象。

```ts
import { getPluginHost } from "@skoll/plugin-sdk";

const host = getPluginHost({
	pluginId: "example_plugin",
	requiredCapabilities: ["request", "permissions"]
});

host.permissions.require("example.read");
const records = await host.request("/v1/plugins/example_plugin/api/records");
```

`getPluginHost` 会校验契约版本、插件身份、宿主结构和能力声明。缺失或伪造的宿主能力会抛出 `PluginHostError`；需要受控展示错误状态时，使用 `resolvePluginHost`。
