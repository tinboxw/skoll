import { describe, expect, it } from "vitest";

import type { PluginControlSnapshot } from "./api";
import { deriveRuntimeState } from "./model";

function snapshot(overrides: Partial<PluginControlSnapshot["runtime"]> = {}, staleAfter = "2099-01-01T00:00:00Z"): PluginControlSnapshot {
	return {
		plugin: { id: "demo", name: "Demo", version: "1.0.0", enabled: true },
		capturedAt: "2026-07-23T00:00:00Z",
		staleAfter,
		runtime: {
			state: "enabled",
			installedAt: "2026-07-23T00:00:00Z",
			health: { pluginId: "demo", status: "healthy", code: "health_ok", checkedAt: "2026-07-23T00:00:00Z", latencyMillis: 2 },
			...overrides
		},
		capabilities: { hostServices: [], permissions: [], routes: [], dependencies: [], extensions: { routes: 0, middlewares: 0, events: 0, menus: 0, widgets: 0, settings: 0 } }
	};
}

describe("deriveRuntimeState", () => {
	it("keeps loading, disabled, ready, degraded, crashed, and stale distinct", () => {
		expect(deriveRuntimeState(null)).toBe("loading");
		expect(deriveRuntimeState(snapshot({ state: "disabled" }))).toBe("disabled");
		expect(deriveRuntimeState(snapshot())).toBe("ready");
		expect(deriveRuntimeState(snapshot({ health: undefined }))).toBe("degraded");
		expect(deriveRuntimeState(snapshot({ health: { pluginId: "demo", status: "unhealthy", code: "health_http_status", checkedAt: "2026-07-23T00:00:00Z", latencyMillis: 3 } }))).toBe("degraded");
		expect(deriveRuntimeState(snapshot({ health: { pluginId: "demo", status: "unhealthy", code: "health_unreachable", checkedAt: "2026-07-23T00:00:00Z", latencyMillis: 3 } }))).toBe("crashed");
		expect(deriveRuntimeState(snapshot({}, "2026-07-23T00:00:01Z"), Date.parse("2026-07-23T00:00:02Z"))).toBe("stale");
	});
});
