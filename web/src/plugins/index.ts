import { defineComponent, h, onMounted, ref } from "vue";
import type { RouteRecordRaw, Router } from "vue-router";

import type { usePluginStore } from "../stores/plugins";
import { builtinAuthPlugin } from "./builtin/auth";
import type { BackendPluginRecord, FrontendPlugin, FrontendPluginManifest } from "./types";

type PluginStore = ReturnType<typeof usePluginStore>;

const builtinPlugins: FrontendPlugin[] = [builtinAuthPlugin];

export async function bootstrapPlugins(
	router: Router,
	store: PluginStore,
	fetcher: typeof fetch = fetch
): Promise<void> {
	for (const plugin of builtinPlugins) {
		registerPlugin(plugin.manifest, router, store);
		plugin.setup({
			router,
			registerRoute: (route: RouteRecordRaw) => addRouteIfMissing(route, router)
		});
	}

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
		store.markSynced(null);
	} catch (error) {
		const msg = normalizeSyncError(error);
		store.setBackendRecords([]);
		store.markSynced(msg);
	}
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
	let resp: Response;
	try {
		resp = await fetcher("/v1/plugins", { signal: controller.signal });
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
			const debugPayload = ref<Record<string, unknown> | null>(null);
			const pageBroken = ref(false);
			const pageURL = `/v1/plugins/${record.id}/page`;

			onMounted(async () => {
				loading.value = true;
				loadError.value = "";
				try {
					const resp = await fetch(`/v1/plugins/${record.id}/debug`);
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

			return () =>
				h("section", { class: "remote-plugin-card" }, [
					h("h3", `${record.name} (${record.id})`),
					h("p", `Version: ${record.version}`),
					h("p", `Enabled: ${record.enabled === false ? "no" : "yes"}`),
					pageBroken.value
						? h("p", { class: "plugin-detail-error" }, "Plugin page failed to load. Fallback details are shown below.")
						: h("iframe", {
							title: `${record.id}-page`,
							src: pageURL,
							class: "plugin-page-frame",
							style: "width:100%;min-height:360px;border:1px solid var(--color-border);border-radius:8px;background:#fff;",
							onError: () => {
								pageBroken.value = true;
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

