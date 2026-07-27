import { computed, defineComponent, h, onMounted, onUnmounted, ref, watch } from "vue";
import type { RouteRecordRaw, Router } from "vue-router";

import {
	PLUGIN_HOST_CONTRACT,
	PLUGIN_HOST_VERSION,
	type PluginHostIdentity,
	type PluginHostLifecycle
} from "@skoll/plugin-sdk";

import { useI18n } from "../i18n";
import { withRouteAccessMeta } from "../permissions/route";
import { getDefaultHomeTarget, type DefaultHomeTarget, type usePluginStore } from "../stores/plugins";
import { useThemeStore, type ThemeBridgePayload } from "../stores/theme";
import { useUserStore } from "../stores/user";
import { getToken } from "../utils/auth";
import { API_BASE_PREFIX } from "../utils/api-base-prefix";
import { builtinAuthPlugin } from "./builtin/auth";
import { buildPluginHostBridgeScript } from "./host-sdk";
import type { BackendPluginRecord, FrontendPlugin, FrontendPluginManifest } from "./types";

type PluginStore = ReturnType<typeof usePluginStore>;

const builtinPlugins: FrontendPlugin[] = [builtinAuthPlugin];
let latestPluginSyncTask: Promise<void> = Promise.resolve();
let backendPluginRouteDisposers: Array<() => void> = [];
let markInitialBootstrapDone: (() => void) | null = null;
const initialBootstrapTask = new Promise<void>((resolve) => {
	markInitialBootstrapDone = resolve;
});

function normalizePluginRoutePath(path: string): string {
	const value = path.trim();
	if (value.startsWith("/plugins/")) {
		return `/skoll${value}`;
	}
	return value;
}

