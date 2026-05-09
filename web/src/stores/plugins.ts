import { defineStore } from "pinia";

import type { FrontendPluginManifest } from "../plugins/types";

type PluginState = {
	items: FrontendPluginManifest[];
	syncedFromServer: boolean;
	lastSyncError: string | null;
};

export const usePluginStore = defineStore("plugins", {
	state: (): PluginState => ({
		items: [],
		syncedFromServer: false,
		lastSyncError: null
	}),
	actions: {
		registerPlugin(plugin: FrontendPluginManifest): void {
			if (this.items.some((item) => item.id === plugin.id)) {
				return;
			}
			this.items.push(plugin);
		},
		markSynced(error: string | null = null): void {
			this.syncedFromServer = true;
			this.lastSyncError = error;
		}
	}
});

