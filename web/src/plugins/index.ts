import { computed, defineComponent, h, onMounted, onUnmounted, ref, watch } from "vue";
import type { RouteRecordRaw, Router } from "vue-router";

import { useI18n } from "../i18n";
import { getDefaultHomeTarget, type DefaultHomeTarget, type usePluginStore } from "../stores/plugins";
import { getToken } from "../utils/auth";
import { API_BASE_PREFIX } from "../utils/api-base-prefix";
import { builtinAuthPlugin } from "./builtin/auth";
import type { BackendPluginRecord, FrontendPlugin, FrontendPluginManifest } from "./types";

type PluginStore = ReturnType<typeof usePluginStore>;

const builtinPlugins: FrontendPlugin[] = [builtinAuthPlugin];
let latestPluginSyncTask: Promise<void> = Promise.resolve();
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

function injectLocaleBridgeScript(content: string, activeLocale: string, pluginLocales: string[], authToken: string): string {
	const bridgeScript = `<script>(function(){var active=${JSON.stringify(activeLocale)};var locales=${JSON.stringify(pluginLocales)};var token=${JSON.stringify(authToken)};window.__SKOLL_LOCALE=active;window.__SKOLL_LOCALES=locales;window.__SKOLL_TOKEN=token;window.__SKOLL_PLUGIN_CONTEXT={locale:active,locales:locales,token:token};if(document&&document.documentElement){document.documentElement.setAttribute('lang',active);}window.dispatchEvent(new CustomEvent('skoll:locale',{detail:{locale:active,locales:locales}}));window.addEventListener('message',function(event){var data=event&&event.data;if(!data||data.type!=='skoll:locale'){return;}var nextLocale=String(data.locale||'').trim()||active;var nextLocales=Array.isArray(data.locales)?data.locales:locales;window.__SKOLL_LOCALE=nextLocale;window.__SKOLL_LOCALES=nextLocales;window.__SKOLL_PLUGIN_CONTEXT={locale:nextLocale,locales:nextLocales,token:token};if(document&&document.documentElement){document.documentElement.setAttribute('lang',nextLocale);}window.dispatchEvent(new CustomEvent('skoll:locale',{detail:{locale:nextLocale,locales:nextLocales}}));});})();</script>`;
	if (/<head>/i.test(content)) {
		return content.replace(/<head>/i, `<head>\n${bridgeScript}`);
	}
	return `${bridgeScript}${content}`;
}

function pushLocaleToIframe(iframe: HTMLIFrameElement | null, locale: string, locales: string[]): void {
	if (!iframe?.contentWindow) {
		return;
	}
	iframe.contentWindow.postMessage({
		type: "skoll:locale",
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
				if (record.level === "app") {
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
						route: hasFrontend && record.level !== "app" && record.uiOpenMode !== "standalone"
							? {
								path: routePath,
								name: `plugin-${record.id}`,
								component: createRemotePluginView(record)
							}
							: undefined
					},
					router,
					store
				);
			}
			for (const appId of appIds) {
				addRouteIfMissing({
					path: `/${appId}`,
					name: `app-home-${appId}`,
					component: createAppHomeView(appId, store)
				}, router);
			}
			store.finishSync(null, false);
		} catch (error) {
			const msg = normalizeSyncError(error);
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

function registerPlugin(manifest: FrontendPluginManifest, router: Router, store: PluginStore): void {
	store.registerPlugin(manifest);
	if (manifest.route) {
		addRouteIfMissing(withPluginAccessMeta(manifest.route, manifest), router);
	}
}

function withPluginAccessMeta(route: RouteRecordRaw, manifest: FrontendPluginManifest): RouteRecordRaw {
	const requiredRoles = manifest.uiMenu?.requiredRoles ?? [];
	const requiredPermissions = manifest.uiMenu?.requiredPermissions ?? [];
	if (requiredRoles.length === 0 && requiredPermissions.length === 0) {
		return route;
	}
	return {
		...route,
		meta: {
			...(route.meta ?? {}),
			roles: requiredRoles,
			permissions: requiredPermissions
		}
	};
}

function addRouteIfMissing(route: RouteRecordRaw, router: Router): void {
	if (route.name && router.hasRoute(route.name)) {
		return;
	}
	router.addRoute(route);
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

function createRemotePluginView(record: BackendPluginRecord) {
	return defineComponent({
		name: `RemotePluginView_${record.id}`,
		setup() {
			const { locale } = useI18n();
			const pageError = ref("");
			const pageBroken = ref(false);
			const pageLoading = ref(true);
			const frameURL = ref("");
			const iframeRef = ref<HTMLIFrameElement | null>(null);
			const pluginLocales = normalizePluginLocales(record.i18nLocales);
			const resolvedLocale = computed(() => resolveActiveLocale(locale.value, pluginLocales));
			const pageURL = computed(() => `${API_BASE_PREFIX}/v1/plugins/${record.id}/page?locale=${encodeURIComponent(resolvedLocale.value)}`);

			function patchPluginHTML(content: string): string {
				const basePath = `${API_BASE_PREFIX}/v1/plugins/${record.id}/assets/`;
				const absoluteBasePath = `${window.location.origin}${basePath}`;
				const absoluteAssetsBase = `${absoluteBasePath}assets/`;
				const authToken = getToken().trim();
				let patched = content;
				patched = injectLocaleBridgeScript(patched, resolvedLocale.value, pluginLocales, authToken);
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

			onMounted(async () => {
				await loadPluginPage();
			});

			watch(
				() => resolvedLocale.value,
				async () => {
					await loadPluginPage();
				}
			);

			onUnmounted(() => {
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
								pushLocaleToIframe(iframeRef.value, resolvedLocale.value, pluginLocales);
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

function createAppHomeView(appId: string, store: PluginStore) {
	return defineComponent({
		name: `AppHomeView_${appId}`,
		setup() {
			const { locale } = useI18n();
			const pageError = ref("");
			const pageBroken = ref(false);
			const pageLoading = ref(true);
			const frameURL = ref("");
			const iframeRef = ref<HTMLIFrameElement | null>(null);
			const selectedPlugin = computed(() => pickAppHomePlugin(appId, store));
			const pluginLocales = computed(() => normalizePluginLocales(selectedPlugin.value?.i18nLocales));
			const resolvedLocale = computed(() => resolveActiveLocale(locale.value, pluginLocales.value));

			function patchPluginHTML(content: string, pluginID: string): string {
				const basePath = `${API_BASE_PREFIX}/v1/plugins/${pluginID}/assets/`;
				const absoluteBasePath = `${window.location.origin}${basePath}`;
				const absoluteAssetsBase = `${absoluteBasePath}assets/`;
				const authToken = getToken().trim();
				let patched = content;
				patched = injectLocaleBridgeScript(patched, resolvedLocale.value, pluginLocales.value, authToken);
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

			onUnmounted(() => {
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
								pushLocaleToIframe(iframeRef.value, resolvedLocale.value, pluginLocales.value);
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

