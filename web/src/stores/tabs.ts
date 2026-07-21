import { defineStore } from "pinia";

const TABS_STATE_KEY = "skoll.ui.tabsState";
const MAX_RECENT_TABS = 8;

export type PinnedTab = {
	id: string;
	label: string;
	path: string;
};

type TabsState = {
	pinnedItems: PinnedTab[];
	recentItems: PinnedTab[];
};

function normalizePath(path: string): string {
	const value = path.trim();
	if (!value.startsWith("/")) {
		return `/${value}`;
	}
	return value;
}

function normalizeTab(tab: PinnedTab): PinnedTab | null {
	const id = tab?.id?.trim() || "";
	const label = tab?.label?.trim() || "";
	const path = tab?.path?.trim() || "";
	if (!id || !label || !path) {
		return null;
	}
	return {
		id,
		label,
		path: normalizePath(path)
	};
}

function normalizeTabArray(items: unknown): PinnedTab[] {
	if (!Array.isArray(items)) {
		return [];
	}
	const tabs: PinnedTab[] = [];
	for (const item of items) {
		const normalized = normalizeTab(item as PinnedTab);
		if (normalized) {
			tabs.push(normalized);
		}
	}
	return tabs;
}

function loadState(): TabsState {
	const raw = localStorage.getItem(TABS_STATE_KEY);
	if (raw) {
		try {
			const parsed = JSON.parse(raw) as { pinnedItems?: unknown; recentItems?: unknown };
			return {
				pinnedItems: normalizeTabArray(parsed?.pinnedItems),
				recentItems: normalizeTabArray(parsed?.recentItems).slice(0, MAX_RECENT_TABS)
			};
		} catch {}
	}
	return {
		pinnedItems: [],
		recentItems: []
	};
}

function saveState(state: TabsState): void {
	localStorage.setItem(TABS_STATE_KEY, JSON.stringify(state));
}

function upsertByPath(items: PinnedTab[], tab: PinnedTab): PinnedTab[] {
	const idx = items.findIndex((item) => item.path === tab.path);
	if (idx < 0) {
		return [...items, tab];
	}
	const next = items.slice();
	next[idx] = tab;
	return next;
}

export const useTabsStore = defineStore("tabs", {
	state: (): TabsState => ({
		...loadState()
	}),
	getters: {
		items(state): PinnedTab[] {
			return state.pinnedItems;
		}
	},
	actions: {
		pinPluginTab(tab: PinnedTab): void {
			const normalized = normalizeTab(tab);
			if (!normalized) {
				return;
			}
			this.pinnedItems = upsertByPath(this.pinnedItems, normalized);
			this.recentItems = this.recentItems.filter((item) => item.path !== normalized.path);
			saveState(this.$state);
		},
		unpinByPath(path: string): void {
			const normalized = normalizePath(path);
			this.pinnedItems = this.pinnedItems.filter((item) => item.path !== normalized);
			saveState(this.$state);
		},
		touchRecentTab(tab: PinnedTab): void {
			const normalized = normalizeTab(tab);
			if (!normalized) {
				return;
			}
			if (this.pinnedItems.some((item) => item.path === normalized.path)) {
				this.pinnedItems = upsertByPath(this.pinnedItems, normalized);
				saveState(this.$state);
				return;
			}
			this.recentItems = [normalized, ...this.recentItems.filter((item) => item.path !== normalized.path)].slice(0, MAX_RECENT_TABS);
			saveState(this.$state);
		},
		removeRecentByPath(path: string): void {
			const normalized = normalizePath(path);
			this.recentItems = this.recentItems.filter((item) => item.path !== normalized);
			saveState(this.$state);
		},
		prunePluginTabs(allowedPluginIDs: string[]): void {
			const allowed = new Set(
				allowedPluginIDs
					.map((value) => value.trim())
					.filter((value) => value !== "")
			);
			const keepTab = (item: PinnedTab): boolean => {
				if (!item.path.startsWith("/skoll/plugins/")) {
					return true;
				}
				return allowed.has(item.id);
			};

			this.pinnedItems = this.pinnedItems.filter(keepTab);
			this.recentItems = this.recentItems.filter(keepTab).slice(0, MAX_RECENT_TABS);
			saveState(this.$state);
		},
		isPinned(path: string): boolean {
			const normalized = normalizePath(path);
			return this.pinnedItems.some((item) => item.path === normalized);
		}
	}
});
