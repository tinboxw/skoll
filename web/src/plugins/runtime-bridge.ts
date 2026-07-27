import type { Router } from "vue-router";

import {
	PLUGIN_HOST_CONTRACT,
	PLUGIN_HOST_VERSION,
	type PluginHostIdentity,
	type PluginHostLifecycle
} from "@skoll/plugin-sdk";

import type { ThemeBridgePayload } from "../stores/theme";
import { API_BASE_PREFIX } from "../utils/api-base-prefix";
import type { FrontendPluginManifest } from "./types";
import { buildPluginHostBridgeScript } from "./host-sdk";

export function normalizePluginLocales(locales?: string[]): string[] {
	if (!Array.isArray(locales) || locales.length === 0) {
		return ["zh-CN", "en-US"];
	}
	const normalized: string[] = [];
	for (const locale of locales) {
		const trimmed = String(locale || "").trim();
		if (trimmed !== "" && !normalized.includes(trimmed)) {
			normalized.push(trimmed);
		}
	}
	return normalized.length > 0 ? normalized : ["zh-CN", "en-US"];
}

export function resolveActiveLocale(hostLocale: string, pluginLocales: string[]): string {
	const normalizedHostLocale = String(hostLocale || "").trim();
	if (normalizedHostLocale !== "" && pluginLocales.includes(normalizedHostLocale)) {
		return normalizedHostLocale;
	}
	return pluginLocales[0] || "zh-CN";
}

export function injectHostBridgeScript(
	content: string,
	plugin: FrontendPluginManifest,
	activeLocale: string,
	pluginLocales: string[],
	authToken: string,
	theme: ThemeBridgePayload,
	identity: PluginHostIdentity
): string {
	const bridgeScript = buildPluginHostBridgeScript({
		pluginId: plugin.id,
		pluginVersion: plugin.version,
		apiBasePrefix: API_BASE_PREFIX,
		locale: activeLocale,
		locales: pluginLocales,
		token: authToken,
		theme,
		identity,
		lifecycle: pluginHostLifecycle(plugin)
	});
	if (/<head>/i.test(content)) {
		return content.replace(/<head>/i, `<head>\n${bridgeScript}`);
	}
	return `${bridgeScript}${content}`;
}

export function pushLocaleToIframe(
	iframe: HTMLIFrameElement | null,
	pluginId: string,
	locale: string,
	locales: string[]
): void {
	if (!iframe?.contentWindow) {
		return;
	}
	iframe.contentWindow.postMessage({
		type: "skoll:locale",
		contract: PLUGIN_HOST_CONTRACT,
		version: PLUGIN_HOST_VERSION,
		pluginId,
		locale,
		locales
	}, window.location.origin);
}

export function pushThemeToIframe(
	iframe: HTMLIFrameElement | null,
	pluginId: string,
	theme: ThemeBridgePayload
): void {
	if (!iframe?.contentWindow) {
		return;
	}
	iframe.contentWindow.postMessage({
		type: "skoll:theme",
		contract: PLUGIN_HOST_CONTRACT,
		version: PLUGIN_HOST_VERSION,
		pluginId,
		theme
	}, window.location.origin);
}

export type PluginHostMessageContext = {
	pluginId: string;
	source: Window | null;
	router: Router;
	reload: () => void;
	fail: (message: string) => void;
};

export function handlePluginHostMessage(event: MessageEvent, context: PluginHostMessageContext): boolean {
	if (!context.source || event.source !== context.source || event.origin !== window.location.origin) {
		return false;
	}
	const data = event.data as Record<string, unknown> | null;
	if (
		!data ||
		data.contract !== PLUGIN_HOST_CONTRACT ||
		data.version !== PLUGIN_HOST_VERSION ||
		data.pluginId !== context.pluginId
	) {
		return false;
	}
	if (data.type === "skoll:plugin-error") {
		const error = data.error as Record<string, unknown> | undefined;
		const code = typeof error?.code === "string" ? error.code : "HOST_ERROR";
		const message = typeof error?.message === "string" ? error.message : "Plugin host bridge failed";
		context.fail(`${code}: ${message}`);
		return true;
	}
	if (data.type === "skoll:command") {
		if (data.command === "reload") {
			context.reload();
			return true;
		}
		if (data.command === "home") {
			void context.router.push("/skoll");
			return true;
		}
		return false;
	}
	if (data.type !== "skoll:navigation") {
		return false;
	}
	if (data.mode === "back") {
		context.router.back();
		return true;
	}
	const path = typeof data.path === "string" ? data.path.trim() : "";
	if ((path !== "/skoll" && !path.startsWith("/skoll/")) || path.includes("..") || path.includes("://")) {
		context.fail("CONTRACT_MISMATCH: Plugin navigation target is invalid");
		return true;
	}
	if (data.mode === "push") {
		void context.router.push(path);
		return true;
	}
	if (data.mode === "replace") {
		void context.router.replace(path);
		return true;
	}
	return false;
}

export function pluginHostIdentity(
	profile: { id?: string; roles?: string[] } | null,
	permissions: string[]
): PluginHostIdentity {
	return {
		subject: profile?.id?.trim() ?? "",
		roles: [...(profile?.roles ?? [])],
		permissions: [...permissions]
	};
}

function pluginHostLifecycle(plugin: FrontendPluginManifest): PluginHostLifecycle {
	const health = typeof plugin.health === "string"
		? plugin.health
		: plugin.health?.status ?? plugin.healthStatus ?? "";
	if (plugin.enabled === false) {
		return { state: "disabled", health };
	}
	const normalized = health.trim().toLowerCase();
	const degraded = normalized !== "" && !["healthy", "ready", "ok", "running"].includes(normalized);
	return { state: degraded ? "degraded" : "enabled", health };
}