function normalizePluginLocales(locales?: string[]): string[] {
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

function resolveActiveLocale(hostLocale: string, pluginLocales: string[]): string {
	const normalizedHostLocale = String(hostLocale || "").trim();
	if (normalizedHostLocale !== "" && pluginLocales.includes(normalizedHostLocale)) {
		return normalizedHostLocale;
	}
	return pluginLocales[0] || "zh-CN";
}

function injectHostBridgeScript(
	content: string,
	pluginId: string,
	pluginVersion: string,
	activeLocale: string,
	pluginLocales: string[],
	authToken: string,
	theme: ThemeBridgePayload,
	identity: PluginHostIdentity,
	lifecycle: PluginHostLifecycle
): string {
	const bridgeScript = buildPluginHostBridgeScript({
		pluginId,
		pluginVersion,
		apiBasePrefix: API_BASE_PREFIX,
		locale: activeLocale,
		locales: pluginLocales,
		token: authToken,
		theme,
		identity,
		lifecycle
	});
	if (/<head>/i.test(content)) {
		return content.replace(/<head>/i, `<head>\n${bridgeScript}`);
	}
	return `${bridgeScript}${content}`;
}

function pushLocaleToIframe(iframe: HTMLIFrameElement | null, pluginId: string, locale: string, locales: string[]): void {
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

export async function waitForPluginBootstrap(): Promise<void> {
	await initialBootstrapTask;
	await latestPluginSyncTask;
}

export async function bootstrapPlugins(
	router: Router,
	store: PluginStore,
	fetcher: typeof fetch = fetch
): Promise<void> {
	try {
		for (const plugin of builtinPlugins) {
			registerPlugin(plugin.manifest, router, store);
			plugin.setup({
				router,
				registerRoute: (route: RouteRecordRaw) => addRouteIfMissing(route, router)
			});
		}

		await syncBackendPlugins(router, store, fetcher);
	} finally {
		markInitialBootstrapDone?.();
		markInitialBootstrapDone = null;
	}
}

export async function syncBackendPlugins(
	router: Router,
	store: PluginStore,
	fetcher: typeof fetch = fetch
): Promise<void> {
	const task = (async () => {
		store.beginSync();
		try {
			const records = await syncPluginsFromBackend(fetcher);
			clearBackendPluginRoutes();
			const appIds = new Set<string>();
			store.setBackendRecords(
				records.map((record) => ({
					id: record.id,
					name: record.name,
					nameZhCN: record.nameZhCN,
					nameEnUS: record.nameEnUS,
					version: record.version,
					enabled: record.enabled,
					uiMode: record.uiMode,
					level: record.level,
					appId: record.appId,
					mountPolicy: record.mountPolicy,
					uiNavPosition: record.uiNavPosition,
					uiOpenMode: record.uiOpenMode,
					uiTabMode: record.uiTabMode,
					i18nLocales: record.i18nLocales,
					uiMenu: record.uiMenu,
					configSchema: record.configSchema,
					entryPath: typeof record.frontendEntry === "string" ? normalizePluginRoutePath(record.frontendEntry) : record.frontendEntry,
					systemBuiltin: record.systemBuiltin
				}))
			);
			for (const record of records) {
				const hasFrontend = record.uiMode !== "backend_only";
				const enabled = record.enabled !== false;
				if (record.level === "app" && enabled) {
					const appId = (record.appId || "").trim();
					if (appId !== "" && hasFrontend && record.uiOpenMode !== "standalone") {
						appIds.add(appId);
					}
				}
				const routePath = typeof record.frontendEntry === "string" && record.frontendEntry.trim().startsWith("/")
					? normalizePluginRoutePath(record.frontendEntry)
					: `/skoll/plugins/${record.id}`;
				registerPlugin(
					{
						id: record.id,
						name: record.name,
						nameZhCN: record.nameZhCN,
						nameEnUS: record.nameEnUS,
						version: record.version,
						enabled: record.enabled,
						uiMode: record.uiMode,
						level: record.level,
						appId: record.appId,
						mountPolicy: record.mountPolicy,
						uiNavPosition: record.uiNavPosition,
						uiOpenMode: record.uiOpenMode,
						uiTabMode: record.uiTabMode,
						i18nLocales: record.i18nLocales,
						uiMenu: record.uiMenu,
						configSchema: record.configSchema,
						entryPath: typeof record.frontendEntry === "string" ? normalizePluginRoutePath(record.frontendEntry) : record.frontendEntry,
						systemBuiltin: record.systemBuiltin,
						route: enabled && hasFrontend && record.level !== "app" && record.uiOpenMode !== "standalone"
							? {
								path: routePath,
								name: `plugin-${record.id}`,
								component: createRemotePluginView(record, router)
							}
							: undefined
					},
					router,
					store,
					true
				);
			}
			for (const appId of appIds) {
				addRouteIfMissing({
					path: `/${appId}`,
					name: `app-home-${appId}`,
					component: createAppHomeView(appId, store, router)
				}, router, true);
			}
			store.finishSync(null, false);
		} catch (error) {
			const msg = normalizeSyncError(error);
			clearBackendPluginRoutes();
			store.setBackendRecords([]);
			store.finishSync(msg, true);
		}
	})();
	latestPluginSyncTask = task.catch(() => undefined);
	await task;
}

function normalizeSyncError(error: unknown): string {
	if (error instanceof DOMException && error.name === "AbortError") {
		return "backend plugin sync timeout";
	}
	if (error instanceof Error) {
		const message = error.message.trim();
		if (message.toLowerCase().includes("aborted")) {
			return "backend plugin sync timeout";
		}
		if (message.toLowerCase().includes("failed to fetch")) {
			return "backend plugin service unavailable";
		}
		return message || "plugin sync failed";
	}
	return "plugin sync failed";
}

function registerPlugin(manifest: FrontendPluginManifest, router: Router, store: PluginStore, backendRoute = false): void {
	store.registerPlugin(manifest);
	if (manifest.route) {
		addRouteIfMissing(withPluginAccessMeta(manifest.route, manifest), router, backendRoute);
	}
}

function pushThemeToIframe(iframe: HTMLIFrameElement | null, pluginId: string, theme: ThemeBridgePayload): void {
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

type PluginHostMessageContext = {
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

function pluginHostIdentity(userStore: ReturnType<typeof useUserStore>): PluginHostIdentity {
	return {
		subject: userStore.profile?.id?.trim() ?? "",
		roles: [...(userStore.profile?.roles ?? [])],
		permissions: [...userStore.permissions]
	};
}

function pluginHostLifecycle(record: FrontendPluginManifest): PluginHostLifecycle {
	const health = typeof record.health === "string"
		? record.health
		: record.health?.status ?? record.healthStatus ?? "";
	if (record.enabled === false) {
		return { state: "disabled", health };
	}
	const normalized = health.trim().toLowerCase();
	const degraded = normalized !== "" && !["healthy", "ready", "ok", "running"].includes(normalized);
	return { state: degraded ? "degraded" : "enabled", health };
}

function withPluginAccessMeta(route: RouteRecordRaw, manifest: FrontendPluginManifest): RouteRecordRaw {
	const requiredRoles = manifest.uiMenu?.requiredRoles ?? [];
	const requiredPermissions = manifest.uiMenu?.requiredPermissions ?? [];
	return withRouteAccessMeta(route, {
		roles: requiredRoles,
		permissions: requiredPermissions
	});
}

function addRouteIfMissing(route: RouteRecordRaw, router: Router, backendRoute = false): void {
	if (route.name && router.hasRoute(route.name)) {
		return;
	}
	const removeRoute = router.addRoute(route);
	if (backendRoute) {
		backendPluginRouteDisposers.push(removeRoute);
	}
}

function clearBackendPluginRoutes(): void {
	for (const dispose of backendPluginRouteDisposers.splice(0)) {
		dispose();
	}
}

async function syncPluginsFromBackend(fetcher: typeof fetch): Promise<BackendPluginRecord[]> {
	const controller = new AbortController();
	const timeout = setTimeout(() => controller.abort(), 2500);
	const headers = new Headers();
	const token = getToken().trim();
	if (token !== "") {
		const authValue = token.toLowerCase().startsWith("bearer ") ? token : `Bearer ${token}`;
		headers.set("Authorization", authValue);
	}
	let resp: Response;
	try {
		resp = await fetcher(`${API_BASE_PREFIX}/v1/plugins`, {
			signal: controller.signal,
			headers
		});
	} finally {
		clearTimeout(timeout);
	}
	if (!resp.ok) {
		throw new Error(`plugin sync failed with status ${resp.status}`);
	}

	const payload = await resp.json() as { data?: BackendPluginRecord[] };
	return Array.isArray(payload.data) ? payload.data : [];
}

function createRemotePluginView(record: BackendPluginRecord, router: Router) {
	return defineComponent({
		name: `RemotePluginView_${record.id}`,
		setup() {
			const { locale } = useI18n();
			const themeStore = useThemeStore();
			const userStore = useUserStore();
			const pageError = ref("");
			const pageBroken = ref(false);
			const pageLoading = ref(true);
			const frameURL = ref("");
			const iframeRef = ref<HTMLIFrameElement | null>(null);
			const pluginLocales = normalizePluginLocales(record.i18nLocales);
			const resolvedLocale = computed(() => resolveActiveLocale(locale.value, pluginLocales));
			const bridgeTheme = computed(() => themeStore.pluginBridgeTheme);
			const pageURL = computed(() => `${API_BASE_PREFIX}/v1/plugins/${record.id}/page?locale=${encodeURIComponent(resolvedLocale.value)}`);

			function patchPluginHTML(content: string): string {
				const basePath = `${API_BASE_PREFIX}/v1/plugins/${record.id}/assets/`;
				const absoluteBasePath = `${window.location.origin}${basePath}`;
				const absoluteAssetsBase = `${absoluteBasePath}assets/`;
				const authToken = getToken().trim();
				let patched = content;
				patched = injectHostBridgeScript(
					patched,
					record.id,
					record.version,
					resolvedLocale.value,
					pluginLocales,
					authToken,
					bridgeTheme.value,
					pluginHostIdentity(userStore),
					pluginHostLifecycle(record)
				);
				if (!/<base\s+/i.test(patched)) {
					patched = patched.replace(/<head>/i, `<head>\n<base href="${absoluteBasePath}">`);
				}
				patched = patched.replace(/(["'])\/assets\//g, `$1${absoluteAssetsBase}`);
				return patched;
			}

			async function loadPluginPage(): Promise<void> {
				pageLoading.value = true;
				pageError.value = "";
				pageBroken.value = false;
				try {
					const resp = await fetch(pageURL.value);
					if (!resp.ok) {
						throw new Error(`plugin page request failed: ${resp.status}`);
					}
					const html = await resp.text();
					const blob = new Blob([patchPluginHTML(html)], { type: "text/html" });
					if (frameURL.value) {
						URL.revokeObjectURL(frameURL.value);
					}
					frameURL.value = URL.createObjectURL(blob);
				} catch (error) {
					pageBroken.value = true;
					pageError.value = error instanceof Error ? error.message : "failed to load plugin page";
				} finally {
					pageLoading.value = false;
				}
			}

			function onHostMessage(event: MessageEvent): void {
				handlePluginHostMessage(event, {
					pluginId: record.id,
					source: iframeRef.value?.contentWindow ?? null,
					router,
					reload: () => void loadPluginPage(),
					fail: (message) => {
						pageBroken.value = true;
						pageError.value = message;
					}
				});
			}

			onMounted(async () => {
				window.addEventListener("message", onHostMessage);
				await loadPluginPage();
			});

			watch(
				() => resolvedLocale.value,
				async () => {
					await loadPluginPage();
				}
			);

			watch(
				() => bridgeTheme.value,
				(theme) => {
					pushThemeToIframe(iframeRef.value, record.id, theme);
				}
			);

			onUnmounted(() => {
				window.removeEventListener("message", onHostMessage);
				if (frameURL.value) {
					URL.revokeObjectURL(frameURL.value);
				}
			});

			return () =>
				h("section", { class: "remote-plugin-fullpage", "data-loading": pageLoading.value ? "true" : "false" }, [
					pageBroken.value
						? h("p", { class: "plugin-detail-error" }, `Plugin page failed to load: ${pageError.value || "unknown error"}.`)
						: h("iframe", {
							title: `${record.id}-page`,
							src: frameURL.value,
							class: "plugin-page-frame",
							ref: (el: unknown) => {
								iframeRef.value = el as HTMLIFrameElement | null;
							},
							onLoad: () => {
								pushLocaleToIframe(iframeRef.value, record.id, resolvedLocale.value, pluginLocales);
								pushThemeToIframe(iframeRef.value, record.id, bridgeTheme.value);
							},
							onError: () => {
								pageBroken.value = true;
								pageError.value = "iframe render failed";
							}
						})
				]);
		}
	});
}

function pickAppHomePlugin(appId: string, store: PluginStore): FrontendPluginManifest | null {
	const candidates = store.items.filter((item) => {
		if (item.level !== "app") {
			return false;
		}
		if ((item.appId || "").trim() !== appId) {
			return false;
		}
		if (item.enabled === false || item.uiMode === "backend_only" || item.uiOpenMode === "standalone") {
			return false;
		}
		return true;
	});
	if (candidates.length === 0) {
		return null;
	}
	const target = getDefaultHomeTarget();
	if (isAppDefaultTargetFor(target, appId)) {
		const preferred = candidates.find((item) => item.id === target.pluginId);
		if (preferred) {
			return preferred;
		}
	}
	return candidates[0];
}

function isAppDefaultTargetFor(target: DefaultHomeTarget | null, appId: string): target is DefaultHomeTarget {
	return Boolean(target && target.level === "app" && target.appId === appId);
}

function createAppHomeView(appId: string, store: PluginStore, router: Router) {
	return defineComponent({
		name: `AppHomeView_${appId}`,
		setup() {
			const { locale } = useI18n();
			const themeStore = useThemeStore();
			const userStore = useUserStore();
			const pageError = ref("");
			const pageBroken = ref(false);
			const pageLoading = ref(true);
			const frameURL = ref("");
			const iframeRef = ref<HTMLIFrameElement | null>(null);
			const selectedPlugin = computed(() => pickAppHomePlugin(appId, store));
			const pluginLocales = computed(() => normalizePluginLocales(selectedPlugin.value?.i18nLocales));
			const resolvedLocale = computed(() => resolveActiveLocale(locale.value, pluginLocales.value));
			const bridgeTheme = computed(() => themeStore.pluginBridgeTheme);

			function patchPluginHTML(content: string, pluginID: string): string {
				const basePath = `${API_BASE_PREFIX}/v1/plugins/${pluginID}/assets/`;
				const absoluteBasePath = `${window.location.origin}${basePath}`;
				const absoluteAssetsBase = `${absoluteBasePath}assets/`;
				const authToken = getToken().trim();
				const plugin = selectedPlugin.value;
				if (!plugin || plugin.id !== pluginID) {
					throw new Error("selected plugin host identity changed");
				}
				let patched = content;
				patched = injectHostBridgeScript(
					patched,
					pluginID,
					plugin.version,
					resolvedLocale.value,
					pluginLocales.value,
					authToken,
					bridgeTheme.value,
					pluginHostIdentity(userStore),
					pluginHostLifecycle(plugin)
				);
				if (!/<base\s+/i.test(patched)) {
					patched = patched.replace(/<head>/i, `<head>\n<base href="${absoluteBasePath}">`);
				}
				patched = patched.replace(/(["'])\/assets\//g, `$1${absoluteAssetsBase}`);
				return patched;
			}

			async function loadPluginPage(pluginID: string): Promise<void> {
				pageLoading.value = true;
				pageError.value = "";
				pageBroken.value = false;
				try {
					const resp = await fetch(`${API_BASE_PREFIX}/v1/plugins/${pluginID}/page?locale=${encodeURIComponent(resolvedLocale.value)}`);
					if (!resp.ok) {
						throw new Error(`plugin page request failed: ${resp.status}`);
					}
					const html = await resp.text();
					const blob = new Blob([patchPluginHTML(html, pluginID)], { type: "text/html" });
					if (frameURL.value) {
						URL.revokeObjectURL(frameURL.value);
					}
					frameURL.value = URL.createObjectURL(blob);
				} catch (error) {
					pageBroken.value = true;
					pageError.value = error instanceof Error ? error.message : "failed to load plugin page";
				} finally {
					pageLoading.value = false;
				}
			}

			watch(
				() => selectedPlugin.value?.id || "",
				async (pluginID) => {
					if (!pluginID) {
						pageLoading.value = false;
						pageBroken.value = true;
						pageError.value = `no enabled app plugin found for app '${appId}'`;
						if (frameURL.value) {
							URL.revokeObjectURL(frameURL.value);
							frameURL.value = "";
						}
						return;
					}
					await loadPluginPage(pluginID);
				},
				{ immediate: true }
			);

			watch(
				() => resolvedLocale.value,
				async () => {
					const pluginID = selectedPlugin.value?.id || "";
					if (!pluginID) {
						return;
					}
					await loadPluginPage(pluginID);
				}
			);

			watch(
				() => bridgeTheme.value,
				(theme) => {
					const pluginID = selectedPlugin.value?.id ?? "";
					if (pluginID) {
						pushThemeToIframe(iframeRef.value, pluginID, theme);
					}
				}
			);

			function onHostMessage(event: MessageEvent): void {
				const pluginID = selectedPlugin.value?.id ?? "";
				if (!pluginID) {
					return;
				}
				handlePluginHostMessage(event, {
					pluginId: pluginID,
					source: iframeRef.value?.contentWindow ?? null,
					router,
					reload: () => void loadPluginPage(pluginID),
					fail: (message) => {
						pageBroken.value = true;
						pageError.value = message;
					}
				});
			}

			onMounted(() => {
				window.addEventListener("message", onHostMessage);
			});

			onUnmounted(() => {
				window.removeEventListener("message", onHostMessage);
				if (frameURL.value) {
					URL.revokeObjectURL(frameURL.value);
				}
			});

			return () =>
				h("section", { class: "remote-plugin-fullpage", "data-loading": pageLoading.value ? "true" : "false" }, [
					pageBroken.value
						? h("p", { class: "plugin-detail-error" }, `Plugin page failed to load: ${pageError.value || "unknown error"}.`)
						: h("iframe", {
							title: `${appId}-home-page`,
							src: frameURL.value,
							class: "plugin-page-frame",
							ref: (el: unknown) => {
								iframeRef.value = el as HTMLIFrameElement | null;
							},
							onLoad: () => {
								const pluginID = selectedPlugin.value?.id ?? "";
								if (pluginID) {
									pushLocaleToIframe(iframeRef.value, pluginID, resolvedLocale.value, pluginLocales.value);
									pushThemeToIframe(iframeRef.value, pluginID, bridgeTheme.value);
								}
							},
							onError: () => {
								pageBroken.value = true;
								pageError.value = "iframe render failed";
							}
						})
				]);
		}
	});
}

