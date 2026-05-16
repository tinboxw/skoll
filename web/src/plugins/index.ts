import { defineComponent, h, onMounted, onUnmounted, ref } from "vue";
import type { RouteRecordRaw, Router } from "vue-router";

import type { usePluginStore } from "../stores/plugins";
import { getToken } from "../utils/auth";
import { builtinAuthPlugin } from "./builtin/auth";
import type { BackendPluginRecord, FrontendPlugin, FrontendPluginManifest } from "./types";

type PluginStore = ReturnType<typeof usePluginStore>;
const API_PREFIX = "/api";

const builtinPlugins: FrontendPlugin[] = [builtinAuthPlugin];
let latestPluginSyncTask: Promise<void> = Promise.resolve();
let markInitialBootstrapDone: (() => void) | null = null;
const initialBootstrapTask = new Promise<void>((resolve) => {
	markInitialBootstrapDone = resolve;
});

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
			store.setBackendRecords(
				records.map((record) => ({
					id: record.id,
					name: record.name,
					version: record.version,
					enabled: record.enabled,
					uiMode: record.uiMode,
					entryPath: record.frontendEntry,
					systemBuiltin: record.systemBuiltin
				}))
			);
			for (const record of records) {
				const hasFrontend = record.uiMode !== "backend_only";
				const routePath = typeof record.frontendEntry === "string" && record.frontendEntry.trim().startsWith("/")
					? record.frontendEntry.trim()
					: `/plugins/${record.id}`;
				registerPlugin(
					{
						id: record.id,
						name: record.name,
						version: record.version,
						enabled: record.enabled,
						uiMode: record.uiMode,
						entryPath: record.frontendEntry,
						systemBuiltin: record.systemBuiltin,
						route: hasFrontend
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
		resp = await fetcher(`${API_PREFIX}/v1/plugins`, {
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
			const loading = ref(true);
			const loadError = ref("");
			const pageError = ref("");
			const debugPayload = ref<Record<string, unknown> | null>(null);
			const pageBroken = ref(false);
			const frameURL = ref("");
			const pageURL = `${API_PREFIX}/v1/plugins/${record.id}/page`;

			function patchPluginHTML(content: string): string {
				const basePath = `${API_PREFIX}/v1/plugins/${record.id}/assets/`;
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
				loading.value = true;
				loadError.value = "";
				await loadPluginPage();
				try {
					const resp = await fetch(`${API_PREFIX}/v1/plugins/${record.id}/debug`);
					if (!resp.ok) {
						throw new Error(`debug request failed: ${resp.status}`);
					}
					const payload = await resp.json() as { data?: Record<string, unknown> };
					debugPayload.value = payload.data ?? null;
				} catch (error) {
					loadError.value = error instanceof Error ? error.message : "failed to load plugin details";
				} finally {
					loading.value = false;
				}
			});

			onUnmounted(() => {
				if (frameURL.value) {
					URL.revokeObjectURL(frameURL.value);
				}
			});

			return () =>
				h("section", { class: "remote-plugin-card" }, [
					h("h3", `${record.name} (${record.id})`),
					h("p", `Version: ${record.version}`),
					h("p", `Enabled: ${record.enabled === false ? "no" : "yes"}`),
					pageBroken.value
						? h("p", { class: "plugin-detail-error" }, `Plugin page failed to load: ${pageError.value || "unknown error"}. Fallback details are shown below.`)
						: h("iframe", {
							title: `${record.id}-page`,
							src: frameURL.value,
							class: "plugin-page-frame",
							style: "width:100%;min-height:360px;border:1px solid var(--color-border);border-radius:8px;background:#fff;",
							onError: () => {
								pageBroken.value = true;
								pageError.value = "iframe render failed";
							}
						}),
					loading.value
						? h("p", "Loading plugin runtime details...")
						: loadError.value
							? h("p", { class: "plugin-detail-error" }, `Failed to load plugin details: ${loadError.value}`)
							: h("pre", { class: "plugin-detail-pre" }, JSON.stringify(debugPayload.value, null, 2))
				]);
		}
	});
}

