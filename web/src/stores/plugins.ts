import { defineStore } from "pinia";

import type { FrontendPluginManifest } from "../plugins/types";

const DEFAULT_HOME_KEY = "skoll.ui.defaultHome";

type PluginState = {
	items: FrontendPluginManifest[];
	backendRecords: FrontendPluginManifest[];
	syncedFromServer: boolean;
	lastSyncError: string | null;
};

export const usePluginStore = defineStore("plugins", {
	state: (): PluginState => ({
		items: [],
		backendRecords: [],
		syncedFromServer: false,
		lastSyncError: null
	}),
	getters: {
		enabledItems: (state): FrontendPluginManifest[] => state.items.filter((item) => item.enabled !== false)
	},
	actions: {
		registerPlugin(plugin: FrontendPluginManifest): void {
			const index = this.items.findIndex((item) => item.id === plugin.id);
			if (index >= 0) {
				this.items[index] = { ...this.items[index], ...plugin };
				return;
			}
			this.items.push(plugin);
		},
		setBackendRecords(records: FrontendPluginManifest[]): void {
			this.backendRecords = records;
		},
		markSynced(error: string | null = null): void {
			this.syncedFromServer = true;
			this.lastSyncError = error;
		}
	}
});

export function resolvePluginEntryPath(plugin: FrontendPluginManifest): string {
	if (plugin.route && typeof plugin.route.path === "string" && plugin.route.path.trim().startsWith("/")) {
		return plugin.route.path.trim();
	}
	if (typeof plugin.entryPath === "string" && plugin.entryPath.trim().startsWith("/")) {
		return plugin.entryPath.trim();
	}
	return "";
}

export function getDefaultHomePath(fallback = "/dashboard"): string {
	const raw = localStorage.getItem(DEFAULT_HOME_KEY);
	if (typeof raw !== "string") {
		return fallback;
	}
	const value = raw.trim();
	if (!value.startsWith("/") || value === "/login") {
		return fallback;
	}
	return value;
}

export function setDefaultHomePath(path: string): void {
	const value = path.trim();
	if (!value.startsWith("/") || value === "/login") {
		return;
	}
	localStorage.setItem(DEFAULT_HOME_KEY, value);
}

