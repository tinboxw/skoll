import { defineStore } from "pinia";

import type { FrontendPluginManifest } from "../plugins/types";

const DEFAULT_HOME_KEY = "skoll.ui.defaultHome";
const SYSTEM_DEFAULT_HOME = "/skoll/dashboard";

export type DefaultHomeTarget = {
	level: "system" | "app";
	pluginId: string;
	appId?: string;
	path: string;
};

type PluginState = {
	items: FrontendPluginManifest[];
	backendRecords: FrontendPluginManifest[];
	syncedFromServer: boolean;
	syncStatus: "idle" | "loading" | "success" | "error";
	syncAttempts: number;
	lastSyncedAt: string | null;
	degradedMode: boolean;
	lastSyncError: string | null;
};

export const usePluginStore = defineStore("plugins", {
	state: (): PluginState => ({
		items: [],
		backendRecords: [],
		syncedFromServer: false,
		syncStatus: "idle",
		syncAttempts: 0,
		lastSyncedAt: null,
		degradedMode: false,
		lastSyncError: null
	}),
	getters: {
		enabledItems: (state): FrontendPluginManifest[] => state.items.filter((item) => item.enabled !== false),
		isSyncing: (state): boolean => state.syncStatus === "loading",
		hasSyncError: (state): boolean => state.syncStatus === "error",
		systemPlugins: (state): FrontendPluginManifest[] => state.items.filter((item) => item.level !== "app"),
		appPlugins: (state): FrontendPluginManifest[] => state.items.filter((item) => item.level === "app"),
		pluginsByAppId: (state) => (appId: string): FrontendPluginManifest[] => state.items.filter((item) => item.level === "app" && item.appId === appId)
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
		beginSync(): void {
			this.syncStatus = "loading";
			this.syncAttempts += 1;
			this.lastSyncError = null;
		},
		finishSync(error: string | null = null, degradedMode = false): void {
			this.syncedFromServer = true;
			this.lastSyncError = error;
			this.degradedMode = degradedMode;
			this.syncStatus = error ? "error" : "success";
			this.lastSyncedAt = new Date().toISOString();
		},
		markSynced(error: string | null = null): void {
			this.finishSync(error, Boolean(error));
		}
	}
});

export function resolvePluginEntryPath(plugin: FrontendPluginManifest): string {
	if (plugin.level === "app") {
		const appId = plugin.appId?.trim() ?? "";
		return appId === "" ? "" : `/${appId}`;
	}
	if (plugin.route && typeof plugin.route.path === "string" && plugin.route.path.trim().startsWith("/")) {
		return plugin.route.path.trim();
	}
	if (typeof plugin.entryPath === "string" && plugin.entryPath.trim().startsWith("/")) {
		return plugin.entryPath.trim();
	}
	return "";
}

function isValidDefaultHomePath(path: string): boolean {
	const value = path.trim();
	return value.startsWith("/") && value !== "/login" && value !== "/skoll/login";
}

function normalizePluginHomePath(path: string): string {
	const value = path.trim();
	if (value.startsWith("/plugins/")) {
		return `/skoll${value}`;
	}
	return value;
}

function normalizeDefaultHomeTarget(value: unknown): DefaultHomeTarget | null {
	if (!value || typeof value !== "object") {
		return null;
	}
	const candidate = value as Record<string, unknown>;
	const level = candidate.level === "app" ? "app" : candidate.level === "system" ? "system" : null;
	const pluginId = typeof candidate.pluginId === "string" ? candidate.pluginId.trim() : "";
	const path = typeof candidate.path === "string" ? normalizePluginHomePath(candidate.path) : "";
	const appId = typeof candidate.appId === "string" ? candidate.appId.trim() : "";
	if (!level || pluginId === "" || !isValidDefaultHomePath(path)) {
		return null;
	}
	if (level === "app") {
		if (appId === "") {
			return null;
		}
		return {
			level,
			pluginId,
			appId,
			path
		};
	}
	return {
		level,
		pluginId,
		path
	};
}

export function createDefaultHomeTarget(plugin: FrontendPluginManifest): DefaultHomeTarget | null {
	const path = plugin.level === "app"
		? `/${plugin.appId?.trim() || ""}`
		: resolvePluginEntryPath(plugin);
	if (!isValidDefaultHomePath(path) || plugin.enabled === false || plugin.uiMode === "backend_only") {
		return null;
	}
	if (plugin.level === "app") {
		const appId = plugin.appId?.trim() ?? "";
		if (appId === "") {
			return null;
		}
		return {
			level: "app",
			pluginId: plugin.id,
			appId,
			path
		};
	}
	return {
		level: "system",
		pluginId: plugin.id,
		path
	};
}

export function getDefaultHomeTarget(): DefaultHomeTarget | null {
	const raw = localStorage.getItem(DEFAULT_HOME_KEY);
	if (typeof raw !== "string" || raw.trim() === "") {
		return null;
	}
	const value = raw.trim();
	if (value.startsWith("{")) {
		try {
			return normalizeDefaultHomeTarget(JSON.parse(value));
		} catch {
			return null;
		}
	}
	return null;
}

export function setDefaultHomeTarget(target: DefaultHomeTarget): void {
	const normalized = normalizeDefaultHomeTarget(target);
	if (!normalized) {
		return;
	}
	localStorage.setItem(DEFAULT_HOME_KEY, JSON.stringify(normalized));
}

export function isDefaultHomePlugin(plugin: FrontendPluginManifest): boolean {
	const target = getDefaultHomeTarget();
	if (!target) {
		return false;
	}
	if (target.pluginId !== plugin.id) {
		return false;
	}
	if (target.level === "app") {
		return plugin.level === "app" && plugin.appId === target.appId;
	}
	return plugin.level !== "app";
}

export function resolveValidatedDefaultHomePath(items: FrontendPluginManifest[], fallback = SYSTEM_DEFAULT_HOME): string {
	const target = getDefaultHomeTarget();
	if (!target) {
		return fallback;
	}
	const plugin = items.find((item) => item.id === target.pluginId);
	if (!plugin || plugin.enabled === false) {
		return fallback;
	}
	const entryPath = normalizePluginHomePath(resolvePluginEntryPath(plugin));
	if (!isValidDefaultHomePath(entryPath) || entryPath !== normalizePluginHomePath(target.path)) {
		return fallback;
	}
	if (target.level === "app") {
		if (plugin.level !== "app" || plugin.appId !== target.appId) {
			return fallback;
		}
		return entryPath;
	}
	if (plugin.level === "app") {
		return fallback;
	}
	return entryPath;
}

export function getDefaultHomePath(fallback = "/skoll/dashboard"): string {
	return getDefaultHomeTarget()?.path ?? fallback;
}

export function getSystemDefaultHomePath(): string {
	return SYSTEM_DEFAULT_HOME;
}

export function clearDefaultHomePath(): void {
	localStorage.removeItem(DEFAULT_HOME_KEY);
}

export function isDefaultHomePath(path: string, fallback = SYSTEM_DEFAULT_HOME): boolean {
	const value = path.trim();
	if (!isValidDefaultHomePath(value)) {
		return false;
	}
	return getDefaultHomePath(fallback) === value;
}

