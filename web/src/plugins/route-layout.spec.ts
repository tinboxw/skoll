import { describe, expect, it } from "vitest";

import { isHostedPluginRoute } from "./route-layout";

describe("plugin route layout", () => {
	it("keeps the plugin control center in the scrollable admin layout", () => {
		expect(isHostedPluginRoute("plugin-center", "/skoll/plugin-center")).toBe(false);
		expect(isHostedPluginRoute("plugin-center-runtime", "/skoll/plugin-center/pharma_oa/runtime")).toBe(false);
	});

	it("uses the full-height host layout only for plugin-owned pages", () => {
		expect(isHostedPluginRoute("plugin-developer-portal", "/skoll/plugins/developer-portal")).toBe(true);
		expect(isHostedPluginRoute("app-home-pharma_oa", "/pharma_oa")).toBe(true);
		expect(isHostedPluginRoute(undefined, "/skoll/plugins/demo")).toBe(true);
	});
});
