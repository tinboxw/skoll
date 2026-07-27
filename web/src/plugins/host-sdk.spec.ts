import { beforeEach, describe, expect, it, vi } from "vitest";
import {
	PLUGIN_HOST_CAPABILITIES,
	PLUGIN_HOST_CONTRACT,
	PLUGIN_HOST_VERSION,
	PluginHostError,
	resolvePluginHost,
	type PluginHostCapability,
	type PluginHostSDK
} from "@skoll/plugin-sdk";

import { buildPluginHostBridgeScript } from "./host-sdk";

function bridgeInput() {
	return {
		pluginId: "demo",
		pluginVersion: "1.2.3",
		apiBasePrefix: "/skoll",
		locale: "zh-CN",
		locales: ["zh-CN", "en-US"],
		token: "token",
		theme: {
			colorScheme: "dark" as const,
			density: "compact" as const,
			tokens: { "--color-primary": "test-primary" }
		},
		identity: {
			subject: "user-1",
			roles: ["operator"],
			permissions: ["plugin.demo.*"]
		},
		lifecycle: {
			state: "enabled" as const,
			health: "healthy"
		}
	};
}

function validHost(): PluginHostSDK {
	const capabilities = [...PLUGIN_HOST_CAPABILITIES];
	return {
		contract: PLUGIN_HOST_CONTRACT,
		version: PLUGIN_HOST_VERSION,
		pluginId: "demo",
		pluginVersion: "1.2.3",
		capabilities,
		identity: { subject: "user-1", roles: ["operator"], permissions: ["plugin.demo.*"] },
		locale: "zh-CN",
		locales: ["zh-CN", "en-US"],
		theme: { colorScheme: "dark", density: "compact", tokens: {} },
		lifecycle: { state: "enabled", health: "healthy" },
		hasCapability: (capability) => capabilities.includes(capability),
		requireCapability: () => undefined,
		request: async <T>() => ({} as T),
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
	};
}

beforeEach(() => {
	delete window.__SKOLL_HOST__;
});

describe("plugin host bridge script", () => {
	it("emits the immutable versioned host contract and current theme", () => {
		const script = buildPluginHostBridgeScript(bridgeInput());

		expect(script).toContain(`"contract":"${PLUGIN_HOST_CONTRACT}"`);
		expect(script).toContain(`"version":"${PLUGIN_HOST_VERSION}"`);
		expect(script).toContain('"pluginVersion":"1.2.3"');
		expect(script).toContain('"colorScheme":"dark"');
		expect(script).toContain('"density":"compact"');
		expect(script).toContain("Object.freeze(host)");
		expect(script).toContain("Object.defineProperty(window, '__SKOLL_HOST__'");
		expect(script).toContain("event.source !== window.parent");
		expect(script).toContain("data.pluginId !== ctx.pluginId");
		expect(script).toContain("allowedThemeTokens");
		expect(script).not.toContain("__SKOLL_TOKEN");
		expect(script).not.toContain("__SKOLL_PLUGIN_CONTEXT");
		expect(script).not.toContain("data-theme-mode");
	});

	it("rejects invalid plugin identities before producing executable content", () => {
		expect(() => buildPluginHostBridgeScript({ ...bridgeInput(), pluginId: "Demo Plugin" })).toThrow(
			"plugin host identity is invalid"
		);
	});
});

describe("plugin host SDK resolver", () => {
	it("accepts one complete host contract", () => {
		window.__SKOLL_HOST__ = validHost();

		const resolution = resolvePluginHost({ pluginId: "demo", requiredCapabilities: ["request", "permissions"] });

		expect(resolution.ok).toBe(true);
	});

	it.each([
		["missing host", undefined, "HOST_UNAVAILABLE"],
		["wrong version", { ...validHost(), version: "9.0" }, "VERSION_UNSUPPORTED"],
		["wrong identity", { ...validHost(), pluginId: "forged" }, "PLUGIN_IDENTITY_MISMATCH"],
		["missing request shape", { ...validHost(), request: undefined }, "CONTRACT_MISMATCH"],
		["forged theme token", { ...validHost(), theme: { ...validHost().theme, tokens: { "--forged-root": "1" } } }, "CONTRACT_MISMATCH"],
		[
			"forged capability",
			{ ...validHost(), capabilities: [...PLUGIN_HOST_CAPABILITIES, "root"] as PluginHostCapability[] },
			"CONTRACT_MISMATCH"
		]
	])("fails closed for %s", (_label, candidate, code) => {
		window.__SKOLL_HOST__ = candidate as PluginHostSDK | undefined;

		const resolution = resolvePluginHost({ pluginId: "demo", requiredCapabilities: ["request"] });

		expect(resolution.ok).toBe(false);
		if (!resolution.ok) {
			expect(resolution.error).toBeInstanceOf(PluginHostError);
			expect(resolution.error.code).toBe(code);
		}
	});

	it("rejects an undeclared required capability even when the host method lies", () => {
		const host = validHost();
		window.__SKOLL_HOST__ = {
			...host,
			capabilities: host.capabilities.filter((capability) => capability !== "audit"),
			hasCapability: vi.fn(() => true)
		};

		const resolution = resolvePluginHost({ pluginId: "demo", requiredCapabilities: ["audit"] });

		expect(resolution.ok).toBe(false);
		if (!resolution.ok) expect(resolution.error.code).toBe("CAPABILITY_MISSING");
	});
});
