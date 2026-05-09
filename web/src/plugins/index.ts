import { defineComponent, h } from "vue";
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
		const msg = error instanceof Error ? error.message : "plugin sync failed";
		store.setBackendRecords([]);
		store.markSynced(msg);
	}
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
	const resp = await fetcher("/v1/plugins");
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
			return () =>
				h("section", { class: "remote-plugin-card" }, [
					h("h3", `${record.name} (${record.id})`),
					h("p", `Version: ${record.version}`),
					h("p", `Enabled: ${record.enabled === false ? "no" : "yes"}`),
					h("p", "This page is loaded from backend plugin metadata and can be replaced by real plugin bundle loader.")
				]);
		}
	});
}

