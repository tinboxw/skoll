import { describe, expect, it } from "vitest";

import { buildPluginHostBridgeScript } from "./host-sdk";

describe("plugin host theme bridge", () => {
	it("emits only the current color scheme and density contract", () => {
		const script = buildPluginHostBridgeScript({
			pluginId: "demo",
			apiBasePrefix: "/skoll",
			locale: "zh-CN",
			locales: ["zh-CN", "en-US"],
			token: "token",
			theme: {
				colorScheme: "dark",
				density: "compact",
				tokens: { "--color-primary": "test-primary" }
			}
		});

		expect(script).toContain('"colorScheme":"dark"');
		expect(script).toContain('"density":"compact"');
		expect(script).toContain("setAttribute('data-theme', ctx.theme.colorScheme");
		expect(script).toContain("setAttribute('data-density', ctx.theme.density");
		expect(script).not.toContain("data-theme-mode");
		expect(script).not.toContain('"mode"');
	});
});
