import {
	PLUGIN_HOST_CAPABILITIES,
	PLUGIN_HOST_CONTRACT,
	PLUGIN_HOST_VERSION,
	type PluginHostRequestOptions,
	type PluginHostSDK,
	type PluginHostTheme
} from "@skoll/plugin-sdk";

type TestRequest = (path: string, options?: PluginHostRequestOptions) => Promise<unknown>;

type TestHostOptions = {
	pluginId?: string;
	theme?: PluginHostTheme;
};

export function installPharmaTestHost(request: TestRequest, options: TestHostOptions = {}): PluginHostSDK {
	const capabilities = [...PLUGIN_HOST_CAPABILITIES];
	const host = {
		contract: PLUGIN_HOST_CONTRACT,
		version: PLUGIN_HOST_VERSION,
		pluginId: options.pluginId ?? "pharma_oa",
		pluginVersion: "test",
		capabilities,
		identity: { subject: "test-user", roles: ["tester"], permissions: ["*"] },
		locale: "zh-CN",
		locales: ["zh-CN", "en-US"],
		theme: options.theme ?? { colorScheme: "light", density: "comfortable", tokens: {} },
		lifecycle: { state: "enabled", health: "healthy" },
		hasCapability: (capability: (typeof capabilities)[number]) => capabilities.includes(capability),
		requireCapability: (capability: (typeof capabilities)[number]) => {
			if (!capabilities.includes(capability)) throw new Error(`missing capability: ${capability}`);
		},
		request: request as PluginHostSDK["request"],
		navigation: { push: () => undefined, replace: () => undefined, back: () => undefined },
		commands: { execute: () => undefined },
		permissions: { has: () => true, require: () => undefined },
		auth: { me: async () => ({}) },
		user: { me: async () => ({}) },
		organization: { departments: async () => [], positions: async () => [] },
		dictionary: { list: async () => [], items: async () => [] },
		file: { list: async () => [] },
		audit: { list: async () => [] },
		config: { get: async () => ({}), update: async () => ({}) }
	} satisfies PluginHostSDK;
	window.__SKOLL_HOST__ = host;
	return host;
}
