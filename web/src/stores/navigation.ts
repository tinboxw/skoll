import { defineStore } from "pinia";

import { apiGet, apiPut, type ApiResponse } from "../utils/api";

export type SystemMenuRecord = {
	id: string;
	label: string;
	path: string;
	icon?: string;
	order: number;
	visible: boolean;
	requiredRoles?: string[];
	requiredPermissions?: string[];
	children?: SystemMenuRecord[];
};

type SystemMenuPayload = {
	items?: SystemMenuRecord[];
	customized?: boolean;
};

type NavigationState = {
	systemMenus: SystemMenuRecord[];
	customized: boolean;
	syncStatus: "idle" | "loading" | "success" | "error";
	lastError: string | null;
};

export const useNavigationStore = defineStore("navigation", {
	state: (): NavigationState => ({
		systemMenus: [],
		customized: false,
		syncStatus: "idle",
		lastError: null
	}),
	getters: {
		hasRemoteMenus: (state): boolean => state.systemMenus.length > 0
	},
	actions: {
		async loadSystemMenus(): Promise<void> {
			this.syncStatus = "loading";
			this.lastError = null;
			try {
				const payload = await apiGet<ApiResponse<SystemMenuPayload>>("/v1/system/menus");
				this.systemMenus = normalizeSystemMenus(payload.data?.items ?? []);
				this.customized = Boolean(payload.data?.customized);
				this.syncStatus = "success";
			} catch (error) {
				this.systemMenus = [];
				this.customized = false;
				this.syncStatus = "error";
				this.lastError = error instanceof Error ? error.message : "menu_sync_failed";
			}
		},
		async saveSystemMenus(items: SystemMenuRecord[]): Promise<void> {
			this.syncStatus = "loading";
			this.lastError = null;
			try {
				const payload = await apiPut<ApiResponse<SystemMenuPayload>>("/v1/system/menus", {
					items: normalizeSystemMenus(items)
				});
				this.systemMenus = normalizeSystemMenus(payload.data?.items ?? []);
				this.customized = Boolean(payload.data?.customized);
				this.syncStatus = "success";
			} catch (error) {
				this.syncStatus = "error";
				this.lastError = error instanceof Error ? error.message : "menu_save_failed";
				throw error;
			}
		}
	}
});

function normalizeSystemMenus(items: SystemMenuRecord[]): SystemMenuRecord[] {
	return items
		.map((item) => ({
			id: String(item.id || "").trim(),
			label: String(item.label || "").trim(),
			path: String(item.path || "").trim(),
			icon: String(item.icon || "").trim(),
			order: Number.isFinite(item.order) ? Number(item.order) : 0,
			visible: item.visible !== false,
			requiredRoles: normalizeStringList(item.requiredRoles),
			requiredPermissions: normalizeStringList(item.requiredPermissions),
			children: Array.isArray(item.children) ? normalizeSystemMenus(item.children) : []
		}))
		.filter((item) => item.id !== "" && item.label !== "" && item.path.startsWith("/"));
}

function normalizeStringList(items?: string[]): string[] {
	return Array.isArray(items) ? items.map((item) => String(item || "").trim()).filter((item) => item !== "") : [];
}
