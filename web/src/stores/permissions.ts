import { defineStore } from "pinia";

import {
	disablePermissionResource,
	enablePermissionResource,
	listPermissionResources,
	type PermissionDiffRequest,
	type PermissionDiffResult,
	type PermissionListQuery,
	type PermissionResource,
	type PermissionResourceType,
	diffPermissionResources
} from "../permissions/api";

type PermissionStoreState = {
	items: PermissionResource[];
	syncStatus: "idle" | "loading" | "success" | "error";
	lastError: string | null;
	lastLoadedAt: string | null;
	lastQuery: PermissionListQuery;
};

export const usePermissionStore = defineStore("permissions", {
	state: (): PermissionStoreState => ({
		items: [],
		syncStatus: "idle",
		lastError: null,
		lastLoadedAt: null,
		lastQuery: {}
	}),
	getters: {
		isLoading: (state): boolean => state.syncStatus === "loading",
		hasError: (state): boolean => state.syncStatus === "error",
		enabledItems: (state): PermissionResource[] => state.items.filter((item) => item.enabled),
		byKey: (state) => (key: string): PermissionResource | undefined => state.items.find((item) => item.key === key.trim()),
		byType: (state) => (type: PermissionResourceType): PermissionResource[] => state.items.filter((item) => item.type === type),
		bySource: (state) => (source: string): PermissionResource[] => {
			const normalized = source.trim().toLowerCase();
			return state.items.filter((item) => item.source.toLowerCase() === normalized);
		}
	},
	actions: {
		async load(query: PermissionListQuery = {}, options: { force?: boolean } = {}): Promise<void> {
			const normalizedQuery = normalizeQuery(query);
			if (!options.force && this.syncStatus === "success" && this.lastLoadedAt && sameQuery(this.lastQuery, normalizedQuery)) {
				return;
			}
			this.syncStatus = "loading";
			this.lastError = null;
			this.lastQuery = normalizedQuery;
			try {
				const result = await listPermissionResources(normalizedQuery);
				this.items = result.items;
				this.syncStatus = "success";
				this.lastLoadedAt = new Date().toISOString();
			} catch (error) {
				this.syncStatus = "error";
				this.lastError = error instanceof Error ? error.message : "permission_load_failed";
				throw error;
			}
		},
		async retry(): Promise<void> {
			await this.load(this.lastQuery, { force: true });
		},
		async refresh(): Promise<void> {
			await this.load(this.lastQuery, { force: true });
		},
		async enable(key: string): Promise<void> {
			const result = await enablePermissionResource(key);
			this.patchEnabled(result.key, result.enabled);
		},
		async disable(key: string): Promise<void> {
			const result = await disablePermissionResource(key);
			this.patchEnabled(result.key, result.enabled);
		},
		async diff(request: PermissionDiffRequest): Promise<PermissionDiffResult> {
			return diffPermissionResources(request);
		},
		clear(): void {
			this.items = [];
			this.syncStatus = "idle";
			this.lastError = null;
			this.lastLoadedAt = null;
			this.lastQuery = {};
		},
		patchEnabled(key: string, enabled: boolean): void {
			const normalized = key.trim();
			const index = this.items.findIndex((item) => item.key === normalized);
			if (index >= 0) {
				this.items[index] = { ...this.items[index], enabled };
			}
		}
	}
});

function normalizeQuery(query: PermissionListQuery): PermissionListQuery {
	const out: PermissionListQuery = {};
	if (query.type) {
		out.type = query.type;
	}
	if (query.source?.trim()) {
		out.source = query.source.trim();
	}
	if (typeof query.enabled === "boolean") {
		out.enabled = query.enabled;
	}
	if (typeof query.offset === "number" && Number.isFinite(query.offset)) {
		out.offset = Math.max(0, Math.trunc(query.offset));
	}
	if (typeof query.limit === "number" && Number.isFinite(query.limit)) {
		out.limit = Math.max(0, Math.trunc(query.limit));
	}
	return out;
}

function sameQuery(left: PermissionListQuery, right: PermissionListQuery): boolean {
	return left.type === right.type &&
		left.source === right.source &&
		left.enabled === right.enabled &&
		left.offset === right.offset &&
		left.limit === right.limit;
}
