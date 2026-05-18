import { computed, defineComponent, h, onMounted, onUnmounted, ref, watch } from "vue";
import type { RouteRecordRaw, Router } from "vue-router";

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
					version: record.version,
					enabled: record.enabled,
					uiMode: record.uiMode,
					level: record.level,
					appId: record.appId,
					mountPolicy: record.mountPolicy,
					entryPath: typeof record.frontendEntry === "string" ? normalizePluginRoutePath(record.frontendEntry) : record.frontendEntry,
					systemBuiltin: record.systemBuiltin
				}))
			);
			for (const record of records) {
				const hasFrontend = record.uiMode !== "backend_only";
				if (record.level === "app") {
					const appId = (record.appId || "").trim();
					if (appId !== "" && hasFrontend) {
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
						version: record.version,
						enabled: record.enabled,
						uiMode: record.uiMode,
						level: record.level,
						appId: record.appId,
						mountPolicy: record.mountPolicy,
						entryPath: typeof record.frontendEntry === "string" ? normalizePluginRoutePath(record.frontendEntry) : record.frontendEntry,
						systemBuiltin: record.systemBuiltin,
						route: hasFrontend && record.level !== "app"
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
		addRouteIfMissing(manifest.route, router);
	}
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
			const pageError = ref("");
			const pageBroken = ref(false);
			const frameURL = ref("");
			const pageURL = `${API_BASE_PREFIX}/v1/plugins/${record.id}/page`;

			function patchPluginHTML(content: string): string {
				const basePath = `${API_BASE_PREFIX}/v1/plugins/${record.id}/assets/`;
				const absoluteAssetsBase = `${basePath}assets/`;
				let patched = content;
				if (!/<base\s+/i.test(patched)) {
					patched = patched.replace(/<head>/i, `<head>\n<base href="${basePath}">`);
				}
				patched = patched.replace(/(["'])\/assets\//g, `$1${absoluteAssetsBase}`);
				return patched;
			}

			async function loadPluginPage(): Promise<void> {
				pageError.value = "";
				pageBroken.value = false;
				try {
					const resp = await fetch(pageURL);
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
				}
			}

			onMounted(async () => {
				await loadPluginPage();
			});

			onUnmounted(() => {
				if (frameURL.value) {
					URL.revokeObjectURL(frameURL.value);
				}
			});

			return () =>
				h("section", { class: "remote-plugin-fullpage", style: "min-height:100vh;background:#fff;" }, [
					pageBroken.value
						? h("p", { class: "plugin-detail-error", style: "padding:20px;" }, `Plugin page failed to load: ${pageError.value || "unknown error"}.`)
						: h("iframe", {
							title: `${record.id}-page`,
							src: frameURL.value,
							class: "plugin-page-frame",
							style: "width:100%;min-height:100vh;border:0;display:block;background:#fff;",
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
		if (item.enabled === false || item.uiMode === "backend_only") {
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
			const pageError = ref("");
			const pageBroken = ref(false);
			const frameURL = ref("");
			const selectedPlugin = computed(() => pickAppHomePlugin(appId, store));

			function patchPluginHTML(content: string, pluginID: string): string {
				const basePath = `${API_BASE_PREFIX}/v1/plugins/${pluginID}/assets/`;
				const absoluteAssetsBase = `${basePath}assets/`;
				let patched = content;
				if (!/<base\s+/i.test(patched)) {
					patched = patched.replace(/<head>/i, `<head>\n<base href="${basePath}">`);
				}
				patched = patched.replace(/(["'])\/assets\//g, `$1${absoluteAssetsBase}`);
				return patched;
			}

			async function loadPluginPage(pluginID: string): Promise<void> {
				pageError.value = "";
				pageBroken.value = false;
				try {
					const resp = await fetch(`${API_BASE_PREFIX}/v1/plugins/${pluginID}/page`);
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
				}
			}

			watch(
				() => selectedPlugin.value?.id || "",
				async (pluginID) => {
					if (!pluginID) {
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

			onUnmounted(() => {
				if (frameURL.value) {
					URL.revokeObjectURL(frameURL.value);
				}
			});

			return () =>
				h("section", { class: "remote-plugin-fullpage", style: "min-height:100vh;background:#fff;" }, [
					pageBroken.value
						? h("p", { class: "plugin-detail-error", style: "padding:20px;" }, `Plugin page failed to load: ${pageError.value || "unknown error"}.`)
						: h("iframe", {
							title: `${appId}-home-page`,
							src: frameURL.value,
							class: "plugin-page-frame",
							style: "width:100%;min-height:100vh;border:0;display:block;background:#fff;",
							onError: () => {
								pageBroken.value = true;
								pageError.value = "iframe render failed";
							}
						})
				]);
		}
	});
}

