import { defineStore } from "pinia";

import {
	getMenuTree,
	reorderMenuNodes,
	saveMenuNodes,
	setMenuVisibility,
	type MenuNodeRecord
} from "../navigation/api";

export type SystemMenuRecord = {
	id: string;
	label: string;
	path: string;
	parentKey?: string;
	source?: string;
	component?: string;
	icon?: string;
	order: number;
	visible: boolean;
	requiredRoles?: string[];
	requiredPermissions?: string[];
	children?: SystemMenuRecord[];
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
				const nodes = await getMenuTree();
				this.systemMenus = menuNodesToSystemMenus(nodes);
				this.customized = nodes.length > 0;
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
				const nodes = await saveMenuNodes(systemMenusToMenuNodes(items));
				this.systemMenus = menuNodesToSystemMenus(nodes);
				this.customized = nodes.length > 0;
				this.syncStatus = "success";
			} catch (error) {
				this.syncStatus = "error";
				this.lastError = error instanceof Error ? error.message : "menu_save_failed";
				throw error;
			}
		},
		async reorderSystemMenus(parentKey: string, orderedKeys: string[]): Promise<void> {
			this.syncStatus = "loading";
			this.lastError = null;
			try {
				await reorderMenuNodes({ parentKey, orderedKeys });
				await this.loadSystemMenus();
			} catch (error) {
				this.syncStatus = "error";
				this.lastError = error instanceof Error ? error.message : "menu_reorder_failed";
				throw error;
			}
		},
		async setSystemMenuVisibility(key: string, visible: boolean): Promise<void> {
			this.syncStatus = "loading";
			this.lastError = null;
			try {
				await setMenuVisibility({ key, visible });
				this.patchMenuVisibility(key, visible);
				this.syncStatus = "success";
			} catch (error) {
				this.syncStatus = "error";
				this.lastError = error instanceof Error ? error.message : "menu_visibility_failed";
				throw error;
			}
		},
		patchMenuVisibility(key: string, visible: boolean): void {
			const target = findSystemMenu(this.systemMenus, key);
			if (target) {
				target.visible = visible;
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
			parentKey: String(item.parentKey || "").trim() || undefined,
			source: String(item.source || "").trim() || "system",
			component: String(item.component || "").trim() || undefined,
			icon: String(item.icon || "").trim(),
			order: Number.isFinite(item.order) ? Number(item.order) : 0,
			visible: item.visible !== false,
			requiredRoles: normalizeStringList(item.requiredRoles),
			requiredPermissions: normalizeStringList(item.requiredPermissions),
			children: Array.isArray(item.children) ? normalizeSystemMenus(item.children) : []
		}))
		.filter((item) => item.id !== "" && item.label !== "" && item.path.startsWith("/"));
}

function menuNodesToSystemMenus(nodes: MenuNodeRecord[]): SystemMenuRecord[] {
	const records = nodes.map((node) => ({
		id: node.key,
		label: node.name,
		path: node.path,
		parentKey: node.parentKey,
		source: node.source,
		component: node.component,
		icon: node.icon,
		order: node.sort,
		visible: node.visible,
		requiredRoles: node.requiredRoles,
		requiredPermissions: node.requiredPermissions,
		children: []
	} satisfies SystemMenuRecord));
	return buildSystemMenuTree(normalizeSystemMenus(records));
}

function buildSystemMenuTree(items: SystemMenuRecord[]): SystemMenuRecord[] {
	const byID = new Map<string, SystemMenuRecord>();
	const roots: SystemMenuRecord[] = [];

	for (const item of items) {
		byID.set(item.id, { ...item, children: [] });
	}

	for (const item of byID.values()) {
		const parentKey = item.parentKey || "";
		const parent = parentKey ? byID.get(parentKey) : undefined;
		if (parent) {
			parent.children = [...(parent.children ?? []), item];
			continue;
		}
		roots.push(item);
	}

	return sortSystemMenus(roots);
}

function sortSystemMenus(items: SystemMenuRecord[]): SystemMenuRecord[] {
	return [...items]
		.sort((a, b) => a.order - b.order || a.label.localeCompare(b.label))
		.map((item) => ({
			...item,
			children: sortSystemMenus(item.children ?? [])
		}));
}

function systemMenusToMenuNodes(items: SystemMenuRecord[], parentKey = ""): MenuNodeRecord[] {
	const normalized = normalizeSystemMenus(items);
	const out: MenuNodeRecord[] = [];
	for (const item of normalized) {
		const key = item.id.trim();
		out.push({
			key,
			parentKey: parentKey || item.parentKey,
			source: item.source || "system",
			name: item.label,
			path: item.path,
			component: item.component,
			icon: item.icon,
			sort: item.order,
			visible: item.visible,
			requiredRoles: item.requiredRoles,
			requiredPermissions: item.requiredPermissions
		});
		out.push(...systemMenusToMenuNodes(item.children ?? [], key));
	}
	return out;
}

function findSystemMenu(items: SystemMenuRecord[], key: string): SystemMenuRecord | null {
	const normalizedKey = key.trim();
	for (const item of items) {
		if (item.id === normalizedKey) {
			return item;
		}
		const child = findSystemMenu(item.children ?? [], normalizedKey);
		if (child) {
			return child;
		}
	}
	return null;
}

function normalizeStringList(items?: string[]): string[] {
	return Array.isArray(items) ? items.map((item) => String(item || "").trim()).filter((item) => item !== "") : [];
}
