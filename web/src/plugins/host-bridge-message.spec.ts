import { describe, expect, it, vi } from "vitest";
import type { Router } from "vue-router";
import { PLUGIN_HOST_CONTRACT, PLUGIN_HOST_VERSION } from "@skoll/plugin-sdk";

import { handlePluginHostMessage } from ".";

function message(data: Record<string, unknown>, source: Window = window): MessageEvent {
	return new MessageEvent("message", {
		data,
		origin: window.location.origin,
		source
	});
}

function validMessage(data: Record<string, unknown>): MessageEvent {
	return message({
		contract: PLUGIN_HOST_CONTRACT,
		version: PLUGIN_HOST_VERSION,
		pluginId: "demo",
		...data
	});
}

function context() {
	const push = vi.fn();
	const replace = vi.fn();
	const back = vi.fn();
	const reload = vi.fn();
	const fail = vi.fn();
	return {
		value: {
			pluginId: "demo",
			source: window,
			router: { push, replace, back } as unknown as Router,
			reload,
			fail
		},
		push,
		replace,
		back,
		reload,
		fail
	};
}

describe("plugin host message boundary", () => {
	it("ignores forged version, identity, source, and origin", () => {
		const test = context();
		const forged = [
			message({ contract: PLUGIN_HOST_CONTRACT, version: "9.0", pluginId: "demo", type: "skoll:command", command: "reload" }),
			message({ contract: PLUGIN_HOST_CONTRACT, version: PLUGIN_HOST_VERSION, pluginId: "other", type: "skoll:command", command: "reload" }),
			new MessageEvent("message", {
				data: { contract: PLUGIN_HOST_CONTRACT, version: PLUGIN_HOST_VERSION, pluginId: "demo", type: "skoll:command", command: "reload" },
				origin: "https://forged.example",
				source: window
			})
		];

		for (const event of forged) {
			expect(handlePluginHostMessage(event, test.value)).toBe(false);
		}
		expect(test.reload).not.toHaveBeenCalled();
	});

	it("routes validated navigation and commands through the host router", () => {
		const test = context();

		expect(handlePluginHostMessage(validMessage({ type: "skoll:navigation", mode: "push", path: "/skoll/todo" }), test.value)).toBe(true);
		expect(handlePluginHostMessage(validMessage({ type: "skoll:navigation", mode: "replace", path: "/skoll/plugins" }), test.value)).toBe(true);
		expect(handlePluginHostMessage(validMessage({ type: "skoll:navigation", mode: "back" }), test.value)).toBe(true);
		expect(handlePluginHostMessage(validMessage({ type: "skoll:command", command: "reload" }), test.value)).toBe(true);

		expect(test.push).toHaveBeenCalledWith("/skoll/todo");
		expect(test.replace).toHaveBeenCalledWith("/skoll/plugins");
		expect(test.back).toHaveBeenCalledOnce();
		expect(test.reload).toHaveBeenCalledOnce();
	});

	it("turns invalid navigation and reported host errors into controlled failures", () => {
		const test = context();

		expect(handlePluginHostMessage(validMessage({ type: "skoll:navigation", mode: "push", path: "https://forged.example" }), test.value)).toBe(true);
		expect(handlePluginHostMessage(validMessage({
			type: "skoll:plugin-error",
			error: { code: "CAPABILITY_MISSING", message: "request unavailable" }
		}), test.value)).toBe(true);

		expect(test.push).not.toHaveBeenCalled();
		expect(test.fail).toHaveBeenNthCalledWith(1, "CONTRACT_MISMATCH: Plugin navigation target is invalid");
		expect(test.fail).toHaveBeenNthCalledWith(2, "CAPABILITY_MISSING: request unavailable");
	});
});
