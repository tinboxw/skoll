import { computed, defineAsyncComponent, defineComponent, h } from "vue";
import type { RouteRecordRaw, Router } from "vue-router";

import { withRouteAccessMeta } from "../permissions/route";
import { getDefaultHomeTarget, type DefaultHomeTarget, type usePluginStore } from "../stores/plugins";
import { getToken } from "../utils/auth";
import { API_BASE_PREFIX } from "../utils/api-base-prefix";
import { builtinAuthPlugin } from "./builtin/auth";
import type { BackendPluginRecord, FrontendPlugin, FrontendPluginManifest } from "./types";

type PluginStore = ReturnType<typeof usePluginStore>;

const builtinPlugins: FrontendPlugin[] = [builtinAuthPlugin];
const RemotePluginRuntime = defineAsyncComponent(() => import("./RemotePluginRuntime.vue"));
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
									component: createRemotePluginView(record)
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
						component: createAppHomeView(appId, store)
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

function createRemotePluginView(record: BackendPluginRecord) {
	return defineComponent({
		name: `RemotePluginView_${record.id}`,
		setup: () => () => h(RemotePluginRuntime, {
			plugin: record
		})
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
			const selectedPlugin = computed(() => pickAppHomePlugin(appId, store));
			return () => h(RemotePluginRuntime, {
				plugin: selectedPlugin.value
			});
		}
	});
}

export { handlePluginHostMessage } from "./runtime-bridge";

